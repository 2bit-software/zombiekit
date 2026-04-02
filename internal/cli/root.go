// Package cli implements the brains command-line interface.
package cli

import (
	"github.com/2bit-software/zombiekit/internal/version"
	"github.com/urfave/cli/v2"
)

// NewApp creates the root CLI application with all commands and global flags.
func NewApp(info *version.BuildInfo) *cli.App {
	return &cli.App{
		Name:    "brains",
		Usage:   "ZombieKit - AI assistant profile and conversation management with MCP tools",
		Version: info.Short(),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose output",
			},
			&cli.StringFlag{
				Name:    "db-type",
				Value:   "sqlite",
				Usage:   "Database backend: sqlite, postgres",
				EnvVars: []string{"BRAINS_BACKEND"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "info",
				Usage:   "Log level: debug, info, warn, error",
				EnvVars: []string{"BRAINS_LOG_LEVEL"},
			},
		},
		Commands: []*cli.Command{
			newVersionCommand(info),
			newStartCommand(),
			newServeCommand(),
			newGUICommand(),
			newMemoryCommand(),
			newRecallCommand(),
			newDBCommand(),
			newProfileCommand(),
			newHookCommand(),
			newInitCommand(),
			newSkillCommand(),
		},
	}
}
