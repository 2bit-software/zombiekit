package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	internalgit "github.com/zombiekit/brains/internal/git"
)

// ToolDefinition represents an MCP tool definition.
type ToolDefinition struct {
	Name        string
	Description string
}

// Tool implements the MCP git tool for local repository operations.
type Tool struct {
	runner *internalgit.Runner
}

// NewTool creates a new git tool backed by the given runner.
func NewTool(runner *internalgit.Runner) *Tool {
	return &Tool{runner: runner}
}

// Definition returns the tool definition for MCP registration.
func (t *Tool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "git",
		Description: "Local git operations: status, log, diff, stage, commit, push. All operations run in the server's working directory.",
	}
}

// Execute runs the git tool and returns the response as JSON.
func (t *Tool) Execute(ctx context.Context, args map[string]any) (string, error) {
	action := getStringArg(args, "action")
	if action == "" {
		return "", &ToolError{
			Code:    "MISSING_REQUIRED_PARAM",
			Message: "missing required parameter: action",
			Hint:    "Provide action (status|log|diff|stage|commit|push)",
		}
	}

	switch action {
	case "status":
		return t.handleStatus(ctx)
	case "log":
		return t.handleLog(ctx, args)
	case "diff":
		return t.handleDiff(ctx, args)
	case "stage":
		return t.handleStage(ctx, args)
	case "commit":
		return t.handleCommit(ctx, args)
	case "push":
		return t.handlePush(ctx, args)
	default:
		return "", &ToolError{
			Code:    "INVALID_ACTION",
			Message: fmt.Sprintf("invalid action: %q", action),
			Hint:    "Valid actions: status, log, diff, stage, commit, push",
		}
	}
}

// handleStatus returns branch, status lines, tracking info, and staged changes flag.
func (t *Tool) handleStatus(ctx context.Context) (string, error) {
	branch, err := t.runner.Run(ctx, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("getting branch: %w", err)
	}

	statusOut, err := t.runner.Run(ctx, "status", "--short")
	if err != nil {
		return "", fmt.Errorf("getting status: %w", err)
	}

	var statusLines []string
	if statusOut != "" {
		statusLines = strings.Split(statusOut, "\n")
	}

	// Check for staged changes: `git diff --cached --quiet` exits non-zero if staged changes exist
	hasStagedChanges := t.runner.RunSilent(ctx, "diff", "--cached", "--quiet") != nil

	// Get tracking info
	trackingOut, _ := t.runner.Run(ctx, "status", "-sb")
	trackingInfo := ""
	if trackingOut != "" {
		lines := strings.SplitN(trackingOut, "\n", 2)
		trackingInfo = lines[0]
	}

	return marshalResponse(StatusResponse{
		Action:           "status",
		Branch:           branch,
		StatusLines:      statusLines,
		HasStagedChanges: hasStagedChanges,
		TrackingInfo:     trackingInfo,
	})
}

// handleLog returns recent commits.
func (t *Tool) handleLog(ctx context.Context, args map[string]any) (string, error) {
	count := getIntArg(args, "count", 10)
	rangeSpec := getStringArg(args, "range")
	base := getStringArg(args, "base")

	gitArgs := []string{"log", "--oneline", fmt.Sprintf("-%d", count)}

	if rangeSpec != "" {
		gitArgs = []string{"log", "--oneline", rangeSpec}
	} else if base != "" {
		gitArgs = []string{"log", "--oneline", fmt.Sprintf("%s..HEAD", base)}
	}

	out, err := t.runner.Run(ctx, gitArgs...)
	if err != nil {
		return "", fmt.Errorf("getting log: %w", err)
	}

	var commits []LogEntry
	if out != "" {
		for _, line := range strings.Split(out, "\n") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				commits = append(commits, LogEntry{Hash: parts[0], Message: parts[1]})
			}
		}
	}

	return marshalResponse(LogResponse{
		Action:  "log",
		Commits: commits,
		Count:   len(commits),
	})
}

// handleDiff returns diff content.
func (t *Tool) handleDiff(ctx context.Context, args map[string]any) (string, error) {
	scope := getStringArg(args, "scope")
	base := getStringArg(args, "base")
	statOnly := getBoolArg(args, "stat_only")
	pathsStr := getStringArg(args, "paths")

	var gitArgs []string

	if base != "" {
		gitArgs = []string{"diff", fmt.Sprintf("%s...HEAD", base)}
	} else {
		switch scope {
		case "staged":
			gitArgs = []string{"diff", "--cached"}
		case "unstaged":
			gitArgs = []string{"diff"}
		case "all", "":
			gitArgs = []string{"diff", "HEAD"}
		default:
			return "", &ToolError{
				Code:    "VALIDATION_ERROR",
				Message: fmt.Sprintf("invalid scope: %q", scope),
				Hint:    "Valid scopes: all, staged, unstaged",
			}
		}
	}

	if statOnly {
		gitArgs = append(gitArgs, "--stat")
	}

	if pathsStr != "" {
		gitArgs = append(gitArgs, "--")
		for _, p := range strings.Split(pathsStr, ",") {
			gitArgs = append(gitArgs, strings.TrimSpace(p))
		}
	}

	out, err := t.runner.Run(ctx, gitArgs...)
	if err != nil {
		return "", fmt.Errorf("getting diff: %w", err)
	}

	return marshalResponse(DiffResponse{
		Action:   "diff",
		Content:  out,
		StatOnly: statOnly,
	})
}

// handleStage validates and stages specific files.
func (t *Tool) handleStage(ctx context.Context, args map[string]any) (string, error) {
	filesStr := getStringArg(args, "files")
	if filesStr == "" {
		return "", &ToolError{
			Code:    "MISSING_REQUIRED_PARAM",
			Message: "missing required parameter: files",
			Hint:    "Provide comma-separated file paths to stage",
		}
	}

	var files []string
	for _, f := range strings.Split(filesStr, ",") {
		trimmed := strings.TrimSpace(f)
		if trimmed != "" {
			files = append(files, trimmed)
		}
	}

	if err := validateFiles(t.runner.WorkDir(), files); err != nil {
		return "", err
	}

	gitArgs := append([]string{"add", "--"}, files...)
	if _, err := t.runner.Run(ctx, gitArgs...); err != nil {
		return "", fmt.Errorf("staging files: %w", err)
	}

	return marshalResponse(StageResponse{
		Action:      "stage",
		StagedFiles: files,
	})
}

// handleCommit creates a commit with the given message.
func (t *Tool) handleCommit(ctx context.Context, args map[string]any) (string, error) {
	message := getStringArg(args, "message")
	if err := validateMessage(message); err != nil {
		return "", err
	}

	// Verify staged changes exist
	if err := t.runner.RunSilent(ctx, "diff", "--cached", "--quiet"); err == nil {
		return "", &ToolError{
			Code:    "NO_STAGED_CHANGES",
			Message: "nothing staged for commit",
			Hint:    "Stage files first with action=stage",
		}
	}

	// Write message to temp file
	tmpFile, err := os.CreateTemp("", "git-commit-msg-*.txt")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(message); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("writing commit message: %w", err)
	}
	tmpFile.Close()

	// Commit
	if _, err := t.runner.Run(ctx, "commit", "-F", tmpPath); err != nil {
		return "", fmt.Errorf("committing: %w", err)
	}

	// Get commit info
	hash, _ := t.runner.Run(ctx, "rev-parse", "--short", "HEAD")
	branch, _ := t.runner.Run(ctx, "branch", "--show-current")
	logLine, _ := t.runner.Run(ctx, "log", "--oneline", "-1")

	return marshalResponse(CommitResponse{
		Action:  "commit",
		Hash:    hash,
		Branch:  branch,
		Summary: logLine,
	})
}

// handlePush pushes the current branch to a remote.
func (t *Tool) handlePush(ctx context.Context, args map[string]any) (string, error) {
	setUpstream := getBoolArg(args, "set_upstream")
	remote := getStringArg(args, "remote")
	if remote == "" {
		remote = "origin"
	}

	branch, err := t.runner.Run(ctx, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("getting branch: %w", err)
	}

	if err := validateBranchForPush(branch); err != nil {
		return "", err
	}

	gitArgs := []string{"push"}
	if setUpstream {
		gitArgs = append(gitArgs, "-u")
	}
	gitArgs = append(gitArgs, remote, branch)

	out, pushErr := t.runner.Run(ctx, gitArgs...)

	// git push often writes to stderr even on success, so check if it actually failed
	if pushErr != nil {
		return "", fmt.Errorf("pushing: %w", pushErr)
	}

	absPath, _ := filepath.Abs(t.runner.WorkDir())

	return marshalResponse(PushResponse{
		Action:  "push",
		Success: true,
		Remote:  remote,
		Branch:  branch,
		Output:  fmt.Sprintf("pushed %s to %s (%s)", branch, remote, absPath) + "\n" + out,
	})
}
