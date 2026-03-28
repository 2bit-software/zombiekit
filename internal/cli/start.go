package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"path/filepath"

	"github.com/urfave/cli/v2"
	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/shutdown"
	"github.com/zombiekit/brains/internal/startup"
	"github.com/zombiekit/brains/internal/state"
)

// newStartCommand creates the start command for running all services.
func newStartCommand() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start all configured services (GUI and recall watcher)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "info",
				Usage:   "Log level: debug, info, warn, error",
				EnvVars: []string{"BRAINS_LOG_LEVEL"},
			},
		},
		Action: runStart,
	}
}

func runStart(c *cli.Context) error {
	// Set up logging
	logLevel := c.String("log-level")
	logging.InitLogger(logLevel, false, os.Stderr)
	log := logging.Logger()

	// Load configuration
	cfg, err := config.LoadStartupConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Build list of enabled services
	var services []shutdown.ServiceFunc

	if cfg.Services.GUI.Enabled {
		guiService := startup.NewGUIService(cfg.Services.GUI)
		services = append(services, func(ctx context.Context) error {
			return guiService.Run(ctx)
		})
		log.Info("gui service enabled", "port", cfg.Services.GUI.Port)
	}

	if cfg.Services.Recall.Enabled {
		recallService := startup.NewRecallService(cfg.Services.Recall)
		services = append(services, func(ctx context.Context) error {
			return recallService.Run(ctx)
		})
		log.Info("recall service enabled",
			"source", cfg.Services.Recall.Source,
			"interval", cfg.Services.Recall.Interval,
		)
	}

	if len(services) == 0 {
		log.Warn("no services enabled, nothing to start")
		return nil
	}

	// Initialize state store for crash-recovery reconciliation
	dataDir := os.Getenv("BRAINS_DATA_DIR")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".brains")
	}
	statePath := filepath.Join(dataDir, "state.db")
	stateStore, err := state.NewSQLiteStore(c.Context, statePath)
	if err != nil {
		return fmt.Errorf("initialize state store: %w", err)
	}
	defer stateStore.Close()

	// Run crash-recovery reconciliation before launching services
	if err := state.ApplyReconciliation(c.Context, stateStore, log); err != nil {
		return fmt.Errorf("startup reconciliation: %w", err)
	}

	log.Info("starting services", "count", len(services))

	// Run services with shutdown manager
	mgr := shutdown.New(10 * time.Second)
	return mgr.Run(services...)
}
