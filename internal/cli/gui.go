package cli

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/urfave/cli/v2"
	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/memory/sqlite"
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/web"
	"github.com/zombiekit/brains/internal/webplugins/memory"
	"github.com/zombiekit/brains/internal/webplugins/profiles"
)

// newGUICommand creates the gui command for starting the web interface.
func newGUICommand() *cli.Command {
	return &cli.Command{
		Name:  "gui",
		Usage: "Start the web GUI interface",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Value:   8080,
				Usage:   "Port for HTTP server",
				EnvVars: []string{"BRAINS_GUI_PORT"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "info",
				Usage:   "Log level: debug, info, warn, error",
				EnvVars: []string{"BRAINS_LOG_LEVEL"},
			},
		},
		Action: runGUI,
	}
}

func runGUI(c *cli.Context) error {
	// Set up logging
	logLevel := c.String("log-level")
	logger := logging.SetupLogger(logLevel, false, os.Stderr)

	// Create profile service
	profileService, err := profile.NewService("")
	if err != nil {
		logger.Warn("failed to initialize profile service, profiles plugin will show errors",
			"error", err,
		)
		// Continue anyway - the plugin will handle the error gracefully
	}

	// Create plugin registry
	registry := web.NewPluginRegistry(logger)

	// Register plugins
	if profileService != nil {
		profilesPlugin := profiles.NewPlugin(profileService)
		registry.Register("profiles", profilesPlugin)
	}

	// Create memory storage (SQLite default)
	homeDir, _ := os.UserHomeDir()
	memoryDBPath := filepath.Join(homeDir, ".brains", "memory.db")
	memoryStorage, err := sqlite.NewSQLiteStorage(context.Background(), memoryDBPath)
	if err != nil {
		logger.Warn("failed to initialize memory storage, memory plugin will show errors",
			"error", err,
		)
	}
	if memoryStorage != nil {
		memoryPlugin := memory.NewPlugin(memoryStorage)
		registry.Register("memory", memoryPlugin)
	}

	// Load storage config for status display
	storageConfig := config.LoadStorageConfigFromEnv()
	// Override SQLite path if using default local storage
	if storageConfig.Backend == config.BackendSQLite && storageConfig.SQLitePath == config.DefaultSQLitePath() {
		storageConfig.SQLitePath = memoryDBPath
	}

	// Create server config
	serverConfig := web.ServerConfig{
		Port: c.Int("port"),
		StatusConfig: web.StatusConfig{
			ServerPort:    c.Int("port"),
			LogLevel:      logLevel,
			StorageConfig: storageConfig,
		},
	}

	// Create server
	server, err := web.NewServer(registry, serverConfig, logger)
	if err != nil {
		return err
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-done
		logger.Info("shutting down web server")
		cancel()
	}()

	logger.Info("starting web GUI",
		"port", serverConfig.Port,
		"url", "http://localhost:"+c.String("port"),
	)

	return server.Start(ctx)
}
