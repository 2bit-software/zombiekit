package orchestrator

import (
	"context"
	"fmt"

	"github.com/2bit-software/zombiekit/internal/archival"
	"github.com/2bit-software/zombiekit/internal/callback"
	"github.com/2bit-software/zombiekit/internal/cmux"
	"github.com/2bit-software/zombiekit/internal/friction"
	"github.com/2bit-software/zombiekit/internal/github"
	"github.com/2bit-software/zombiekit/internal/linear"
	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/2bit-software/zombiekit/internal/shutdown"
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

// Run executes the orchestrator lifecycle:
//  1. Reconciliation (synchronous, fail-fast)
//  2. Build callback server and watcher stubs
//  3. Run all services under the shutdown manager
//
// Returns nil on clean shutdown, non-nil on service or reconciliation failure.
func (o *Orchestrator) Run() error {
	logger := logging.Logger()

	if err := state.ApplyReconciliation(context.Background(), o.store, logger); err != nil {
		return fmt.Errorf("reconciliation: %w", err)
	}

	callbackSrv := callback.New(o.cfg.CallbackPort)
	dispatcher := NewCommentDispatcher(logger)

	router := NewRouter(
		callbackSrv.Events(),
		o.store, o.github, o.linear,
		archival.NoopArchiver{}, friction.NoopAuditor{},
		dispatcher,
		o.cfg, logger,
	)

	linearPoller := o.NewLinearPoller()
	prWatcher := o.NewPRWatcher()
	commentWatcher := o.NewCommentWatcher(dispatcher)

	logger.Info("starting services")
	mgr := shutdown.New(o.cfg.ShutdownTimeout)
	return mgr.Run(callbackSrv.Run, router.Run, linearPoller, prWatcher, commentWatcher)
}
