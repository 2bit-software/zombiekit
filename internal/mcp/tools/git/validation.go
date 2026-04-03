package git

import (
	"context"
	"fmt"
	"os"
	"strings"

	internalgit "github.com/2bit-software/zombiekit/internal/git"
)

// validateFiles checks that file paths are safe to stage.
// Files must either exist on disk or be tracked by git (to support staging deletions).
func validateFiles(ctx context.Context, runner *internalgit.Runner, files []string) error {
	if len(files) == 0 {
		return &ToolError{
			Code:    "VALIDATION_ERROR",
			Message: "files parameter is required and must not be empty",
			Hint:    "Provide a list of file paths to stage",
		}
	}

	for _, f := range files {
		if strings.HasPrefix(f, "-") {
			return &ToolError{
				Code:    "VALIDATION_ERROR",
				Message: fmt.Sprintf("file path looks like a flag: %q", f),
				Hint:    "File paths must not start with '-'",
			}
		}

		// Resolve path relative to workDir for existence check
		path := f
		if !strings.HasPrefix(f, "/") {
			path = runner.WorkDir() + "/" + f
		}
		if _, err := os.Stat(path); err != nil {
			// File not on disk — check if git tracks it (supports staging deletions)
			out, gitErr := runner.Run(ctx, "ls-files", f)
			if gitErr != nil || strings.TrimSpace(out) == "" {
				return &ToolError{
					Code:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("file does not exist: %s", f),
					Hint:    "Check the file path and try again",
				}
			}
		}
	}
	return nil
}

// validateMessage checks that a commit message is non-empty.
func validateMessage(message string) error {
	if strings.TrimSpace(message) == "" {
		return &ToolError{
			Code:    "VALIDATION_ERROR",
			Message: "commit message must not be empty",
			Hint:    "Provide a non-empty commit message",
		}
	}
	return nil
}
