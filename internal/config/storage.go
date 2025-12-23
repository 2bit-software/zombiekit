// Package config provides configuration management for the brains CLI.
package config

import (
	"os"
	"path/filepath"
	"strconv"
)

// BackendType represents the database backend type.
type BackendType string

const (
	// BackendSQLite uses SQLite as the storage backend (default).
	BackendSQLite BackendType = "sqlite"
	// BackendPostgres uses PostgreSQL as the storage backend.
	BackendPostgres BackendType = "postgres"
)

// StorageConfig holds configuration for the storage backend.
type StorageConfig struct {
	// Backend is the storage backend type (sqlite or postgres).
	Backend BackendType

	// SQLitePath is the path to the SQLite database file.
	// Only used when Backend is BackendSQLite.
	SQLitePath string

	// PostgresURL is the PostgreSQL connection string.
	// Only used when Backend is BackendPostgres.
	PostgresURL string

	// MaxConns is the maximum number of connections in the pool.
	// Only used for PostgreSQL.
	MaxConns int32

	// MinConns is the minimum number of connections in the pool.
	// Only used for PostgreSQL.
	MinConns int32
}

// DefaultSQLitePath returns the default SQLite database path.
// Uses ~/.brains/memories.db as the default location.
func DefaultSQLitePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "memories.db"
	}
	return filepath.Join(homeDir, ".brains", "memories.db")
}

// LoadStorageConfigFromEnv loads storage configuration from environment variables.
//
// Environment variables:
//   - BRAINS_BACKEND: Backend type (sqlite or postgres, default: sqlite)
//   - BRAINS_SQLITE_PATH: Path to SQLite database (default: ~/.brains/memories.db)
//   - BRAINS_POSTGRES_URL: PostgreSQL connection string
//   - BRAINS_POSTGRES_MAX_CONNS: Max connections (default: 10)
//   - BRAINS_POSTGRES_MIN_CONNS: Min connections (default: 2)
func LoadStorageConfigFromEnv() StorageConfig {
	backend := os.Getenv("BRAINS_BACKEND")
	if backend == "" {
		backend = string(BackendSQLite)
	}

	sqlitePath := os.Getenv("BRAINS_SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = DefaultSQLitePath()
	}

	maxConns := int32(10)
	if v := os.Getenv("BRAINS_POSTGRES_MAX_CONNS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			maxConns = int32(n)
		}
	}

	minConns := int32(2)
	if v := os.Getenv("BRAINS_POSTGRES_MIN_CONNS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			minConns = int32(n)
		}
	}

	return StorageConfig{
		Backend:     BackendType(backend),
		SQLitePath:  sqlitePath,
		PostgresURL: os.Getenv("BRAINS_POSTGRES_URL"),
		MaxConns:    maxConns,
		MinConns:    minConns,
	}
}

// ExpandPath expands the path, handling home directory (~) expansion.
func ExpandPath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
