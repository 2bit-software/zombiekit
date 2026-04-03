package cli

import (
	"encoding/json"
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

	// Without --claude, .claude/commands/ should not exist
	commandsDir := filepath.Join(tmpDir, ".claude", "commands")
	assert.NoDirExists(t, commandsDir, "should not create .claude/commands/ without --claude")

	// Verify .brains/templates/ exists with files
	templatesDir := filepath.Join(tmpDir, ".brains", "templates")
	entries, err := os.ReadDir(templatesDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 1, "should have at least one template file")
}

func TestInitCommand_ClaudeFlag(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	// With --claude, .claude/commands/ should exist with files
	commandsDir := filepath.Join(tmpDir, ".claude", "commands")
	entries, err := os.ReadDir(commandsDir)
	require.NoError(t, err)
	assert.Equal(t, 4, len(entries), "should have 4 command files")

	// Templates should also be installed
	templatesDir := filepath.Join(tmpDir, ".brains", "templates")
	entries, err = os.ReadDir(templatesDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 1, "should have at least one template file")
}

func TestInitCommand_SkipExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// First init with --claude
	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	// Modify a file to verify it's not overwritten
	testFile := filepath.Join(tmpDir, ".claude", "commands", "brains.new.md")
	originalContent := []byte("ORIGINAL CONTENT")
	err = os.WriteFile(testFile, originalContent, 0o644)
	require.NoError(t, err)

	// Second init - should skip existing files
	err = runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	// Verify file was not overwritten
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, content, "file should not be overwritten without --force")
}

func TestInitCommand_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// First init with --claude
	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	// Modify a file
	testFile := filepath.Join(tmpDir, ".claude", "commands", "brains.new.md")
	originalContent := []byte("ORIGINAL CONTENT")
	err = os.WriteFile(testFile, originalContent, 0o644)
	require.NoError(t, err)

	// Second init with --force --claude - should overwrite
	err = runInitCmd(t, tmpDir, "--force", "--claude")
	require.NoError(t, err)

	// Verify file was overwritten
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.NotEqual(t, originalContent, content, "file should be overwritten with --force")
}

func TestInitCommand_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	// Verify directories exist
	assert.DirExists(t, filepath.Join(tmpDir, ".claude"))
	assert.DirExists(t, filepath.Join(tmpDir, ".claude", "commands"))
	assert.DirExists(t, filepath.Join(tmpDir, ".brains"))
	assert.DirExists(t, filepath.Join(tmpDir, ".brains", "templates"))
}

func TestInitCommand_DirectoryAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// First init with --claude
	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	// Remove all command files but keep directories
	commandsDir := filepath.Join(tmpDir, ".claude", "commands")
	entries, _ := os.ReadDir(commandsDir)
	for _, entry := range entries {
		os.Remove(filepath.Join(commandsDir, entry.Name()))
	}

	// Second init with --claude - should re-copy files since they were removed
	err = runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	// Verify files were copied again
	entries, err = os.ReadDir(commandsDir)
	require.NoError(t, err)
	assert.Equal(t, 4, len(entries), "should have 4 command files")
}

func TestInitCommand_FileCount(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	// Count command files
	commandsDir := filepath.Join(tmpDir, ".claude", "commands")
	entries, err := os.ReadDir(commandsDir)
	require.NoError(t, err)
	assert.Equal(t, 4, len(entries), "should have 4 command files")

	// Count template entries (7 files + init-spec-creator/ subdirectory = 8)
	templatesDir := filepath.Join(tmpDir, ".brains", "templates")
	entries, err = os.ReadDir(templatesDir)
	require.NoError(t, err)
	assert.Equal(t, 8, len(entries), "should have 7 template files + 1 subdirectory")
}

func TestInitCommand_SpecificFiles(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	// Verify specific command files exist
	expectedCommands := []string{
		"brains.new.md",
		"brains.next.md",
		"brains.complete.md",
		"brains.help.md",
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

func TestInitCommand_Global(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runInitCmd(t, tmpDir, "--global")
	require.NoError(t, err)

	brainsDir := filepath.Join(tmpDir, ".brains")
	assert.DirExists(t, filepath.Join(brainsDir, "scripts"))
	assert.DirExists(t, filepath.Join(brainsDir, "templates"))

	// Verify scripts were installed with subdirectories
	assert.DirExists(t, filepath.Join(brainsDir, "scripts", "commit-message"))
	assert.DirExists(t, filepath.Join(brainsDir, "scripts", "permissions-audit"))
	assert.DirExists(t, filepath.Join(brainsDir, "scripts", "repo-auditor"))

	// Verify scripts are executable
	gitInfoPath := filepath.Join(brainsDir, "scripts", "commit-message", "git-info.sh")
	assert.FileExists(t, gitInfoPath)
	info, err := os.Stat(gitInfoPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&0o111 != 0, "script should be executable")

	// Verify templates include init-spec-creator subdirectory
	assert.DirExists(t, filepath.Join(brainsDir, "templates", "init-spec-creator"))
	assert.FileExists(t, filepath.Join(brainsDir, "templates", "init-spec-creator", "DOMAIN-TEMPLATE.md"))
}

func TestInitCommand_GlobalClaude(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runInitCmd(t, tmpDir, "--global", "--claude")
	require.NoError(t, err)

	// Verify claude commands installed to ~/.claude/commands/
	commandsDir := filepath.Join(tmpDir, ".claude", "commands")
	entries, err := os.ReadDir(commandsDir)
	require.NoError(t, err)
	assert.Equal(t, 4, len(entries), "should have 4 command files")

	// Without --claude, commands should not be installed
	tmpDir2 := t.TempDir()
	t.Setenv("HOME", tmpDir2)

	err = runInitCmd(t, tmpDir2, "--global")
	require.NoError(t, err)
	assert.NoDirExists(t, filepath.Join(tmpDir2, ".claude", "commands"))
}

func TestInitCommand_GlobalForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// First init
	err := runInitCmd(t, tmpDir, "--global")
	require.NoError(t, err)

	// Modify a script
	scriptPath := filepath.Join(tmpDir, ".brains", "scripts", "commit-message", "git-info.sh")
	original := []byte("MODIFIED")
	err = os.WriteFile(scriptPath, original, 0o755)
	require.NoError(t, err)

	// Second init without --force — should skip
	err = runInitCmd(t, tmpDir, "--global")
	require.NoError(t, err)
	content, _ := os.ReadFile(scriptPath)
	assert.Equal(t, original, content)

	// Third init with --force — should overwrite
	err = runInitCmd(t, tmpDir, "--global", "--force")
	require.NoError(t, err)
	content, _ = os.ReadFile(scriptPath)
	assert.NotEqual(t, original, content)
}

func TestInitCommand_FileContentsNotEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir, "--claude")
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

func TestInitCommand_ClaudeMCPServer(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))

	servers, ok := settings["mcpServers"].(map[string]any)
	require.True(t, ok, "mcpServers should exist")

	zk, ok := servers["zombiekit"].(map[string]any)
	require.True(t, ok, "zombiekit server should exist")
	assert.Equal(t, "brains", zk["command"])
}

func TestInitCommand_ClaudeMCPServer_PreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0o755))

	existing := map[string]any{
		"env": map[string]any{"FOO": "bar"},
		"mcpServers": map[string]any{
			"other-server": map[string]any{"command": "other"},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644))

	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	result, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(result, &settings))

	// Existing env preserved
	env, _ := settings["env"].(map[string]any)
	assert.Equal(t, "bar", env["FOO"])

	// Existing MCP server preserved
	servers, _ := settings["mcpServers"].(map[string]any)
	_, ok := servers["other-server"]
	assert.True(t, ok, "existing server should be preserved")

	// New server added
	_, ok = servers["zombiekit"]
	assert.True(t, ok, "zombiekit server should be added")
}

func TestInitCommand_ClaudeMCPServer_SkipsIfExists(t *testing.T) {
	tmpDir := t.TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0o755))

	custom := map[string]any{
		"mcpServers": map[string]any{
			"zombiekit": map[string]any{"command": "custom-brains", "args": []any{"serve"}},
		},
	}
	data, _ := json.MarshalIndent(custom, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644))

	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	result, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(result, &settings))

	servers, _ := settings["mcpServers"].(map[string]any)
	zk, _ := servers["zombiekit"].(map[string]any)
	assert.Equal(t, "custom-brains", zk["command"], "should not overwrite existing zombiekit config")
}

func TestInitCommand_ClaudeHooks(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))

	hooks, ok := settings["hooks"].(map[string]any)
	require.True(t, ok, "hooks should exist")

	// Verify all three hook events are present
	for _, event := range []string{"SessionStart", "PreToolUse", "SessionEnd"} {
		entries, ok := hooks[event].([]any)
		require.True(t, ok, "hook event %s should exist", event)
		assert.GreaterOrEqual(t, len(entries), 1, "hook event %s should have at least one entry", event)
	}
}

func TestInitCommand_ClaudeHooks_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Run twice
	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)
	err = runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))

	hooks, _ := settings["hooks"].(map[string]any)

	// Each event should have exactly one entry (no duplicates)
	for _, event := range []string{"SessionStart", "PreToolUse", "SessionEnd"} {
		entries, _ := hooks[event].([]any)
		assert.Equal(t, 1, len(entries), "hook event %s should have exactly one entry after two runs", event)
	}
}

func TestInitCommand_ClaudeHooks_PreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0o755))

	existing := map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{
				map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "echo custom-hook",
							"timeout": 5,
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644))

	err := runInitCmd(t, tmpDir, "--claude")
	require.NoError(t, err)

	result, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(result, &settings))

	hooks, _ := settings["hooks"].(map[string]any)
	sessionStart, _ := hooks["SessionStart"].([]any)

	// Should have 2 entries: the existing custom one + the brains one
	assert.Equal(t, 2, len(sessionStart), "should preserve existing hook and add brains hook")

	// Verify custom hook is still there
	first, _ := sessionStart[0].(map[string]any)
	firstHooks, _ := first["hooks"].([]any)
	firstHook, _ := firstHooks[0].(map[string]any)
	assert.Equal(t, "echo custom-hook", firstHook["command"], "existing hook should be preserved")
}

func TestInitCommand_GlobalClaudeMCPServer(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runInitCmd(t, tmpDir, "--global", "--claude")
	require.NoError(t, err)

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))

	servers, ok := settings["mcpServers"].(map[string]any)
	require.True(t, ok)
	_, ok = servers["zombiekit"]
	assert.True(t, ok, "zombiekit server should be configured globally")
}
