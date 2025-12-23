package config

import (
	"os"
	"testing"
	"time"
)

func TestFileStorageConfig_ToStorageConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    *FileStorageConfig
		expected StorageConfig
	}{
		{
			name:  "nil config returns defaults",
			input: nil,
			expected: StorageConfig{
				Backend:           BackendSQLite,
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: DefaultConnectionTimeout,
				MaxConns:          10,
				MinConns:          2,
			},
		},
		{
			name:  "empty config returns defaults",
			input: &FileStorageConfig{},
			expected: StorageConfig{
				Backend:           BackendSQLite,
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: DefaultConnectionTimeout,
				MaxConns:          10,
				MinConns:          2,
			},
		},
		{
			name: "postgres backend with all settings",
			input: &FileStorageConfig{
				Backend:           "postgres",
				PostgresURL:       "postgres://user:pass@localhost:5432/db",
				ConnectionTimeout: 10,
				MaxConnections:    20,
				MinConnections:    5,
			},
			expected: StorageConfig{
				Backend:           BackendPostgres,
				SQLitePath:        DefaultSQLitePath(),
				PostgresURL:       "postgres://user:pass@localhost:5432/db",
				ConnectionTimeout: 10 * time.Second,
				MaxConns:          20,
				MinConns:          5,
			},
		},
		{
			name: "sqlite backend with custom path",
			input: &FileStorageConfig{
				Backend:    "sqlite",
				SQLitePath: "/tmp/test.db",
			},
			expected: StorageConfig{
				Backend:           BackendSQLite,
				SQLitePath:        "/tmp/test.db",
				ConnectionTimeout: DefaultConnectionTimeout,
				MaxConns:          10,
				MinConns:          2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.ToStorageConfig()

			if result.Backend != tt.expected.Backend {
				t.Errorf("Backend = %v, want %v", result.Backend, tt.expected.Backend)
			}
			if result.SQLitePath != tt.expected.SQLitePath {
				t.Errorf("SQLitePath = %v, want %v", result.SQLitePath, tt.expected.SQLitePath)
			}
			if result.PostgresURL != tt.expected.PostgresURL {
				t.Errorf("PostgresURL = %v, want %v", result.PostgresURL, tt.expected.PostgresURL)
			}
			if result.ConnectionTimeout != tt.expected.ConnectionTimeout {
				t.Errorf("ConnectionTimeout = %v, want %v", result.ConnectionTimeout, tt.expected.ConnectionTimeout)
			}
			if result.MaxConns != tt.expected.MaxConns {
				t.Errorf("MaxConns = %v, want %v", result.MaxConns, tt.expected.MaxConns)
			}
			if result.MinConns != tt.expected.MinConns {
				t.Errorf("MinConns = %v, want %v", result.MinConns, tt.expected.MinConns)
			}
		})
	}
}

func TestFileStorageConfig_MergeFrom(t *testing.T) {
	tests := []struct {
		name     string
		dst      *FileStorageConfig
		src      *FileStorageConfig
		expected *FileStorageConfig
	}{
		{
			name: "merge from nil does nothing",
			dst: &FileStorageConfig{
				Backend: "sqlite",
			},
			src: nil,
			expected: &FileStorageConfig{
				Backend: "sqlite",
			},
		},
		{
			name: "merge overwrites non-empty values",
			dst: &FileStorageConfig{
				Backend:    "sqlite",
				SQLitePath: "/default/path.db",
			},
			src: &FileStorageConfig{
				Backend:     "postgres",
				PostgresURL: "postgres://localhost/db",
			},
			expected: &FileStorageConfig{
				Backend:     "postgres",
				SQLitePath:  "/default/path.db",
				PostgresURL: "postgres://localhost/db",
			},
		},
		{
			name: "merge preserves dst values when src is empty",
			dst: &FileStorageConfig{
				Backend:           "postgres",
				PostgresURL:       "postgres://localhost/db",
				ConnectionTimeout: 10,
			},
			src: &FileStorageConfig{},
			expected: &FileStorageConfig{
				Backend:           "postgres",
				PostgresURL:       "postgres://localhost/db",
				ConnectionTimeout: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.dst.MergeFrom(tt.src)

			if tt.dst.Backend != tt.expected.Backend {
				t.Errorf("Backend = %v, want %v", tt.dst.Backend, tt.expected.Backend)
			}
			if tt.dst.PostgresURL != tt.expected.PostgresURL {
				t.Errorf("PostgresURL = %v, want %v", tt.dst.PostgresURL, tt.expected.PostgresURL)
			}
			if tt.dst.SQLitePath != tt.expected.SQLitePath {
				t.Errorf("SQLitePath = %v, want %v", tt.dst.SQLitePath, tt.expected.SQLitePath)
			}
			if tt.dst.ConnectionTimeout != tt.expected.ConnectionTimeout {
				t.Errorf("ConnectionTimeout = %v, want %v", tt.dst.ConnectionTimeout, tt.expected.ConnectionTimeout)
			}
		})
	}
}

func TestStorageConfig_MergeEnvOverrides(t *testing.T) {
	tests := []struct {
		name     string
		base     StorageConfig
		env      StorageConfig
		expected StorageConfig
	}{
		{
			name: "env backend postgres overrides file config sqlite",
			base: StorageConfig{
				Backend:           BackendSQLite,
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: DefaultConnectionTimeout,
				MaxConns:          10,
				MinConns:          2,
			},
			env: StorageConfig{
				Backend:           BackendPostgres, // Different from default sqlite
				PostgresURL:       "postgres://env/db",
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: DefaultConnectionTimeout,
				MaxConns:          10,
				MinConns:          2,
			},
			expected: StorageConfig{
				Backend:           BackendPostgres,
				PostgresURL:       "postgres://env/db",
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: DefaultConnectionTimeout,
				MaxConns:          10,
				MinConns:          2,
			},
		},
		{
			name: "env postgres URL overrides file config",
			base: StorageConfig{
				Backend:           BackendPostgres,
				PostgresURL:       "postgres://file/db",
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: DefaultConnectionTimeout,
				MaxConns:          10,
				MinConns:          2,
			},
			env: StorageConfig{
				Backend:           BackendSQLite, // Default value
				PostgresURL:       "postgres://env/db",
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: DefaultConnectionTimeout,
				MaxConns:          10,
				MinConns:          2,
			},
			expected: StorageConfig{
				Backend:           BackendPostgres, // Not overridden (env has default)
				PostgresURL:       "postgres://env/db",
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: DefaultConnectionTimeout,
				MaxConns:          10,
				MinConns:          2,
			},
		},
		{
			name: "env timeout overrides file config",
			base: StorageConfig{
				Backend:           BackendPostgres,
				ConnectionTimeout: 10 * time.Second,
				SQLitePath:        DefaultSQLitePath(),
				MaxConns:          10,
				MinConns:          2,
			},
			env: StorageConfig{
				Backend:           BackendSQLite, // Default
				ConnectionTimeout: 30 * time.Second,
				SQLitePath:        DefaultSQLitePath(),
				MaxConns:          10,
				MinConns:          2,
			},
			expected: StorageConfig{
				Backend:           BackendPostgres, // Not overridden
				ConnectionTimeout: 30 * time.Second,
				SQLitePath:        DefaultSQLitePath(),
				MaxConns:          10,
				MinConns:          2,
			},
		},
		{
			name: "default env does not override file config",
			base: StorageConfig{
				Backend:           BackendPostgres,
				PostgresURL:       "postgres://file/db",
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: 10 * time.Second,
				MaxConns:          20,
				MinConns:          5,
			},
			env: StorageConfig{
				Backend:           BackendSQLite, // Default
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: DefaultConnectionTimeout, // Default
				MaxConns:          10,                       // Default
				MinConns:          2,                        // Default
			},
			expected: StorageConfig{
				Backend:           BackendPostgres,
				PostgresURL:       "postgres://file/db",
				SQLitePath:        DefaultSQLitePath(),
				ConnectionTimeout: 10 * time.Second,
				MaxConns:          20,
				MinConns:          5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.base
			result.MergeEnvOverrides(tt.env)

			if result.Backend != tt.expected.Backend {
				t.Errorf("Backend = %v, want %v", result.Backend, tt.expected.Backend)
			}
			if result.PostgresURL != tt.expected.PostgresURL {
				t.Errorf("PostgresURL = %v, want %v", result.PostgresURL, tt.expected.PostgresURL)
			}
			if result.ConnectionTimeout != tt.expected.ConnectionTimeout {
				t.Errorf("ConnectionTimeout = %v, want %v", result.ConnectionTimeout, tt.expected.ConnectionTimeout)
			}
		})
	}
}

func TestLoadStorageConfigFromEnv(t *testing.T) {
	// Save original env vars
	origBackend := os.Getenv("BRAINS_BACKEND")
	origSQLitePath := os.Getenv("BRAINS_SQLITE_PATH")
	origPostgresURL := os.Getenv("BRAINS_POSTGRES_URL")
	origMaxConns := os.Getenv("BRAINS_POSTGRES_MAX_CONNS")
	origMinConns := os.Getenv("BRAINS_POSTGRES_MIN_CONNS")
	origTimeout := os.Getenv("BRAINS_POSTGRES_TIMEOUT")

	// Restore env vars after test
	defer func() {
		os.Setenv("BRAINS_BACKEND", origBackend)
		os.Setenv("BRAINS_SQLITE_PATH", origSQLitePath)
		os.Setenv("BRAINS_POSTGRES_URL", origPostgresURL)
		os.Setenv("BRAINS_POSTGRES_MAX_CONNS", origMaxConns)
		os.Setenv("BRAINS_POSTGRES_MIN_CONNS", origMinConns)
		os.Setenv("BRAINS_POSTGRES_TIMEOUT", origTimeout)
	}()

	t.Run("defaults when no env vars set", func(t *testing.T) {
		os.Unsetenv("BRAINS_BACKEND")
		os.Unsetenv("BRAINS_SQLITE_PATH")
		os.Unsetenv("BRAINS_POSTGRES_URL")
		os.Unsetenv("BRAINS_POSTGRES_MAX_CONNS")
		os.Unsetenv("BRAINS_POSTGRES_MIN_CONNS")
		os.Unsetenv("BRAINS_POSTGRES_TIMEOUT")

		cfg := LoadStorageConfigFromEnv()

		if cfg.Backend != BackendSQLite {
			t.Errorf("Backend = %v, want %v", cfg.Backend, BackendSQLite)
		}
		if cfg.SQLitePath != DefaultSQLitePath() {
			t.Errorf("SQLitePath = %v, want %v", cfg.SQLitePath, DefaultSQLitePath())
		}
		if cfg.ConnectionTimeout != DefaultConnectionTimeout {
			t.Errorf("ConnectionTimeout = %v, want %v", cfg.ConnectionTimeout, DefaultConnectionTimeout)
		}
	})

	t.Run("postgres backend from env", func(t *testing.T) {
		os.Setenv("BRAINS_BACKEND", "postgres")
		os.Setenv("BRAINS_POSTGRES_URL", "postgres://env:secret@localhost:5432/testdb")
		os.Setenv("BRAINS_POSTGRES_MAX_CONNS", "25")
		os.Setenv("BRAINS_POSTGRES_MIN_CONNS", "3")
		os.Setenv("BRAINS_POSTGRES_TIMEOUT", "15")
		defer func() {
			os.Unsetenv("BRAINS_BACKEND")
			os.Unsetenv("BRAINS_POSTGRES_URL")
			os.Unsetenv("BRAINS_POSTGRES_MAX_CONNS")
			os.Unsetenv("BRAINS_POSTGRES_MIN_CONNS")
			os.Unsetenv("BRAINS_POSTGRES_TIMEOUT")
		}()

		cfg := LoadStorageConfigFromEnv()

		if cfg.Backend != BackendPostgres {
			t.Errorf("Backend = %v, want %v", cfg.Backend, BackendPostgres)
		}
		if cfg.PostgresURL != "postgres://env:secret@localhost:5432/testdb" {
			t.Errorf("PostgresURL = %v, want postgres://env:secret@localhost:5432/testdb", cfg.PostgresURL)
		}
		if cfg.MaxConns != 25 {
			t.Errorf("MaxConns = %v, want 25", cfg.MaxConns)
		}
		if cfg.MinConns != 3 {
			t.Errorf("MinConns = %v, want 3", cfg.MinConns)
		}
		if cfg.ConnectionTimeout != 15*time.Second {
			t.Errorf("ConnectionTimeout = %v, want %v", cfg.ConnectionTimeout, 15*time.Second)
		}
	})

	t.Run("sqlite path from env", func(t *testing.T) {
		os.Setenv("BRAINS_SQLITE_PATH", "/custom/path/memories.db")
		defer os.Unsetenv("BRAINS_SQLITE_PATH")

		cfg := LoadStorageConfigFromEnv()

		if cfg.SQLitePath != "/custom/path/memories.db" {
			t.Errorf("SQLitePath = %v, want /custom/path/memories.db", cfg.SQLitePath)
		}
	})
}

func TestValidateConnectionTimeout(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected time.Duration
	}{
		{
			name:     "below minimum gets clamped",
			input:    500 * time.Millisecond,
			expected: MinConnectionTimeout,
		},
		{
			name:     "at minimum stays same",
			input:    MinConnectionTimeout,
			expected: MinConnectionTimeout,
		},
		{
			name:     "normal value stays same",
			input:    30 * time.Second,
			expected: 30 * time.Second,
		},
		{
			name:     "at maximum stays same",
			input:    MaxConnectionTimeout,
			expected: MaxConnectionTimeout,
		},
		{
			name:     "above maximum gets clamped",
			input:    500 * time.Second,
			expected: MaxConnectionTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateConnectionTimeout(tt.input)
			if result != tt.expected {
				t.Errorf("ValidateConnectionTimeout(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "absolute path unchanged",
			input:    "/absolute/path/file.db",
			expected: "/absolute/path/file.db",
		},
		{
			name:     "relative path unchanged",
			input:    "relative/path/file.db",
			expected: "relative/path/file.db",
		},
		{
			name:  "tilde path expanded",
			input: "~/path/file.db",
			// Expected: home directory + /path/file.db
			// We can't test exact value without knowing home dir
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)

			if tt.input == "~/path/file.db" {
				// For tilde expansion, just verify it starts with home dir
				home, _ := os.UserHomeDir()
				if home != "" && result[:len(home)] != home {
					t.Errorf("ExpandPath(%v) = %v, expected to start with %v", tt.input, result, home)
				}
			} else if result != tt.expected {
				t.Errorf("ExpandPath(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
