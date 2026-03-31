package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/2bit-software/zombiekit/internal/config"
	"github.com/2bit-software/zombiekit/internal/memory/sqlite"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/recall"
	"github.com/2bit-software/zombiekit/internal/recall/postgres"
	"github.com/2bit-software/zombiekit/internal/step"
	"github.com/2bit-software/zombiekit/internal/web"
	"github.com/2bit-software/zombiekit/internal/workflow"
)

// GUIService wraps the web GUI server as a Service.
type GUIService struct {
	config config.GUIConfig
}

// NewGUIService creates a new GUI service with the given configuration.
func NewGUIService(cfg config.GUIConfig) *GUIService {
	return &GUIService{config: cfg}
}

// Name returns the service identifier.
func (s *GUIService) Name() string {
	return "gui"
}

// memoryDBPath returns the path to the memory SQLite database.
func memoryDBPath() string {
	dataDir := os.Getenv("BRAINS_DATA_DIR")
	if dataDir == "" {
		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".brains")
	}
	return filepath.Join(dataDir, "memory.db")
}

// registerRecallPlugin creates and registers the recall plugin if PostgreSQL is configured.
func registerRecallPlugin(ctx context.Context, log *slog.Logger, registry *web.PluginRegistry, cfg config.StorageConfig) {
	if cfg.Backend != config.BackendPostgres {
		return
	}
	recallStorage, err := postgres.New(ctx, cfg)
	if err != nil {
		log.Warn("failed to initialize recall storage, conversations plugin will be unavailable",
			"error", err,
		)
		return
	}

	var embedder recall.Embedder
	if cfg.OllamaURL != "" {
		e, err := recall.NewOllamaEmbedder(cfg.OllamaURL, cfg.EmbeddingModel)
		if err != nil {
			log.Warn("recall embedder unavailable, search disabled", "error", err)
		} else {
			embedder = e
		}
	}
	registry.Register("recall", web.NewRecallPlugin(recallStorage, embedder))
}

// Run starts the GUI server and blocks until the context is cancelled.
func (s *GUIService) Run(ctx context.Context) error {
	log := ServiceLogger(s.Name())

	profileService, err := profile.NewService("")
	if err != nil {
		log.Warn("failed to initialize profile service, profiles plugin will show errors",
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

	dbPath := memoryDBPath()
	memoryStorage, err := sqlite.NewSQLiteStorage(ctx, dbPath)
	if err != nil {
		log.Warn("failed to initialize memory storage, memory plugin will show errors",
			"error", err,
		)
	}
	if memoryStorage != nil {
		registry.Register("memory", web.NewMemoryPlugin(memoryStorage))
	}

	storageConfig := config.LoadStorageConfigFromEnv()
	registerRecallPlugin(ctx, log, registry, storageConfig)

	if storageConfig.Backend == config.BackendSQLite && storageConfig.SQLitePath == config.DefaultSQLitePath() {
		storageConfig.SQLitePath = dbPath
	}

	serverConfig := web.ServerConfig{
		Port: s.config.Port,
		StatusConfig: web.StatusConfig{
			ServerPort:    s.config.Port,
			LogLevel:      "info", // TODO: make configurable
			StorageConfig: storageConfig,
		},
	}

	server, err := web.NewServer(registry, serverConfig)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	log.Info("starting web server",
		"port", s.config.Port,
		"url", fmt.Sprintf("http://localhost:%d", s.config.Port),
	)

	return server.Start(ctx)
}
