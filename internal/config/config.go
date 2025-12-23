// Package config provides configuration management for the brains CLI.
// It handles loading, merging, and validating configuration from files
// and environment variables.
package config

// ToolConfig holds configuration for a single tool or tool category.
type ToolConfig struct {
	// Enabled indicates whether the tool is enabled.
	// nil means "not set" - inherit from category or default.
	Enabled *bool `toml:"enabled"`
}

// Config holds the merged configuration from all sources.
type Config struct {
	// Tools maps tool/category names to their configuration.
	Tools map[string]ToolConfig `toml:"tools"`
}

// NewDefaultConfig creates a new Config with all tools enabled by default.
// Tools not explicitly configured are enabled by default via IsToolEnabled().
func NewDefaultConfig() *Config {
	return &Config{
		Tools: make(map[string]ToolConfig),
	}
}
