// Package database provides database connection management.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // SQLite driver

	"github.com/2bit-software/zombiekit/internal/config"
)

// SQLiteDB wraps a sql.DB for SQLite connection management.
type SQLiteDB struct {
	db *sql.DB
}

// NewSQLiteDB creates a new SQLite database connection.
// It ensures the parent directory exists and enables WAL mode.
func NewSQLiteDB(ctx context.Context, cfg config.StorageConfig) (*SQLiteDB, error) {
	dbPath := config.ExpandPath(cfg.SQLitePath)

	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	// Open database with SQLite driver
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	// Verify connectivity
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	return &SQLiteDB{db: db}, nil
}

// DB returns the underlying sql.DB.
func (s *SQLiteDB) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
