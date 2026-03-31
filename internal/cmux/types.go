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
	cmuxBin  string
	command  string
	mu       sync.Mutex
	sessions map[string]sessionEntry
}

type sessionEntry struct {
	ref  string // e.g. "workspace:9"
	name string // e.g. "DEV-186: implement session manager"
}

// Option configures a CmuxManager.
type Option func(*CmuxManager)

// WithCommand overrides the default launch command (default: "claude").
func WithCommand(cmd string) Option {
	return func(m *CmuxManager) {
		m.command = cmd
	}
}
