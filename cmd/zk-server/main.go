package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/2bit-software/zombiekit/internal/server"
	"github.com/2bit-software/zombiekit/internal/version"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "zk-server",
		Usage:   "ZombieKit central server",
		Version: version.Get().Short(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "listen",
				Aliases: []string{"l"},
				Usage:   "Listen address",
				Value:   ":8443",
				EnvVars: []string{"ZK_LISTEN"},
			},
			&cli.StringFlag{
				Name:    "tls-cert",
				Usage:   "Path to TLS certificate file",
				EnvVars: []string{"ZK_TLS_CERT"},
			},
			&cli.StringFlag{
				Name:    "tls-key",
				Usage:   "Path to TLS key file",
				EnvVars: []string{"ZK_TLS_KEY"},
			},
			&cli.StringFlag{
				Name:    "postgres-url",
				Usage:   "PostgreSQL connection string",
				EnvVars: []string{"ZK_POSTGRES_URL", "BRAINS_POSTGRES_URL"},
			},
			&cli.StringFlag{
				Name:    "ollama-url",
				Usage:   "Ollama API URL for LLM proxy",
				Value:   "http://localhost:11434",
				EnvVars: []string{"ZK_OLLAMA_URL", "BRAINS_OLLAMA_URL"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level (debug, info, warn, error)",
				Value:   "info",
				EnvVars: []string{"ZK_LOG_LEVEL"},
			},
			&cli.BoolFlag{
				Name:    "log-json",
				Usage:   "Output logs as JSON",
				EnvVars: []string{"ZK_LOG_JSON"},
			},
			&cli.BoolFlag{
				Name:    "migrate",
				Usage:   "Run database migrations on startup",
				EnvVars: []string{"ZK_MIGRATE"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	logging.InitLogger(c.String("log-level"), c.Bool("log-json"), nil)

	cfg := &server.Config{
		ListenAddr:    c.String("listen"),
		TLSCertPath:   c.String("tls-cert"),
		TLSKeyPath:    c.String("tls-key"),
		PostgresURL:   c.String("postgres-url"),
		OllamaURL:     c.String("ollama-url"),
		RunMigrations: c.Bool("migrate"),
	}

	srv, err := server.New(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return srv.Run(ctx)
}
