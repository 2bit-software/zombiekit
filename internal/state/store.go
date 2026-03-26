// Package state provides persistent state management for the orchestrator daemon.
// It owns the StateStore interface and its SQLite implementation.
package state

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// StateStore defines the interface for orchestrator persistent state.
// DEV-153 scope: initialization and lifecycle only.
// DEV-154 will add CRUD methods for jobs, watermarks, and slots.
type StateStore interface {
	Migrate(ctx context.Context) error
	Close() error
}

// SQLiteStore implements StateStore backed by a local SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed state store.
// It creates parent directories, opens the connection with
// appropriate pragmas, and runs any pending migrations.
func NewSQLiteStore(ctx context.Context, dbPath string) (*SQLiteStore, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("open state store: %w", ErrInvalidDBPath)
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create state directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open state store: %w", err)
	}

	db.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("set %s: %w", pragma, err)
		}
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("state store connection failed: %w", err)
	}

	if err := RunMigrations(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run state migrations: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// DB returns the underlying *sql.DB for use by CRUD operations (DEV-154).
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// Migrate runs any pending database migrations.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	return RunMigrations(ctx, s.db)
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
