package workspace

import "errors"

// ErrNoMarker indicates ReadMarker found no .ai/workspace.json at the
// requested worktree path. Callers may treat this as a soft failure
// (e.g., fall back to convention-based discovery) or a hard one.
var ErrNoMarker = errors.New("workspace marker not found")
