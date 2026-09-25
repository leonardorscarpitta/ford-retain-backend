// Package postgres opens the database and applies each module's migrations.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

// Open connects to Postgres and checks the connection.
func Open(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxIdleTime(5 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	return db, nil
}

// Migrate applies a module's *.sql files, in name order, that haven't run
// yet. Each file runs in its own transaction and is recorded in
// schema_migrations, so running Migrate again is a no-op.
func Migrate(ctx context.Context, db *sql.DB, module string, files fs.FS) error {
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("no *.sql migrations found for module %q", module)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		module     text NOT NULL,
		version    text NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now(),
		PRIMARY KEY (module, version)
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		if err := apply(ctx, db, module, version, files, name); err != nil {
			return fmt.Errorf("migration %s/%s: %w", module, name, err)
		}
	}
	return nil
}

func apply(ctx context.Context, db *sql.DB, module, version string, files fs.FS, name string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serialise concurrent starts of the API so a migration only runs once.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('schema_migrations', 0))`); err != nil {
		return err
	}
	var done bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE module = $1 AND version = $2)`, module, version,
	).Scan(&done); err != nil {
		return err
	}
	if done {
		return nil
	}
	body, err := fs.ReadFile(files, name)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (module, version) VALUES ($1, $2)`, module, version); err != nil {
		return err
	}
	return tx.Commit()
}
