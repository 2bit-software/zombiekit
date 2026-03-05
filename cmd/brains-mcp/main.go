package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"

	zombiekit "github.com/zombiekit/brains"
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/proxy"
	"github.com/zombiekit/brains/internal/step"
	"github.com/zombiekit/brains/internal/version"
	"github.com/zombiekit/brains/internal/workflow"
)

func init() {
	profile.SetEmbeddedFS(zombiekit.EmbeddedProfiles)
	workflow.SetEmbeddedFS(zombiekit.EmbeddedWorkflows)
	step.SetTemplateFS(zombiekit.EmbeddedTemplates)
}

func main() {
	app := &cli.App{
		Name:    "brains-mcp",
		Usage:   "ZombieKit MCP proxy (stdio server, gRPC client)",
		Version: version.Get().Short(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "server-url",
				Usage:   "ZK server URL (empty = local-only mode)",
				EnvVars: []string{"ZK_SERVER_URL"},
			},
			&cli.StringFlag{
				Name:    "tls-ca",
				Usage:   "Path to CA cert for TLS verification",
				EnvVars: []string{"ZK_TLS_CA"},
			},
			&cli.StringFlag{
				Name:    "api-key",
				Usage:   "API key for server auth",
				EnvVars: []string{"ZK_API_KEY"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level (debug, info, warn, error)",
				Value:   "info",
				EnvVars: []string{"ZK_LOG_LEVEL"},
			},
			&cli.StringFlag{
				Name:    "env-file",
				Usage:   "Path to .env file",
				EnvVars: []string{"ZK_ENV_FILE"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	if envFile := c.String("env-file"); envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return err
		}
	}

	logLevel := parseLogLevel(c.String("log-level"))
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	cfg := &proxy.ProxyConfig{
		ServerURL: c.String("server-url"),
		TLSCAPath: c.String("tls-ca"),
		APIKey:    c.String("api-key"),
		LogLevel:  c.String("log-level"),
	}

	p, err := proxy.NewProxy(cfg, logger)
	if err != nil {
		return err
	}

	logger.Info("starting brains-mcp",
		slog.String("server_url", cfg.ServerURL),
		slog.Bool("local_only", cfg.ServerURL == ""),
	)

	return p.ServeStdio()
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
