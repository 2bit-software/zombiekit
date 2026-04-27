package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

var projectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Duration wraps time.Duration for TOML string decoding.
// BurntSushi/toml v1.6 does not natively decode time.Duration from strings.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

// OrchestratorConfig is the top-level TOML config for multi-project mode.
type OrchestratorConfig struct {
	Global   GlobalConfig    `toml:"global"`
	Projects []ProjectConfig `toml:"project"`
}

// GlobalConfig holds shared settings inherited by all projects.
type GlobalConfig struct {
	LinearAPIKey    string   `toml:"linear_api_key"`
	GitHubToken     string   `toml:"github_token"`
	CallbackPort    int      `toml:"callback_port"`
	DBPath          string   `toml:"db_path"`
	PollInterval    Duration `toml:"poll_interval"`
	LogLevel        string   `toml:"log_level"`
	LogJSON         bool     `toml:"log_json"`
	ShutdownTimeout Duration `toml:"shutdown_timeout"`
	BotUsername     string   `toml:"bot_username"`
	Sandbox         string   `toml:"sandbox"`
}

// ProjectConfig holds per-project settings.
type ProjectConfig struct {
	ID               string   `toml:"id"`
	LinearProjectID  string   `toml:"linear_project_id"`
	GitHubOwner      string   `toml:"github_owner"`
	GitHubRepo       string   `toml:"github_repo"`
	RepoDir          string   `toml:"repo_dir"`
	WorktreesRoot    string   `toml:"worktrees_root"`
	BaseBranch       string   `toml:"base_branch"`
	ConcurrencyLimit int      `toml:"concurrency_limit"`
	TrackingLabel    string   `toml:"tracking_label"`
	CopyFiles        []string `toml:"copy_files"`
	ClosedPRStatus   string   `toml:"closed_pr_status"`

	// Per-project credential overrides (optional, falls back to global).
	LinearAPIKey string `toml:"linear_api_key"`
	GitHubToken  string `toml:"github_token"`

	// Inherited from global during applyDefaults. Not in TOML.
	PollInterval Duration `toml:"-"`
	CallbackPort int      `toml:"-"`
	BotUsername  string   `toml:"-"`
	SandboxMode  string   `toml:"-"`
}

// LoadOrchestratorConfig reads and validates a TOML config file.
func LoadOrchestratorConfig(path string) (*OrchestratorConfig, error) {
	var cfg OrchestratorConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("decode config %s: %w", path, err)
	}
	applyDefaults(&cfg)
	inheritCredentials(&cfg)
	if err := validateOrchestratorConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyDefaults(cfg *OrchestratorConfig) {
	for i := range cfg.Projects {
		p := &cfg.Projects[i]
		if p.BaseBranch == "" {
			p.BaseBranch = "main"
		}
		if p.TrackingLabel == "" {
			p.TrackingLabel = "ai-managed"
		}
		if p.ConcurrencyLimit == 0 {
			p.ConcurrencyLimit = 1
		}
		p.PollInterval = cfg.Global.PollInterval
		p.CallbackPort = cfg.Global.CallbackPort
		p.BotUsername = cfg.Global.BotUsername
		p.SandboxMode = cfg.Global.Sandbox
	}
}

func inheritCredentials(cfg *OrchestratorConfig) {
	for i := range cfg.Projects {
		p := &cfg.Projects[i]
		if p.LinearAPIKey == "" {
			p.LinearAPIKey = cfg.Global.LinearAPIKey
		}
		if p.GitHubToken == "" {
			p.GitHubToken = cfg.Global.GitHubToken
		}
	}
}

type orchestratorConfigRule struct {
	msg   string
	check func(*OrchestratorConfig) bool
}

var orchestratorConfigRules = []orchestratorConfigRule{
	{"global.linear_api_key is required", func(c *OrchestratorConfig) bool { return c.Global.LinearAPIKey == "" }},
	{"global.github_token is required", func(c *OrchestratorConfig) bool { return c.Global.GitHubToken == "" }},
	{"global.callback_port must be 1-65535", func(c *OrchestratorConfig) bool {
		return c.Global.CallbackPort < 1 || c.Global.CallbackPort > 65535
	}},
	{"global.db_path is required", func(c *OrchestratorConfig) bool { return c.Global.DBPath == "" }},
	{"global.poll_interval must be > 0", func(c *OrchestratorConfig) bool { return c.Global.PollInterval.Duration <= 0 }},
	{"global.log_level must be one of: debug, info, warn, error", func(c *OrchestratorConfig) bool {
		return !validLogLevels[c.Global.LogLevel]
	}},
	{"global.shutdown_timeout must be > 0", func(c *OrchestratorConfig) bool { return c.Global.ShutdownTimeout.Duration <= 0 }},
	{"global.bot_username is required", func(c *OrchestratorConfig) bool { return c.Global.BotUsername == "" }},
	{"at least one [[project]] is required", func(c *OrchestratorConfig) bool { return len(c.Projects) == 0 }},
}

func validateOrchestratorConfig(cfg *OrchestratorConfig) error {
	var errs []string

	for _, r := range orchestratorConfigRules {
		if r.check(cfg) {
			errs = append(errs, r.msg)
		}
	}

	seen := make(map[string]bool)
	repos := make(map[string]bool)

	for i, p := range cfg.Projects {
		prefix := fmt.Sprintf("project[%d]", i)

		if p.ID == "" {
			errs = append(errs, prefix+".id is required")
		} else if !projectIDPattern.MatchString(p.ID) {
			errs = append(errs, prefix+".id must match [a-z0-9][a-z0-9-]*, got "+p.ID)
		} else if seen[p.ID] {
			errs = append(errs, "duplicate project id: "+p.ID)
		} else {
			seen[p.ID] = true
		}

		if p.LinearProjectID == "" {
			errs = append(errs, prefix+".linear_project_id is required")
		}
		if p.GitHubOwner == "" {
			errs = append(errs, prefix+".github_owner is required")
		}
		if p.GitHubRepo == "" {
			errs = append(errs, prefix+".github_repo is required")
		}
		if p.RepoDir == "" {
			errs = append(errs, prefix+".repo_dir is required")
		}
		if p.WorktreesRoot == "" {
			errs = append(errs, prefix+".worktrees_root is required")
		}

		repoKey := p.GitHubOwner + "/" + p.GitHubRepo
		if p.GitHubOwner != "" && p.GitHubRepo != "" {
			if repos[repoKey] {
				errs = append(errs, "duplicate (github_owner, github_repo): "+repoKey)
			}
			repos[repoKey] = true
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}

	// Filesystem checks (only after field validation passes).
	for _, p := range cfg.Projects {
		if err := os.MkdirAll(p.WorktreesRoot, 0o755); err != nil {
			return fmt.Errorf("create worktrees directory for %s: %w", p.ID, err)
		}
		gitPath := filepath.Join(p.RepoDir, ".git")
		if _, err := os.Stat(gitPath); err != nil {
			return fmt.Errorf("project %s: repo_dir %q does not contain a .git directory: %w", p.ID, p.RepoDir, err)
		}
	}

	return nil
}
