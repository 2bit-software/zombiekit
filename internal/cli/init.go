package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"github.com/zombiekit/brains/internal/profile"
)

func newInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize profile directory structure",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "global",
				Usage: "Create in global directory instead of current directory",
			},
			&cli.StringFlag{
				Name:    "source",
				Aliases: []string{"s"},
				Value:   "brains",
				Usage:   "Profile source: brains (default) or claude",
			},
		},
		Action: func(c *cli.Context) error {
			sourceType, err := profile.ParseSourceType(c.String("source"))
			if err != nil {
				return err
			}

			svc, err := profile.NewServiceWithSource(sourceType, "")
			if err != nil {
				return fmt.Errorf("initializing profile service: %w", err)
			}

			targetDir, err := svc.GetInitDir(c.Bool("global"))
			if err != nil {
				return fmt.Errorf("getting init directory: %w", err)
			}

			// Create display path
			displayPath := targetDir
			homeDir, _ := os.UserHomeDir()
			if c.Bool("global") && homeDir != "" {
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
		},
	}
}
