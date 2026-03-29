package git

import (
	"fmt"
	"os"
	"strings"
)

// validateFiles checks that file paths are safe to stage.
func validateFiles(workDir string, files []string) error {
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
			path = workDir + "/" + f
		}
		if _, err := os.Stat(path); err != nil {
			return &ToolError{
				Code:    "VALIDATION_ERROR",
				Message: fmt.Sprintf("file does not exist: %s", f),
				Hint:    "Check the file path and try again",
			}
		}
	}
	return nil
}

// protectedBranches that must not be pushed to directly.
var protectedBranches = map[string]bool{
	"main":   true,
	"master": true,
}

// validateBranchForPush checks that the current branch is safe to push.
func validateBranchForPush(branch string) error {
	if protectedBranches[branch] {
		return &ToolError{
			Code:    "BRANCH_PROTECTED",
			Message: fmt.Sprintf("cannot push to %s", branch),
			Hint:    "Use a feature branch instead",
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
