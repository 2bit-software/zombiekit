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
	"github.com/zombiekit/brains/internal/recall"
	"github.com/zombiekit/brains/internal/recall/postgres"
	"github.com/zombiekit/brains/internal/web"
	"github.com/zombiekit/brains/internal/webplugins/memory"
	"github.com/zombiekit/brains/internal/webplugins/profiles"
	recallweb "github.com/zombiekit/brains/internal/webplugins/recall"
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
	logging.InitLogger(logLevel, false, os.Stderr)

	// Create profile service
	profileService, err := profile.NewService("")
	if err != nil {
		logging.Logger().Warn("failed to initialize profile service, profiles plugin will show errors",
			"error", err,
		)
		// Continue anyway - the plugin will handle the error gracefully
	}

	// Create plugin registry
	registry := web.NewPluginRegistry()

	// Register plugins
	if profileService != nil {
		profilesPlugin := profiles.NewPlugin(profileService)
		registry.Register("profiles", profilesPlugin)
	}

	// Create memory storage (SQLite default)
	// Use BRAINS_DATA_DIR if set (for containerized environments), otherwise default to ~/.brains
	dataDir := os.Getenv("BRAINS_DATA_DIR")
	if dataDir == "" {
		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".brains")
	}
	memoryDBPath := filepath.Join(dataDir, "memory.db")
	memoryStorage, err := sqlite.NewSQLiteStorage(context.Background(), memoryDBPath)
	if err != nil {
		logging.Logger().Warn("failed to initialize memory storage, memory plugin will show errors",
			"error", err,
		)
	}
	if memoryStorage != nil {
		memoryPlugin := memory.NewPlugin(memoryStorage)
		registry.Register("memory", memoryPlugin)
	}

	// Load storage config for status display and recall plugin
	storageConfig := config.LoadStorageConfigFromEnv()

	// Register recall plugin (requires PostgreSQL for semantic search)
	if storageConfig.Backend == config.BackendPostgres {
		recallStorage, err := postgres.New(context.Background(), storageConfig)
		if err != nil {
			logging.Logger().Warn("failed to initialize recall storage, conversations plugin will be unavailable",
				"error", err,
			)
		} else {
			// Try to create embedder for semantic search (optional)
			var embedder recall.Embedder
			if storageConfig.OllamaURL != "" {
				e, err := recall.NewOllamaEmbedder(storageConfig.OllamaURL, storageConfig.EmbeddingModel)
				if err != nil {
					logging.Logger().Warn("recall embedder unavailable, search disabled",
						"error", err,
					)
				} else {
					embedder = e
				}
			}
			recallPlugin := recallweb.NewPlugin(recallStorage, embedder)
			registry.Register("recall", recallPlugin)
		}
	}
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
	server, err := web.NewServer(registry, serverConfig)
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
		logging.Logger().Info("shutting down web server")
		cancel()
	}()

	logging.Logger().Info("starting web GUI",
		"port", serverConfig.Port,
		"url", "http://localhost:"+c.String("port"),
	)

	return server.Start(ctx)
}
