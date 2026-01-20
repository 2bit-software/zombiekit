package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultStartupConfig(t *testing.T) {
	cfg := DefaultStartupConfig()

	if !cfg.Services.GUI.Enabled {
		t.Error("expected GUI to be enabled by default")
	}
	if cfg.Services.GUI.Port != 9981 {
		t.Errorf("expected GUI port 9981, got %d", cfg.Services.GUI.Port)
	}

	if !cfg.Services.Recall.Enabled {
		t.Error("expected Recall to be enabled by default")
	}
	if cfg.Services.Recall.Source != "claude" {
		t.Errorf("expected Recall source 'claude', got '%s'", cfg.Services.Recall.Source)
	}
	if cfg.Services.Recall.Interval != 30*time.Second {
		t.Errorf("expected Recall interval 30s, got %v", cfg.Services.Recall.Interval)
	}
	if cfg.Services.Recall.Verbose {
		t.Error("expected Recall verbose to be false by default")
	}
}

func TestStartupConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*StartupConfig)
		wantErr bool
	}{
		{
			name:    "valid default config",
			modify:  func(cfg *StartupConfig) {},
			wantErr: false,
		},
		{
			name: "invalid GUI port zero",
			modify: func(cfg *StartupConfig) {
				cfg.Services.GUI.Port = 0
			},
			wantErr: true,
		},
		{
			name: "invalid GUI port too high",
			modify: func(cfg *StartupConfig) {
				cfg.Services.GUI.Port = 70000
			},
			wantErr: true,
		},
		{
			name: "invalid recall source",
			modify: func(cfg *StartupConfig) {
				cfg.Services.Recall.Source = "unknown"
			},
			wantErr: true,
		},
		{
			name: "invalid recall interval too short",
			modify: func(cfg *StartupConfig) {
				cfg.Services.Recall.Interval = 500 * time.Millisecond
			},
			wantErr: true,
		},
		{
			name: "disabled services skip validation",
			modify: func(cfg *StartupConfig) {
				cfg.Services.GUI.Enabled = false
				cfg.Services.GUI.Port = 0 // Would be invalid if enabled
				cfg.Services.Recall.Enabled = false
				cfg.Services.Recall.Source = "invalid" // Would be invalid if enabled
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultStartupConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadStartupConfig_FromYAML(t *testing.T) {
	// Create a temp directory with a .brains/config.yml
	tmpDir := t.TempDir()
	brainsDir := filepath.Join(tmpDir, ".brains")
	if err := os.MkdirAll(brainsDir, 0755); err != nil {
		t.Fatalf("failed to create .brains dir: %v", err)
	}

	configContent := `
services:
  gui:
    enabled: false
    port: 8888
  recall:
    enabled: true
    source: claude
    interval: 60s
    verbose: true
`
	configPath := filepath.Join(brainsDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })

	cfg, err := LoadStartupConfig()
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v", err)
	}

	if cfg.Services.GUI.Enabled {
		t.Error("expected GUI to be disabled from config")
	}
	if cfg.Services.GUI.Port != 8888 {
		t.Errorf("expected GUI port 8888, got %d", cfg.Services.GUI.Port)
	}
	if cfg.Services.Recall.Interval != 60*time.Second {
		t.Errorf("expected Recall interval 60s, got %v", cfg.Services.Recall.Interval)
	}
	if !cfg.Services.Recall.Verbose {
		t.Error("expected Recall verbose to be true from config")
	}
}

func TestLoadStartupConfig_EnvOverrides(t *testing.T) {
	// Set environment variables
	t.Setenv("BRAINS_GUI_PORT", "7777")
	t.Setenv("BRAINS_GUI_ENABLED", "false")
	t.Setenv("BRAINS_RECALL_INTERVAL", "2m")

	// Use a non-existent directory so we get defaults + env overrides
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })

	cfg, err := LoadStartupConfig()
	if err != nil {
		t.Fatalf("LoadStartupConfig() error = %v", err)
	}

	if cfg.Services.GUI.Port != 7777 {
		t.Errorf("expected GUI port 7777 from env, got %d", cfg.Services.GUI.Port)
	}
	if cfg.Services.GUI.Enabled {
		t.Error("expected GUI to be disabled from env")
	}
	if cfg.Services.Recall.Interval != 2*time.Minute {
		t.Errorf("expected Recall interval 2m from env, got %v", cfg.Services.Recall.Interval)
	}
}

func TestLoadStartupConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	brainsDir := filepath.Join(tmpDir, ".brains")
	if err := os.MkdirAll(brainsDir, 0755); err != nil {
		t.Fatalf("failed to create .brains dir: %v", err)
	}

	// Invalid YAML
	configPath := filepath.Join(brainsDir, "config.yml")
	if err := os.WriteFile(configPath, []byte("services: [invalid yaml"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })

	_, err = LoadStartupConfig()
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}
