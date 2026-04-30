package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/2bit-software/zombiekit/internal/orchestrator"
	"github.com/urfave/cli/v2"
)

// configFlag returns the standard --config flag used by worktree, sandbox, and
// workspace subcommands. Mirrors cmd/orchestrator/main.go conventions but is
// not Required so commands can fall back to cwd auto-detection.
func configFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "Path to orchestrator TOML config (default: ./orchestrator.toml if present, else cwd auto-detect)",
		EnvVars: []string{"ORCH_CONFIG", "BRAINS_CONFIG"},
	}
}

// projectFlag returns the --project flag used to disambiguate when a config
// holds multiple projects.
func projectFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    "project",
		Aliases: []string{"p"},
		Usage:   "Project ID (required when --config has multiple projects and cwd does not match one)",
		EnvVars: []string{"BRAINS_PROJECT"},
	}
}

// loadProjectConfig resolves a single ProjectConfig the CLI subcommand should
// operate on. Resolution order:
//  1. If --config is set, load it and select the project named by --project,
//     or auto-pick if there's only one, or auto-pick by cwd-match.
//  2. If --config is unset and ./orchestrator.toml exists, treat it as if
//     --config pointed at it.
//  3. Otherwise return a synthetic ProjectConfig derived from the current
//     working directory: RepoDir = nearest .git ancestor, WorktreesRoot =
//     {RepoDir}/../worktrees, no copy_files, ID = "default". This lets
//     `brains worktree create DEV-1 "title"` work in any git repo without
//     needing a daemon config.
func loadProjectConfig(c *cli.Context) (*orchestrator.ProjectConfig, error) {
	configPath := c.String("config")
	if configPath == "" {
		if _, err := os.Stat("orchestrator.toml"); err == nil {
			configPath = "orchestrator.toml"
		}
	}

	if configPath == "" {
		return autoDetectProject()
	}

	cfg, err := orchestrator.LoadOrchestratorConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config %s: %w", configPath, err)
	}

	return selectProject(cfg, c.String("project"))
}

// selectProject picks one ProjectConfig from a multi-project config.
// Disambiguation: explicit --project flag wins; then cwd-match against
// project.RepoDir; then single-project default; otherwise error listing IDs.
func selectProject(cfg *orchestrator.OrchestratorConfig, projectID string) (*orchestrator.ProjectConfig, error) {
	if projectID != "" {
		for i := range cfg.Projects {
			if cfg.Projects[i].ID == projectID {
				return &cfg.Projects[i], nil
			}
		}
		return nil, fmt.Errorf("project %q not found in config; available: %s",
			projectID, strings.Join(projectIDs(cfg), ", "))
	}

	cwd, _ := os.Getwd()
	if cwd != "" {
		if p := matchProjectByCwd(cfg, cwd); p != nil {
			return p, nil
		}
	}

	if len(cfg.Projects) == 1 {
		return &cfg.Projects[0], nil
	}

	return nil, fmt.Errorf("config has %d projects; specify one with --project (available: %s)",
		len(cfg.Projects), strings.Join(projectIDs(cfg), ", "))
}

// matchProjectByCwd returns the project whose RepoDir contains cwd, or nil if none.
func matchProjectByCwd(cfg *orchestrator.OrchestratorConfig, cwd string) *orchestrator.ProjectConfig {
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return nil
	}
	for i := range cfg.Projects {
		repoAbs, err := filepath.Abs(cfg.Projects[i].RepoDir)
		if err != nil {
			continue
		}
		if cwdAbs == repoAbs || strings.HasPrefix(cwdAbs, repoAbs+string(filepath.Separator)) {
			return &cfg.Projects[i]
		}
	}
	return nil
}

func projectIDs(cfg *orchestrator.OrchestratorConfig) []string {
	ids := make([]string, len(cfg.Projects))
	for i := range cfg.Projects {
		ids[i] = cfg.Projects[i].ID
	}
	return ids
}

// autoDetectProject builds a synthetic ProjectConfig from the current
// working directory's git repo, suitable for ad-hoc CLI use without a
// daemon config. Returns ErrNoGitRepo if cwd is not inside a git repo.
func autoDetectProject() (*orchestrator.ProjectConfig, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git not found on PATH: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting cwd: %w", err)
	}

	repoDir, err := findGitRoot(cwd)
	if err != nil {
		return nil, err
	}

	worktreesRoot := filepath.Join(filepath.Dir(repoDir), "worktrees")

	return &orchestrator.ProjectConfig{
		ID:            "default",
		RepoDir:       repoDir,
		WorktreesRoot: worktreesRoot,
		BaseBranch:    "main",
	}, nil
}

// ErrNoGitRepo is returned when auto-detection cannot find a git repository
// from the current working directory.
var ErrNoGitRepo = errors.New("no git repository found from cwd; pass --config or run from inside a git repo")

// findGitRoot walks up from start looking for a .git directory. Returns
// ErrNoGitRepo if none is found before reaching the filesystem root.
func findGitRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && (info.IsDir() || !info.IsDir()) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNoGitRepo
		}
		dir = parent
	}
}
