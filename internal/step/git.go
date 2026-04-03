package step

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// GitService provides git branch management with graceful degradation.
type GitService struct {
	workDir string
}

// NewGitService creates a new GitService for the given working directory.
func NewGitService(workDir string) *GitService {
	return &GitService{workDir: workDir}
}

// EnsureBranch creates or switches to a branch for the initiative.
// If git is not available or the directory is not a git repository,
// it returns nil (graceful degradation).
func (g *GitService) EnsureBranch(initType, name string) error {
	if !g.isGitAvailable() || !g.isGitRepository() {
		return nil // Graceful degradation
	}

	branchName, err := g.formatBranchName(initType, name)
	if err != nil {
		return err
	}

	if g.branchExists(branchName) {
		return g.switchToBranch(branchName)
	}
	return g.createBranch(branchName)
}

// isGitAvailable checks if the git command is available in PATH.
func (g *GitService) isGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// isGitRepository checks if the working directory is a git repository.
func (g *GitService) isGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = g.workDir
	return cmd.Run() == nil
}

// branchExists checks if the branch already exists locally.
func (g *GitService) branchExists(branchName string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	cmd.Dir = g.workDir
	return cmd.Run() == nil
}

// switchToBranch switches to an existing branch.
func (g *GitService) switchToBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = g.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("switching to branch %s: %w\nOutput: %s", branchName, err, output)
	}
	return nil
}

// createBranch creates and switches to a new branch.
func (g *GitService) createBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = g.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating branch %s: %w\nOutput: %s", branchName, err, output)
	}
	return nil
}

// EnsureBranchGraphite creates a branch using graphite stacking, with fallback to git.
// Returns:
//   - method: "graphite", "git", or "" (graceful degradation)
//   - warning: non-empty when graphite failed but git succeeded (non-fatal)
//   - err: only for truly fatal errors (both graphite and git failed)
func (g *GitService) EnsureBranchGraphite(initType, name string) (method, warning string, err error) {
	if !g.isGitAvailable() || !g.isGitRepository() {
		return "", "", nil
	}

	branchName, err := g.formatBranchName(initType, name)
	if err != nil {
		return "", "", err
	}

	if g.branchExists(branchName) {
		if switchErr := g.switchToBranch(branchName); switchErr != nil {
			return "", "", switchErr
		}
		// Best-effort: track existing branch with graphite
		if g.isGraphiteAvailable() {
			_ = g.trackBranchGraphite()
		}
		return "git", "", nil
	}

	if !g.isGraphiteAvailable() {
		if createErr := g.createBranch(branchName); createErr != nil {
			return "", "", createErr
		}
		return "git", "", nil
	}

	if gtErr := g.createBranchGraphite(branchName); gtErr != nil {
		// Graphite failed — fall back to git
		if createErr := g.createBranch(branchName); createErr != nil {
			return "", "", fmt.Errorf("graphite failed: %w; git fallback also failed: %w", gtErr, createErr)
		}
		return "git", fmt.Sprintf("graphite branch creation failed: %v; fell back to git", gtErr), nil
	}

	return "graphite", "", nil
}

// isGraphiteAvailable checks if the gt CLI is in PATH.
func (g *GitService) isGraphiteAvailable() bool {
	_, err := exec.LookPath("gt")
	return err == nil
}

// createBranchGraphite creates a branch using gt create.
func (g *GitService) createBranchGraphite(branchName string) error {
	cmd := exec.Command("gt", "create", branchName, "--no-interactive")
	cmd.Dir = g.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gt create %s: %w\nOutput: %s", branchName, err, output)
	}
	return nil
}

// trackBranchGraphite attempts to track the current branch with graphite.
// Best-effort: errors are silently ignored.
func (g *GitService) trackBranchGraphite() error {
	cmd := exec.Command("gt", "track", "--force", "--no-interactive")
	cmd.Dir = g.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gt track: %w\nOutput: %s", err, output)
	}
	return nil
}

// formatBranchName formats the branch name based on initiative type.
// Maps: feature → feat/, bug → fix/, refactor → ref/
func (g *GitService) formatBranchName(initType, name string) (string, error) {
	// Normalize name to slug format
	slug := strings.ToLower(name)
	slug = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	if slug == "" {
		return "", fmt.Errorf("name normalizes to empty string")
	}

	prefixMap := map[string]string{
		"feature":  "feat",
		"bug":      "fix",
		"refactor": "ref",
	}

	prefix, ok := prefixMap[initType]
	if !ok {
		return "", fmt.Errorf("unknown initiative type '%s'", initType)
	}

	return fmt.Sprintf("%s/%s", prefix, slug), nil
}
