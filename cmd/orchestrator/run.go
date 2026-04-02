package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v2"
	"github.com/2bit-software/zombiekit/internal/cmux"
	"github.com/2bit-software/zombiekit/internal/github"
	"github.com/2bit-software/zombiekit/internal/linear"
	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/2bit-software/zombiekit/internal/orchestrator"
	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/state"
	"github.com/2bit-software/zombiekit/internal/version"
	"github.com/2bit-software/zombiekit/internal/worktree"
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

	worktreeMgr, err := worktree.New(cfg.RepoDir,
		worktree.WithWorktreesRoot(cfg.WorktreesRoot),
		worktree.WithCopyFiles(cfg.CopyFiles),
	)
	if err != nil {
		return err
	}

	var cmuxOpts []cmux.Option
	useSandbox, err := resolveSandboxMode(c.String("sandbox"))
	if err != nil {
		return err
	}
	if useSandbox {
		sbxCfg := sandbox.DefaultConfig()
		cfg.SandboxAvailable = true
		cfg.SandboxConfig = sbxCfg
		cmuxOpts = append(cmuxOpts, cmux.WithCommandBuilder(sandbox.NewCommandBuilder(sbxCfg)))
		logging.Logger().Info("docker sandbox mode enabled")
	}

	sessionMgr, err := cmux.New(cmuxOpts...)
	if err != nil {
		return err
	}

	ghClient, err := github.NewClient(cfg.GitHubToken, cfg.GitHubOwner, cfg.GitHubRepo)
	if err != nil {
		return err
	}

	return orchestrator.New(cfg, store, linearClient, ghClient, worktreeMgr, sessionMgr).Run()
}

// resolveSandboxMode interprets the --sandbox flag value.
func resolveSandboxMode(mode string) (bool, error) {
	switch mode {
	case "auto":
		return sandbox.Available(), nil
	case "enabled":
		if !sandbox.Available() {
			return false, fmt.Errorf("--sandbox=enabled but sbx is not on PATH")
		}
		return true, nil
	case "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("--sandbox must be auto, enabled, or disabled (got %q)", mode)
	}
}
