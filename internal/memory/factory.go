// Package memory provides persistent memory storage functionality.
package memory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/2bit-software/zombiekit/internal/config"
)

// StorageFactory creates storage instances based on configuration.
type StorageFactory struct {
	cfg config.StorageConfig
}

// NewStorageFactory creates a new storage factory with the given configuration.
func NewStorageFactory(cfg config.StorageConfig) *StorageFactory {
	return &StorageFactory{cfg: cfg}
}

// SQLiteStorageCreator is a function type for creating SQLite storage.
type SQLiteStorageCreator func(ctx context.Context, dbPath string) (Storage, error)

// PostgresStorageCreator is a function type for creating PostgreSQL storage.
type PostgresStorageCreator func(ctx context.Context, pool *pgxpool.Pool) (Storage, error)

// NewStorage creates a storage instance based on the factory's configuration.
// This requires providing the backend-specific creator functions to avoid import cycles.
func NewStorage(
	ctx context.Context,
	cfg config.StorageConfig,
	sqliteCreator SQLiteStorageCreator,
	postgresCreator PostgresStorageCreator,
	pool *pgxpool.Pool, // Only used for PostgreSQL
) (Storage, error) {
	switch cfg.Backend {
	case config.BackendSQLite:
		if sqliteCreator == nil {
			return nil, fmt.Errorf("SQLite storage creator not provided")
		}
		return sqliteCreator(ctx, cfg.SQLitePath)

	case config.BackendPostgres:
		if postgresCreator == nil {
			return nil, fmt.Errorf("PostgreSQL storage creator not provided")
		}
		if pool == nil {
			return nil, fmt.Errorf("PostgreSQL connection pool not provided")
		}
		return postgresCreator(ctx, pool)

	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidBackend, cfg.Backend)
	}
}
