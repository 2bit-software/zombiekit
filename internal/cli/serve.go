// Package cli implements the brains command-line interface.
package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/mcp"
	"github.com/zombiekit/brains/internal/memory/sqlite"
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
		},
		Action: runServe,
	}
}

func runServe(c *cli.Context) error {
	ctx := context.Background()

	// Set up logging
	logLevel := c.String("log-level")
	logger := logging.SetupLogger(logLevel, false, os.Stderr)

	// Load tool configuration from config files
	toolCfg := config.LoadConfig(logger)

	// Apply CLI flag overrides
	enabledTools := c.StringSlice("enable-tool")
	disabledTools := c.StringSlice("disable-tool")
	toolCfg.ApplyCLIOverrides(enabledTools, disabledTools)

	// Warn about unknown tool names
	config.WarnUnknownTools(logger, enabledTools)
	config.WarnUnknownTools(logger, disabledTools)

	// Load storage configuration
	storageCfg := config.LoadStorageConfigFromEnv()

	// Override with CLI flags
	if dbType := c.String("db-type"); dbType != "" {
		storageCfg.Backend = config.BackendType(dbType)
	}

	// For now, only support SQLite (PostgreSQL can be added later)
	if storageCfg.Backend != config.BackendSQLite {
		return fmt.Errorf("only sqlite backend is currently supported, got: %s", storageCfg.Backend)
	}

	// Create storage
	storage, err := sqlite.NewSQLiteStorage(ctx, storageCfg.SQLitePath)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer storage.Close()

	logger.Info("Storage initialized",
		"backend", storageCfg.Backend,
		"path", storageCfg.SQLitePath,
	)

	// Create MCP server with tool configuration
	server := mcp.NewServer(storage, toolCfg)
	defer server.Close()

	mode := c.String("mode")
	port := c.Int("port")

	logger.Info("Starting MCP server",
		"mode", mode,
		"port", port,
	)

	switch mode {
	case "stdio":
		return runStdio(server, logger)
	case "sse":
		return runSSE(server, port, logger)
	case "http":
		return runHTTP(server, port, logger)
	default:
		return fmt.Errorf("unsupported transport mode: %s", mode)
	}
}

func runStdio(s *mcp.Server, logger interface{}) error {
	return s.ServeStdio()
}

func runSSE(s *mcp.Server, port int, logger interface{}) error {
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

func runHTTP(s *mcp.Server, port int, logger interface{}) error {
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
