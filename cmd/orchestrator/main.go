package main

import (
	"context"
	"log/slog"
	"os"
	"time"

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

func main() {
	app := &cli.App{
		Name:    "orchestrator",
		Usage:   "ZombieKit autonomous development orchestrator",
		Version: version.Get().Short(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "linear-api-key",
				Usage:   "Linear API key",
				EnvVars: []string{"ORCH_LINEAR_API_KEY"},
			},
			&cli.StringFlag{
				Name:    "github-token",
				Usage:   "GitHub personal access token",
				EnvVars: []string{"ORCH_GITHUB_TOKEN"},
			},
			&cli.IntFlag{
				Name:    "callback-port",
				Usage:   "HTTP callback server port",
				Value:   8666,
				EnvVars: []string{"ORCH_CALLBACK_PORT"},
			},
			&cli.StringFlag{
				Name:    "worktrees-root",
				Usage:   "Root directory for git worktrees",
				EnvVars: []string{"ORCH_WORKTREES_ROOT"},
			},
			&cli.StringFlag{
				Name:    "db-path",
				Usage:   "Path to SQLite database file",
				EnvVars: []string{"ORCH_DB_PATH"},
			},
			&cli.IntFlag{
				Name:    "concurrency-limit",
				Usage:   "Max concurrent jobs per project",
				Value:   1,
				EnvVars: []string{"ORCH_CONCURRENCY_LIMIT"},
			},
			&cli.DurationFlag{
				Name:    "poll-interval",
				Usage:   "Watcher polling interval",
				Value:   30 * time.Second,
				EnvVars: []string{"ORCH_POLL_INTERVAL"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level (debug, info, warn, error)",
				Value:   "info",
				EnvVars: []string{"ORCH_LOG_LEVEL"},
			},
			&cli.BoolFlag{
				Name:    "log-json",
				Usage:   "Output logs as JSON",
				EnvVars: []string{"ORCH_LOG_JSON"},
			},
			&cli.DurationFlag{
				Name:    "shutdown-timeout",
				Usage:   "Max time to drain on shutdown",
				Value:   30 * time.Second,
				EnvVars: []string{"ORCH_SHUTDOWN_TIMEOUT"},
			},
			&cli.StringFlag{
				Name:    "project-id",
				Usage:   "Linear project identifier for concurrency slot scoping",
				EnvVars: []string{"ORCH_PROJECT_ID"},
			},
			&cli.StringFlag{
				Name:    "repo-dir",
				Usage:   "Git repository root directory (must contain .git)",
				EnvVars: []string{"ORCH_REPO_DIR"},
			},
			&cli.StringFlag{
				Name:    "github-owner",
				Usage:   "GitHub repository owner",
				EnvVars: []string{"ORCH_GITHUB_OWNER"},
			},
			&cli.StringFlag{
				Name:    "github-repo",
				Usage:   "GitHub repository name",
				EnvVars: []string{"ORCH_GITHUB_REPO"},
			},
			&cli.StringFlag{
				Name:    "base-branch",
				Usage:   "Default base branch for PRs",
				Value:   "main",
				EnvVars: []string{"ORCH_BASE_BRANCH"},
			},
			&cli.StringFlag{
				Name:    "tracking-label",
				Usage:   "GitHub label applied to agent-created PRs",
				Value:   "ai-managed",
				EnvVars: []string{"ORCH_TRACKING_LABEL"},
			},
			&cli.StringFlag{
				Name:    "bot-username",
				Usage:   "GitHub username of the bot account (for filtering self-authored comments)",
				EnvVars: []string{"ORCH_BOT_USERNAME"},
			},
			&cli.StringFlag{
				Name:    "closed-pr-status",
				Usage:   "Linear ticket status for PRs closed without merge",
				Value:   "cancelled",
				EnvVars: []string{"ORCH_CLOSED_PR_STATUS"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("orchestrator failed", "error", err)
		os.Exit(1)
	}
}

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
	defer store.Close()

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
