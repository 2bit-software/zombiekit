// Package config provides configuration management for the brains CLI.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// BackendType represents the database backend type.
type BackendType string

const (
	// BackendSQLite uses SQLite as the storage backend (default).
	BackendSQLite BackendType = "sqlite"
	// BackendPostgres uses PostgreSQL as the storage backend.
	BackendPostgres BackendType = "postgres"
)

// Default configuration values.
const (
	DefaultConnectionTimeout = 5 * time.Second
	MinConnectionTimeout     = 1 * time.Second
	MaxConnectionTimeout     = 300 * time.Second
)

// FileStorageConfig represents the [storage] section in TOML config files.
// Field names use snake_case TOML tags to match the config file format.
type FileStorageConfig struct {
	// Backend is the storage backend type ("sqlite" or "postgres").
	Backend string `toml:"backend"`

	// PostgresURL is the PostgreSQL connection string.
	PostgresURL string `toml:"postgres_url"`

	// SQLitePath is the path to the SQLite database file.
	SQLitePath string `toml:"sqlite_path"`

	// ConnectionTimeout is the timeout for PostgreSQL connection attempts in seconds.
	ConnectionTimeout int `toml:"connection_timeout"`

	// MaxConnections is the maximum number of connections in the PostgreSQL pool.
	MaxConnections int `toml:"max_connections"`

	// MinConnections is the minimum number of connections in the PostgreSQL pool.
	MinConnections int `toml:"min_connections"`

	// OllamaURL is the URL for the Ollama API server.
	OllamaURL string `toml:"ollama_url"`

	// EmbeddingModel is the Ollama model to use for generating embeddings.
	EmbeddingModel string `toml:"embedding_model"`
}

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

	// ConnectionTimeout is the timeout for PostgreSQL connection attempts.
	// Defaults to 5 seconds if not configured.
	ConnectionTimeout time.Duration

	// OllamaURL is the URL for the Ollama API server.
	// Defaults to http://localhost:11434.
	OllamaURL string

	// EmbeddingModel is the Ollama model to use for generating embeddings.
	// Defaults to nomic-embed-text.
	EmbeddingModel string
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

// Default Ollama configuration values.
const (
	DefaultOllamaURL       = "http://localhost:11434"
	DefaultEmbeddingModel  = "nomic-embed-text"
)

// LoadStorageConfigFromEnv loads storage configuration from environment variables.
//
// Environment variables:
//   - BRAINS_BACKEND: Backend type (sqlite or postgres, default: sqlite)
//   - BRAINS_SQLITE_PATH: Path to SQLite database (default: ~/.brains/memories.db)
//   - BRAINS_POSTGRES_URL: PostgreSQL connection string
//   - BRAINS_POSTGRES_MAX_CONNS: Max connections (default: 10)
//   - BRAINS_POSTGRES_MIN_CONNS: Min connections (default: 2)
//   - BRAINS_POSTGRES_TIMEOUT: Connection timeout in seconds (default: 5)
//   - BRAINS_OLLAMA_URL: Ollama API URL (default: http://localhost:11434)
//   - BRAINS_EMBEDDING_MODEL: Embedding model (default: nomic-embed-text)
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

	timeout := DefaultConnectionTimeout
	if v := os.Getenv("BRAINS_POSTGRES_TIMEOUT"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			timeout = time.Duration(n) * time.Second
		}
	}

	ollamaURL := os.Getenv("BRAINS_OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = DefaultOllamaURL
	}

	embeddingModel := os.Getenv("BRAINS_EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = DefaultEmbeddingModel
	}

	return StorageConfig{
		Backend:           BackendType(backend),
		SQLitePath:        sqlitePath,
		PostgresURL:       os.Getenv("BRAINS_POSTGRES_URL"),
		MaxConns:          maxConns,
		MinConns:          minConns,
		ConnectionTimeout: timeout,
		OllamaURL:         ollamaURL,
		EmbeddingModel:    embeddingModel,
	}
}

// NewDefaultStorageConfig returns a StorageConfig with default values.
func NewDefaultStorageConfig() StorageConfig {
	return StorageConfig{
		Backend:           BackendSQLite,
		SQLitePath:        DefaultSQLitePath(),
		ConnectionTimeout: DefaultConnectionTimeout,
		MaxConns:          10,
		MinConns:          2,
		OllamaURL:         DefaultOllamaURL,
		EmbeddingModel:    DefaultEmbeddingModel,
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
