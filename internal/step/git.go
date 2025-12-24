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
