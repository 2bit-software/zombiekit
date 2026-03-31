package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"github.com/2bit-software/zombiekit"
	"github.com/2bit-software/zombiekit/internal/profile"
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

// initGlobal implements the --global flag behavior (existing functionality).
// Creates the global profile directory at ~/.brains/profiles/
func initGlobal(c *cli.Context) error {
	sourceType, err := profile.ParseSourceType(c.String("source"))
	if err != nil {
		return err
	}

	svc, err := profile.NewServiceWithSource(sourceType, "")
	if err != nil {
		return fmt.Errorf("initializing profile service: %w", err)
	}

	targetDir, err := svc.GetInitDir(true)
	if err != nil {
		return fmt.Errorf("getting init directory: %w", err)
	}

	// Create display path
	displayPath := targetDir
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		displayPath = "~" + targetDir[len(homeDir):]
	}

	// Check if already exists
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("Directory already exists: %s\n", displayPath)
		return nil
	}

	// Create the directory
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Register in registry (best effort, only for brains source)
	if sourceType == profile.SourceTypeBrains {
		brainsDir := filepath.Dir(targetDir)
		rm, err := profile.NewRegistryManager()
		if err == nil {
			_ = rm.Register(brainsDir) // Ignore errors, this is optional
		}
	}

	fmt.Printf("Initialized %s\n", displayPath)
	return nil
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
func initLocal(force bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	if err := validateEmbeddedFS(zombiekit.EmbeddedCommands, "integrations/claude/commands", "commands"); err != nil {
		return err
	}
	if err := validateEmbeddedFS(zombiekit.EmbeddedTemplates, "templates", "templates"); err != nil {
		return err
	}

	total := copyResult{}

	claudeDir := filepath.Join(cwd, ".claude")
	commandsDir := filepath.Join(claudeDir, "commands")
	if err := copyToDir(&total, zombiekit.EmbeddedCommands, "integrations/claude/commands", []string{claudeDir, commandsDir}, force); err != nil {
		return err
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
				Usage: "Create global profile directory (~/.brains/) only",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Overwrite existing files",
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

			// If --global is specified, use existing behavior (profile directory only)
			if c.Bool("global") {
				return initGlobal(c)
			}

			// Full local setup: .claude/commands/ and .brains/templates/
			return initLocal(force)
		},
	}
}
