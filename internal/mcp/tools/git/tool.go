package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	internalgit "github.com/2bit-software/zombiekit/internal/git"
)

// resolveRunner returns the effective runner for this call.
// If directory is specified in args, creates a new runner for that directory.
// Otherwise returns the tool's default runner.
func (t *Tool) resolveRunner(ctx context.Context, args map[string]any) (*internalgit.Runner, error) {
	dir := getStringArg(args, "directory")
	if dir == "" {
		return t.runner, nil
	}

	if !filepath.IsAbs(dir) {
		dir = filepath.Join(t.runner.WorkDir(), dir)
	}

	info, err := os.Stat(dir)
	if err != nil {
		return nil, &ToolError{
			Code:    "INVALID_DIRECTORY",
			Message: fmt.Sprintf("directory does not exist: %s", dir),
			Hint:    "Provide a valid absolute or relative directory path",
		}
	}
	if !info.IsDir() {
		return nil, &ToolError{
			Code:    "INVALID_DIRECTORY",
			Message: fmt.Sprintf("path is not a directory: %s", dir),
			Hint:    "Provide a path to a directory, not a file",
		}
	}

	runner, err := internalgit.NewRunner(dir)
	if err != nil {
		return nil, fmt.Errorf("creating runner: %w", err)
	}

	if _, err := runner.Run(ctx, "rev-parse", "--git-dir"); err != nil {
		return nil, &ToolError{
			Code:    "NOT_GIT_REPOSITORY",
			Message: fmt.Sprintf("not a git repository: %s", dir),
			Hint:    "Provide a path to a directory inside a git repository",
		}
	}

	return runner, nil
}

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
		Description: "Local git operations: status, log, diff, stage, commit, push. Operations run in the server's working directory by default, or in the specified directory.",
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

	runner, err := t.resolveRunner(ctx, args)
	if err != nil {
		return "", err
	}

	switch action {
	case "status":
		return t.handleStatus(ctx, runner)
	case "log":
		return t.handleLog(ctx, args, runner)
	case "diff":
		return t.handleDiff(ctx, args, runner)
	case "stage":
		return t.handleStage(ctx, args, runner)
	case "commit":
		return t.handleCommit(ctx, args, runner)
	case "push":
		return t.handlePush(ctx, args, runner)
	default:
		return "", &ToolError{
			Code:    "INVALID_ACTION",
			Message: fmt.Sprintf("invalid action: %q", action),
			Hint:    "Valid actions: status, log, diff, stage, commit, push",
		}
	}
}

// handleStatus returns branch, status lines, tracking info, and staged changes flag.
func (t *Tool) handleStatus(ctx context.Context, runner *internalgit.Runner) (string, error) {
	branch, err := runner.Run(ctx, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("getting branch: %w", err)
	}

	statusOut, err := runner.Run(ctx, "status", "--short")
	if err != nil {
		return "", fmt.Errorf("getting status: %w", err)
	}

	var statusLines []string
	if statusOut != "" {
		statusLines = strings.Split(statusOut, "\n")
	}

	// Check for staged changes: `git diff --cached --quiet` exits non-zero if staged changes exist
	hasStagedChanges := runner.RunSilent(ctx, "diff", "--cached", "--quiet") != nil

	// Get tracking info
	trackingOut, _ := runner.Run(ctx, "status", "-sb")
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
func (t *Tool) handleLog(ctx context.Context, args map[string]any, runner *internalgit.Runner) (string, error) {
	count := getIntArg(args, "count", 10)
	rangeSpec := getStringArg(args, "range")
	base := getStringArg(args, "base")

	gitArgs := []string{"log", "--oneline", fmt.Sprintf("-%d", count)}

	if rangeSpec != "" {
		gitArgs = []string{"log", "--oneline", rangeSpec}
	} else if base != "" {
		gitArgs = []string{"log", "--oneline", fmt.Sprintf("%s..HEAD", base)}
	}

	out, err := runner.Run(ctx, gitArgs...)
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
func (t *Tool) handleDiff(ctx context.Context, args map[string]any, runner *internalgit.Runner) (string, error) {
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

	out, err := runner.Run(ctx, gitArgs...)
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
func (t *Tool) handleStage(ctx context.Context, args map[string]any, runner *internalgit.Runner) (string, error) {
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

	if err := validateFiles(runner.WorkDir(), files); err != nil {
		return "", err
	}

	gitArgs := append([]string{"add", "--"}, files...)
	if _, err := runner.Run(ctx, gitArgs...); err != nil {
		return "", fmt.Errorf("staging files: %w", err)
	}

	return marshalResponse(StageResponse{
		Action:      "stage",
		StagedFiles: files,
	})
}

// handleCommit creates a commit with the given message.
func (t *Tool) handleCommit(ctx context.Context, args map[string]any, runner *internalgit.Runner) (string, error) {
	message := getStringArg(args, "message")
	if err := validateMessage(message); err != nil {
		return "", err
	}

	// Verify staged changes exist
	if err := runner.RunSilent(ctx, "diff", "--cached", "--quiet"); err == nil {
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
	if _, err := runner.Run(ctx, "commit", "-F", tmpPath); err != nil {
		return "", fmt.Errorf("committing: %w", err)
	}

	// Get commit info
	hash, _ := runner.Run(ctx, "rev-parse", "--short", "HEAD")
	branch, _ := runner.Run(ctx, "branch", "--show-current")
	logLine, _ := runner.Run(ctx, "log", "--oneline", "-1")

	return marshalResponse(CommitResponse{
		Action:  "commit",
		Hash:    hash,
		Branch:  branch,
		Summary: logLine,
	})
}

// handlePush pushes the current branch to a remote.
func (t *Tool) handlePush(ctx context.Context, args map[string]any, runner *internalgit.Runner) (string, error) {
	setUpstream := getBoolArg(args, "set_upstream")
	remote := getStringArg(args, "remote")
	if remote == "" {
		remote = "origin"
	}

	branch, err := runner.Run(ctx, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("getting branch: %w", err)
	}

	gitArgs := []string{"push"}
	if setUpstream {
		gitArgs = append(gitArgs, "-u")
	}
	gitArgs = append(gitArgs, remote, branch)

	out, pushErr := runner.Run(ctx, gitArgs...)

	// git push often writes to stderr even on success, so check if it actually failed
	if pushErr != nil {
		return "", fmt.Errorf("pushing: %w", pushErr)
	}

	absPath, _ := filepath.Abs(runner.WorkDir())

	return marshalResponse(PushResponse{
		Action:  "push",
		Success: true,
		Remote:  remote,
		Branch:  branch,
		Output:  fmt.Sprintf("pushed %s to %s (%s)", branch, remote, absPath) + "\n" + out,
	})
}
