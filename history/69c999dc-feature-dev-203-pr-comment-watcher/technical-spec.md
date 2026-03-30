# Technical Spec: Watcher 2 — PR Comment Queue

## New Types

### `SessionResult` and `CommentDispatcher`

**File**: `internal/orchestrator/comment_dispatcher.go`

```go
package orchestrator

import (
    "log/slog"
    "sync"

    "github.com/2bit-software/zombiekit/internal/github"
)

type SessionResultKind string

const (
    SessionResolved SessionResultKind = "resolved"
    SessionFailed   SessionResultKind = "failed"
)

type SessionResult struct {
    Kind     SessionResultKind
    TicketID string
    PRNumber int
}

// prQueue holds the per-PR goroutine state.
type prQueue struct {
    comments chan github.PRComment
    cancel   context.CancelFunc
}

// CommentDispatcher coordinates between the polling loop, per-PR goroutines,
// and the callback router. It owns the session completion signaling contract.
type CommentDispatcher struct {
    mu       sync.Mutex
    queues   map[int]*prQueue              // PR number -> active queue
    sessions map[string]chan SessionResult  // ticketID -> completion channel (buffered 1)
    logger   *slog.Logger
}

func NewCommentDispatcher(logger *slog.Logger) *CommentDispatcher {
    return &CommentDispatcher{
        queues:   make(map[int]*prQueue),
        sessions: make(map[string]chan SessionResult),
        logger:   logger,
    }
}

// RegisterSession creates a completion channel for a session. The per-PR
// goroutine calls this before SpawnSession and blocks on the returned channel.
// Channel is buffered(1) so NotifyResult never blocks.
func (d *CommentDispatcher) RegisterSession(ticketID string, prNumber int) <-chan SessionResult {
    d.mu.Lock()
    defer d.mu.Unlock()
    ch := make(chan SessionResult, 1)
    d.sessions[ticketID] = ch
    return ch
}

// NotifyResult signals that a session completed. Called by the Router after
// handling CommentResolvedEvent or FailureEvent. Safe to call for sessions
// not registered (logs warning, no-op).
func (d *CommentDispatcher) NotifyResult(ticketID string, result SessionResult) {
    d.mu.Lock()
    ch, ok := d.sessions[ticketID]
    if ok {
        delete(d.sessions, ticketID)
    }
    d.mu.Unlock()

    if !ok {
        // Expected for Watcher 1 failures — those sessions aren't registered with the dispatcher
        d.logger.Debug("notify for unregistered session", slog.String("ticket_id", ticketID))
        return
    }
    ch <- result
}

// CreateQueue creates a per-PR goroutine queue. Returns the queue for the
// caller to start the goroutine. Channel capacity 100.
func (d *CommentDispatcher) CreateQueue(prNumber int, cancel context.CancelFunc) *prQueue {
    d.mu.Lock()
    defer d.mu.Unlock()
    q := &prQueue{
        comments: make(chan github.PRComment, 100),
        cancel:   cancel,
    }
    d.queues[prNumber] = q
    return q
}

// GetQueue returns the queue for a PR, or nil if none exists.
func (d *CommentDispatcher) GetQueue(prNumber int) *prQueue {
    d.mu.Lock()
    defer d.mu.Unlock()
    return d.queues[prNumber]
}

// RemoveQueue cancels the per-PR context and removes the queue entry.
func (d *CommentDispatcher) RemoveQueue(prNumber int) {
    d.mu.Lock()
    q, ok := d.queues[prNumber]
    if ok {
        delete(d.queues, prNumber)
    }
    d.mu.Unlock()

    if ok && q.cancel != nil {
        q.cancel()
    }
}

// ActivePRs returns the PR numbers with active queues.
func (d *CommentDispatcher) ActivePRs() []int {
    d.mu.Lock()
    defer d.mu.Unlock()
    prs := make([]int, 0, len(d.queues))
    for pr := range d.queues {
        prs = append(prs, pr)
    }
    return prs
}
```

## Comment Watcher

**File**: `internal/orchestrator/watcher_comment.go`

```go
// NewCommentWatcher returns a ServiceFunc that polls tracked PRs for new
// review comments and dispatches them to per-PR goroutines for serial
// processing via AI sessions.
func (o *Orchestrator) NewCommentWatcher(dispatcher *CommentDispatcher) shutdown.ServiceFunc {
    return func(ctx context.Context) error {
        logger := logging.Logger().With(slog.String("watcher", WatcherCommentWatcher))
        logger.Info("comment watcher started", slog.Duration("poll_interval", o.cfg.PollInterval))

        ticker := time.NewTicker(o.cfg.PollInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                logger.Info("comment watcher stopping")
                return nil
            case <-ticker.C:
                o.pollComments(ctx, dispatcher, logger)
            }
        }
    }
}
```

### Poll Cycle

```go
func (o *Orchestrator) pollComments(ctx context.Context, dispatcher *CommentDispatcher, logger *slog.Logger) {
    prs, err := o.github.ListOpenPRs(ctx, o.cfg.TrackingLabel)
    if err != nil {
        logger.Error("failed to list open PRs", slog.String("err", err.Error()))
        return
    }

    activePRSet := make(map[int]bool, len(prs))

    for _, pr := range prs {
        activePRSet[pr.Number] = true
        o.pollPRComments(ctx, dispatcher, pr, logger)
    }

    // Reap queues for PRs no longer in the tracked set
    for _, prNumber := range dispatcher.ActivePRs() {
        if !activePRSet[prNumber] {
            logger.Info("reaping stale PR queue", slog.Int("pr_number", prNumber))
            dispatcher.RemoveQueue(prNumber)
        }
    }
}
```

### Per-PR Comment Polling

```go
func (o *Orchestrator) pollPRComments(ctx context.Context, dispatcher *CommentDispatcher, pr github.PRSummary, logger *slog.Logger) {
    prLog := logger.With(slog.Int("pr_number", pr.Number))

    // Look up job by PR number
    job, err := o.store.GetJobByPR(ctx, int64(pr.Number))
    if err != nil {
        prLog.Error("failed to get job by PR", slog.String("err", err.Error()))
        return
    }
    if job == nil {
        return // PR exists but no job — not our concern
    }

    // Skip terminal states
    switch job.Status {
    case state.StatusComplete, state.StatusClosed, state.StatusNeedsAttention:
        return
    }

    // Get watermark
    watermark, err := o.store.GetCommentWatermark(ctx, int64(pr.Number))
    if err != nil {
        prLog.Error("failed to get watermark", slog.String("err", err.Error()))
        return
    }

    // Fetch new review comments
    comments, err := o.github.GetCommentsSince(ctx, pr.Number, github.CommentKindReview, watermark)
    if err != nil {
        prLog.Error("failed to get comments", slog.String("err", err.Error()))
        return
    }

    // Filter bot comments
    var filtered []github.PRComment
    for _, c := range comments {
        if c.Author == o.cfg.BotUsername {
            continue
        }
        filtered = append(filtered, c)
    }

    if len(filtered) == 0 {
        return
    }

    // Get or create per-PR queue
    q := dispatcher.GetQueue(pr.Number)
    if q == nil {
        prCtx, prCancel := context.WithCancel(ctx)
        q = dispatcher.CreateQueue(pr.Number, prCancel)
        go o.runPRQueue(prCtx, dispatcher, pr.Number, job, q, prLog)
    }

    // Enqueue comments
    for _, c := range filtered {
        select {
        case q.comments <- c:
        default:
            prLog.Warn("PR comment queue full, skipping — will retry next poll", slog.Int64("comment_id", c.ID))
        }
    }
}
```

### Per-PR Goroutine

```go
func (o *Orchestrator) runPRQueue(
    ctx context.Context,
    dispatcher *CommentDispatcher,
    prNumber int,
    job *state.Job,
    q *prQueue,
    logger *slog.Logger,
) {
    defer dispatcher.RemoveQueue(prNumber)
    logger.Info("per-PR queue started")

    var highestEnqueuedID int64

    for {
        select {
        case <-ctx.Done():
            logger.Info("per-PR queue stopping (context cancelled)")
            return

        case comment, ok := <-q.comments:
            if !ok {
                return // channel closed
            }

            if comment.ID > highestEnqueuedID {
                highestEnqueuedID = comment.ID
            }

            // Check PR state before dispatching
            merged, err := o.github.IsMerged(ctx, prNumber)
            if err != nil {
                logger.Error("IsMerged check failed", slog.String("err", err.Error()))
                continue
            }
            if merged {
                logger.Info("PR merged, aborting queue")
                o.drainChannel(q.comments)
                return
            }

            closed, err := o.github.IsClosed(ctx, prNumber)
            if err != nil {
                logger.Error("IsClosed check failed", slog.String("err", err.Error()))
                continue
            }
            if closed {
                logger.Info("PR closed, aborting queue")
                o.drainChannel(q.comments)
                return
            }

            // Acquire concurrency slot (blocking with context check)
            if !o.acquireSlotBlocking(ctx, logger) {
                return // context cancelled while waiting
            }

            // Write comment payload to worktree
            if err := writeCommentJSON(job.WorktreePath, comment); err != nil {
                logger.Error("failed to write comment.json", slog.String("err", err.Error()))
                o.store.ReleaseSlot(ctx, o.cfg.ProjectID)
                continue
            }

            // Register completion channel before spawning
            done := dispatcher.RegisterSession(job.TicketID, prNumber)

            // Spawn session
            _, err = o.sessions.SpawnSession(ctx, job.TicketID, "comment-resolution", job.WorktreePath, nil)
            if err != nil {
                logger.Error("failed to spawn session", slog.String("err", err.Error()))
                o.store.ReleaseSlot(ctx, o.cfg.ProjectID)
                continue
            }

            // Block until session completes
            select {
            case <-ctx.Done():
                return
            case result := <-done:
                if result.Kind == SessionFailed {
                    logger.Info("session failed, draining queue")
                    // Drain buffered comments and track highest ID
                    drainedMax := o.drainChannel(q.comments)
                    if drainedMax > highestEnqueuedID {
                        highestEnqueuedID = drainedMax
                    }
                    // Advance watermark past all enqueued comments to prevent reprocessing
                    o.store.SetCommentWatermark(ctx, int64(prNumber), highestEnqueuedID)
                    return
                }
                // SessionResolved: continue to next comment
                logger.Info("comment resolved", slog.Int64("comment_id", comment.ID))
            }
        }
    }
}
```

### Helpers

```go
func (o *Orchestrator) acquireSlotBlocking(ctx context.Context, logger *slog.Logger) bool {
    for {
        acquired, err := o.store.TryAcquireSlot(ctx, o.cfg.ProjectID, o.cfg.ConcurrencyLimit)
        if err != nil {
            logger.Error("slot acquisition error", slog.String("err", err.Error()))
        }
        if acquired {
            return true
        }
        select {
        case <-ctx.Done():
            return false
        case <-time.After(5 * time.Second):
            // retry
        }
    }
}

// drainChannel reads and discards all buffered comments, returning the highest
// comment ID seen. Used to advance the watermark past drained comments.
func (o *Orchestrator) drainChannel(ch chan github.PRComment) int64 {
    var maxID int64
    for {
        select {
        case c, ok := <-ch:
            if !ok {
                return maxID
            }
            if c.ID > maxID {
                maxID = c.ID
            }
        default:
            return maxID
        }
    }
}

func writeCommentJSON(worktreePath string, comment github.PRComment) error {
    payload := map[string]any{
        "id":        comment.ID,
        "author":    comment.Author,
        "body":      comment.Body,
        "path":      comment.Path,
        "diff_hunk": comment.DiffHunk,
    }
    data, err := json.MarshalIndent(payload, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal comment: %w", err)
    }
    dir := filepath.Join(worktreePath, ".ai")
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return fmt.Errorf("create .ai dir: %w", err)
    }
    return os.WriteFile(filepath.Join(dir, "comment.json"), data, 0o644)
}
```

## Changes to Existing Code

### `router.go` — Add dispatcher + slot release

```go
type Router struct {
    // ... existing fields ...
    dispatcher *CommentDispatcher  // NEW
}

func NewRouter(
    events <-chan callback.Event,
    store state.StateStore,
    gh github.Client,
    lc linear.Client,
    arch archival.Archiver,
    aud friction.Auditor,
    dispatcher *CommentDispatcher,  // NEW
    cfg *Config,
    logger *slog.Logger,
) *Router {
    return &Router{
        // ... existing ...
        dispatcher: dispatcher,
    }
}
```

**`handleCommentResolved`** — add after archive/audit block:
```go
// Release concurrency slot (acquired by comment watcher before SpawnSession)
if err := r.store.ReleaseSlot(ctx, r.cfg.ProjectID); err != nil {
    logger.Error("failed to release slot", slog.String("step", "ReleaseSlot"), slog.String("err", err.Error()))
}

// Signal comment watcher that session completed
if r.dispatcher != nil {
    r.dispatcher.NotifyResult(evt.TicketID, SessionResult{
        Kind:     SessionResolved,
        TicketID: evt.TicketID,
        PRNumber: prNumber,
    })
}
```

**`handleFailed`** — add after existing slot release:
```go
if r.dispatcher != nil {
    r.dispatcher.NotifyResult(evt.TicketID, SessionResult{
        Kind:     SessionFailed,
        TicketID: evt.TicketID,
    })
}
```

### `orchestrator.go` — Wire dispatcher

```go
func (o *Orchestrator) Run() error {
    // ... existing reconciliation ...

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
    prWatcher := NewWatcherStub(WatcherPRWatcher, o.cfg.PollInterval)
    commentWatcher := o.NewCommentWatcher(dispatcher)

    logger.Info("starting services")
    mgr := shutdown.New(o.cfg.ShutdownTimeout)
    return mgr.Run(callbackSrv.Run, router.Run, linearPoller, prWatcher, commentWatcher)
}
```

### `config.go` — Add BotUsername

```go
type Config struct {
    // ... existing fields ...
    BotUsername string
}
```

Add to `Validate()`:
```go
if c.BotUsername == "" {
    errs = append(errs, "--bot-username/ORCH_BOT_USERNAME is required")
}
```

### `state/store.go` — Add GetJobByPR

Interface addition:
```go
GetJobByPR(ctx context.Context, prNumber int64) (*Job, error)
```

Implementation:
```go
func (s *SQLiteStore) GetJobByPR(ctx context.Context, prNumber int64) (*Job, error) {
    var job Job
    var prNum sql.NullInt64
    err := s.db.QueryRowContext(ctx,
        `SELECT ticket_id, worktree_path, cmux_session, pr_number, status, created_at, updated_at
         FROM jobs WHERE pr_number = ?`,
        prNumber,
    ).Scan(&job.TicketID, &job.WorktreePath, &job.CmuxSession, &prNum, &job.Status, &job.CreatedAt, &job.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("get job by PR %d: %w", prNumber, err)
    }
    if prNum.Valid {
        job.PRNumber = &prNum.Int64
    }
    return &job, nil
}
```

## File Inventory

| File | Action | Description |
|------|--------|-------------|
| `internal/orchestrator/comment_dispatcher.go` | Create | CommentDispatcher, SessionResult, prQueue types |
| `internal/orchestrator/comment_dispatcher_test.go` | Create | Unit tests for dispatcher |
| `internal/orchestrator/watcher_comment.go` | Create | Comment watcher polling loop + per-PR goroutine |
| `internal/orchestrator/watcher_comment_test.go` | Create | Integration tests for watcher |
| `internal/orchestrator/router.go` | Modify | Add dispatcher field, ReleaseSlot in handleCommentResolved, NotifyResult calls |
| `internal/orchestrator/orchestrator.go` | Modify | Wire dispatcher into Router and comment watcher |
| `internal/orchestrator/config.go` | Modify | Add BotUsername field + validation |
| `internal/state/store.go` | Modify | Add GetJobByPR to interface + SQLite impl |
| `internal/state/store_test.go` | Modify | Add GetJobByPR test |
| CLI flag definition file | Modify | Add --bot-username / ORCH_BOT_USERNAME |

## FR Traceability

| FR | Implementation Step |
|----|-------------------|
| FR-001 | Step 2.1 (poll loop calls ListOpenPRs) |
| FR-002 | Step 2.1 (pollPRComments calls GetCommentsSince with CommentKindReview) |
| FR-003 | Steps 1.1 + 2.2 (prQueue + runPRQueue goroutine) |
| FR-004 | Step 2.2 (writeCommentJSON + SpawnSession) |
| FR-005 | Step 2.2 (acquireSlotBlocking before SpawnSession) |
| FR-006 | Step 2.1 (filter by BotUsername in pollPRComments) |
| FR-007 | Step 2.1 (skip terminal statuses in pollPRComments) |
| FR-008 | Step 2.2 (IsMerged/IsClosed check + drain in runPRQueue) |
| FR-009 | Step 2.2 (SessionFailed branch in runPRQueue) |
| FR-010 | Step 2.1 (reaping loop in pollComments) |
| FR-011 | Step 2.1 (context cancellation in NewCommentWatcher) |
| FR-012 | Steps 1.1 + 2.2 + 3.1 (RegisterSession/NotifyResult contract) |
