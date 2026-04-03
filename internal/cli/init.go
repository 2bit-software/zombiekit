package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/2bit-software/zombiekit"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/urfave/cli/v2"
)

// copyResult tracks the outcome of file copying operations.
type copyResult struct {
	copied      int
	skipped     int
	overwritten int
	errors      []error
}

// copyEmbeddedFiles copies files from an embedded filesystem to a target directory.
// srcPrefix is stripped from the embedded path to get the relative filename.
// If force is true, existing files are overwritten; otherwise they are skipped.
func copyEmbeddedFiles(fsys fs.FS, srcPrefix, destDir string, force bool) copyResult {
	result := copyResult{}

	err := fs.WalkDir(fsys, srcPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			result.errors = append(result.errors, fmt.Errorf("walking %s: %w", path, err))
			return nil
		}
		if d.IsDir() {
			return nil
		}

		copyOneEmbeddedFile(fsys, path, srcPrefix, destDir, force, &result)
		return nil
	})

	if err != nil {
		result.errors = append(result.errors, fmt.Errorf("walking filesystem: %w", err))
	}

	return result
}

// copyOneEmbeddedFile copies a single file from the embedded FS to destDir,
// updating result counters. Errors are appended to result.errors rather than
// returned so the walk can continue.
func copyOneEmbeddedFile(fsys fs.FS, path, srcPrefix, destDir string, force bool, result *copyResult) {
	relPath, err := filepath.Rel(srcPrefix, path)
	if err != nil {
		result.errors = append(result.errors, fmt.Errorf("getting relative path for %s: %w", path, err))
		return
	}

	destPath := filepath.Join(destDir, relPath)

	// Ensure parent directory exists for nested files
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		result.errors = append(result.errors, fmt.Errorf("creating directory for %s: %w", destPath, err))
		return
	}

	exists := fileExists(destPath)

	if exists && !force {
		fmt.Printf("  Skipped %s (exists)\n", relPath)
		result.skipped++
		return
	}

	content, err := fs.ReadFile(fsys, path)
	if err != nil {
		result.errors = append(result.errors, fmt.Errorf("reading %s: %w", path, err))
		return
	}

	if err := os.WriteFile(destPath, content, 0o644); err != nil {
		result.errors = append(result.errors, fmt.Errorf("writing %s: %w", destPath, err))
		return
	}

	if exists && force {
		fmt.Printf("  Overwrote %s\n", relPath)
		result.overwritten++
	} else {
		fmt.Printf("  Copied %s\n", relPath)
		result.copied++
	}
}

// fileExists reports whether a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// initGlobal implements the --global flag behavior.
// Creates ~/.brains/ with profiles/, scripts/, and templates/.
func initGlobal(c *cli.Context, claude bool) error {
	force := c.Bool("force")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	brainsDir := filepath.Join(homeDir, ".brains")
	profilesDir := filepath.Join(brainsDir, "profiles")
	scriptsDir := filepath.Join(brainsDir, "scripts")
	templatesDir := filepath.Join(brainsDir, "templates")

	// Create directory structure
	for _, dir := range []string{brainsDir, profilesDir, scriptsDir, templatesDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	total := copyResult{}

	// Install scripts
	if err := validateEmbeddedFS(zombiekit.EmbeddedScripts, "scripts", "scripts"); err != nil {
		return err
	}
	if err := copyToDir(&total, zombiekit.EmbeddedScripts, "scripts", []string{scriptsDir}, force); err != nil {
		return err
	}
	if err := makeExecutable(scriptsDir); err != nil {
		return fmt.Errorf("setting script permissions: %w", err)
	}

	// Install templates
	if err := validateEmbeddedFS(zombiekit.EmbeddedTemplates, "templates", "templates"); err != nil {
		return err
	}
	if err := copyToDir(&total, zombiekit.EmbeddedTemplates, "templates", []string{templatesDir}, force); err != nil {
		return err
	}

	if claude {
		if err := validateEmbeddedFS(zombiekit.EmbeddedClaudeCommands, "integrations/claude/commands", "commands"); err != nil {
			return err
		}
		claudeDir := filepath.Join(homeDir, ".claude")
		commandsDir := filepath.Join(claudeDir, "commands")
		if err := copyToDir(&total, zombiekit.EmbeddedClaudeCommands, "integrations/claude/commands", []string{claudeDir, commandsDir}, force); err != nil {
			return err
		}

		settingsPath := filepath.Join(claudeDir, "settings.json")
		if err := ensureMCPServer(settingsPath); err != nil {
			total.errors = append(total.errors, fmt.Errorf("configuring MCP server: %w", err))
		}
		if err := ensureHooks(settingsPath); err != nil {
			total.errors = append(total.errors, fmt.Errorf("configuring hooks: %w", err))
		}
	}

	// Register in registry (best effort)
	rm, err := profile.NewRegistryManager()
	if err == nil {
		_ = rm.Register(brainsDir)
	}

	fmt.Printf("\nInitialized ~/.brains/\n")
	printInitSummary(total)
	return nil
}

// makeExecutable sets executable permissions on script files (.sh, .py) within a directory tree.
func makeExecutable(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext == ".sh" || ext == ".py" {
			if err := os.Chmod(path, 0o755); err != nil {
				return fmt.Errorf("chmod %s: %w", path, err)
			}
		}
		return nil
	})
}

// validateEmbeddedFS checks that the embedded filesystem contains entries at the given path.
func validateEmbeddedFS(fsys fs.FS, path, label string) error {
	entries, err := fs.ReadDir(fsys, path)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("embedded %s filesystem is empty - binary may be corrupted, please reinstall", label)
	}
	return nil
}

// copyToDir creates the directory tree and copies embedded files, accumulating into total.
func copyToDir(total *copyResult, fsys fs.FS, srcPrefix string, dirs []string, force bool) error {
	for _, dir := range dirs {
		if err := createDirIfNeeded(dir); err != nil {
			return err
		}
	}

	result := copyEmbeddedFiles(fsys, srcPrefix, dirs[len(dirs)-1], force)
	total.copied += result.copied
	total.skipped += result.skipped
	total.overwritten += result.overwritten
	total.errors = append(total.errors, result.errors...)
	return nil
}

// printInitSummary prints the final init result.
func printInitSummary(total copyResult) {
	fmt.Println()
	if total.overwritten > 0 {
		fmt.Printf("Initialized ZombieKit: %d files copied, %d skipped, %d overwritten\n",
			total.copied, total.skipped, total.overwritten)
	} else {
		fmt.Printf("Initialized ZombieKit: %d files copied, %d skipped\n",
			total.copied, total.skipped)
	}

	if len(total.errors) > 0 {
		fmt.Println("\nWarnings:")
		for _, e := range total.errors {
			fmt.Printf("  - %v\n", e)
		}
	}
}

// initLocal performs full ZombieKit setup in the current directory.
// Creates .claude/commands/ with embedded commands and .brains/templates/ with templates.
func initLocal(force, claude bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	if err := validateEmbeddedFS(zombiekit.EmbeddedTemplates, "templates", "templates"); err != nil {
		return err
	}

	total := copyResult{}

	if claude {
		if err := validateEmbeddedFS(zombiekit.EmbeddedClaudeCommands, "integrations/claude/commands", "commands"); err != nil {
			return err
		}
		claudeDir := filepath.Join(cwd, ".claude")
		commandsDir := filepath.Join(claudeDir, "commands")
		if err := copyToDir(&total, zombiekit.EmbeddedClaudeCommands, "integrations/claude/commands", []string{claudeDir, commandsDir}, force); err != nil {
			return err
		}

		settingsPath := filepath.Join(claudeDir, "settings.json")
		if err := ensureMCPServer(settingsPath); err != nil {
			total.errors = append(total.errors, fmt.Errorf("configuring MCP server: %w", err))
		}
		if err := ensureHooks(settingsPath); err != nil {
			total.errors = append(total.errors, fmt.Errorf("configuring hooks: %w", err))
		}
	}

	brainsDir := filepath.Join(cwd, ".brains")
	templatesDir := filepath.Join(brainsDir, "templates")
	if err := copyToDir(&total, zombiekit.EmbeddedTemplates, "templates", []string{brainsDir, templatesDir}, force); err != nil {
		return err
	}

	rm, err := profile.NewRegistryManager()
	if err == nil {
		_ = rm.Register(brainsDir)
	}

	printInitSummary(total)
	return nil
}

// ensureMCPServer adds the zombiekit MCP server entry to a Claude settings.json file.
// Creates the file if it doesn't exist. Preserves all existing settings.
func ensureMCPServer(settingsPath string) error {
	var settings map[string]any

	data, err := os.ReadFile(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", settingsPath, err)
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parsing %s: %w", settingsPath, err)
		}
	}

	if settings == nil {
		settings = make(map[string]any)
	}

	servers, _ := settings["mcpServers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}

	if _, exists := servers["zombiekit"]; exists {
		fmt.Println("  MCP server zombiekit already configured")
		return nil
	}

	servers["zombiekit"] = map[string]any{
		"command": "brains",
		"args":    []string{"serve", "--mode", "stdio"},
	}
	settings["mcpServers"] = servers

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", settingsPath, err)
	}

	fmt.Printf("  Configured MCP server in %s\n", settingsPath)
	return nil
}

// brainsHookDefs defines the hook entries that brains init --claude installs.
var brainsHookDefs = []struct {
	event   string // Claude hook event name (e.g. "SessionStart")
	matcher string // tool matcher pattern ("" = match all)
	command string
	timeout int
}{
	{event: "SessionStart", matcher: "startup|resume|compact", command: "brains hook --event session-start", timeout: 10},
	{event: "PreToolUse", matcher: "Read|Write|Edit|MultiEdit", command: "brains hook --event pre-tool-use", timeout: 10},
	{event: "SessionEnd", matcher: "", command: "brains hook --event session-end", timeout: 5},
}

// ensureHooks idempotently adds brains hook entries to a Claude settings.json file.
// Existing hooks (including other brains hooks with different matchers) are preserved.
func ensureHooks(settingsPath string) error {
	var settings map[string]any

	data, err := os.ReadFile(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", settingsPath, err)
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parsing %s: %w", settingsPath, err)
		}
	}

	if settings == nil {
		settings = make(map[string]any)
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
	}

	added := 0
	for _, def := range brainsHookDefs {
		entries, _ := hooks[def.event].([]any)

		if hookEntryExists(entries, def.command, def.matcher) {
			continue
		}

		entry := map[string]any{
			"matcher": def.matcher,
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": def.command,
					"timeout": def.timeout,
				},
			},
		}
		entries = append(entries, entry)
		hooks[def.event] = entries
		added++
	}

	if added == 0 {
		fmt.Println("  Hooks already configured")
		return nil
	}

	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", settingsPath, err)
	}

	fmt.Printf("  Configured %d hook(s) in %s\n", added, settingsPath)
	return nil
}

// hookEntryExists checks whether an entry with the given command and matcher
// already exists in a list of hook event entries.
func hookEntryExists(entries []any, command, matcher string) bool {
	for _, raw := range entries {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		entryMatcher, _ := entry["matcher"].(string)
		if entryMatcher != matcher {
			continue
		}
		hooksList, _ := entry["hooks"].([]any)
		for _, h := range hooksList {
			hookMap, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if hookMap["command"] == command {
				return true
			}
		}
	}
	return false
}

// createDirIfNeeded creates a directory if it doesn't exist, printing status.
func createDirIfNeeded(path string) error {
	if _, err := os.Stat(path); err == nil {
		// Get relative path for display
		cwd, _ := os.Getwd()
		relPath, _ := filepath.Rel(cwd, path)
		if relPath == "" {
			relPath = path
		}
		fmt.Printf("%s/ exists\n", relPath)
		return nil
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", path, err)
	}

	// Get relative path for display
	cwd, _ := os.Getwd()
	relPath, _ := filepath.Rel(cwd, path)
	if relPath == "" {
		relPath = path
	}
	fmt.Printf("Created %s/\n", relPath)
	return nil
}

func newInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize ZombieKit in current directory with commands and templates",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "global",
				Usage: "Install global profiles, scripts, and templates to ~/.brains/",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Overwrite existing files",
			},
			&cli.BoolFlag{
				Name:  "claude",
				Usage: "Install Claude Code slash commands to .claude/commands/",
			},
			&cli.StringFlag{
				Name:    "source",
				Aliases: []string{"s"},
				Value:   "brains",
				Usage:   "Profile source: brains (default) or claude",
			},
		},
		Action: func(c *cli.Context) error {
			force := c.Bool("force")
			claude := c.Bool("claude")

			if c.Bool("global") {
				return initGlobal(c, claude)
			}

			return initLocal(force, claude)
		},
	}
}
