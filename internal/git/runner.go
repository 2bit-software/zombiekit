// Package git provides a reusable git command runner.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner executes git commands in a working directory.
type Runner struct {
	gitBin  string
	workDir string
}

// NewRunner creates a Runner for the given working directory.
// Returns an error if git is not found in PATH.
func NewRunner(workDir string) (*Runner, error) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git not found on PATH: %w", err)
	}
	return &Runner{gitBin: gitBin, workDir: workDir}, nil
}

// Run executes a git command and returns trimmed stdout.
func (r *Runner) Run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, r.gitBin, args...)
	cmd.Dir = r.workDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", &Error{
			Args:    args,
			Stderr:  strings.TrimSpace(stderr.String()),
			Wrapped: err,
		}
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunSilent executes a git command and returns only the exit status.
// Useful for commands like `git diff --cached --quiet` where exit code is the signal.
func (r *Runner) RunSilent(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, r.gitBin, args...)
	cmd.Dir = r.workDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &Error{
			Args:    args,
			Stderr:  strings.TrimSpace(stderr.String()),
			Wrapped: err,
		}
	}
	return nil
}

// WorkDir returns the runner's working directory.
func (r *Runner) WorkDir() string {
	return r.workDir
}

// Error wraps a failed git command with stderr output.
type Error struct {
	Args    []string
	Stderr  string
	Wrapped error
}

func (e *Error) Error() string {
	return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), e.Stderr)
}

func (e *Error) Unwrap() error {
	return e.Wrapped
}
