package main

import (
	"log/slog"
	"os"

	"github.com/2bit-software/zombiekit/internal/version"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "orchestrator",
		Usage:   "ZombieKit autonomous development orchestrator",
		Version: version.Get().Short(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "db-path",
				Usage:   "Path to SQLite database file (overrides config)",
				EnvVars: []string{"ORCH_DB_PATH"},
			},
		},
		Commands: []*cli.Command{
			runCommand(),
			jobsCommand(),
			slotsCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("orchestrator failed", "error", err)
		os.Exit(1)
	}
}

func runCommand() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Start the orchestrator daemon",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Usage:    "Path to TOML config file",
				EnvVars:  []string{"ORCH_CONFIG"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level override (debug, info, warn, error)",
				EnvVars: []string{"ORCH_LOG_LEVEL"},
			},
			&cli.BoolFlag{
				Name:    "log-json",
				Usage:   "Output logs as JSON",
				EnvVars: []string{"ORCH_LOG_JSON"},
			},
			&cli.IntFlag{
				Name:    "callback-port",
				Usage:   "Callback port override",
				EnvVars: []string{"ORCH_CALLBACK_PORT"},
			},
			&cli.StringFlag{
				Name:    "sandbox",
				Usage:   "Docker Sandbox mode override: auto, enabled, disabled",
				EnvVars: []string{"ORCH_SANDBOX"},
			},
		},
		Action: runDaemon,
	}
}

func runDaemon(c *cli.Context) error {
	return run(c)
}
