// Package cli implements the brains command-line interface.
package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"

	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/database"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/mcp"
	"github.com/zombiekit/brains/internal/memory"
	"github.com/zombiekit/brains/internal/memory/postgres"
	"github.com/zombiekit/brains/internal/memory/sqlite"
	"github.com/zombiekit/brains/internal/recall"
	recallpostgres "github.com/zombiekit/brains/internal/recall/postgres"
)

// newServeCommand creates the serve command for starting the MCP server.
func newServeCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start the MCP server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "mode",
				Value:   "http",
				Usage:   "Transport mode: http, sse, stdio",
				EnvVars: []string{"BRAINS_MCP_MODE"},
			},
			&cli.IntFlag{
				Name:    "port",
				Value:   8080,
				Usage:   "Port for HTTP-based transports",
				EnvVars: []string{"BRAINS_MCP_PORT"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "info",
				Usage:   "Log level: debug, info, warn, error",
				EnvVars: []string{"BRAINS_LOG_LEVEL"},
			},
			&cli.StringFlag{
				Name:    "db-type",
				Value:   "sqlite",
				Usage:   "Database backend: sqlite, postgres",
				EnvVars: []string{"BRAINS_BACKEND"},
			},
			&cli.StringSliceFlag{
				Name:  "enable-tool",
				Usage: "Enable specific MCP tool (can be repeated)",
			},
			&cli.StringSliceFlag{
				Name:  "disable-tool",
				Usage: "Disable specific MCP tool (can be repeated)",
			},
			&cli.StringFlag{
				Name:    "env-file",
				Usage:   "Path to environment file to load",
				EnvVars: []string{"BRAINS_ENV_FILE"},
			},
		},
		Action: runServe,
	}
}

func runServe(c *cli.Context) error {
	// Load environment file first (before any other config)
	if envFile := c.String("env-file"); envFile != "" {
		if err := loadEnvFile(envFile); err != nil {
			return err
		}
	}

	ctx := context.Background()

	// Set up logging
	logLevel := c.String("log-level")
	logging.InitLogger(logLevel, false, os.Stderr)

	// Load tool configuration from config files
	toolCfg := config.LoadConfig()

	// Apply CLI flag overrides
	enabledTools := c.StringSlice("enable-tool")
	disabledTools := c.StringSlice("disable-tool")
	toolCfg.ApplyCLIOverrides(enabledTools, disabledTools)

	// Warn about unknown tool names
	config.WarnUnknownTools(enabledTools)
	config.WarnUnknownTools(disabledTools)

	// Load storage configuration from files + env vars
	storageCfg := config.LoadStorageConfig()

	// Override with CLI flags (highest precedence)
	if c.IsSet("db-type") {
		storageCfg.Backend = config.BackendType(c.String("db-type"))
	}

	// Initialize storage with fallback support
	storage, storageCfg, err := initializeStorage(ctx, storageCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer storage.Close()

	// Initialize recall storage if PostgreSQL is configured
	var recallStorage recall.Storage
	if storageCfg.Backend == config.BackendPostgres {
		rs, err := recallpostgres.New(ctx, storageCfg)
		if err != nil {
			logging.Logger().Warn("recall storage unavailable, recall tools will be disabled",
				"error", err.Error(),
			)
		} else {
			recallStorage = rs
			defer recallStorage.Close()
		}
	}

	// Create MCP server with tool configuration
	server := mcp.NewServer(storage, recallStorage, toolCfg)
	defer server.Close()

	mode := c.String("mode")
	port := c.Int("port")

	logging.Logger().Info("Starting MCP server",
		"mode", mode,
		"port", port,
	)

	switch mode {
	case "stdio":
		return runStdio(server)
	case "sse":
		return runSSE(server, port)
	case "http":
		return runHTTP(server, port)
	default:
		return fmt.Errorf("unsupported transport mode: %s", mode)
	}
}

// initializeStorage creates the appropriate storage backend based on configuration.
// If PostgreSQL is configured but unavailable, it falls back to SQLite with a warning.
// Returns the storage, the (potentially updated) config, and any error.
func initializeStorage(ctx context.Context, cfg config.StorageConfig) (memory.Storage, config.StorageConfig, error) {
	switch cfg.Backend {
	case config.BackendPostgres:
		storage, err := connectPostgres(ctx, cfg)
		if err != nil {
			// Fallback to SQLite
			logging.Logger().Warn("PostgreSQL connection failed, falling back to SQLite",
				"error", err.Error(),
			)
			cfg.Backend = config.BackendSQLite
			return initializeSQLite(ctx, cfg)
		}
		return storage, cfg, nil

	case config.BackendSQLite:
		return initializeSQLite(ctx, cfg)

	default:
		return nil, cfg, fmt.Errorf("unsupported backend: %s", cfg.Backend)
	}
}

// connectPostgres attempts to connect to PostgreSQL with the configured timeout.
func connectPostgres(ctx context.Context, cfg config.StorageConfig) (memory.Storage, error) {
	// Create context with connection timeout
	connCtx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout)
	defer cancel()

	// Attempt PostgreSQL connection
	pool, err := database.NewPostgresPool(connCtx, cfg)
	if err != nil {
		return nil, err
	}

	// Create storage using the pool
	storage, err := postgres.NewPostgresStorage(ctx, pool.Pool())
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize postgres storage: %w", err)
	}

	logging.Logger().Info("Storage initialized",
		"backend", "postgres",
		"host", sanitizePostgresURL(cfg.PostgresURL),
	)

	// Wrap to ensure pool cleanup
	return &postgresStorageWrapper{
		Storage: storage,
		pool:    pool,
	}, nil
}

// initializeSQLite creates a SQLite storage backend.
func initializeSQLite(ctx context.Context, cfg config.StorageConfig) (memory.Storage, config.StorageConfig, error) {
	storage, err := sqlite.NewSQLiteStorage(ctx, cfg.SQLitePath)
	if err != nil {
		return nil, cfg, fmt.Errorf("failed to initialize SQLite storage: %w", err)
	}

	logging.Logger().Info("Storage initialized",
		"backend", "sqlite",
		"path", cfg.SQLitePath,
	)

	return storage, cfg, nil
}

// postgresStorageWrapper wraps PostgresStorage to manage pool lifecycle.
type postgresStorageWrapper struct {
	memory.Storage
	pool *database.PostgresPool
}

// Close closes both the storage and the underlying pool.
func (w *postgresStorageWrapper) Close() error {
	err := w.Storage.Close()
	w.pool.Close()
	return err
}

// sanitizePostgresURL removes credentials from a PostgreSQL URL for logging.
func sanitizePostgresURL(connURL string) string {
	// Simple sanitization - just show host/db, hide credentials
	// More sophisticated parsing is in web/status.go
	if connURL == "" {
		return "(not configured)"
	}
	// This is a simplified version; the full version is in web/status.go
	return "(configured)"
}

func runStdio(s *mcp.Server) error {
	return s.ServeStdio()
}

func runSSE(s *mcp.Server, port int) error {
	addr := fmt.Sprintf(":%d", port)
	sseServer := s.ServeSSE(addr)

	// Set up graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-done
		sseServer.Shutdown(context.Background())
	}()

	return sseServer.Start(addr)
}

func runHTTP(s *mcp.Server, port int) error {
	addr := fmt.Sprintf(":%d", port)

	// Use SSE server for HTTP mode (it supports both SSE and regular HTTP)
	sseServer := s.ServeSSE(addr)

	// Set up graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		if err := sseServer.Start(addr); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-done:
		return sseServer.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

// loadEnvFile loads environment variables from a file.
// Uses godotenv.Load which does NOT override existing environment variables.
func loadEnvFile(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("env file not found: %s", path)
	}
	if err != nil {
		return fmt.Errorf("cannot access env file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("env file path is a directory: %s", path)
	}

	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("failed to load env file: %w", err)
	}

	return nil
}
