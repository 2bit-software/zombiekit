package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/web"
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
	registry := web.NewPluginRegistry()

	// Register plugins
	if profileService != nil {
		profilesPlugin := profiles.NewPlugin(profileService)
		if err := registry.Register(profilesPlugin); err != nil {
			return err
		}
	}

	// Create server config
	config := web.ServerConfig{
		Port: c.Int("port"),
	}

	// Create server
	server, err := web.NewServer(registry, config, logger)
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
		"port", config.Port,
		"url", "http://localhost:"+c.String("port"),
	)

	return server.Start(ctx)
}
