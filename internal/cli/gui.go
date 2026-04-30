package cli

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/2bit-software/zombiekit/internal/config"
	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/2bit-software/zombiekit/internal/memory/sqlite"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/recall"
	"github.com/2bit-software/zombiekit/internal/recall/postgres"
	"github.com/2bit-software/zombiekit/internal/step"
	"github.com/2bit-software/zombiekit/internal/web"
	"github.com/2bit-software/zombiekit/internal/workflow"
	"github.com/urfave/cli/v2"
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

// guiMemoryDBPath returns the path to the memory SQLite database for the GUI.
func guiMemoryDBPath() string {
	dataDir := os.Getenv("BRAINS_DATA_DIR")
	if dataDir == "" {
		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".brains")
	}
	return filepath.Join(dataDir, "memory.db")
}

// registerGUIRecallPlugin creates and registers the recall plugin if PostgreSQL is configured.
func registerGUIRecallPlugin(ctx context.Context, registry *web.PluginRegistry, cfg config.StorageConfig) {
	if cfg.Backend != config.BackendPostgres {
		return
	}
	recallStorage, err := postgres.New(ctx, cfg)
	if err != nil {
		logging.Logger().Warn("failed to initialize recall storage, conversations plugin will be unavailable",
			"error", err,
		)
		return
	}

	var embedder recall.Embedder
	if cfg.OllamaURL != "" {
		e, err := recall.NewOllamaEmbedder(cfg.OllamaURL, cfg.EmbeddingModel)
		if err != nil {
			logging.Logger().Warn("recall embedder unavailable, search disabled", "error", err)
		} else {
			embedder = e
		}
	}
	registry.Register("recall", web.NewRecallPlugin(recallStorage, embedder))
}

func runGUI(c *cli.Context) error {
	logLevel := c.String("log-level")
	logging.InitLogger(logLevel, false, os.Stderr)

	profileService, err := profile.NewService("")
	if err != nil {
		logging.Logger().Warn("failed to initialize profile service, profiles plugin will show errors",
			"error", err,
		)
	}

	registry := web.NewPluginRegistry()

	if profileService != nil {
		registry.Register("profiles", web.NewProfilesPlugin(profileService))
	}

	workflowSvc, _ := workflow.NewService("")
	stepSvc, _ := step.NewService("")
	registry.Register("prompts", web.NewPromptsPlugin(profileService, stepSvc, workflowSvc))

	dbPath := guiMemoryDBPath()
	memoryStorage, err := sqlite.NewSQLiteStorage(context.Background(), dbPath)
	if err != nil {
		logging.Logger().Warn("failed to initialize memory storage, memory plugin will show errors",
			"error", err,
		)
	}
	if memoryStorage != nil {
		registry.Register("memory", web.NewMemoryPlugin(memoryStorage))
	}

	storageConfig := config.LoadStorageConfigFromEnv()
	registerGUIRecallPlugin(context.Background(), registry, storageConfig)

	if storageConfig.Backend == config.BackendSQLite && storageConfig.SQLitePath == config.DefaultSQLitePath() {
		storageConfig.SQLitePath = dbPath
	}

	serverConfig := web.ServerConfig{
		Port: c.Int("port"),
		StatusConfig: web.StatusConfig{
			ServerPort:    c.Int("port"),
			LogLevel:      logLevel,
			StorageConfig: storageConfig,
		},
	}

	server, err := web.NewServer(registry, serverConfig)
	if err != nil {
		return err
	}

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
