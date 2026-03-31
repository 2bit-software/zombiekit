package ghpr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ToolDefinition represents an MCP tool definition.
type ToolDefinition struct {
	Name        string
	Description string
}

// Tool implements the MCP gh-pr tool for GitHub PR operations.
type Tool struct {
	ghBin   string
	workDir string
}

// NewTool creates a new gh-pr tool. Returns an error if gh CLI is not found.
func NewTool(workDir string) (*Tool, error) {
	ghBin, err := exec.LookPath("gh")
	if err != nil {
		return nil, &ToolError{
			Code:    "GH_NOT_FOUND",
			Message: "gh CLI not found on PATH",
			Hint:    "Install: https://cli.github.com",
		}
	}
	return &Tool{ghBin: ghBin, workDir: workDir}, nil
}

// Definition returns the tool definition for MCP registration.
func (t *Tool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "gh-pr",
		Description: "GitHub PR operations via the gh CLI: view (check if PR exists), create (open a new PR), comment (add a comment to a PR).",
	}
}

// Execute runs the gh-pr tool and returns the response as JSON.
func (t *Tool) Execute(ctx context.Context, args map[string]any) (string, error) {
	action := getStringArg(args, "action")
	if action == "" {
		return "", &ToolError{
			Code:    "MISSING_REQUIRED_PARAM",
			Message: "missing required parameter: action",
			Hint:    "Provide action (view|create|comment)",
		}
	}

	switch action {
	case "view":
		return t.handleView(ctx)
	case "create":
		return t.handleCreate(ctx, args)
	case "comment":
		return t.handleComment(ctx, args)
	default:
		return "", &ToolError{
			Code:    "INVALID_ACTION",
			Message: fmt.Sprintf("invalid action: %q", action),
			Hint:    "Valid actions: view, create, comment",
		}
	}
}

// run executes a gh command and returns stdout.
func (t *Tool) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, t.ghBin, args...)
	cmd.Dir = t.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// handleView checks if a PR exists for the current branch.
func (t *Tool) handleView(ctx context.Context) (string, error) {
	out, err := t.run(ctx, "pr", "view", "--json", "url,title,number,state")
	if err != nil {
		// No PR found is not an error -- return exists=false
		return marshalResponse(ViewResponse{
			Action: "view",
			Exists: false,
		})
	}

	var prData struct {
		URL    string `json:"url"`
		Title  string `json:"title"`
		Number int    `json:"number"`
		State  string `json:"state"`
	}
	if err := json.Unmarshal([]byte(out), &prData); err != nil {
		return "", fmt.Errorf("parsing pr view output: %w", err)
	}

	return marshalResponse(ViewResponse{
		Action: "view",
		Exists: true,
		URL:    prData.URL,
		Title:  prData.Title,
		Number: prData.Number,
		State:  prData.State,
	})
}

// createArgs holds validated arguments for PR creation.
type createArgs struct {
	title string
	body  string
	base  string
	draft bool
}

// parseCreateArgs validates and extracts the arguments needed for PR creation.
func parseCreateArgs(args map[string]any) (createArgs, error) {
	title := getStringArg(args, "title")
	if strings.TrimSpace(title) == "" {
		return createArgs{}, &ToolError{
			Code:    "VALIDATION_ERROR",
			Message: "title is required and must not be empty",
		}
	}

	body := getStringArg(args, "body")
	if strings.TrimSpace(body) == "" {
		return createArgs{}, &ToolError{
			Code:    "VALIDATION_ERROR",
			Message: "body is required and must not be empty",
		}
	}

	base := getStringArg(args, "base")
	if base == "" {
		base = "main"
	}

	return createArgs{
		title: title,
		body:  body,
		base:  base,
		draft: getBoolArg(args, "draft"),
	}, nil
}

// checkExistingPR returns an error if a PR already exists for the current branch.
func (t *Tool) checkExistingPR(ctx context.Context) error {
	existing, _ := t.run(ctx, "pr", "view", "--json", "url")
	if existing == "" {
		return nil
	}
	var prData struct {
		URL string `json:"url"`
	}
	if json.Unmarshal([]byte(existing), &prData) == nil && prData.URL != "" {
		return &ToolError{
			Code:    "PR_EXISTS",
			Message: fmt.Sprintf("PR already exists for this branch: %s", prData.URL),
		}
	}
	return nil
}

// handleCreate creates a new PR.
func (t *Tool) handleCreate(ctx context.Context, args map[string]any) (string, error) {
	ca, err := parseCreateArgs(args)
	if err != nil {
		return "", err
	}

	if err := t.checkExistingPR(ctx); err != nil {
		return "", err
	}

	// Write body to temp file to avoid shell escaping issues
	tmpFile, err := os.CreateTemp("", "gh-pr-body-*.md")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(ca.body); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("writing PR body: %w", err)
	}
	tmpFile.Close()

	ghArgs := []string{"pr", "create", "--base", ca.base, "--title", ca.title, "--body-file", tmpPath}
	if ca.draft {
		ghArgs = append(ghArgs, "--draft")
	}

	out, err := t.run(ctx, ghArgs...)
	if err != nil {
		return "", fmt.Errorf("creating PR: %w", err)
	}

	// Try to get structured data from the newly created PR
	prInfo, infoErr := t.run(ctx, "pr", "view", "--json", "url,title,number")
	if infoErr == nil {
		var prData struct {
			URL    string `json:"url"`
			Title  string `json:"title"`
			Number int    `json:"number"`
		}
		if json.Unmarshal([]byte(prInfo), &prData) == nil {
			return marshalResponse(CreateResponse{
				Action: "create",
				URL:    prData.URL,
				Number: prData.Number,
				Title:  prData.Title,
			})
		}
	}

	return marshalResponse(CreateResponse{
		Action: "create",
		URL:    out,
		Title:  ca.title,
	})
}

// handleComment adds a comment to a PR.
func (t *Tool) handleComment(ctx context.Context, args map[string]any) (string, error) {
	prNumber := getIntArg(args, "pr_number", 0)
	if prNumber <= 0 {
		return "", &ToolError{
			Code:    "VALIDATION_ERROR",
			Message: "pr_number is required and must be a positive integer",
		}
	}

	body := getStringArg(args, "body")
	if strings.TrimSpace(body) == "" {
		return "", &ToolError{
			Code:    "VALIDATION_ERROR",
			Message: "body is required and must not be empty",
		}
	}

	// Write body to temp file
	tmpFile, err := os.CreateTemp("", "gh-pr-comment-*.md")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(body); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("writing comment body: %w", err)
	}
	tmpFile.Close()

	if _, err := t.run(ctx, "pr", "comment", fmt.Sprintf("%d", prNumber), "--body-file", tmpPath); err != nil {
		return "", fmt.Errorf("commenting on PR: %w", err)
	}

	return marshalResponse(CommentResponse{
		Action:   "comment",
		Success:  true,
		PRNumber: prNumber,
	})
}
