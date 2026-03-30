// Package sqlite provides SQLite-based memory storage.
// This implementation is compatible with mcp-genie's stickymemory patterns.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // SQLite driver

	"github.com/2bit-software/zombiekit/internal/memory"
	"github.com/2bit-software/zombiekit/internal/mo"
)

// SQLiteStorage implements the memory.Storage interface using SQLite.
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite-based storage.
// It initializes the database schema if it doesn't exist.
func NewSQLiteStorage(ctx context.Context, dbPath string) (*SQLiteStorage, error) {
	// Expand home directory if needed
	dbPath = expandPath(dbPath)

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

	storage := &SQLiteStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the memories table if it doesn't exist.
func (s *SQLiteStorage) initSchema(ctx context.Context) error {
	// Enable WAL mode for better concurrent access
	if _, err := s.db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("enable WAL mode: %w", err)
	}

	// Create memories table (mcp-genie compatible schema)
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS memories (
			name TEXT NOT NULL,
			version INTEGER NOT NULL,
			content TEXT NOT NULL,
			deleted BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (name, version)
		)
	`)
	if err != nil {
		return fmt.Errorf("create memories table: %w", err)
	}

	return nil
}

// Set stores a memory item, creating a new version.
func (s *SQLiteStorage) Set(ctx context.Context, name, content string) error {
	name = memory.SanitizeName(name)
	now := time.Now()

	// Transaction for atomic version generation
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get next version number
	var nextVersion int
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM memories
		WHERE name = ?
	`, name).Scan(&nextVersion)
	if err != nil {
		return fmt.Errorf("get next version: %w", err)
	}

	// Insert new version
	_, err = tx.ExecContext(ctx, `
		INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
		VALUES (?, ?, ?, FALSE, ?, ?)
	`, name, nextVersion, content, now, now)
	if err != nil {
		return fmt.Errorf("insert memory: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Get retrieves the latest non-deleted version of a memory item.
func (s *SQLiteStorage) Get(ctx context.Context, name string) (mo.Maybe[memory.MemoryItem], error) {
	name = memory.SanitizeName(name)

	query := `
		SELECT name, version, content, deleted, created_at, updated_at
		FROM memories
		WHERE name = ? AND deleted = FALSE
		ORDER BY version DESC
		LIMIT 1
	`

	var item memory.MemoryItem
	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&item.Name, &item.Version, &item.Content,
		&item.Deleted, &item.CreatedAt, &item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return mo.Nothing[memory.MemoryItem](), nil
	}
	if err != nil {
		return mo.Nothing[memory.MemoryItem](), fmt.Errorf("query memory: %w", err)
	}

	return mo.Just(item), nil
}

// Delete soft-deletes all versions of a memory item.
func (s *SQLiteStorage) Delete(ctx context.Context, name string) error {
	name = memory.SanitizeName(name)

	_, err := s.db.ExecContext(ctx, `
		UPDATE memories
		SET deleted = TRUE, updated_at = ?
		WHERE name = ? AND deleted = FALSE
	`, time.Now(), name)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}

	return nil
}

// List returns all items, optionally filtered by search query.
func (s *SQLiteStorage) List(ctx context.Context, search string) ([]memory.MemoryMetadata, error) {
	var query string
	var args []interface{}

	if search == "" {
		query = `
			SELECT name, version, length(content) as size, created_at, updated_at
			FROM memories m1
			WHERE deleted = FALSE
			AND version = (
				SELECT MAX(version)
				FROM memories m2
				WHERE m2.name = m1.name AND m2.deleted = FALSE
			)
			ORDER BY updated_at DESC
		`
	} else {
		searchParam := "%" + search + "%"
		query = `
			SELECT name, version, length(content) as size, created_at, updated_at
			FROM memories m1
			WHERE deleted = FALSE
			AND version = (
				SELECT MAX(version)
				FROM memories m2
				WHERE m2.name = m1.name AND m2.deleted = FALSE
			)
			AND (LOWER(name) LIKE LOWER(?) OR LOWER(content) LIKE LOWER(?))
			ORDER BY updated_at DESC
		`
		args = []interface{}{searchParam, searchParam}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query memories: %w", err)
	}
	defer rows.Close()

	var items []memory.MemoryMetadata
	for rows.Next() {
		var item memory.MemoryMetadata
		if err := rows.Scan(&item.Name, &item.Version, &item.Size, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memories: %w", err)
	}

	return items, nil
}

// Clear soft-deletes all items and returns the count of distinct names deleted.
func (s *SQLiteStorage) Clear(ctx context.Context) (int, error) {
	// Count distinct names first
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT name) FROM memories WHERE deleted = FALSE",
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count memories: %w", err)
	}

	// Soft delete all
	_, err = s.db.ExecContext(ctx, `
		UPDATE memories
		SET deleted = TRUE, updated_at = ?
		WHERE deleted = FALSE
	`, time.Now())
	if err != nil {
		return 0, fmt.Errorf("clear memories: %w", err)
	}

	return count, nil
}

// Close closes the database connection.
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// expandPath expands ~ in the path to the user's home directory.
func expandPath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
