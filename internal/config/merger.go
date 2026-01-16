package config

import "time"

// Merge applies settings from src onto the receiver (dst).
// Values from src override values in dst when set (non-nil).
// This allows layered configuration: global -> local -> CLI.
func (dst *Config) Merge(src *Config) {
	if src == nil {
		return
	}

	for name, srcTool := range src.Tools {
		if srcTool.Enabled != nil {
			dstTool := dst.Tools[name]
			dstTool.Enabled = srcTool.Enabled
			dst.Tools[name] = dstTool
		}
	}

	// Merge storage configuration
	if src.Storage != nil {
		if dst.Storage == nil {
			dst.Storage = &FileStorageConfig{}
		}
		dst.Storage.MergeFrom(src.Storage)
	}
}

// MergeFrom applies settings from src onto the receiver (dst).
// Only non-empty/non-zero values from src override values in dst.
func (dst *FileStorageConfig) MergeFrom(src *FileStorageConfig) {
	if src == nil {
		return
	}

	if src.Backend != "" {
		dst.Backend = src.Backend
	}
	if src.PostgresURL != "" {
		dst.PostgresURL = src.PostgresURL
	}
	if src.SQLitePath != "" {
		dst.SQLitePath = src.SQLitePath
	}
	if src.ConnectionTimeout > 0 {
		dst.ConnectionTimeout = src.ConnectionTimeout
	}
	if src.MaxConnections > 0 {
		dst.MaxConnections = src.MaxConnections
	}
	if src.MinConnections > 0 {
		dst.MinConnections = src.MinConnections
	}
}

// ToStorageConfig converts FileStorageConfig to StorageConfig,
// applying defaults where values are not set.
func (f *FileStorageConfig) ToStorageConfig() StorageConfig {
	cfg := StorageConfig{
		Backend:           BackendSQLite, // default
		SQLitePath:        DefaultSQLitePath(),
		ConnectionTimeout: DefaultConnectionTimeout,
		MaxConns:          10,
		MinConns:          2,
	}

	if f == nil {
		return cfg
	}

	if f.Backend != "" {
		cfg.Backend = BackendType(f.Backend)
	}
	if f.PostgresURL != "" {
		cfg.PostgresURL = f.PostgresURL
	}
	if f.SQLitePath != "" {
		cfg.SQLitePath = ExpandPath(f.SQLitePath)
	}
	if f.ConnectionTimeout > 0 {
		cfg.ConnectionTimeout = time.Duration(f.ConnectionTimeout) * time.Second
	}
	if f.MaxConnections > 0 {
		cfg.MaxConns = int32(f.MaxConnections)
	}
	if f.MinConnections > 0 {
		cfg.MinConns = int32(f.MinConnections)
	}

	return cfg
}

// MergeEnvOverrides applies environment variable overrides to the StorageConfig.
// Environment variables take precedence over config file values.
func (cfg *StorageConfig) MergeEnvOverrides(env StorageConfig) {
	// Environment variable overrides always win if they're set to non-default values
	// We check if the env config differs from the default to detect explicit settings
	defaultEnv := NewDefaultStorageConfig()

	if env.Backend != defaultEnv.Backend {
		cfg.Backend = env.Backend
	}
	if env.SQLitePath != defaultEnv.SQLitePath {
		cfg.SQLitePath = env.SQLitePath
	}
	if env.PostgresURL != "" {
		cfg.PostgresURL = env.PostgresURL
	}
	if env.MaxConns != defaultEnv.MaxConns {
		cfg.MaxConns = env.MaxConns
	}
	if env.MinConns != defaultEnv.MinConns {
		cfg.MinConns = env.MinConns
	}
	if env.ConnectionTimeout != defaultEnv.ConnectionTimeout {
		cfg.ConnectionTimeout = env.ConnectionTimeout
	}
}

// ApplyCLIOverrides applies command-line flag overrides to the configuration.
// enabledTools and disabledTools are lists of tool names from CLI flags.
// CLI flags have the highest precedence and always override config file settings.
func (c *Config) ApplyCLIOverrides(enabledTools, disabledTools []string) {
	// Apply disabled tools first
	for _, name := range disabledTools {
		enabled := false
		c.Tools[name] = ToolConfig{Enabled: &enabled}
	}

	// Apply enabled tools (these override disabled if both specified)
	for _, name := range enabledTools {
		enabled := true
		c.Tools[name] = ToolConfig{Enabled: &enabled}
	}
}
