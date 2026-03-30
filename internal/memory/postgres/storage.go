// Package postgres provides PostgreSQL-based memory storage.
// This implementation is compatible with mcp-genie's stickymemory patterns.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/2bit-software/zombiekit/internal/memory"
	"github.com/2bit-software/zombiekit/internal/mo"
)

// PostgresStorage implements the memory.Storage interface using PostgreSQL.
type PostgresStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresStorage creates a new PostgreSQL-based storage.
// It initializes the database schema if it doesn't exist.
func NewPostgresStorage(ctx context.Context, pool *pgxpool.Pool) (*PostgresStorage, error) {
	storage := &PostgresStorage{pool: pool}

	// Initialize schema
	if err := storage.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the memories table if it doesn't exist.
func (s *PostgresStorage) initSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS memories (
			name TEXT NOT NULL,
			version INTEGER NOT NULL,
			content TEXT NOT NULL,
			deleted BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (name, version)
		)
	`)
	if err != nil {
		return fmt.Errorf("create memories table: %w", err)
	}

	// Create index for finding latest version efficiently
	_, err = s.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_memories_name_latest
		ON memories (name, version DESC)
		WHERE deleted = FALSE
	`)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	return nil
}

// Set stores a memory item, creating a new version.
func (s *PostgresStorage) Set(ctx context.Context, name, content string) error {
	name = memory.SanitizeName(name)

	// Transaction for atomic version generation
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get next version number
	var nextVersion int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM memories
		WHERE name = $1
	`, name).Scan(&nextVersion)
	if err != nil {
		return fmt.Errorf("get next version: %w", err)
	}

	// Insert new version
	_, err = tx.Exec(ctx, `
		INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, FALSE, NOW(), NOW())
	`, name, nextVersion, content)
	if err != nil {
		return fmt.Errorf("insert memory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Get retrieves the latest non-deleted version of a memory item.
func (s *PostgresStorage) Get(ctx context.Context, name string) (mo.Maybe[memory.MemoryItem], error) {
	name = memory.SanitizeName(name)

	query := `
		SELECT name, version, content, deleted, created_at, updated_at
		FROM memories
		WHERE name = $1 AND deleted = FALSE
		ORDER BY version DESC
		LIMIT 1
	`

	var item memory.MemoryItem
	err := s.pool.QueryRow(ctx, query, name).Scan(
		&item.Name, &item.Version, &item.Content,
		&item.Deleted, &item.CreatedAt, &item.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return mo.Nothing[memory.MemoryItem](), nil
	}
	if err != nil {
		return mo.Nothing[memory.MemoryItem](), fmt.Errorf("query memory: %w", err)
	}

	return mo.Just(item), nil
}

// Delete soft-deletes all versions of a memory item.
func (s *PostgresStorage) Delete(ctx context.Context, name string) error {
	name = memory.SanitizeName(name)

	_, err := s.pool.Exec(ctx, `
		UPDATE memories
		SET deleted = TRUE, updated_at = NOW()
		WHERE name = $1 AND deleted = FALSE
	`, name)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}

	return nil
}

// List returns all items, optionally filtered by search query.
func (s *PostgresStorage) List(ctx context.Context, search string) ([]memory.MemoryMetadata, error) {
	var query string
	var args []interface{}

	if search == "" {
		query = `
			SELECT DISTINCT ON (name) name, version, LENGTH(content) as size, created_at, updated_at
			FROM memories
			WHERE deleted = FALSE
			ORDER BY name, version DESC
		`
	} else {
		searchParam := "%" + search + "%"
		query = `
			SELECT DISTINCT ON (name) name, version, LENGTH(content) as size, created_at, updated_at
			FROM memories
			WHERE deleted = FALSE
			AND (name ILIKE $1 OR content ILIKE $1)
			ORDER BY name, version DESC
		`
		args = []interface{}{searchParam}
	}

	rows, err := s.pool.Query(ctx, query, args...)
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

	// Sort by updated_at DESC (DISTINCT ON requires ordering by name first, so we sort in Go)
	sortByUpdatedAtDesc(items)

	return items, nil
}

// sortByUpdatedAtDesc sorts items by UpdatedAt in descending order.
func sortByUpdatedAtDesc(items []memory.MemoryMetadata) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].UpdatedAt.After(items[i].UpdatedAt) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// Clear soft-deletes all items and returns the count of distinct names deleted.
func (s *PostgresStorage) Clear(ctx context.Context) (int, error) {
	// Count distinct names first
	var count int
	err := s.pool.QueryRow(ctx,
		"SELECT COUNT(DISTINCT name) FROM memories WHERE deleted = FALSE",
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count memories: %w", err)
	}

	// Soft delete all
	_, err = s.pool.Exec(ctx, `
		UPDATE memories
		SET deleted = TRUE, updated_at = NOW()
		WHERE deleted = FALSE
	`)
	if err != nil {
		return 0, fmt.Errorf("clear memories: %w", err)
	}

	return count, nil
}

// Close is a no-op for PostgresStorage since the pool is managed externally.
func (s *PostgresStorage) Close() error {
	// Pool is managed externally, so we don't close it here
	return nil
}

// Pool returns the underlying connection pool.
func (s *PostgresStorage) Pool() *pgxpool.Pool {
	return s.pool
}

// Ensure PostgresStorage is created with current timestamps.
func init() {
	// This is a simple sanity check that time.Now works
	_ = time.Now()
}
