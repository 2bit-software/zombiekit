package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestTOML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "orchestrator.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func validTOMLConfig(t *testing.T) string {
	t.Helper()

	repoA := filepath.Join(t.TempDir(), "repo-a")
	repoB := filepath.Join(t.TempDir(), "repo-b")
	require.NoError(t, os.MkdirAll(filepath.Join(repoA, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(repoB, ".git"), 0o755))

	wtA := filepath.Join(t.TempDir(), "wt-a")
	wtB := filepath.Join(t.TempDir(), "wt-b")

	return writeTestTOML(t, `
[global]
linear_api_key   = "lin_api_test"
github_token     = "ghp_test"
callback_port    = 8666
db_path          = "`+filepath.Join(t.TempDir(), "state.db")+`"
poll_interval    = "30s"
log_level        = "info"
shutdown_timeout = "30s"
bot_username     = "test-bot"
sandbox          = "auto"

[[project]]
id                = "alpha"
linear_project_id = "aaa-111"
github_owner      = "owner-a"
github_repo       = "repo-a"
repo_dir          = "`+repoA+`"
worktrees_root    = "`+wtA+`"

[[project]]
id                = "beta"
linear_project_id = "bbb-222"
github_owner      = "owner-b"
github_repo       = "repo-b"
repo_dir          = "`+repoB+`"
worktrees_root    = "`+wtB+`"
concurrency_limit = 3
`)
}

func TestLoadOrchestratorConfig_Valid(t *testing.T) {
	path := validTOMLConfig(t)
	cfg, err := LoadOrchestratorConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "lin_api_test", cfg.Global.LinearAPIKey)
	assert.Equal(t, 8666, cfg.Global.CallbackPort)
	assert.Len(t, cfg.Projects, 2)

	alpha := cfg.Projects[0]
	assert.Equal(t, "alpha", alpha.ID)
	assert.Equal(t, "main", alpha.BaseBranch)
	assert.Equal(t, "ai-managed", alpha.TrackingLabel)
	assert.Equal(t, 1, alpha.ConcurrencyLimit)

	beta := cfg.Projects[1]
	assert.Equal(t, "beta", beta.ID)
	assert.Equal(t, 3, beta.ConcurrencyLimit)
}

func TestLoadOrchestratorConfig_DefaultsApplied(t *testing.T) {
	path := validTOMLConfig(t)
	cfg, err := LoadOrchestratorConfig(path)
	require.NoError(t, err)

	for _, p := range cfg.Projects {
		assert.Equal(t, cfg.Global.PollInterval, p.PollInterval)
		assert.Equal(t, cfg.Global.CallbackPort, p.CallbackPort)
		assert.Equal(t, cfg.Global.BotUsername, p.BotUsername)
		assert.Equal(t, cfg.Global.Sandbox, p.SandboxMode)
	}
}

func TestLoadOrchestratorConfig_CredentialInheritance(t *testing.T) {
	path := validTOMLConfig(t)
	cfg, err := LoadOrchestratorConfig(path)
	require.NoError(t, err)

	for _, p := range cfg.Projects {
		assert.Equal(t, "lin_api_test", p.LinearAPIKey)
		assert.Equal(t, "ghp_test", p.GitHubToken)
	}
}

func TestLoadOrchestratorConfig_CredentialOverride(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755))
	wtDir := filepath.Join(t.TempDir(), "wt")

	path := writeTestTOML(t, `
[global]
linear_api_key   = "lin_global"
github_token     = "ghp_global"
callback_port    = 8666
db_path          = "`+filepath.Join(t.TempDir(), "state.db")+`"
poll_interval    = "30s"
log_level        = "info"
shutdown_timeout = "30s"
bot_username     = "bot"

[[project]]
id                = "proj"
linear_project_id = "aaa"
github_owner      = "owner"
github_repo       = "repo"
repo_dir          = "`+repoDir+`"
worktrees_root    = "`+wtDir+`"
github_token      = "ghp_override"
`)

	cfg, err := LoadOrchestratorConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "lin_global", cfg.Projects[0].LinearAPIKey)
	assert.Equal(t, "ghp_override", cfg.Projects[0].GitHubToken)
}

func TestLoadOrchestratorConfig_MissingRequiredFields(t *testing.T) {
	path := writeTestTOML(t, `
[global]
callback_port = 8666
log_level     = "info"
`)

	_, err := LoadOrchestratorConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "global.linear_api_key is required")
	assert.Contains(t, err.Error(), "global.github_token is required")
	assert.Contains(t, err.Error(), "global.db_path is required")
	assert.Contains(t, err.Error(), "global.poll_interval must be > 0")
	assert.Contains(t, err.Error(), "global.shutdown_timeout must be > 0")
	assert.Contains(t, err.Error(), "global.bot_username is required")
	assert.Contains(t, err.Error(), "at least one [[project]] is required")
}

func TestLoadOrchestratorConfig_DuplicateProjectIDs(t *testing.T) {
	repoA := filepath.Join(t.TempDir(), "repo-a")
	repoB := filepath.Join(t.TempDir(), "repo-b")
	require.NoError(t, os.MkdirAll(filepath.Join(repoA, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(repoB, ".git"), 0o755))

	path := writeTestTOML(t, `
[global]
linear_api_key   = "key"
github_token     = "tok"
callback_port    = 8666
db_path          = "state.db"
poll_interval    = "30s"
log_level        = "info"
shutdown_timeout = "30s"
bot_username     = "bot"

[[project]]
id                = "dupe"
linear_project_id = "aaa"
github_owner      = "owner-a"
github_repo       = "repo-a"
repo_dir          = "`+repoA+`"
worktrees_root    = "`+filepath.Join(t.TempDir(), "wt-a")+`"

[[project]]
id                = "dupe"
linear_project_id = "bbb"
github_owner      = "owner-b"
github_repo       = "repo-b"
repo_dir          = "`+repoB+`"
worktrees_root    = "`+filepath.Join(t.TempDir(), "wt-b")+`"
`)

	_, err := LoadOrchestratorConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate project id: dupe")
}

func TestLoadOrchestratorConfig_InvalidProjectID(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755))

	path := writeTestTOML(t, `
[global]
linear_api_key   = "key"
github_token     = "tok"
callback_port    = 8666
db_path          = "state.db"
poll_interval    = "30s"
log_level        = "info"
shutdown_timeout = "30s"
bot_username     = "bot"

[[project]]
id                = "Bad-ID"
linear_project_id = "aaa"
github_owner      = "owner"
github_repo       = "repo"
repo_dir          = "`+repoDir+`"
worktrees_root    = "`+filepath.Join(t.TempDir(), "wt")+`"
`)

	_, err := LoadOrchestratorConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must match [a-z0-9][a-z0-9-]*")
}

func TestLoadOrchestratorConfig_DuplicateRepo(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755))

	path := writeTestTOML(t, `
[global]
linear_api_key   = "key"
github_token     = "tok"
callback_port    = 8666
db_path          = "state.db"
poll_interval    = "30s"
log_level        = "info"
shutdown_timeout = "30s"
bot_username     = "bot"

[[project]]
id                = "proj-a"
linear_project_id = "aaa"
github_owner      = "owner"
github_repo       = "repo"
repo_dir          = "`+repoDir+`"
worktrees_root    = "`+filepath.Join(t.TempDir(), "wt-a")+`"

[[project]]
id                = "proj-b"
linear_project_id = "bbb"
github_owner      = "owner"
github_repo       = "repo"
repo_dir          = "`+repoDir+`"
worktrees_root    = "`+filepath.Join(t.TempDir(), "wt-b")+`"
`)

	_, err := LoadOrchestratorConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate (github_owner, github_repo): owner/repo")
}

func TestLoadOrchestratorConfig_RepoDirNoGit(t *testing.T) {
	repoDir := t.TempDir() // exists but no .git

	path := writeTestTOML(t, `
[global]
linear_api_key   = "key"
github_token     = "tok"
callback_port    = 8666
db_path          = "state.db"
poll_interval    = "30s"
log_level        = "info"
shutdown_timeout = "30s"
bot_username     = "bot"

[[project]]
id                = "proj"
linear_project_id = "aaa"
github_owner      = "owner"
github_repo       = "repo"
repo_dir          = "`+repoDir+`"
worktrees_root    = "`+filepath.Join(t.TempDir(), "wt")+`"
`)

	_, err := LoadOrchestratorConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain a .git directory")
}

func TestLoadOrchestratorConfig_WorktreesDirCreated(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755))
	wtDir := filepath.Join(t.TempDir(), "nested", "worktrees")

	path := writeTestTOML(t, `
[global]
linear_api_key   = "key"
github_token     = "tok"
callback_port    = 8666
db_path          = "state.db"
poll_interval    = "30s"
log_level        = "info"
shutdown_timeout = "30s"
bot_username     = "bot"

[[project]]
id                = "proj"
linear_project_id = "aaa"
github_owner      = "owner"
github_repo       = "repo"
repo_dir          = "`+repoDir+`"
worktrees_root    = "`+wtDir+`"
`)

	_, err := LoadOrchestratorConfig(path)
	require.NoError(t, err)

	info, err := os.Stat(wtDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDuration_UnmarshalText(t *testing.T) {
	var d Duration
	require.NoError(t, d.UnmarshalText([]byte("30s")))
	assert.Equal(t, 30*1000*1000*1000, int(d.Duration))

	require.NoError(t, d.UnmarshalText([]byte("2m")))
	assert.Equal(t, 2*60*1000*1000*1000, int(d.Duration))

	err := d.UnmarshalText([]byte("not-a-duration"))
	assert.Error(t, err)
}

func TestLoadOrchestratorConfig_BadTOML(t *testing.T) {
	path := writeTestTOML(t, `[[[broken`)
	_, err := LoadOrchestratorConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode config")
}
