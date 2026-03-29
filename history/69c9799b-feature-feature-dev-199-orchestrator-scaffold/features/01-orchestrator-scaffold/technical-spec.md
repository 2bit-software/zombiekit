# Technical Specification: Orchestrator Scaffold

## 1. Config (`internal/orchestrator/config.go`)

```go
package orchestrator

import (
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/urfave/cli/v2"
)

var validLogLevels = map[string]bool{
    "debug": true, "info": true, "warn": true, "error": true,
}

type Config struct {
    LinearAPIKey     string
    GitHubToken      string
    CallbackPort     int
    WorktreesRoot    string
    DBPath           string
    ConcurrencyLimit int
    PollInterval     time.Duration
    LogLevel         string
    LogJSON          bool
    ShutdownTimeout  time.Duration
}

func NewConfig(c *cli.Context) (*Config, error) {
    cfg := &Config{
        LinearAPIKey:     c.String("linear-api-key"),
        GitHubToken:      c.String("github-token"),
        CallbackPort:     c.Int("callback-port"),
        WorktreesRoot:    c.String("worktrees-root"),
        DBPath:           c.String("db-path"),
        ConcurrencyLimit: c.Int("concurrency-limit"),
        PollInterval:     c.Duration("poll-interval"),
        LogLevel:         c.String("log-level"),
        LogJSON:          c.Bool("log-json"),
        ShutdownTimeout:  c.Duration("shutdown-timeout"),
    }
    if err := cfg.Validate(); err != nil {
        return nil, err
    }
    return cfg, nil
}

func (c *Config) Validate() error {
    var errs []string

    if c.LinearAPIKey == "" {
        errs = append(errs, "--linear-api-key/ORCH_LINEAR_API_KEY is required")
    }
    if c.GitHubToken == "" {
        errs = append(errs, "--github-token/ORCH_GITHUB_TOKEN is required")
    }
    if c.CallbackPort < 1 || c.CallbackPort > 65535 {
        errs = append(errs, "--callback-port/ORCH_CALLBACK_PORT must be 1-65535")
    }
    if c.WorktreesRoot == "" {
        errs = append(errs, "--worktrees-root/ORCH_WORKTREES_ROOT is required")
    }
    if c.DBPath == "" {
        errs = append(errs, "--db-path/ORCH_DB_PATH is required")
    }
    if c.ConcurrencyLimit < 1 {
        errs = append(errs, "--concurrency-limit/ORCH_CONCURRENCY_LIMIT must be >= 1")
    }
    if c.PollInterval <= 0 {
        errs = append(errs, "--poll-interval/ORCH_POLL_INTERVAL must be > 0")
    }
    if !validLogLevels[c.LogLevel] {
        errs = append(errs, "--log-level/ORCH_LOG_LEVEL must be one of: debug, info, warn, error")
    }
    if c.ShutdownTimeout <= 0 {
        errs = append(errs, "--shutdown-timeout/ORCH_SHUTDOWN_TIMEOUT must be > 0")
    }

    if len(errs) > 0 {
        return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
    }

    // Side effect: create worktrees directory
    if c.WorktreesRoot != "" {
        if err := os.MkdirAll(c.WorktreesRoot, 0o755); err != nil {
            return fmt.Errorf("create worktrees directory: %w", err)
        }
    }

    return nil
}
```

### Design Notes

- `Validate()` is exported separately from `NewConfig` so tests can construct a `Config` directly and validate without needing a `cli.Context`.
- Validation collects all errors into a single string. Uses `fmt.Errorf` rather than `errors.Join` because the output is a single human-readable message, not a chain of wrapped errors.
- `os.MkdirAll` for worktrees root is a side effect in validation — this is intentional per the spec. It only runs after all pure validation passes.

## 2. Watcher Stubs (`internal/orchestrator/watchers.go`)

```go
package orchestrator

import (
    "context"
    "log/slog"
    "time"

    "github.com/zombiekit/brains/internal/logging"
    "github.com/zombiekit/brains/internal/shutdown"
)

const (
    WatcherLinearPoller  = "linear-poller"
    WatcherPRWatcher     = "pr-watcher"
    WatcherCommentWatcher = "comment-watcher"
)

func NewWatcherStub(name string, pollInterval time.Duration) shutdown.ServiceFunc {
    return func(ctx context.Context) error {
        logger := logging.Logger().With(slog.String("watcher", name))
        logger.Info("watcher started", slog.Duration("poll_interval", pollInterval))
        <-ctx.Done()
        logger.Info("watcher stopped")
        return nil
    }
}
```

### Design Notes

- Factory function returns a closure. The `pollInterval` is captured but unused by stubs — it's there so the signature doesn't change when real watchers replace the stubs.
- Returns `nil`, not `ctx.Err()`. This is critical — errgroup propagates non-nil errors.
- Uses the logging singleton. Logger must be initialized before these run (guaranteed by startup sequence).

## 3. Orchestrator (`internal/orchestrator/orchestrator.go`)

```go
package orchestrator

import (
    "context"
    "fmt"

    "github.com/zombiekit/brains/internal/callback"
    "github.com/zombiekit/brains/internal/logging"
    "github.com/zombiekit/brains/internal/shutdown"
    "github.com/zombiekit/brains/internal/state"
)

type Orchestrator struct {
    cfg   *Config
    store state.StateStore
}

func New(cfg *Config, store state.StateStore) *Orchestrator {
    return &Orchestrator{cfg: cfg, store: store}
}

func (o *Orchestrator) Run() error {
    logger := logging.Logger()

    // Step 1: Reconciliation (must complete before services start)
    if err := state.ApplyReconciliation(context.Background(), o.store, logger); err != nil {
        return fmt.Errorf("reconciliation: %w", err)
    }

    // Step 2: Build services
    callbackSrv := callback.New(o.cfg.CallbackPort)

    linearPoller := NewWatcherStub(WatcherLinearPoller, o.cfg.PollInterval)
    prWatcher := NewWatcherStub(WatcherPRWatcher, o.cfg.PollInterval)
    commentWatcher := NewWatcherStub(WatcherCommentWatcher, o.cfg.PollInterval)

    // Step 3: Run all services under shutdown manager
    logger.Info("starting services")
    mgr := shutdown.New(o.cfg.ShutdownTimeout)
    return mgr.Run(callbackSrv.Run, linearPoller, prWatcher, commentWatcher)
}
```

### Design Notes

- `Run()` takes no context. The shutdown manager creates its own. Using `context.Background()` for reconciliation is intentional — it's a short-lived pre-service operation.
- Reconciliation failure short-circuits: no callback server, no watchers, no shutdown manager.
- Service construction (callback.New, NewWatcherStub) happens synchronously before `mgr.Run()`. This separates construction errors (synchronous, fatal) from runtime errors (managed by errgroup).
- The callback server's `Events()` channel is not consumed by this scaffold — that's deferred to future tickets.

## 4. Entry Point (`cmd/orchestrator/main.go`)

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "github.com/urfave/cli/v2"
    "github.com/zombiekit/brains/internal/logging"
    "github.com/zombiekit/brains/internal/orchestrator"
    "github.com/zombiekit/brains/internal/state"
    "github.com/zombiekit/brains/internal/version"
)

func main() {
    app := &cli.App{
        Name:    "orchestrator",
        Usage:   "ZombieKit autonomous development orchestrator",
        Version: version.Get().Short(),
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:    "linear-api-key",
                Usage:   "Linear API key",
                EnvVars: []string{"ORCH_LINEAR_API_KEY"},
            },
            &cli.StringFlag{
                Name:    "github-token",
                Usage:   "GitHub personal access token",
                EnvVars: []string{"ORCH_GITHUB_TOKEN"},
            },
            &cli.IntFlag{
                Name:    "callback-port",
                Usage:   "HTTP callback server port",
                Value:   8666,
                EnvVars: []string{"ORCH_CALLBACK_PORT"},
            },
            &cli.StringFlag{
                Name:    "worktrees-root",
                Usage:   "Root directory for git worktrees",
                EnvVars: []string{"ORCH_WORKTREES_ROOT"},
            },
            &cli.StringFlag{
                Name:    "db-path",
                Usage:   "Path to SQLite database file",
                EnvVars: []string{"ORCH_DB_PATH"},
            },
            &cli.IntFlag{
                Name:    "concurrency-limit",
                Usage:   "Max concurrent jobs per project",
                Value:   1,
                EnvVars: []string{"ORCH_CONCURRENCY_LIMIT"},
            },
            &cli.DurationFlag{
                Name:    "poll-interval",
                Usage:   "Watcher polling interval",
                Value:   30 * time.Second,
                EnvVars: []string{"ORCH_POLL_INTERVAL"},
            },
            &cli.StringFlag{
                Name:    "log-level",
                Usage:   "Log level (debug, info, warn, error)",
                Value:   "info",
                EnvVars: []string{"ORCH_LOG_LEVEL"},
            },
            &cli.BoolFlag{
                Name:    "log-json",
                Usage:   "Output logs as JSON",
                EnvVars: []string{"ORCH_LOG_JSON"},
            },
            &cli.DurationFlag{
                Name:    "shutdown-timeout",
                Usage:   "Max time to drain on shutdown",
                Value:   30 * time.Second,
                EnvVars: []string{"ORCH_SHUTDOWN_TIMEOUT"},
            },
        },
        Action: run,
    }

    if err := app.Run(os.Args); err != nil {
        slog.Error("orchestrator failed", "error", err)
        os.Exit(1)
    }
}

func run(c *cli.Context) error {
    cfg, err := orchestrator.NewConfig(c)
    if err != nil {
        return err
    }

    logging.InitLogger(cfg.LogLevel, cfg.LogJSON, nil)
    logging.Logger().Info("orchestrator starting",
        slog.String("version", version.Get().Short()),
    )

    ctx := context.Background()
    store, err := state.NewSQLiteStore(ctx, cfg.DBPath)
    if err != nil {
        return err
    }
    defer store.Close()

    return orchestrator.New(cfg, store).Run()
}
```

### Design Notes

- `main()` follows the same pattern as `cmd/zk-server/main.go`.
- `slog.Error` in main uses the default logger (before InitLogger is called). This is fine — it only fires if the app fails to start.
- `run()` is ~15 lines. Resource acquisition only: config, logger, store, then delegate to `Orchestrator.Run()`.
- No `signal.NotifyContext` — the shutdown manager handles signals.

## 5. Test Approach

### Config Tests (`internal/orchestrator/config_test.go`)

Test `Validate()` directly with constructed `Config` structs (no need for `cli.Context`):

```go
func TestValidate_ValidConfig(t *testing.T) {
    cfg := validConfig(t) // helper that returns a fully-valid Config
    assert.NoError(t, cfg.Validate())
}

func TestValidate_MissingLinearAPIKey(t *testing.T) {
    cfg := validConfig(t)
    cfg.LinearAPIKey = ""
    err := cfg.Validate()
    assert.ErrorContains(t, err, "--linear-api-key/ORCH_LINEAR_API_KEY is required")
}

func TestValidate_MultipleErrors(t *testing.T) {
    cfg := &Config{} // everything zero
    err := cfg.Validate()
    assert.ErrorContains(t, err, "--linear-api-key")
    assert.ErrorContains(t, err, "--github-token")
    assert.ErrorContains(t, err, "--db-path")
}
```

Use `t.TempDir()` for WorktreesRoot to verify `os.MkdirAll` behavior without polluting the filesystem.

### Orchestrator Tests (`internal/orchestrator/orchestrator_test.go`)

Use a mock `state.StateStore` that records method calls:

```go
type mockStore struct {
    calls         []string
    reconcileErr  error
}

func (m *mockStore) ListJobsByStatus(ctx context.Context, statuses ...string) ([]state.Job, error) {
    m.calls = append(m.calls, "ListJobsByStatus")
    if m.reconcileErr != nil {
        return nil, m.reconcileErr
    }
    return []state.Job{}, nil
}
// ... implement remaining StateStore methods as no-ops
```

Key tests:
- Reconciliation runs (ListJobsByStatus called)
- Reconciliation error prevents services (Run returns error, no port bound)
- Clean lifecycle with immediate shutdown (test that Run returns nil when services stop)

Logger setup for tests:
```go
func setupLogger(t *testing.T) {
    t.Helper()
    logging.ResetLogger()
    logging.InitLogger("debug", false, nil)
    t.Cleanup(logging.ResetLogger)
}
```

### Watcher Tests (`internal/orchestrator/watchers_test.go`)

```go
func TestWatcherStub_ReturnsNilOnCancel(t *testing.T) {
    setupLogger(t)
    ctx, cancel := context.WithCancel(context.Background())
    stub := NewWatcherStub("test-watcher", 30*time.Second)
    cancel()
    err := stub(ctx)
    assert.NoError(t, err)
}

func TestWatcherStub_BlocksUntilCancel(t *testing.T) {
    setupLogger(t)
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    stub := NewWatcherStub("test-watcher", 30*time.Second)
    done := make(chan error, 1)
    go func() { done <- stub(ctx) }()

    select {
    case <-done:
        t.Fatal("stub returned before context cancelled")
    case <-time.After(50 * time.Millisecond):
        // still blocking — correct
    }
    cancel()
    assert.NoError(t, <-done)
}
```

## AC Traceability

| AC | Covered By | How |
|----|-----------|-----|
| AC-1 | T3 (orchestrator.go) + T6 (tests) | Run() calls reconciliation, builds services, passes to shutdown manager |
| AC-2 | Existing shutdown.Manager tests | Manager cancels context on SIGTERM; watchers return nil |
| AC-3 | T1 (config.go) + T5 (tests) | Validate() collects all errors, NewConfig returns before anything else |
| AC-4 | T4 (main.go) | state.NewSQLiteStore handles DB creation + migration |
| AC-5 | T3 (orchestrator.go) + T6 (tests) | Run() calls reconciliation synchronously before mgr.Run() |
| AC-6 | Existing shutdown.Manager tests | Manager handles double-signal with os.Exit(1) |
| AC-7 | T3 (orchestrator.go) | callback.Server.Run fails on net.Listen; error propagates through errgroup |
| AC-8 | T4 (main.go) | logging.InitLogger(level, cfg.LogJSON, nil) |
| AC-9 | T2 (watchers.go) + T7 (tests) | Stubs log start/stop with watcher name |
| AC-10 | T4 (main.go) | defer store.Close() in run(), after orchestrator.New(cfg, store).Run() returns |
