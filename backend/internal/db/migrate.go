package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StampMigrations marks all migration files as applied without executing them.
// Use this once on an existing DB that was set up outside of RunMigrations.
func StampMigrations(ctx context.Context, connString, migrationsDir string) error {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return fmt.Errorf("stamp: connect: %w", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     VARCHAR(255) PRIMARY KEY,
			applied_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("stamp: create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("stamp: read dir %q: %w", migrationsDir, err)
	}

	stamped := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		_, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`,
			e.Name(),
		)
		if err != nil {
			return fmt.Errorf("stamp: insert %s: %w", e.Name(), err)
		}
		stamped++
	}
	log.Printf("stamp: marked %d migration(s) as applied (no SQL executed)", stamped)
	return nil
}

// RunMigrations applies any unapplied *.sql files in migrationsDir, in filename order.
// Applied migrations are tracked in the schema_migrations table (idempotent).
func RunMigrations(ctx context.Context, connString, migrationsDir string) error {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return fmt.Errorf("migrate: connect: %w", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     VARCHAR(255) PRIMARY KEY,
			applied_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("migrate: create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("migrate: read dir %q: %w", migrationsDir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	applied := map[string]bool{}
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("migrate: list applied: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return fmt.Errorf("migrate: scan applied: %w", err)
		}
		applied[v] = true
	}
	rows.Close()

	pending := 0
	for _, name := range files {
		if applied[name] {
			continue
		}
		pending++

		sql, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", name, err)
		}

		if err := runInTx(ctx, pool, name, string(sql)); err != nil {
			return err
		}
		log.Printf("migrate: applied %s", name)
	}

	if pending == 0 {
		log.Printf("migrate: all %d migrations already applied", len(files))
	} else {
		log.Printf("migrate: applied %d new migration(s)", pending)
	}
	return nil
}

func runInTx(ctx context.Context, pool *pgxpool.Pool, name, sql string) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("migrate: begin tx for %s: %w", name, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("migrate: exec %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("migrate: record %s: %w", name, err)
	}
	return tx.Commit(ctx)
}
