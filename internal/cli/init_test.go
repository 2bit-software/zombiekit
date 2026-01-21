package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func runInitCmd(t *testing.T, dir string, args ...string) error {
	t.Helper()

	app := &cli.App{
		Name: "brains",
		Commands: []*cli.Command{
			newInitCommand(),
		},
	}

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(dir)
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	allArgs := append([]string{"brains", "init"}, args...)
	return app.Run(allArgs)
}

func TestInitCommand_FreshDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir)
	require.NoError(t, err)

	// Verify .claude/commands/ exists with files
	commandsDir := filepath.Join(tmpDir, ".claude", "commands")
	entries, err := os.ReadDir(commandsDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 1, "should have at least one command file")

	// Verify .brains/templates/ exists with files
	templatesDir := filepath.Join(tmpDir, ".brains", "templates")
	entries, err = os.ReadDir(templatesDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 1, "should have at least one template file")
}

func TestInitCommand_SkipExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// First init
	err := runInitCmd(t, tmpDir)
	require.NoError(t, err)

	// Modify a file to verify it's not overwritten
	testFile := filepath.Join(tmpDir, ".claude", "commands", "brains.new.md")
	originalContent := []byte("ORIGINAL CONTENT")
	err = os.WriteFile(testFile, originalContent, 0o644)
	require.NoError(t, err)

	// Second init - should skip existing files
	err = runInitCmd(t, tmpDir)
	require.NoError(t, err)

	// Verify file was not overwritten
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, content, "file should not be overwritten without --force")
}

func TestInitCommand_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// First init
	err := runInitCmd(t, tmpDir)
	require.NoError(t, err)

	// Modify a file
	testFile := filepath.Join(tmpDir, ".claude", "commands", "brains.new.md")
	originalContent := []byte("ORIGINAL CONTENT")
	err = os.WriteFile(testFile, originalContent, 0o644)
	require.NoError(t, err)

	// Second init with --force - should overwrite
	err = runInitCmd(t, tmpDir, "--force")
	require.NoError(t, err)

	// Verify file was overwritten
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.NotEqual(t, originalContent, content, "file should be overwritten with --force")
}

func TestInitCommand_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir)
	require.NoError(t, err)

	// Verify directories exist
	assert.DirExists(t, filepath.Join(tmpDir, ".claude"))
	assert.DirExists(t, filepath.Join(tmpDir, ".claude", "commands"))
	assert.DirExists(t, filepath.Join(tmpDir, ".brains"))
	assert.DirExists(t, filepath.Join(tmpDir, ".brains", "templates"))
}

func TestInitCommand_DirectoryAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// First init
	err := runInitCmd(t, tmpDir)
	require.NoError(t, err)

	// Remove all command files but keep directories
	commandsDir := filepath.Join(tmpDir, ".claude", "commands")
	entries, _ := os.ReadDir(commandsDir)
	for _, entry := range entries {
		os.Remove(filepath.Join(commandsDir, entry.Name()))
	}

	// Second init - should re-copy files since they were removed
	err = runInitCmd(t, tmpDir)
	require.NoError(t, err)

	// Verify files were copied again
	entries, err = os.ReadDir(commandsDir)
	require.NoError(t, err)
	assert.Equal(t, 14, len(entries), "should have 14 command files")
}

func TestInitCommand_FileCount(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir)
	require.NoError(t, err)

	// Count command files (should be 15)
	commandsDir := filepath.Join(tmpDir, ".claude", "commands")
	entries, err := os.ReadDir(commandsDir)
	require.NoError(t, err)
	assert.Equal(t, 14, len(entries), "should have 14 command files")

	// Count template files (should be 7)
	templatesDir := filepath.Join(tmpDir, ".brains", "templates")
	entries, err = os.ReadDir(templatesDir)
	require.NoError(t, err)
	assert.Equal(t, 7, len(entries), "should have 7 template files")
}

func TestInitCommand_SpecificFiles(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir)
	require.NoError(t, err)

	// Verify specific command files exist
	expectedCommands := []string{
		"brains.new.md",
		"brains.plan.md",
		"brains.tasks.md",
		"brains.init.md",
	}
	for _, cmd := range expectedCommands {
		path := filepath.Join(tmpDir, ".claude", "commands", cmd)
		assert.FileExists(t, path, "command file should exist: %s", cmd)
	}

	// Verify specific template files exist
	expectedTemplates := []string{
		"spec-template.md",
		"plan-template.md",
		"tasks-template.md",
	}
	for _, tpl := range expectedTemplates {
		path := filepath.Join(tmpDir, ".brains", "templates", tpl)
		assert.FileExists(t, path, "template file should exist: %s", tpl)
	}
}

func TestInitCommand_FileContentsNotEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir)
	require.NoError(t, err)

	// Verify a command file has content
	cmdPath := filepath.Join(tmpDir, ".claude", "commands", "brains.new.md")
	content, err := os.ReadFile(cmdPath)
	require.NoError(t, err)
	assert.Greater(t, len(content), 0, "command file should have content")

	// Verify a template file has content
	tplPath := filepath.Join(tmpDir, ".brains", "templates", "spec-template.md")
	content, err = os.ReadFile(tplPath)
	require.NoError(t, err)
	assert.Greater(t, len(content), 0, "template file should have content")
}
