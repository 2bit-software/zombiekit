package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v2"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/version"
)

func TestStartCommand_Exists(t *testing.T) {
	info := &version.BuildInfo{}
	app := NewApp(info)

	found := false
	for _, cmd := range app.Commands {
		if cmd.Name == "start" {
			found = true
			break
		}
	}

	if !found {
		t.Error("start command not registered in app")
	}
}

func TestStartCommand_NoServicesEnabled(t *testing.T) {
	// Reset logger for test
	logging.ResetLogger()
	defer logging.ResetLogger()

	// Create temp dir with config that disables all services
	tmpDir := t.TempDir()
	brainsDir := filepath.Join(tmpDir, ".brains")
	if err := os.MkdirAll(brainsDir, 0755); err != nil {
		t.Fatalf("failed to create .brains dir: %v", err)
	}

	configContent := `
services:
  gui:
    enabled: false
  recall:
    enabled: false
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

	// Create a minimal app with just the start command
	app := &cli.App{
		Name: "brains",
		Commands: []*cli.Command{
			newStartCommand(),
		},
	}

	// Run start command - should exit gracefully with no services
	err = app.Run([]string{"brains", "start"})
	if err != nil {
		t.Errorf("start command failed with no services: %v", err)
	}
}

func TestStartCommand_InvalidConfig(t *testing.T) {
	// Reset logger for test
	logging.ResetLogger()
	defer logging.ResetLogger()

	// Create temp dir with invalid config
	tmpDir := t.TempDir()
	brainsDir := filepath.Join(tmpDir, ".brains")
	if err := os.MkdirAll(brainsDir, 0755); err != nil {
		t.Fatalf("failed to create .brains dir: %v", err)
	}

	// Invalid port and source
	configContent := `
services:
  gui:
    enabled: true
    port: 0
  recall:
    enabled: true
    source: invalid
`
	configPath := filepath.Join(brainsDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
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

	app := &cli.App{
		Name: "brains",
		Commands: []*cli.Command{
			newStartCommand(),
		},
	}

	// Run start command - should fail with validation error
	err = app.Run([]string{"brains", "start"})
	if err == nil {
		t.Error("expected error for invalid config, got nil")
	}
}
