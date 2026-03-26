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

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Sort entries by name to ensure ordered application
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

		var exists bool
		err := db.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)",
			version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}

		if exists {
			continue
		}

		sqlBytes, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin transaction for migration %d: %w", version, err)
		}
		defer tx.Rollback()

		_, err = tx.ExecContext(ctx, string(sqlBytes))
		if err != nil {
			return fmt.Errorf("apply migration %d: %w", version, err)
		}

		_, err = tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)",
			version, name, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("record migration %d: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}

	return nil
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
