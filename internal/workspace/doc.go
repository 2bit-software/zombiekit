// Package workspace composes the worktree, sandbox, and (optional) cmux
// session steps the orchestrator runs for each ticket pickup.
//
// It exists so the orchestrator daemon and the `brains workspace` CLI both
// drive the same code path. Prep performs each step in order with rollback;
// Teardown performs the inverse for the cleanup path.
//
// The package depends on internal/worktree and internal/sandbox via small
// interfaces (Sandbox, Spawner) so callers can fake out Docker and cmux in
// tests.
package workspace
