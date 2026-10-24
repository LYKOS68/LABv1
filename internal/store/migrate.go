package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5"
)

// Migrate spielt alle noch nicht eingetragenen *.sql-Dateien in Namensreihenfolge ein.
// Jede Datei läuft in einer eigenen Transaktion, damit ein Fehler keine halbe Migration hinterlässt.
func (s *Store) Migrate(ctx context.Context, fsys fs.FS) error {
	if _, err := s.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(names)

	for _, name := range names {
		if err := s.applyMigration(ctx, fsys, name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) applyMigration(ctx context.Context, fsys fs.FS, name string) error {
	sqlBytes, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}

	var applied bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version=$1)", name).Scan(&applied); err != nil {
		return rollback(ctx, tx, fmt.Errorf("check migration %s: %w", name, err))
	}
	if applied {
		return rollback(ctx, tx, nil)
	}

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return rollback(ctx, tx, fmt.Errorf("apply migration %s: %w", name, err))
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", name); err != nil {
		return rollback(ctx, tx, fmt.Errorf("record migration %s: %w", name, err))
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}

// rollback bricht die Transaktion ab, ohne den Rollback-Fehler zu verschlucken (TECH-STACK §5(1)).
func rollback(ctx context.Context, tx pgx.Tx, err error) error {
	if rbErr := tx.Rollback(ctx); rbErr != nil {
		return errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
	}
	return err
}
