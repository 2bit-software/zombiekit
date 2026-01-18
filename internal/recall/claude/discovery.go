package claude

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DefaultClaudePath returns the default Claude config directory path.
func DefaultClaudePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// DiscoverHistoryFiles finds all JSONL history files in the Claude projects directory.
func DiscoverHistoryFiles(claudePath string) ([]string, error) {
	projectsDir := filepath.Join(claudePath, "projects")

	// Check if projects directory exists
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return nil, nil // No projects directory, return empty slice
	}

	var files []string
	err := filepath.WalkDir(projectsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

// DiscoverProjectFiles finds history files for a specific project path.
func DiscoverProjectFiles(claudePath, projectPath string) ([]string, error) {
	encoded := EncodeProjectPath(projectPath)
	projectDir := filepath.Join(claudePath, "projects", encoded)

	// Check if project directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return nil, nil // Project directory doesn't exist
	}

	var files []string
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			files = append(files, filepath.Join(projectDir, entry.Name()))
		}
	}

	return files, nil
}

// EncodeProjectPath converts a filesystem path to Claude's encoded format.
// Example: /Users/foo/bar -> -Users-foo-bar
func EncodeProjectPath(path string) string {
	// Replace path separators with dashes
	return strings.ReplaceAll(path, string(filepath.Separator), "-")
}
