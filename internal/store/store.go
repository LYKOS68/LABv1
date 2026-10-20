package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{Pool: pool} }

var ErrNotFound = errors.New("not found")

func (s *Store) CreateUser(ctx context.Context, username, password string) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	var id int
	err = s.Pool.QueryRow(ctx, "INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id", username, string(hash)).Scan(&id)
	if err != nil {
		return 0, err
	}
	_, _ = s.Pool.Exec(ctx, "INSERT INTO profiles (user_id) VALUES ($1)", id)
	return id, nil
}

func (s *Store) AuthUser(ctx context.Context, username, password string) (int, error) {
	var id int
	var hash string
	err := s.Pool.QueryRow(ctx, "SELECT id, password_hash FROM users WHERE username=$1", username).Scan(&id, &hash)
	if err != nil {
		return 0, ErrNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return 0, errors.New("invalid credentials")
	}
	return id, nil
}

func (s *Store) CreateSession(ctx context.Context, userID int, ttl time.Duration) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	_, err := s.Pool.Exec(ctx, "INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)", token, userID, time.Now().Add(ttl))
	return token, err
}

func (s *Store) SessionUser(ctx context.Context, token string) (int, error) {
	var userID int
	var expires time.Time
	err := s.Pool.QueryRow(ctx, "SELECT user_id, expires_at FROM sessions WHERE token=$1", token).Scan(&userID, &expires)
	if err != nil {
		return 0, ErrNotFound
	}
	if time.Now().After(expires) {
		_, _ = s.Pool.Exec(ctx, "DELETE FROM sessions WHERE token=$1", token)
		return 0, ErrNotFound
	}
	return userID, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.Pool.Exec(ctx, "DELETE FROM sessions WHERE token=$1", token)
	return err
}

func (s *Store) GetProfile(ctx context.Context, userID int) (string, string, error) {
	var name, bio *string
	err := s.Pool.QueryRow(ctx, "SELECT display_name, bio FROM profiles WHERE user_id=$1", userID).Scan(&name, &bio)
	if err != nil {
		return "", "", ErrNotFound
	}
	dn, b := "", ""
	if name != nil {
		dn = *name
	}
	if bio != nil {
		b = *bio
	}
	return dn, b, nil
}

func (s *Store) ListOfferings(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.Pool.Query(ctx, "SELECT id, user_id, title, description, price_cents FROM offerings ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, uid, price int
		var title string
		var desc *string
		if err := rows.Scan(&id, &uid, &title, &desc, &price); err != nil {
			return nil, err
		}
		d := ""
		if desc != nil {
			d = *desc
		}
		out = append(out, map[string]any{"id": id, "user_id": uid, "title": title, "description": d, "price_cents": price})
	}
	return out, rows.Err()
}
