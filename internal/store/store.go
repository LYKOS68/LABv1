package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{Pool: pool} }

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email taken")
	ErrNameTaken          = errors.New("name taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotVerified        = errors.New("email not verified")
)

// CreateUser legt Nutzer, Profil und Bestätigungs-Token in einer Transaktion an (TECH-STACK §4(2)),
// damit es nie einen Nutzer ohne Profil oder ohne Bestätigungsweg gibt.
func (s *Store) CreateUser(ctx context.Context, email, name, password, tokenHash string, tokenTTL time.Duration) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin create user: %w", err)
	}

	var id int
	err = tx.QueryRow(ctx, "INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id", name, email, string(hash)).Scan(&id)
	if err != nil {
		return 0, rollback(ctx, tx, uniqueUserError(err))
	}
	if _, err := tx.Exec(ctx, "INSERT INTO profiles (user_id, display_name) VALUES ($1, $2)", id, name); err != nil {
		return 0, rollback(ctx, tx, fmt.Errorf("insert profile: %w", err))
	}
	if _, err := tx.Exec(ctx, "INSERT INTO email_verifications (token_hash, user_id, expires_at) VALUES ($1, $2, $3)", tokenHash, id, time.Now().Add(tokenTTL)); err != nil {
		return 0, rollback(ctx, tx, fmt.Errorf("insert verification: %w", err))
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit create user: %w", err)
	}
	return id, nil
}

// AuthUser prüft das Passwort vor dem Bestätigungsstatus, damit Fremde nicht erfahren,
// ob eine E-Mail registriert, aber unbestätigt ist.
func (s *Store) AuthUser(ctx context.Context, email, password string) (int, error) {
	var id int
	var hash string
	var verified bool
	err := s.Pool.QueryRow(ctx, "SELECT id, password_hash, email_verified_at IS NOT NULL FROM users WHERE lower(email)=lower($1)", email).Scan(&id, &hash, &verified)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInvalidCredentials
	}
	if err != nil {
		return 0, fmt.Errorf("select user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return 0, ErrInvalidCredentials
	}
	if !verified {
		return 0, ErrNotVerified
	}
	return id, nil
}

// VerifyEmail löst den Token in einem Statement ein: Token löschen und Nutzer bestätigen
// passieren gemeinsam oder gar nicht.
func (s *Store) VerifyEmail(ctx context.Context, tokenHash string) error {
	var id int
	err := s.Pool.QueryRow(ctx, `WITH t AS (
		DELETE FROM email_verifications WHERE token_hash=$1 AND expires_at > now() RETURNING user_id
	)
	UPDATE users SET email_verified_at = COALESCE(email_verified_at, now())
	FROM t WHERE users.id = t.user_id RETURNING users.id`, tokenHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("verify email: %w", err)
	}
	return nil
}

func (s *Store) UserName(ctx context.Context, userID int) (string, error) {
	var name string
	err := s.Pool.QueryRow(ctx, "SELECT username FROM users WHERE id=$1", userID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("select user name: %w", err)
	}
	return name, nil
}

// uniqueUserError übersetzt Unique-Verletzungen in fachliche Fehler, damit der Handler
// dem Nutzer sagen kann, welches Feld schon vergeben ist.
func uniqueUserError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		taken := map[string]error{
			"users_email_key":    ErrEmailTaken,
			"users_username_key": ErrNameTaken,
		}
		if e, ok := taken[pgErr.ConstraintName]; ok {
			return e
		}
	}
	return fmt.Errorf("insert user: %w", err)
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
