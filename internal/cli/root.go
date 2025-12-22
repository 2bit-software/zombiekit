// Package cli implements the brains command-line interface.
package cli

import (
	"github.com/urfave/cli/v2"
)

// NewApp creates the root CLI application with all commands and global flags.
func NewApp(version, commit string) *cli.App {
	return &cli.App{
		Name:    "brains",
		Usage:   "ZombieKit - AI assistant profile and conversation management",
		Version: version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose output",
			},
		},
		Commands: []*cli.Command{
			newVersionCommand(version, commit),
		},
	}
}
