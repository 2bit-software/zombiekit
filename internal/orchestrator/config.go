package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

var validLogLevels = map[string]bool{
	"debug": true, "info": true, "warn": true, "error": true,
}

// Config holds all orchestrator daemon settings.
type Config struct {
	LinearAPIKey     string
	GitHubToken      string
	CallbackPort     int
	WorktreesRoot    string
	DBPath           string
	ConcurrencyLimit int
	PollInterval     time.Duration
	LogLevel         string
	LogJSON          bool
	ShutdownTimeout  time.Duration
	ProjectID        string
	RepoDir          string
	GitHubOwner      string
	GitHubRepo       string
	BaseBranch       string
	TrackingLabel    string
	BotUsername      string
}

// NewConfig parses a urfave/cli context into a validated Config.
func NewConfig(c *cli.Context) (*Config, error) {
	cfg := &Config{
		LinearAPIKey:     c.String("linear-api-key"),
		GitHubToken:      c.String("github-token"),
		CallbackPort:     c.Int("callback-port"),
		WorktreesRoot:    c.String("worktrees-root"),
		DBPath:           c.String("db-path"),
		ConcurrencyLimit: c.Int("concurrency-limit"),
		PollInterval:     c.Duration("poll-interval"),
		LogLevel:         c.String("log-level"),
		LogJSON:          c.Bool("log-json"),
		ShutdownTimeout:  c.Duration("shutdown-timeout"),
		ProjectID:        c.String("project-id"),
		RepoDir:          c.String("repo-dir"),
		GitHubOwner:      c.String("github-owner"),
		GitHubRepo:       c.String("github-repo"),
		BaseBranch:       c.String("base-branch"),
		TrackingLabel:    c.String("tracking-label"),
		BotUsername:      c.String("bot-username"),
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks all config fields and returns a multi-error listing every failure.
func (c *Config) Validate() error {
	var errs []string

	if c.LinearAPIKey == "" {
		errs = append(errs, "--linear-api-key/ORCH_LINEAR_API_KEY is required")
	}
	if c.GitHubToken == "" {
		errs = append(errs, "--github-token/ORCH_GITHUB_TOKEN is required")
	}
	if c.CallbackPort < 1 || c.CallbackPort > 65535 {
		errs = append(errs, "--callback-port/ORCH_CALLBACK_PORT must be 1-65535")
	}
	if c.WorktreesRoot == "" {
		errs = append(errs, "--worktrees-root/ORCH_WORKTREES_ROOT is required")
	}
	if c.DBPath == "" {
		errs = append(errs, "--db-path/ORCH_DB_PATH is required")
	}
	if c.ConcurrencyLimit < 1 {
		errs = append(errs, "--concurrency-limit/ORCH_CONCURRENCY_LIMIT must be >= 1")
	}
	if c.PollInterval <= 0 {
		errs = append(errs, "--poll-interval/ORCH_POLL_INTERVAL must be > 0")
	}
	if !validLogLevels[c.LogLevel] {
		errs = append(errs, "--log-level/ORCH_LOG_LEVEL must be one of: debug, info, warn, error")
	}
	if c.ShutdownTimeout <= 0 {
		errs = append(errs, "--shutdown-timeout/ORCH_SHUTDOWN_TIMEOUT must be > 0")
	}
	if c.ProjectID == "" {
		errs = append(errs, "--project-id/ORCH_PROJECT_ID is required")
	}
	if c.RepoDir == "" {
		errs = append(errs, "--repo-dir/ORCH_REPO_DIR is required")
	}
	if c.GitHubOwner == "" {
		errs = append(errs, "--github-owner/ORCH_GITHUB_OWNER is required")
	}
	if c.GitHubRepo == "" {
		errs = append(errs, "--github-repo/ORCH_GITHUB_REPO is required")
	}
	if c.BotUsername == "" {
		errs = append(errs, "--bot-username/ORCH_BOT_USERNAME is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}

	if err := os.MkdirAll(c.WorktreesRoot, 0o755); err != nil {
		return fmt.Errorf("create worktrees directory: %w", err)
	}

	if c.RepoDir != "" {
		gitPath := filepath.Join(c.RepoDir, ".git")
		if _, err := os.Stat(gitPath); err != nil {
			return fmt.Errorf("repo-dir %q does not contain a .git directory: %w", c.RepoDir, err)
		}
	}

	return nil
}
