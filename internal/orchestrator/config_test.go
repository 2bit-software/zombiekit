package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig(t *testing.T) *Config {
	t.Helper()
	// Create a fake repo dir with .git
	repoDir := filepath.Join(t.TempDir(), "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755))
	return &Config{
		LinearAPIKey:     "lin_test_key",
		GitHubToken:      "ghp_test_token",
		CallbackPort:     8666,
		WorktreesRoot:    filepath.Join(t.TempDir(), "worktrees"),
		DBPath:           filepath.Join(t.TempDir(), "state.db"),
		ConcurrencyLimit: 1,
		PollInterval:     30 * time.Second,
		LogLevel:         "info",
		LogJSON:          false,
		ShutdownTimeout:  30 * time.Second,
		ProjectID:        "test-project",
		RepoDir:          repoDir,
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := validConfig(t)
	assert.NoError(t, cfg.Validate())
}

func TestValidate_MissingLinearAPIKey(t *testing.T) {
	cfg := validConfig(t)
	cfg.LinearAPIKey = ""
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--linear-api-key/ORCH_LINEAR_API_KEY is required")
}

func TestValidate_MissingGitHubToken(t *testing.T) {
	cfg := validConfig(t)
	cfg.GitHubToken = ""
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--github-token/ORCH_GITHUB_TOKEN is required")
}

func TestValidate_InvalidCallbackPort_Zero(t *testing.T) {
	cfg := validConfig(t)
	cfg.CallbackPort = 0
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--callback-port/ORCH_CALLBACK_PORT must be 1-65535")
}

func TestValidate_InvalidCallbackPort_TooHigh(t *testing.T) {
	cfg := validConfig(t)
	cfg.CallbackPort = 65536
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--callback-port/ORCH_CALLBACK_PORT must be 1-65535")
}

func TestValidate_MissingWorktreesRoot(t *testing.T) {
	cfg := validConfig(t)
	cfg.WorktreesRoot = ""
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--worktrees-root/ORCH_WORKTREES_ROOT is required")
}

func TestValidate_MissingDBPath(t *testing.T) {
	cfg := validConfig(t)
	cfg.DBPath = ""
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--db-path/ORCH_DB_PATH is required")
}

func TestValidate_InvalidConcurrencyLimit_Zero(t *testing.T) {
	cfg := validConfig(t)
	cfg.ConcurrencyLimit = 0
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--concurrency-limit/ORCH_CONCURRENCY_LIMIT must be >= 1")
}

func TestValidate_InvalidConcurrencyLimit_Negative(t *testing.T) {
	cfg := validConfig(t)
	cfg.ConcurrencyLimit = -1
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--concurrency-limit/ORCH_CONCURRENCY_LIMIT must be >= 1")
}

func TestValidate_InvalidPollInterval(t *testing.T) {
	cfg := validConfig(t)
	cfg.PollInterval = 0
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--poll-interval/ORCH_POLL_INTERVAL must be > 0")
}

func TestValidate_InvalidShutdownTimeout(t *testing.T) {
	cfg := validConfig(t)
	cfg.ShutdownTimeout = 0
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--shutdown-timeout/ORCH_SHUTDOWN_TIMEOUT must be > 0")
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := validConfig(t)
	cfg.LogLevel = "banana"
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--log-level/ORCH_LOG_LEVEL must be one of: debug, info, warn, error")
}

func TestValidate_MissingProjectID(t *testing.T) {
	cfg := validConfig(t)
	cfg.ProjectID = ""
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--project-id/ORCH_PROJECT_ID is required")
}

func TestValidate_MissingRepoDir(t *testing.T) {
	cfg := validConfig(t)
	cfg.RepoDir = ""
	err := cfg.Validate()
	assert.ErrorContains(t, err, "--repo-dir/ORCH_REPO_DIR is required")
}

func TestValidate_RepoDirNoGit(t *testing.T) {
	cfg := validConfig(t)
	cfg.RepoDir = t.TempDir() // exists but has no .git
	err := cfg.Validate()
	assert.ErrorContains(t, err, "does not contain a .git directory")
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &Config{}
	err := cfg.Validate()
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "--linear-api-key")
	assert.Contains(t, msg, "--github-token")
	assert.Contains(t, msg, "--worktrees-root")
	assert.Contains(t, msg, "--db-path")
	assert.Contains(t, msg, "--callback-port")
	assert.Contains(t, msg, "--concurrency-limit")
	assert.Contains(t, msg, "--poll-interval")
	assert.Contains(t, msg, "--log-level")
	assert.Contains(t, msg, "--shutdown-timeout")
	assert.Contains(t, msg, "--project-id")
	assert.Contains(t, msg, "--repo-dir")
}

func TestValidate_WorktreesDirCreated(t *testing.T) {
	cfg := validConfig(t)
	dir := filepath.Join(t.TempDir(), "nested", "worktrees")
	cfg.WorktreesRoot = dir

	require.NoError(t, cfg.Validate())

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
