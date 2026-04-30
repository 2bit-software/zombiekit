package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Marker captures enough state about a prepped workspace to drive cleanup
// without re-deriving from naming conventions. It is written to
// .ai/workspace.json inside the worktree by Prep and read by Teardown
// and gc.
type Marker struct {
	TicketID     string    `json:"ticket_id"`
	Title        string    `json:"title,omitempty"`
	Branch       string    `json:"branch,omitempty"`
	WorktreePath string    `json:"worktree_path"`
	SandboxName  string    `json:"sandbox_name,omitempty"`
	Spawned      bool      `json:"spawned,omitempty"`
	Prompt       string    `json:"prompt,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// MarkerPath returns the absolute path of the workspace.json marker for a
// worktree.
func MarkerPath(worktreePath string) string {
	return filepath.Join(worktreePath, ".ai", "workspace.json")
}

// ReadMarker returns the marker stored at .ai/workspace.json under
// worktreePath, or ErrNoMarker if the file does not exist.
func ReadMarker(worktreePath string) (Marker, error) {
	path := MarkerPath(worktreePath)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Marker{}, ErrNoMarker
		}
		return Marker{}, fmt.Errorf("read %s: %w", path, err)
	}
	var m Marker
	if err := json.Unmarshal(data, &m); err != nil {
		return Marker{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// writeMarker serializes m to .ai/workspace.json under worktreePath. The
// parent .ai directory must already exist.
func writeMarker(worktreePath string, m Marker) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal marker: %w", err)
	}
	path := MarkerPath(worktreePath)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
