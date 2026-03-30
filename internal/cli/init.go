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
			return nil // continue walking
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Get the filename (strip the prefix)
		relPath, err := filepath.Rel(srcPrefix, path)
		if err != nil {
			result.errors = append(result.errors, fmt.Errorf("getting relative path for %s: %w", path, err))
			return nil
		}

		destPath := filepath.Join(destDir, relPath)

		// Check if file exists
		exists := false
		if _, err := os.Stat(destPath); err == nil {
			exists = true
		}

		// Handle existing files
		if exists && !force {
			fmt.Printf("  Skipped %s (exists)\n", relPath)
			result.skipped++
			return nil
		}

		// Read from embedded filesystem
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			result.errors = append(result.errors, fmt.Errorf("reading %s: %w", path, err))
			return nil
		}

		// Write to destination
		if err := os.WriteFile(destPath, content, 0o644); err != nil {
			result.errors = append(result.errors, fmt.Errorf("writing %s: %w", destPath, err))
			return nil
		}

		if exists && force {
			fmt.Printf("  Overwrote %s\n", relPath)
			result.overwritten++
		} else {
			fmt.Printf("  Copied %s\n", relPath)
			result.copied++
		}

		return nil
	})

	if err != nil {
		result.errors = append(result.errors, fmt.Errorf("walking filesystem: %w", err))
	}

	return result
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

// initLocal performs full ZombieKit setup in the current directory.
// Creates .claude/commands/ with embedded commands and .brains/templates/ with templates.
func initLocal(force bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Validate embedded filesystems are not empty
	commandEntries, err := fs.ReadDir(zombiekit.EmbeddedCommands, "integrations/claude/commands")
	if err != nil || len(commandEntries) == 0 {
		return fmt.Errorf("embedded commands filesystem is empty - binary may be corrupted, please reinstall")
	}

	templateEntries, err := fs.ReadDir(zombiekit.EmbeddedTemplates, "templates")
	if err != nil || len(templateEntries) == 0 {
		return fmt.Errorf("embedded templates filesystem is empty - binary may be corrupted, please reinstall")
	}

	// Aggregate results
	totalResult := copyResult{}

	// Create .claude/commands/ directory and copy commands
	claudeDir := filepath.Join(cwd, ".claude")
	commandsDir := filepath.Join(claudeDir, "commands")

	if err := createDirIfNeeded(claudeDir); err != nil {
		return err
	}
	if err := createDirIfNeeded(commandsDir); err != nil {
		return err
	}

	cmdResult := copyEmbeddedFiles(zombiekit.EmbeddedCommands, "integrations/claude/commands", commandsDir, force)
	totalResult.copied += cmdResult.copied
	totalResult.skipped += cmdResult.skipped
	totalResult.overwritten += cmdResult.overwritten
	totalResult.errors = append(totalResult.errors, cmdResult.errors...)

	// Create .brains/templates/ directory and copy templates
	brainsDir := filepath.Join(cwd, ".brains")
	templatesDir := filepath.Join(brainsDir, "templates")

	if err := createDirIfNeeded(brainsDir); err != nil {
		return err
	}
	if err := createDirIfNeeded(templatesDir); err != nil {
		return err
	}

	tplResult := copyEmbeddedFiles(zombiekit.EmbeddedTemplates, "templates", templatesDir, force)
	totalResult.copied += tplResult.copied
	totalResult.skipped += tplResult.skipped
	totalResult.overwritten += tplResult.overwritten
	totalResult.errors = append(totalResult.errors, tplResult.errors...)

	// Register .brains directory in profile registry (best effort)
	rm, err := profile.NewRegistryManager()
	if err == nil {
		_ = rm.Register(brainsDir)
	}

	// Print summary
	fmt.Println()
	if totalResult.overwritten > 0 {
		fmt.Printf("Initialized ZombieKit: %d files copied, %d skipped, %d overwritten\n",
			totalResult.copied, totalResult.skipped, totalResult.overwritten)
	} else {
		fmt.Printf("Initialized ZombieKit: %d files copied, %d skipped\n",
			totalResult.copied, totalResult.skipped)
	}

	// Report any errors
	if len(totalResult.errors) > 0 {
		fmt.Println("\nWarnings:")
		for _, e := range totalResult.errors {
			fmt.Printf("  - %v\n", e)
		}
	}

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
