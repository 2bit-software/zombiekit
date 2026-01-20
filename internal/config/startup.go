package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// StartupConfig holds configuration for the brains start command.
type StartupConfig struct {
	Services ServiceConfigs `yaml:"services"`
}

// ServiceConfigs holds configuration for all services.
type ServiceConfigs struct {
	GUI    GUIConfig    `yaml:"gui"`
	Recall RecallConfig `yaml:"recall"`
}

// GUIConfig holds configuration for the GUI web server.
type GUIConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

// RecallConfig holds configuration for the recall watcher.
type RecallConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Source   string        `yaml:"source"`
	Interval time.Duration `yaml:"interval"`
	Verbose  bool          `yaml:"verbose"`
}

// DefaultStartupConfig returns a StartupConfig with sensible defaults
// matching current task up behavior.
func DefaultStartupConfig() *StartupConfig {
	return &StartupConfig{
		Services: ServiceConfigs{
			GUI: GUIConfig{
				Enabled: true,
				Port:    9981,
			},
			Recall: RecallConfig{
				Enabled:  true,
				Source:   "claude",
				Interval: 30 * time.Second,
				Verbose:  false,
			},
		},
	}
}

// LoadStartupConfig loads configuration from files and environment variables.
// Discovery order: local (.brains/config.yml) → global (~/.brains/config.yml) → env vars.
// If no config file exists, returns defaults.
func LoadStartupConfig() (*StartupConfig, error) {
	cfg := DefaultStartupConfig()

	// Try local config first
	localPath, err := localConfigPath()
	if err != nil {
		return nil, fmt.Errorf("determine local config path: %w", err)
	}

	if fileExists(localPath) {
		if err := loadConfigFile(localPath, cfg); err != nil {
			return nil, fmt.Errorf("load local config %s: %w", localPath, err)
		}
	} else {
		// Fall back to global config
		globalPath, err := globalConfigPath()
		if err != nil {
			return nil, fmt.Errorf("determine global config path: %w", err)
		}

		if fileExists(globalPath) {
			if err := loadConfigFile(globalPath, cfg); err != nil {
				return nil, fmt.Errorf("load global config %s: %w", globalPath, err)
			}
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	return cfg, nil
}

// Validate checks that the configuration is valid.
func (c *StartupConfig) Validate() error {
	var errs []error

	if c.Services.GUI.Enabled {
		if c.Services.GUI.Port < 1 || c.Services.GUI.Port > 65535 {
			errs = append(errs, fmt.Errorf("gui.port must be 1-65535, got %d", c.Services.GUI.Port))
		}
	}

	if c.Services.Recall.Enabled {
		if c.Services.Recall.Source != "claude" {
			errs = append(errs, fmt.Errorf("recall.source must be 'claude', got '%s'", c.Services.Recall.Source))
		}
		if c.Services.Recall.Interval < time.Second {
			errs = append(errs, fmt.Errorf("recall.interval must be >= 1s, got %v", c.Services.Recall.Interval))
		}
	}

	return errors.Join(errs...)
}

func localConfigPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, ".brains", "config.yml"), nil
}

func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".brains", "config.yml"), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func loadConfigFile(path string, cfg *StartupConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}

	return nil
}

func applyEnvOverrides(cfg *StartupConfig) {
	if port := os.Getenv("BRAINS_GUI_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			cfg.Services.GUI.Port = p
		}
	}

	if enabled := os.Getenv("BRAINS_GUI_ENABLED"); enabled != "" {
		cfg.Services.GUI.Enabled = enabled == "true" || enabled == "1"
	}

	if enabled := os.Getenv("BRAINS_RECALL_ENABLED"); enabled != "" {
		cfg.Services.Recall.Enabled = enabled == "true" || enabled == "1"
	}

	if interval := os.Getenv("BRAINS_RECALL_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			cfg.Services.Recall.Interval = d
		}
	}
}
