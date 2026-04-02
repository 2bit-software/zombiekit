package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/2bit-software/zombiekit/internal/sandbox"
)

var validLogLevels = map[string]bool{
	"debug": true, "info": true, "warn": true, "error": true,
}

// Config holds all orchestrator daemon settings.
type Config struct {
	LinearAPIKey         string
	GitHubToken          string
	CallbackPort         int
	WorktreesRoot        string
	DBPath               string
	ConcurrencyLimit     int
	PollInterval         time.Duration
	LogLevel             string
	LogJSON              bool
	ShutdownTimeout      time.Duration
	ProjectID            string
	RepoDir              string
	GitHubOwner          string
	GitHubRepo           string
	BaseBranch           string
	TrackingLabel        string
	BotUsername          string
	ClosedPRTicketStatus string

	// CopyFiles lists files (relative to repo root) to copy into each
	// new worktree. Used for untracked files like .env or .mcp.json.
	CopyFiles []string

	// SandboxAvailable is true when sbx is detected on PATH at startup.
	// When true, agent sessions run inside Docker Sandboxes for isolation.
	SandboxAvailable bool
	SandboxConfig    sandbox.Config
}

// NewConfig parses a urfave/cli context into a validated Config.
func NewConfig(c *cli.Context) (*Config, error) {
	cfg := &Config{
		LinearAPIKey:         c.String("linear-api-key"),
		GitHubToken:          c.String("github-token"),
		CallbackPort:         c.Int("callback-port"),
		WorktreesRoot:        c.String("worktrees-root"),
		DBPath:               c.String("db-path"),
		ConcurrencyLimit:     c.Int("concurrency-limit"),
		PollInterval:         c.Duration("poll-interval"),
		LogLevel:             c.String("log-level"),
		LogJSON:              c.Bool("log-json"),
		ShutdownTimeout:      c.Duration("shutdown-timeout"),
		ProjectID:            c.String("project-id"),
		RepoDir:              c.String("repo-dir"),
		GitHubOwner:          c.String("github-owner"),
		GitHubRepo:           c.String("github-repo"),
		BaseBranch:           c.String("base-branch"),
		TrackingLabel:        c.String("tracking-label"),
		BotUsername:          c.String("bot-username"),
		ClosedPRTicketStatus: c.String("closed-pr-status"),
		CopyFiles:            c.StringSlice("copy-files"),
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

type configRule struct {
	msg   string
	check func(*Config) bool
}

var configRules = []configRule{
	{"--linear-api-key/ORCH_LINEAR_API_KEY is required", func(c *Config) bool { return c.LinearAPIKey == "" }},
	{"--github-token/ORCH_GITHUB_TOKEN is required", func(c *Config) bool { return c.GitHubToken == "" }},
	{"--callback-port/ORCH_CALLBACK_PORT must be 1-65535", func(c *Config) bool { return c.CallbackPort < 1 || c.CallbackPort > 65535 }},
	{"--worktrees-root/ORCH_WORKTREES_ROOT is required", func(c *Config) bool { return c.WorktreesRoot == "" }},
	{"--db-path/ORCH_DB_PATH is required", func(c *Config) bool { return c.DBPath == "" }},
	{"--concurrency-limit/ORCH_CONCURRENCY_LIMIT must be >= 1", func(c *Config) bool { return c.ConcurrencyLimit < 1 }},
	{"--poll-interval/ORCH_POLL_INTERVAL must be > 0", func(c *Config) bool { return c.PollInterval <= 0 }},
	{"--log-level/ORCH_LOG_LEVEL must be one of: debug, info, warn, error", func(c *Config) bool { return !validLogLevels[c.LogLevel] }},
	{"--shutdown-timeout/ORCH_SHUTDOWN_TIMEOUT must be > 0", func(c *Config) bool { return c.ShutdownTimeout <= 0 }},
	{"--project-id/ORCH_PROJECT_ID is required", func(c *Config) bool { return c.ProjectID == "" }},
	{"--repo-dir/ORCH_REPO_DIR is required", func(c *Config) bool { return c.RepoDir == "" }},
	{"--github-owner/ORCH_GITHUB_OWNER is required", func(c *Config) bool { return c.GitHubOwner == "" }},
	{"--github-repo/ORCH_GITHUB_REPO is required", func(c *Config) bool { return c.GitHubRepo == "" }},
	{"--bot-username/ORCH_BOT_USERNAME is required", func(c *Config) bool { return c.BotUsername == "" }},
}

// Validate checks all config fields and returns a multi-error listing every failure.
func (c *Config) Validate() error {
	var errs []string
	for _, r := range configRules {
		if r.check(c) {
			errs = append(errs, r.msg)
		}
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
