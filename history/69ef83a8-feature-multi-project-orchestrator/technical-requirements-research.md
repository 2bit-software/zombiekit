# Technical Requirements: Multi-Project Orchestrator

## Architecture Decision: N Watcher Goroutine Sets

Spawn one `(LinearPoller, PRWatcher, CommentWatcher, Router)` tuple per project. Each tuple is wrapped in a `ProjectRunner` with its own restart-capable lifecycle.

### Shared Components (singletons)
- **State store** (`state.StateStore`): Single SQLite DB, queries gain `projectID` parameter
- **Linear client** (`linear.Client`): Project-agnostic — `PollReadyTickets` already takes `projectID` param. One client per unique API key.
- **Callback server** (`callback.CallbackServer`): Single HTTP server, single port. New URL format only.
- **EventDemuxer**: New component — receives all callback events, routes to per-project channels by `projectID`
- **Session manager** (`cmux.SessionManager`): Project-agnostic, shared

### Per-Project Components (one per `[[project]]`)
- **GitHub client** (`github.Client`): Constructor takes `(token, owner, repo)` — one per project
- **Worktree manager** (`worktree.Manager`): Holds `repoDir`, `worktreesRoot` — one per project
- **ProjectRunner**: New type wrapping the 4 watcher/router goroutines + per-project config

## Config Layer

### TOML Structure
BurntSushi/toml v1.6.0 already in go.mod. `[[project]]` array-of-tables maps to `[]ProjectConfig`. All struct fields require explicit `toml:"..."` tags to ensure TOML field names match the documented snake_case format.

```toml
[global]
db_path = ".data/orchestrator.db"
callback_port = 8666
poll_interval = "30s"
log_level = "info"
log_json = false
shutdown_timeout = "30s"
bot_username = "zombiekit-bot"
sandbox = ""
linear_api_key = "lin_api_..."
github_token = "ghp_..."

[[project]]
id = "zombiekit"
linear_project_id = "707aafa8-..."
github_owner = "2bit-software"
github_repo = "zombiekit"
repo_dir = "/path/to/zombiekit"
worktrees_root = ".claude/worktrees"
base_branch = "main"
tracking_label = "ai-ready"
concurrency_limit = 1
copy_files = [".claude/settings.json"]
closed_pr_status = "Done"
# Optional per-project credential overrides:
# linear_api_key = "lin_api_different..."
# github_token = "ghp_different..."
```

### Config Go Types

```go
type OrchestratorConfig struct {
    Global   GlobalConfig    `toml:"global"`
    Projects []ProjectConfig `toml:"project"`
}

type GlobalConfig struct {
    LinearAPIKey    string        `toml:"linear_api_key"`
    GitHubToken     string        `toml:"github_token"`
    CallbackPort    int           `toml:"callback_port"`
    DBPath          string        `toml:"db_path"`
    PollInterval    time.Duration `toml:"poll_interval"`
    LogLevel        string        `toml:"log_level"`
    LogJSON         bool          `toml:"log_json"`
    ShutdownTimeout time.Duration `toml:"shutdown_timeout"`
    BotUsername     string        `toml:"bot_username"`
    Sandbox        string        `toml:"sandbox"`
}

type ProjectConfig struct {
    ID               string        `toml:"id"`
    LinearAPIKey     string        `toml:"linear_api_key"`     // optional, falls back to global
    LinearProjectID  string        `toml:"linear_project_id"`
    GitHubToken      string        `toml:"github_token"`       // optional, falls back to global
    GitHubOwner      string        `toml:"github_owner"`
    GitHubRepo       string        `toml:"github_repo"`
    RepoDir          string        `toml:"repo_dir"`
    WorktreesRoot    string        `toml:"worktrees_root"`
    BaseBranch       string        `toml:"base_branch"`        // defaults to "main"
    TrackingLabel    string        `toml:"tracking_label"`
    ConcurrencyLimit int           `toml:"concurrency_limit"`
    CopyFiles        []string      `toml:"copy_files"`
    ClosedPRStatus   string        `toml:"closed_pr_status"`
}
```

### Config Loading and Validation

```go
func LoadOrchestratorConfig(path string) (*OrchestratorConfig, error) {
    var cfg OrchestratorConfig
    if _, err := toml.DecodeFile(path, &cfg); err != nil {
        return nil, fmt.Errorf("parse orchestrator config %s: %w", path, err)
    }
    cfg.applyDefaults()  // base_branch="main", etc.
    cfg.inheritCredentials()  // per-project inherits from global if empty
    if err := cfg.Validate(); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

Validation (all at parse time, fail-fast):
- Project IDs match `[a-z0-9][a-z0-9-]*`
- No duplicate project IDs
- No duplicate `(github_owner, github_repo)` pairs
- Required fields present: `id`, `linear_project_id`, `github_owner`, `github_repo`, `repo_dir`
- Each `repo_dir` contains `.git`
- Create `worktrees_root` dirs (or verify they exist)
- Warn on overlapping `worktrees_root` paths
- At least one credential source (global or per-project) for linear_api_key and github_token

### CLI Flags

Old single-project CLI flags are removed. The only way to run the orchestrator is with `--config <path>`. A few global overrides remain for operational convenience:

```go
func runDaemon(c *cli.Context) error {
    configPath := c.String("config")
    if configPath == "" {
        return fmt.Errorf("--config is required")
    }
    cfg, err := LoadOrchestratorConfig(configPath)
    // Optional CLI overrides for globals only, using c.IsSet()
    if c.IsSet("log-level") { cfg.Global.LogLevel = c.String("log-level") }
    if c.IsSet("callback-port") { cfg.Global.CallbackPort = c.Int("callback-port") }
    if c.IsSet("db-path") { cfg.Global.DBPath = c.String("db-path") }
}
```

## Updated StateStore Interface

Full interface with project scoping. Changes marked with `// CHANGED`.

```go
type StateStore interface {
    Migrate(ctx context.Context) error
    Close() error

    // Jobs — composite key (projectID, ticketID)
    CreateJob(ctx context.Context, projectID, ticketID, worktreePath, cmuxSession string) error  // CHANGED: projectID is first param (was separate)
    GetJob(ctx context.Context, projectID, ticketID string) (*Job, error)                        // CHANGED: added projectID
    GetJobByPR(ctx context.Context, projectID string, prNumber int64) (*Job, error)              // CHANGED: added projectID
    ListAllJobs(ctx context.Context) ([]Job, error)                                              // UNCHANGED: admin use, returns all projects
    ListJobsByStatus(ctx context.Context, projectID string, statuses ...string) ([]Job, error)   // CHANGED: added projectID filter
    DeleteJob(ctx context.Context, projectID, ticketID string) error                             // CHANGED: added projectID
    SetJobStatus(ctx context.Context, projectID, ticketID string, status string) error           // CHANGED: added projectID
    SetPR(ctx context.Context, projectID, ticketID string, prNumber int64) error                 // CHANGED: added projectID

    // Comment watermarks — composite key (projectID, prNumber)
    GetCommentWatermark(ctx context.Context, projectID string, prNumber int64) (int64, error)    // CHANGED: added projectID
    SetCommentWatermark(ctx context.Context, projectID string, prNumber int64, commentID int64) error // CHANGED: added projectID

    // Concurrency slots — already per-project, no change
    TryAcquireSlot(ctx context.Context, projectID string, limit int) (bool, error)
    ReleaseSlot(ctx context.Context, projectID string) error
    ResetAllSlots(ctx context.Context) (int, error)
    ListSlots(ctx context.Context) ([]ConcurrencySlot, error)
}
```

All job/watermark methods now require `projectID` as first param after `ctx`. `ListAllJobs` remains global for admin use.

## Schema Changes

### Migration 003: Composite Primary Keys (clean break)

Drops existing data and recreates tables with composite PKs. No backfill — restart agents after upgrade.

```sql
-- Drop and recreate jobs with composite PK
DROP TABLE IF EXISTS jobs;
CREATE TABLE jobs (
    project_id    TEXT NOT NULL,
    ticket_id     TEXT NOT NULL,
    worktree_path TEXT NOT NULL,
    cmux_session  TEXT NOT NULL,
    pr_number     INTEGER,
    status        TEXT NOT NULL DEFAULT 'queued',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, ticket_id)
);
CREATE INDEX idx_jobs_status ON jobs (status);
CREATE INDEX idx_jobs_pr_number ON jobs (pr_number);
CREATE INDEX idx_jobs_project_status ON jobs (project_id, status);

-- Drop and recreate comment_watermarks with composite PK
DROP TABLE IF EXISTS comment_watermarks;
CREATE TABLE comment_watermarks (
    project_id                TEXT NOT NULL,
    pr_number                 INTEGER NOT NULL,
    last_processed_comment_id INTEGER NOT NULL,
    updated_at                TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, pr_number)
);
```

No PRAGMA handling needed — no data to preserve. No `--migrate-project-id` flag needed.

## ProjectRunner Type

Extracted from current `Orchestrator`. Takes per-project deps, shares global deps.

```go
type ProjectRunner struct {
    id        string
    cfg       ProjectConfig
    store     state.StateStore      // shared
    linear    linear.Client         // shared (project-agnostic)
    github    github.Client         // per-project
    worktrees worktree.Manager      // per-project
    sessions  cmux.SessionManager   // shared
    events    <-chan callback.Event  // per-project channel from EventDemuxer
    logger    *slog.Logger          // with slog.String("project", id)
}
```

All current `*Orchestrator` methods (`pollAndProcess`, `processTicket`, `setupWorktree`, `pollPRLifecycle`, `cleanupPR`, `pollComments`, `processComment`) move to `*ProjectRunner`. They replace `o.cfg.ProjectID` with `p.id` and `o.cfg.*` with `p.cfg.*`. The per-project GitHub client and worktree manager mean no cross-project data leakage.

### RunSupervised Pattern

```go
func (p *ProjectRunner) RunSupervised(ctx context.Context) error {
    watchers := map[string]func(context.Context) error{
        "linear-poller":   p.linearPoller,
        "pr-watcher":      p.prWatcher,
        "comment-watcher": p.commentWatcher,
        "event-router":    p.eventRouter,
    }

    var wg sync.WaitGroup
    for name, fn := range watchers {
        wg.Add(1)
        go func(name string, fn func(context.Context) error) {
            defer wg.Done()
            p.runWithRestart(ctx, name, fn)
        }(name, fn)
    }

    wg.Wait()
    return nil  // only reached after ctx cancellation — NEVER returns error
}
```

Key behavior:
- `RunSupervised` **never returns an error** to the top-level errgroup. This ensures one project's total failure doesn't trigger process-wide shutdown.
- Each watcher runs in `runWithRestart` which retries indefinitely with capped exponential backoff.
- `return nil` is reached only when `ctx` is cancelled (shutdown signal).
- Project health is surfaced via the `/healthz` endpoint, not via shutdown.

### runWithRestart Pattern

```go
func (p *ProjectRunner) runWithRestart(ctx context.Context, name string, fn func(context.Context) error) {
    backoff := time.Second
    maxBackoff := 2 * time.Minute
    lastSuccess := time.Now()

    for {
        err := fn(ctx)
        if ctx.Err() != nil {
            p.logger.Info("watcher stopped", slog.String("watcher", name))
            return
        }
        // If watcher ran for at least one poll interval, reset backoff
        if time.Since(lastSuccess) > p.cfg.PollInterval {
            backoff = time.Second
        }
        p.logger.Error("watcher failed, restarting",
            slog.String("watcher", name),
            slog.String("err", err.Error()),
            slog.Duration("backoff", backoff),
        )
        select {
        case <-ctx.Done():
            return
        case <-time.After(backoff):
        }
        lastSuccess = time.Now()
        backoff = min(backoff*2, maxBackoff)
    }
}
```

Backoff resets to 1s after a watcher runs successfully for at least one poll interval. A watcher that crashes immediately on every attempt escalates to 2min backoff. A watcher that runs for 30s+ before crashing resets.

## EventDemuxer

New component between callback server and per-project routers.

```go
type EventDemuxer struct {
    mu       sync.RWMutex
    channels map[string]chan<- callback.Event  // projectID → per-project channel
    logger   *slog.Logger
}

func NewEventDemuxer(logger *slog.Logger) *EventDemuxer {
    return &EventDemuxer{
        channels: make(map[string]chan<- callback.Event),
        logger:   logger,
    }
}
```

### Registration API

```go
// Register adds a per-project event channel. Called by each ProjectRunner
// during setup, before the callback server starts accepting connections.
// Returns the receive end of the channel.
func (d *EventDemuxer) Register(projectID string) <-chan callback.Event {
    ch := make(chan callback.Event, 64)  // same buffer as current single-channel design
    d.mu.Lock()
    d.channels[projectID] = ch
    d.mu.Unlock()
    return ch
}

// Deregister removes a project's channel and closes it.
// Called during shutdown cleanup.
func (d *EventDemuxer) Deregister(projectID string) {
    d.mu.Lock()
    if ch, ok := d.channels[projectID]; ok {
        close(ch)
        delete(d.channels, projectID)
    }
    d.mu.Unlock()
}
```

### Run Loop

```go
// Run consumes events from the callback server and routes to per-project channels.
// Runs as a ServiceFunc in the top-level errgroup.
func (d *EventDemuxer) Run(ctx context.Context, allEvents <-chan callback.Event) error {
    for {
        select {
        case <-ctx.Done():
            // Close all registered channels on shutdown
            d.mu.Lock()
            for id, ch := range d.channels {
                close(ch)
                delete(d.channels, id)
            }
            d.mu.Unlock()
            return nil
        case evt, ok := <-allEvents:
            if !ok {
                return nil
            }
            d.mu.RLock()
            ch, found := d.channels[evt.ProjectID]
            d.mu.RUnlock()
            if !found {
                d.logger.Warn("event for unknown project",
                    slog.String("project_id", evt.ProjectID),
                    slog.String("ticket_id", evt.TicketID),
                )
                continue  // drop event, don't crash
            }
            select {
            case ch <- evt:
            default:
                d.logger.Warn("project event queue full, dropping event",
                    slog.String("project_id", evt.ProjectID),
                    slog.String("ticket_id", evt.TicketID),
                )
            }
        }
    }
}
```

Behavior:
- Events for unknown project IDs are logged and dropped (handles stale agents, typos)
- Full per-project channels cause event drops with a warning (backpressure isolation — project A's full queue doesn't block project B)
- Buffer size: 64 per project (matches current single-channel design)
- Registration is thread-safe (RWMutex)
- On shutdown: all channels closed, preventing goroutine leaks in ProjectRunners

### Startup Ordering

1. Create EventDemuxer
2. For each project: `demuxer.Register(projectID)` → returns `<-chan Event` → passed to ProjectRunner
3. Start callback server (begins accepting HTTP connections)
4. Start EventDemuxer.Run (begins consuming from callback server's event channel)
5. Start all ProjectRunners (begin consuming from their per-project channels)

This ordering ensures channels are registered before events can arrive.

## Callback Server Changes

### Routes

```go
func (s *CallbackServer) registerRoutes() {
    s.mux.HandleFunc("POST /project/{projectID}/{ticketID}/complete", s.handleComplete)
    s.mux.HandleFunc("POST /project/{projectID}/{ticketID}/comment-resolved", s.handleCommentResolved)
    s.mux.HandleFunc("POST /project/{projectID}/{ticketID}/failed", s.handleFailed)
    s.mux.HandleFunc("GET /healthz", s.handleHealthz)
}
```

No legacy URL support. No store dependency on the callback server.

### Event Struct Change

```go
type Event struct {
    Kind      EventKind
    ProjectID string    // NEW: extracted from URL path
    TicketID  string
    Timestamp time.Time
    // ...existing fields...
}
```

### Callback URL Construction

In `watcher_linear.go` (now on ProjectRunner), the callback URL changes to:

```go
fmt.Sprintf("http://localhost:%d/project/%s/%s", p.cfg.CallbackPort, p.id, ticket.Identifier)
```

## Composition Root (cmd/orchestrator/run.go)

```go
func runDaemon(c *cli.Context) error {
    configPath := c.String("config")
    if configPath == "" {
        return fmt.Errorf("--config is required")
    }
    cfg, err := LoadOrchestratorConfig(configPath)
    // Apply optional CLI overrides to globals
    initLogger(cfg.Global)

    store := state.NewSQLiteStore(cfg.Global.DBPath)
    defer store.Close()

    state.ApplyReconciliation(ctx, store, cfg.projectIDs(), logger)  // global, once

    linearClient := linear.NewClient(cfg.Global.LinearAPIKey)
    callbackSrv := callback.New(cfg.Global.CallbackPort)
    demuxer := callback.NewEventDemuxer(logger)

    var runners []shutdown.ServiceFunc
    runners = append(runners, callbackSrv.Run)
    runners = append(runners, func(ctx context.Context) error {
        return demuxer.Run(ctx, callbackSrv.Events())
    })

    for _, projCfg := range cfg.Projects {
        events := demuxer.Register(projCfg.ID)
        ghClient := github.NewClient(projCfg.GitHubToken, projCfg.GitHubOwner, projCfg.GitHubRepo)
        wtMgr := worktree.New(projCfg.RepoDir, projCfg.WorktreesRoot, projCfg.CopyFiles)
        sessMgr := cmux.New(/* sandbox config from global */)

        runner := NewProjectRunner(projCfg, store, linearClient, ghClient, wtMgr, sessMgr, events, logger)
        runners = append(runners, runner.RunSupervised)
    }

    mgr := shutdown.New(cfg.Global.ShutdownTimeout)
    return mgr.Run(runners...)
}
```

## Reconciliation Changes

```go
func ApplyReconciliation(ctx context.Context, store StateStore, configuredProjects []string, logger *slog.Logger) {
    // Existing reconciliation logic (reset stale jobs, etc.)
    // NEW: check for orphaned jobs
    allJobs, _ := store.ListAllJobs(ctx)
    configured := toSet(configuredProjects)
    for _, job := range allJobs {
        if !configured[job.ProjectID] {
            logger.Warn("orphaned job from unconfigured project",
                slog.String("project_id", job.ProjectID),
                slog.String("ticket_id", job.TicketID),
            )
            store.ReleaseSlot(ctx, job.ProjectID)
        }
    }
}
```

## Bug Fix: handleComplete Slot Release

The router's `handleComplete` must call `r.store.ReleaseSlot(ctx, projectID)` after successfully creating the PR. Currently only `handleFailed` and `handleCommentResolved` release slots. The PR watcher also releases on merge/close, but there's a gap between PR creation and merge where the slot is held unnecessarily.

## Files That Need Changes

| File | Change |
|------|--------|
| `internal/orchestrator/config.go` | Replace with TOML config types (`OrchestratorConfig`, `GlobalConfig`, `ProjectConfig`) |
| `cmd/orchestrator/run.go` | New composition root: load TOML, construct per-project runners, EventDemuxer |
| `cmd/orchestrator/main.go` | Replace CLI flags with `--config`; update `jobs`/`slots` subcommand output |
| `internal/orchestrator/orchestrator.go` | Extract `ProjectRunner` type; move watcher methods |
| `internal/orchestrator/watcher_linear.go` | Methods on `*ProjectRunner`, new callback URL format |
| `internal/orchestrator/watcher_pr.go` | Methods on `*ProjectRunner`, project-scoped `ListJobsByStatus` |
| `internal/orchestrator/watcher_comment.go` | Methods on `*ProjectRunner`, project-scoped queries |
| `internal/orchestrator/comment_dispatcher.go` | Per-project scoping |
| `internal/orchestrator/router.go` | Methods on `*ProjectRunner`, project-scoped store calls, slot release fix |
| `internal/callback/server.go` | New URL pattern only, `ProjectID` in Event |
| `internal/callback/event.go` | Add `ProjectID` field |
| `internal/callback/demuxer.go` | New file: `EventDemuxer` |
| `internal/state/store.go` | Updated interface with `projectID` params |
| `internal/state/migrations/003_composite_pks.sql` | Table recreation for composite PKs |
| `internal/state/migrations.go` | No special handling needed for migration 003 (clean drop+create) |
| `internal/state/reconciliation.go` | Orphaned job detection |
| All `_test.go` files for changed packages | Updated to pass `projectID` params |
