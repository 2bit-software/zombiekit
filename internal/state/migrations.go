package state

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations runs all pending migrations against the state database.
func RunMigrations(ctx context.Context, db *sql.DB) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	if err := ensureMigrationsTable(ctx, db); err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, name := parseMigrationFilename(entry.Name())
		if version == 0 {
			continue
		}

		if err := applyMigration(ctx, db, entry.Name(), version, name); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, filename string, version int, name string) error {
	var exists bool
	err := db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)",
		version,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check migration %d: %w", version, err)
	}
	if exists {
		return nil
	}

	sqlBytes, err := migrationsFS.ReadFile("migrations/" + filename)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", filename, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction for migration %d: %w", version, err)
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("apply migration %d: %w", version, err)
	}

	if _, err = tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)",
		version, name, time.Now(),
	); err != nil {
		return fmt.Errorf("record migration %d: %w", version, err)
	}

	return tx.Commit()
}

func parseMigrationFilename(filename string) (int, string) {
	name := strings.TrimSuffix(filename, ".sql")

	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return 0, ""
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, ""
	}

	return version, parts[1]
}
