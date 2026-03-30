package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/2bit-software/zombiekit/internal/logging"
)

// LocalConfigPath returns the path to the local configuration file.
// The local config is located at .brains/config.toml relative to the working directory.
func LocalConfigPath() string {
	return filepath.Join(".brains", "config.toml")
}

// GlobalConfigPath returns the path to the global configuration file.
// The path varies by platform:
//   - Linux: $XDG_CONFIG_HOME/brains/config.toml or ~/.config/brains/config.toml
//   - macOS: $XDG_CONFIG_HOME/brains/config.toml or ~/.config/brains/config.toml
//   - Windows: %APPDATA%\brains\config.toml
func GlobalConfigPath() (string, error) {
	// On macOS, prefer XDG-style paths for CLI tools
	if runtime.GOOS == "darwin" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "brains", "config.toml"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "brains", "config.toml"), nil
	}

	// On Windows, use APPDATA; on Linux, use XDG_CONFIG_HOME or ~/.config
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "brains", "config.toml"), nil
}

// LoadFile loads configuration from a TOML file at the given path.
// Returns an error if the file cannot be read or parsed.
// Returns os.ErrNotExist if the file does not exist.
func LoadFile(path string) (*Config, error) {
	cfg := &Config{
		Tools: make(map[string]ToolConfig),
	}

	_, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadLocalConfig loads configuration from the local config file (.brains/config.toml).
// Returns nil config if the file does not exist (not an error condition).
// Logs debug message when config is loaded, warning on parse errors.
func LoadLocalConfig() *Config {
	path := LocalConfigPath()

	cfg, err := LoadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logging.Logger().Warn("failed to load local config",
				"path", path,
				"error", err.Error(),
			)
		}
		return nil
	}

	logging.Logger().Debug("loaded local config", "path", path)
	return cfg
}

// LoadGlobalConfig loads configuration from the global config file.
// Returns nil config if the file does not exist (not an error condition).
// Logs debug message when config is loaded, warning on parse errors.
func LoadGlobalConfig() *Config {
	path, err := GlobalConfigPath()
	if err != nil {
		logging.Logger().Warn("failed to determine global config path",
			"error", err.Error(),
		)
		return nil
	}

	cfg, err := LoadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logging.Logger().Warn("failed to load global config",
				"path", path,
				"error", err.Error(),
			)
		}
		return nil
	}

	logging.Logger().Debug("loaded global config", "path", path)
	return cfg
}

// LoadConfig loads and merges configuration from all sources.
// Precedence order (highest to lowest): CLI flags > local > global > defaults.
// This function handles global and local; CLI flags should be applied separately.
func LoadConfig() *Config {
	cfg := NewDefaultConfig()

	// Load global config first (lowest precedence of file configs)
	if globalCfg := LoadGlobalConfig(); globalCfg != nil {
		cfg.Merge(globalCfg)
	}

	// Load local config second (overrides global)
	if localCfg := LoadLocalConfig(); localCfg != nil {
		cfg.Merge(localCfg)
	}

	return cfg
}

// LoadStorageConfig loads and merges storage configuration from all sources.
// Precedence order (highest to lowest): env vars > local file > global file > defaults.
//
// Returns a StorageConfig ready for use with the database layer.
func LoadStorageConfig() StorageConfig {
	// Start with defaults
	cfg := NewDefaultStorageConfig()

	// Load and merge from config files (global -> local precedence)
	if fileCfg := LoadConfig(); fileCfg != nil && fileCfg.Storage != nil {
		cfg = fileCfg.Storage.ToStorageConfig()
		logging.Logger().Debug("loaded storage config from file",
			"backend", cfg.Backend,
			"postgres_url_set", cfg.PostgresURL != "",
		)
	}

	// Apply environment variable overrides (highest precedence)
	envCfg := LoadStorageConfigFromEnv()
	cfg.MergeEnvOverrides(envCfg)

	// Validate and clamp connection timeout
	cfg.ConnectionTimeout = ValidateConnectionTimeout(cfg.ConnectionTimeout)

	return cfg
}

// ValidateConnectionTimeout ensures the connection timeout is within valid bounds.
// Returns the clamped value within [MinConnectionTimeout, MaxConnectionTimeout].
func ValidateConnectionTimeout(d time.Duration) time.Duration {
	if d < MinConnectionTimeout {
		return MinConnectionTimeout
	}
	if d > MaxConnectionTimeout {
		return MaxConnectionTimeout
	}
	return d
}
