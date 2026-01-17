// Package postgres implements recall storage using PostgreSQL with pgvector.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"

	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/recall"
)

// Storage implements recall.Storage using PostgreSQL with pgvector.
type Storage struct {
	pool *pgxpool.Pool
}

// New creates a new PostgreSQL storage instance.
func New(ctx context.Context, cfg config.StorageConfig) (*Storage, error) {
	if cfg.PostgresURL == "" {
		return nil, fmt.Errorf("postgres URL is required for recall storage")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres URL: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	// Register pgvector types after each connection
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	// Verify connectivity immediately (fail-fast)
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("cannot connect to PostgreSQL database: %w", err)
	}

	return &Storage{pool: pool}, nil
}

// Save stores content with its embedding.
// Returns (id, created, error) where created=false indicates duplicate.
func (s *Storage) Save(ctx context.Context, content string, embedding []float32) (string, bool, error) {
	hash := recall.ContentHash(content)
	id := uuid.New().String()

	result, err := s.pool.Exec(ctx, `
		INSERT INTO recall_chunks (id, content, content_hash, embedding)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (content_hash) DO NOTHING
	`, id, content, hash, pgvector.NewVector(embedding))
	if err != nil {
		return "", false, fmt.Errorf("save chunk: %w", err)
	}

	// RowsAffected() == 0 means duplicate
	if result.RowsAffected() == 0 {
		// Duplicate - return silently (no output per spec)
		return "", false, nil
	}

	return id, true, nil
}

// List returns all chunks ordered by created_at DESC.
func (s *Storage) List(ctx context.Context, limit int) ([]recall.Chunk, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, content, created_at
		FROM recall_chunks
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list chunks: %w", err)
	}
	defer rows.Close()

	var chunks []recall.Chunk
	for rows.Next() {
		var chunk recall.Chunk
		if err := rows.Scan(&chunk.ID, &chunk.Content, &chunk.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		chunks = append(chunks, chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chunks: %w", err)
	}

	return chunks, nil
}

// Search finds chunks by cosine similarity to the query embedding.
func (s *Storage) Search(ctx context.Context, embedding []float32, limit int) ([]recall.SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, content, created_at, 1 - (embedding <=> $1) AS similarity
		FROM recall_chunks
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $2
	`, pgvector.NewVector(embedding), limit)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer rows.Close()

	var results []recall.SearchResult
	for rows.Next() {
		var result recall.SearchResult
		if err := rows.Scan(
			&result.Chunk.ID,
			&result.Chunk.Content,
			&result.Chunk.CreatedAt,
			&result.Similarity,
		); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate results: %w", err)
	}

	return results, nil
}

// Close releases any resources held by the storage.
func (s *Storage) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}

// Ping verifies the database connection is alive.
func (s *Storage) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
