package orchestrator

import (
	"context"
	"fmt"

	"github.com/zombiekit/brains/internal/callback"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/shutdown"
	"github.com/zombiekit/brains/internal/state"
)

// Orchestrator owns the daemon lifecycle: reconciliation, service assembly,
// and shutdown coordination. Resource acquisition (store, logger) is handled
// by the caller.
type Orchestrator struct {
	cfg   *Config
	store state.StateStore
}

// New creates an Orchestrator with the given config and state store.
func New(cfg *Config, store state.StateStore) *Orchestrator {
	return &Orchestrator{cfg: cfg, store: store}
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

	linearPoller := NewWatcherStub(WatcherLinearPoller, o.cfg.PollInterval)
	prWatcher := NewWatcherStub(WatcherPRWatcher, o.cfg.PollInterval)
	commentWatcher := NewWatcherStub(WatcherCommentWatcher, o.cfg.PollInterval)

	logger.Info("starting services")
	mgr := shutdown.New(o.cfg.ShutdownTimeout)
	return mgr.Run(callbackSrv.Run, linearPoller, prWatcher, commentWatcher)
}
