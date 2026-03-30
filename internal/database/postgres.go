// Package database provides database connection management.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/2bit-software/zombiekit/internal/config"
)

// PostgresPool wraps a pgxpool.Pool for connection management.
type PostgresPool struct {
	pool *pgxpool.Pool
}

// NewPostgresPool creates a new PostgreSQL connection pool.
// It verifies connectivity immediately (fail-fast behavior).
func NewPostgresPool(ctx context.Context, cfg config.StorageConfig) (*PostgresPool, error) {
	if cfg.PostgresURL == "" {
		return nil, fmt.Errorf("postgres URL is required")
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

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	// Verify connectivity immediately (fail-fast per spec)
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	return &PostgresPool{pool: pool}, nil
}

// Pool returns the underlying pgxpool.Pool.
func (p *PostgresPool) Pool() *pgxpool.Pool {
	return p.pool
}

// Close closes the connection pool.
func (p *PostgresPool) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// Ping verifies the connection is still alive.
func (p *PostgresPool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}
