package main

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v2"
	"github.com/zombiekit/brains/internal/cmux"
	"github.com/zombiekit/brains/internal/github"
	"github.com/zombiekit/brains/internal/linear"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/orchestrator"
	"github.com/zombiekit/brains/internal/state"
	"github.com/zombiekit/brains/internal/version"
	"github.com/zombiekit/brains/internal/worktree"
)

func run(c *cli.Context) error {
	cfg, err := orchestrator.NewConfig(c)
	if err != nil {
		return err
	}

	logging.InitLogger(cfg.LogLevel, cfg.LogJSON, nil)
	logging.Logger().Info("orchestrator starting",
		slog.String("version", version.Get().Short()),
	)

	ctx := context.Background()
	store, err := state.NewSQLiteStore(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	linearClient, err := linear.NewClient(cfg.LinearAPIKey)
	if err != nil {
		return err
	}

	worktreeMgr, err := worktree.New(cfg.RepoDir, worktree.WithWorktreesRoot(cfg.WorktreesRoot))
	if err != nil {
		return err
	}

	sessionMgr, err := cmux.New()
	if err != nil {
		return err
	}

	ghClient, err := github.NewClient(cfg.GitHubToken, cfg.GitHubOwner, cfg.GitHubRepo)
	if err != nil {
		return err
	}

	return orchestrator.New(cfg, store, linearClient, ghClient, worktreeMgr, sessionMgr).Run()
}
