package orchestrator

import (
	"fmt"

	"github.com/2bit-software/zombiekit/internal/cmux"
	"github.com/2bit-software/zombiekit/internal/github"
	"github.com/2bit-software/zombiekit/internal/linear"
	"github.com/2bit-software/zombiekit/internal/state"
	"github.com/2bit-software/zombiekit/internal/worktree"
)

// Orchestrator owns the daemon lifecycle: reconciliation, service assembly,
// and shutdown coordination. Resource acquisition (store, logger) is handled
// by the caller.
type Orchestrator struct {
	cfg       *Config
	store     state.StateStore
	linear    linear.Client
	github    github.Client
	worktrees worktree.Manager
	sessions  cmux.SessionManager
}

// New creates an Orchestrator with the given config and dependencies.
func New(cfg *Config, store state.StateStore, lc linear.Client, gh github.Client, wt worktree.Manager, sm cmux.SessionManager) *Orchestrator {
	return &Orchestrator{
		cfg:       cfg,
		store:     store,
		linear:    lc,
		github:    gh,
		worktrees: wt,
		sessions:  sm,
	}
}

// Run is deprecated. Use ProjectRunner.RunSupervised for multi-project mode.
// Kept temporarily for compilation; will be deleted in T010.
func (o *Orchestrator) Run() error {
	return fmt.Errorf("orchestrator.Run is deprecated: use ProjectRunner.RunSupervised")
}
