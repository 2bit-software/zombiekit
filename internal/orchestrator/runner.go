package orchestrator

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/2bit-software/zombiekit/internal/callback"
	"github.com/2bit-software/zombiekit/internal/cmux"
	"github.com/2bit-software/zombiekit/internal/github"
	"github.com/2bit-software/zombiekit/internal/linear"
	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/state"
	"github.com/2bit-software/zombiekit/internal/workspace"
	"github.com/2bit-software/zombiekit/internal/worktree"
)

const (
	backoffInitial = 1 * time.Second
	backoffMax     = 2 * time.Minute
)

// ProjectRunner manages the watcher goroutines for a single project.
// Each project gets its own runner with independent failure isolation.
type ProjectRunner struct {
	id         string
	cfg        ProjectConfig
	store      state.StateStore
	linear     linear.Client
	github     github.Client
	worktrees  worktree.Manager
	sessions   cmux.SessionManager
	workspace  *workspace.Manager
	events     <-chan callback.Event
	dispatcher *CommentDispatcher
	logger     *slog.Logger

	sandboxAvailable bool
	sandboxConfig    sandbox.Config
	archiver         Archiver
	auditor          Auditor

	mu      sync.Mutex
	healths map[string]*watcherHealth
}

type watcherHealth struct {
	LastSuccess      time.Time
	LastError        time.Time
	LastErrorMsg     string
	ConsecutiveFails int
	CurrentBackoff   time.Duration
}

// ProjectHealth is the externally-visible health snapshot for a project.
type ProjectHealth struct {
	ProjectID string                   `json:"project_id"`
	Watchers  map[string]WatcherHealth `json:"watchers"`
}

// WatcherHealth is the health snapshot for a single watcher.
type WatcherHealth struct {
	LastSuccess      time.Time     `json:"last_success,omitempty"`
	LastError        time.Time     `json:"last_error,omitempty"`
	LastErrorMsg     string        `json:"last_error_msg,omitempty"`
	ConsecutiveFails int           `json:"consecutive_fails"`
	CurrentBackoff   time.Duration `json:"current_backoff_ms"`
}

func NewProjectRunner(
	cfg ProjectConfig,
	store state.StateStore,
	lc linear.Client,
	gh github.Client,
	wt worktree.Manager,
	sm cmux.SessionManager,
	events <-chan callback.Event,
	sandboxAvailable bool,
	sandboxCfg sandbox.Config,
	logger *slog.Logger,
) *ProjectRunner {
	projectLogger := logger.With(slog.String("project", cfg.ID))
	ws := workspace.NewManager(wt, sandboxCfg,
		workspace.WithSpawner(sm),
		workspace.WithLogger(projectLogger),
		workspace.WithWorktreesRoot(cfg.WorktreesRoot),
	)

	return &ProjectRunner{
		id:               cfg.ID,
		cfg:              cfg,
		store:            store,
		linear:           lc,
		github:           gh,
		worktrees:        wt,
		sessions:         sm,
		workspace:        ws,
		events:           events,
		dispatcher:       NewCommentDispatcher(logger),
		logger:           projectLogger,
		sandboxAvailable: sandboxAvailable,
		sandboxConfig:    sandboxCfg,
		archiver:         NoopArchiver{},
		auditor:          NoopAuditor{},
		healths:          make(map[string]*watcherHealth),
	}
}

// RunSupervised starts all watcher goroutines and blocks until ctx is
// cancelled. Individual watcher failures are restarted with exponential
// backoff. This method never returns an error — only nil on ctx done.
func (p *ProjectRunner) RunSupervised(ctx context.Context) error {
	p.logger.Info("project runner starting")

	type watcherDef struct {
		name string
		fn   func(context.Context) error
	}

	watchers := []watcherDef{
		{"linear-poller", p.linearPoller},
		{"pr-watcher", p.prWatcher},
		{"comment-watcher", p.commentWatcher},
		{"event-router", p.eventRouter},
	}

	var wg sync.WaitGroup
	for _, w := range watchers {
		wg.Add(1)
		go func(def watcherDef) {
			defer wg.Done()
			p.runWithRestart(ctx, def.name, def.fn)
		}(w)
	}

	wg.Wait()
	p.logger.Info("project runner stopped")
	return nil
}

// runWithRestart runs fn in a loop with exponential backoff on failure.
// Backoff resets to initial after fn runs successfully for at least one
// poll interval.
func (p *ProjectRunner) runWithRestart(ctx context.Context, name string, fn func(context.Context) error) {
	backoff := backoffInitial
	logger := p.logger.With(slog.String("watcher", name))

	for {
		if ctx.Err() != nil {
			return
		}

		start := time.Now()
		err := fn(ctx)

		if ctx.Err() != nil {
			return
		}

		if err == nil {
			logger.Info("watcher returned without error, restarting")
			backoff = backoffInitial
			p.recordSuccess(name)
			continue
		}

		elapsed := time.Since(start)
		if elapsed > p.cfg.PollInterval.Duration {
			backoff = backoffInitial
		}

		p.recordError(name, err, backoff)
		logger.Error("watcher failed, restarting after backoff",
			slog.String("err", err.Error()),
			slog.Duration("backoff", backoff),
		)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, backoffMax)
	}
}

func (p *ProjectRunner) recordSuccess(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	h, ok := p.healths[name]
	if !ok {
		h = &watcherHealth{}
		p.healths[name] = h
	}
	h.LastSuccess = time.Now()
	h.ConsecutiveFails = 0
	h.CurrentBackoff = 0
}

func (p *ProjectRunner) recordError(name string, err error, backoff time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	h, ok := p.healths[name]
	if !ok {
		h = &watcherHealth{}
		p.healths[name] = h
	}
	h.LastError = time.Now()
	h.LastErrorMsg = err.Error()
	h.ConsecutiveFails++
	h.CurrentBackoff = backoff
}

// Health returns a snapshot of this project's watcher health states.
func (p *ProjectRunner) Health() ProjectHealth {
	p.mu.Lock()
	defer p.mu.Unlock()

	watchers := make(map[string]WatcherHealth, len(p.healths))
	for name, h := range p.healths {
		watchers[name] = WatcherHealth{
			LastSuccess:      h.LastSuccess,
			LastError:        h.LastError,
			LastErrorMsg:     h.LastErrorMsg,
			ConsecutiveFails: h.ConsecutiveFails,
			CurrentBackoff:   h.CurrentBackoff,
		}
	}
	return ProjectHealth{
		ProjectID: p.id,
		Watchers:  watchers,
	}
}

// linearPoller is the watcher function for the linear polling loop.
// Implementations are in watcher_linear.go.
func (p *ProjectRunner) linearPoller(ctx context.Context) error {
	p.logger.Info("linear poller started", "pollInterval", p.cfg.PollInterval.Duration)

	ticker := time.NewTicker(p.cfg.PollInterval.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("linear poller stopping")
			return nil
		case <-ticker.C:
			p.pollAndProcess(ctx)
		}
	}
}

// prWatcher is the watcher function for the PR lifecycle polling loop.
// Implementations are in watcher_pr.go.
func (p *ProjectRunner) prWatcher(ctx context.Context) error {
	logger := p.logger.With(slog.String("watcher", WatcherPRWatcher))
	logger.Info("pr watcher started", slog.Duration("poll_interval", p.cfg.PollInterval.Duration))

	ticker := time.NewTicker(p.cfg.PollInterval.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("pr watcher stopping")
			return nil
		case <-ticker.C:
			p.pollPRLifecycle(ctx, logger)
		}
	}
}

// commentWatcher is the watcher function for the comment polling loop.
// Implementations are in watcher_comment.go.
func (p *ProjectRunner) commentWatcher(ctx context.Context) error {
	logger := p.logger.With(slog.String("watcher", WatcherCommentWatcher))
	logger.Info("comment watcher started", slog.Duration("poll_interval", p.cfg.PollInterval.Duration))

	ticker := time.NewTicker(p.cfg.PollInterval.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("comment watcher stopping")
			return nil
		case <-ticker.C:
			p.pollComments(ctx, p.dispatcher, logger)
		}
	}
}

// eventRouter is the watcher function for the callback event routing loop.
// It replaces the old Router struct's Run method.
func (p *ProjectRunner) eventRouter(ctx context.Context) error {
	p.logger.Info("event router started")
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("event router stopped", slog.String("reason", "context cancelled"))
			return nil
		case evt, ok := <-p.events:
			if !ok {
				p.logger.Info("event router stopped", slog.String("reason", "channel closed"))
				return nil
			}
			p.handleEvent(ctx, evt)
		}
	}
}
