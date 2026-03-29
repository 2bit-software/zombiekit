// Package cmux manages cmux workspace lifecycles for agent sessions.
//
// The orchestrator uses this package to spawn isolated Claude Code instances
// in git worktrees, check their status, and terminate them. Each workspace
// is identified by a ticket ID and given a human-readable display name.
//
// cmux >= 0.63.0 is required (--name flag support on new-workspace).
//
// # Usage
//
//	mgr, err := cmux.New()
//	if err != nil {
//		// cmux binary not found or not running
//	}
//
//	ref, err := mgr.SpawnSession(ctx, "DEV-186", "implement session manager", "/path/to/worktree", map[string]string{
//		"WORK_CALLBACK_URL": "http://localhost:8666/DEV-186",
//	})
//	if err != nil {
//		// handle error
//	}
//
//	exists, err := mgr.SessionExists(ctx, "DEV-186")
//	// exists == true
//
//	err = mgr.KillSession(ctx, "DEV-186")
package cmux
