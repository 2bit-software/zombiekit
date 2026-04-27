package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v2"

	"github.com/2bit-software/zombiekit/internal/callback"
	"github.com/2bit-software/zombiekit/internal/cmux"
	"github.com/2bit-software/zombiekit/internal/github"
	"github.com/2bit-software/zombiekit/internal/linear"
	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/2bit-software/zombiekit/internal/orchestrator"
	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/shutdown"
	"github.com/2bit-software/zombiekit/internal/state"
	"github.com/2bit-software/zombiekit/internal/version"
	"github.com/2bit-software/zombiekit/internal/worktree"
)

func initProjectRunner(
	p *orchestrator.ProjectConfig,
	store state.StateStore,
	sessionMgr cmux.SessionManager,
	demuxer *callback.EventDemuxer,
	useSandbox bool,
	sbxCfg sandbox.Config,
	logger *slog.Logger,
) (*orchestrator.ProjectRunner, shutdown.ServiceFunc, error) {
	events := demuxer.Register(p.ID)

	lc, err := linear.NewClient(p.LinearAPIKey)
	if err != nil {
		return nil, nil, fmt.Errorf("project %s: %w", p.ID, err)
	}

	gh, err := github.NewClient(p.GitHubToken, p.GitHubOwner, p.GitHubRepo)
	if err != nil {
		return nil, nil, fmt.Errorf("project %s: %w", p.ID, err)
	}

	var wtOpts []worktree.Option
	if p.WorktreesRoot != "" {
		wtOpts = append(wtOpts, worktree.WithWorktreesRoot(p.WorktreesRoot))
	}
	if len(p.CopyFiles) > 0 {
		wtOpts = append(wtOpts, worktree.WithCopyFiles(p.CopyFiles))
	}
	wt, err := worktree.New(p.RepoDir, wtOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("project %s: %w", p.ID, err)
	}

	runner := orchestrator.NewProjectRunner(
		*p, store, lc, gh, wt, sessionMgr, events,
		useSandbox, sbxCfg, logger,
	)

	logger.Info("project configured",
		slog.String("project", p.ID),
		slog.String("repo", p.GitHubOwner+"/"+p.GitHubRepo),
		slog.Int("concurrency", p.ConcurrencyLimit),
	)
	return runner, runner.RunSupervised, nil
}

func run(c *cli.Context) error {
	cfg, err := orchestrator.LoadOrchestratorConfig(c.String("config"))
	if err != nil {
		return err
	}

	applyCLIOverrides(c, cfg)

	logging.InitLogger(cfg.Global.LogLevel, cfg.Global.LogJSON, nil)
	logger := logging.Logger()
	logger.Info("orchestrator starting",
		slog.String("version", version.Get().Short()),
		slog.Int("projects", len(cfg.Projects)),
	)

	ctx := context.Background()
	dbPath := cfg.Global.DBPath
	if override := c.String("db-path"); override != "" {
		dbPath = override
	}
	store, err := state.NewSQLiteStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	projectIDs := make([]string, len(cfg.Projects))
	for i, p := range cfg.Projects {
		projectIDs[i] = p.ID
	}
	if err := state.ApplyReconciliation(ctx, store, logger, projectIDs...); err != nil {
		return fmt.Errorf("reconciliation: %w", err)
	}

	useSandbox, err := resolveSandboxMode(cfg.Global.Sandbox)
	if err != nil {
		return err
	}
	var sbxCfg sandbox.Config
	var cmuxOpts []cmux.Option
	if useSandbox {
		sbxCfg = sandbox.DefaultConfig()
		cmuxOpts = append(cmuxOpts, cmux.WithCommandBuilder(sandbox.NewCommandBuilder(sbxCfg)))
		logger.Info("docker sandbox mode enabled")
	}

	sessionMgr, err := cmux.New(cmuxOpts...)
	if err != nil {
		return err
	}

	callbackSrv := callback.New(cfg.Global.CallbackPort)
	demuxer := callback.NewEventDemuxer(logger)

	var services []shutdown.ServiceFunc
	services = append(services, callbackSrv.Run)
	services = append(services, func(ctx context.Context) error {
		return demuxer.Run(ctx, callbackSrv.Events())
	})

	var runners []*orchestrator.ProjectRunner
	for i := range cfg.Projects {
		runner, svc, initErr := initProjectRunner(&cfg.Projects[i], store, sessionMgr, demuxer, useSandbox, sbxCfg, logger)
		if initErr != nil {
			return initErr
		}
		runners = append(runners, runner)
		services = append(services, svc)
	}

	callbackSrv.SetHealthProvider(func() any {
		status := "healthy"
		projects := make(map[string]any, len(runners))
		for _, r := range runners {
			h := r.Health()
			projects[h.ProjectID] = h.Watchers
			for _, w := range h.Watchers {
				if w.ConsecutiveFails > 0 {
					status = "degraded"
				}
			}
		}
		return map[string]any{"status": status, "projects": projects}
	})

	logger.Info("starting services",
		slog.Int("total", len(services)),
	)
	mgr := shutdown.New(cfg.Global.ShutdownTimeout.Duration)
	return mgr.Run(services...)
}

func applyCLIOverrides(c *cli.Context, cfg *orchestrator.OrchestratorConfig) {
	if v := c.String("log-level"); v != "" {
		cfg.Global.LogLevel = v
	}
	if c.IsSet("log-json") {
		cfg.Global.LogJSON = c.Bool("log-json")
	}
	if v := c.Int("callback-port"); v != 0 {
		cfg.Global.CallbackPort = v
		for i := range cfg.Projects {
			cfg.Projects[i].CallbackPort = v
		}
	}
	if v := c.String("sandbox"); v != "" {
		cfg.Global.Sandbox = v
		for i := range cfg.Projects {
			cfg.Projects[i].SandboxMode = v
		}
	}
}

func resolveSandboxMode(mode string) (bool, error) {
	switch mode {
	case "auto", "":
		return sandbox.Available(), nil
	case "enabled":
		if !sandbox.Available() {
			return false, fmt.Errorf("sandbox=enabled but sbx is not on PATH")
		}
		return true, nil
	case "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("sandbox must be auto, enabled, or disabled (got %q)", mode)
	}
}
