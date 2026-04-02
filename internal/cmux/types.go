package cmux

import (
	"context"
	"sync"
)

// SessionManager manages cmux workspace lifecycles for agent sessions.
type SessionManager interface {
	SpawnSession(ctx context.Context, ticketID, title, worktreePath string, env map[string]string, prompt string) (workspaceRef string, err error)
	KillSession(ctx context.Context, ticketID string) error
	SessionExists(ctx context.Context, ticketID string) (bool, error)
}

// CmuxManager implements SessionManager by shelling out to the cmux CLI.
//
// The mutex serializes all mutating operations (SpawnSession, KillSession).
// Concurrent spawns for different tickets block each other, which is acceptable
// because spawn operations are infrequent and correctness requires atomicity
// across the check-create-rename sequence.
type CmuxManager struct {
	cmuxBin        string
	command        string
	commandBuilder CommandBuilder
	mu             sync.Mutex
	sessions       map[string]sessionEntry
}

type sessionEntry struct {
	ref  string // e.g. "workspace:9"
	name string // e.g. "DEV-186: implement session manager"
}

// CommandBuilder overrides default command construction in SpawnSession.
// It receives the worktree path, environment variables, the base agent command,
// and the prompt. It returns the shell command to pass to cmux and the working
// directory for the cmux workspace. When cwd is empty, worktreePath is used.
type CommandBuilder func(worktreePath string, env map[string]string, baseCmd, prompt string) (cmd, cwd string, err error)

// Option configures a CmuxManager.
type Option func(*CmuxManager)

// WithCommand overrides the default launch command (default: "claude").
func WithCommand(cmd string) Option {
	return func(m *CmuxManager) {
		m.command = cmd
	}
}

// WithCommandBuilder sets a custom command builder that replaces the default
// buildCommand logic. Use this to wrap agent sessions in additional tooling
// (e.g., Docker Sandboxes) without modifying the core cmux integration.
func WithCommandBuilder(fn CommandBuilder) Option {
	return func(m *CmuxManager) {
		m.commandBuilder = fn
	}
}
