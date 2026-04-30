// Package database provides database connection and migration management.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/2bit-software/zombiekit/internal/config"
)

//go:embed migrations/postgres/*.sql
var postgresMigrationsFS embed.FS

//go:embed migrations/sqlite/*.sql
var sqliteMigrationsFS embed.FS

// MigrationStatus represents the status of a single migration.
type MigrationStatus struct {
	Version   int       `json:"version"`
	Name      string    `json:"name"`
	Applied   bool      `json:"applied"`
	AppliedAt time.Time `json:"applied_at,omitempty"`
}

// migrationRunner abstracts the database-specific operations needed to run migrations.
type migrationRunner struct {
	fs        embed.FS
	dir       string
	isApplied func(version int) (bool, error)
	readSQL   func(filename string) ([]byte, error)
	apply     func(version int, name string, sqlBytes []byte) error
}

// runPendingMigrations iterates over migration files and applies any that haven't been run.
func runPendingMigrations(runner migrationRunner) error {
	entries, err := runner.fs.ReadDir(runner.dir)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, name := parseMigrationFilename(entry.Name())
		if version == 0 {
			continue
		}

		exists, err := runner.isApplied(version)
		if err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if exists {
			continue
		}

		sqlBytes, err := runner.readSQL(entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		if err := runner.apply(version, name, sqlBytes); err != nil {
			return err
		}
	}

	return nil
}

// RunPostgresMigrations runs all pending PostgreSQL migrations.
func RunPostgresMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	return runPendingMigrations(migrationRunner{
		fs:  postgresMigrationsFS,
		dir: "migrations/postgres",
		isApplied: func(version int) (bool, error) {
			var exists bool
			err := pool.QueryRow(ctx,
				"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
				version,
			).Scan(&exists)
			return exists, err
		},
		readSQL: func(filename string) ([]byte, error) {
			return postgresMigrationsFS.ReadFile("migrations/postgres/" + filename)
		},
		apply: func(version int, name string, sqlBytes []byte) error {
			tx, err := pool.Begin(ctx)
			if err != nil {
				return fmt.Errorf("begin transaction for migration %d: %w", version, err)
			}
			defer tx.Rollback(ctx)

			if _, err = tx.Exec(ctx, string(sqlBytes)); err != nil {
				return fmt.Errorf("apply migration %d: %w", version, err)
			}
			if _, err = tx.Exec(ctx,
				"INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, NOW())",
				version, name,
			); err != nil {
				return fmt.Errorf("record migration %d: %w", version, err)
			}
			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("commit migration %d: %w", version, err)
			}
			return nil
		},
	})
}

// RunSQLiteMigrations runs all pending SQLite migrations.
func RunSQLiteMigrations(ctx context.Context, db *sql.DB) error {
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

	return runPendingMigrations(migrationRunner{
		fs:  sqliteMigrationsFS,
		dir: "migrations/sqlite",
		isApplied: func(version int) (bool, error) {
			var exists bool
			err := db.QueryRowContext(ctx,
				"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)",
				version,
			).Scan(&exists)
			return exists, err
		},
		readSQL: func(filename string) ([]byte, error) {
			return sqliteMigrationsFS.ReadFile("migrations/sqlite/" + filename)
		},
		apply: func(version int, name string, sqlBytes []byte) error {
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
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("commit migration %d: %w", version, err)
			}
			return nil
		},
	})
}

// GetPostgresMigrationStatus returns the status of all PostgreSQL migrations.
func GetPostgresMigrationStatus(ctx context.Context, pool *pgxpool.Pool) ([]MigrationStatus, error) {
	return getMigrationStatus(ctx, "postgres", func(query string, version int) (bool, time.Time, error) {
		var appliedAt time.Time
		err := pool.QueryRow(ctx, query, version).Scan(&appliedAt)
		if err == sql.ErrNoRows {
			return false, time.Time{}, nil
		}
		if err != nil {
			return false, time.Time{}, err
		}
		return true, appliedAt, nil
	})
}

// GetSQLiteMigrationStatus returns the status of all SQLite migrations.
func GetSQLiteMigrationStatus(ctx context.Context, db *sql.DB) ([]MigrationStatus, error) {
	// Check if schema_migrations table exists
	var tableExists int
	err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'",
	).Scan(&tableExists)
	if err != nil {
		return nil, fmt.Errorf("check schema_migrations table: %w", err)
	}

	// If table doesn't exist, all migrations are pending
	if tableExists == 0 {
		return getMigrationStatus(ctx, "sqlite", func(query string, version int) (bool, time.Time, error) {
			return false, time.Time{}, nil // All pending
		})
	}

	return getMigrationStatus(ctx, "sqlite", func(query string, version int) (bool, time.Time, error) {
		var appliedAt time.Time
		err := db.QueryRowContext(ctx, strings.ReplaceAll(query, "$1", "?"), version).Scan(&appliedAt)
		if err == sql.ErrNoRows {
			return false, time.Time{}, nil
		}
		if err != nil {
			return false, time.Time{}, err
		}
		return true, appliedAt, nil
	})
}

func getMigrationStatus(ctx context.Context, backend string, checkApplied func(query string, version int) (bool, time.Time, error)) ([]MigrationStatus, error) {
	var fs embed.FS
	var dir string
	if backend == "postgres" {
		fs = postgresMigrationsFS
		dir = "migrations/postgres"
	} else {
		fs = sqliteMigrationsFS
		dir = "migrations/sqlite"
	}

	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	var statuses []MigrationStatus
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, name := parseMigrationFilename(entry.Name())
		if version == 0 {
			continue
		}

		applied, appliedAt, err := checkApplied(
			"SELECT applied_at FROM schema_migrations WHERE version = $1",
			version,
		)
		if err != nil {
			return nil, fmt.Errorf("check migration %d: %w", version, err)
		}

		statuses = append(statuses, MigrationStatus{
			Version:   version,
			Name:      name,
			Applied:   applied,
			AppliedAt: appliedAt,
		})
	}

	// Sort by version
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Version < statuses[j].Version
	})

	return statuses, nil
}

// parseMigrationFilename extracts version and name from a migration filename.
// Expected format: 001_name.sql
func parseMigrationFilename(filename string) (int, string) {
	// Remove .sql extension
	name := strings.TrimSuffix(filename, ".sql")

	// Split by underscore
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

// RunMigrations runs migrations for the specified backend.
func RunMigrations(ctx context.Context, cfg config.StorageConfig) error {
	switch cfg.Backend {
	case config.BackendPostgres:
		pool, err := NewPostgresPool(ctx, cfg)
		if err != nil {
			return err
		}
		defer pool.Close()
		return RunPostgresMigrations(ctx, pool.Pool())

	case config.BackendSQLite:
		db, err := NewSQLiteDB(ctx, cfg)
		if err != nil {
			return err
		}
		defer db.Close()
		return RunSQLiteMigrations(ctx, db.DB())

	default:
		return fmt.Errorf("unknown backend: %s", cfg.Backend)
	}
}

// GetMigrationStatus returns the status of all migrations for the specified backend.
func GetMigrationStatus(ctx context.Context, cfg config.StorageConfig) ([]MigrationStatus, error) {
	switch cfg.Backend {
	case config.BackendPostgres:
		pool, err := NewPostgresPool(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer pool.Close()
		return GetPostgresMigrationStatus(ctx, pool.Pool())

	case config.BackendSQLite:
		db, err := NewSQLiteDB(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return GetSQLiteMigrationStatus(ctx, db.DB())

	default:
		return nil, fmt.Errorf("unknown backend: %s", cfg.Backend)
	}
}
