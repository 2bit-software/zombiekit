package startup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/memory/sqlite"
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/recall"
	"github.com/zombiekit/brains/internal/recall/postgres"
	"github.com/zombiekit/brains/internal/step"
	"github.com/zombiekit/brains/internal/web"
	"github.com/zombiekit/brains/internal/webplugins/memory"
	"github.com/zombiekit/brains/internal/webplugins/profiles"
	"github.com/zombiekit/brains/internal/webplugins/prompts"
	recallweb "github.com/zombiekit/brains/internal/webplugins/recall"
	"github.com/zombiekit/brains/internal/workflow"
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

// Run starts the GUI server and blocks until the context is cancelled.
func (s *GUIService) Run(ctx context.Context) error {
	log := ServiceLogger(s.Name())

	// Create profile service
	profileService, err := profile.NewService("")
	if err != nil {
		log.Warn("failed to initialize profile service, profiles plugin will show errors",
			"error", err,
		)
	}

	// Create plugin registry
	registry := web.NewPluginRegistry()

	// Register plugins
	if profileService != nil {
		profilesPlugin := profiles.NewPlugin(profileService)
		registry.Register("profiles", profilesPlugin)
	}

	// Register prompts plugin (unified view of workflows, profiles, and steps)
	workflowSvc, _ := workflow.NewService("")
	stepSvc, _ := step.NewService("")
	promptsPlugin := prompts.NewPlugin(profileService, stepSvc, workflowSvc)
	registry.Register("prompts", promptsPlugin)

	// Create memory storage (SQLite default)
	dataDir := os.Getenv("BRAINS_DATA_DIR")
	if dataDir == "" {
		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".brains")
	}
	memoryDBPath := filepath.Join(dataDir, "memory.db")
	memoryStorage, err := sqlite.NewSQLiteStorage(ctx, memoryDBPath)
	if err != nil {
		log.Warn("failed to initialize memory storage, memory plugin will show errors",
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
		recallStorage, err := postgres.New(ctx, storageConfig)
		if err != nil {
			log.Warn("failed to initialize recall storage, conversations plugin will be unavailable",
				"error", err,
			)
		} else {
			// Try to create embedder for semantic search (optional)
			var embedder recall.Embedder
			if storageConfig.OllamaURL != "" {
				e, err := recall.NewOllamaEmbedder(storageConfig.OllamaURL, storageConfig.EmbeddingModel)
				if err != nil {
					log.Warn("recall embedder unavailable, search disabled",
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
		Port: s.config.Port,
		StatusConfig: web.StatusConfig{
			ServerPort:    s.config.Port,
			LogLevel:      "info", // TODO: make configurable
			StorageConfig: storageConfig,
		},
	}

	// Create server
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
