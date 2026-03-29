# Tasks: Orchestrator Scaffold (DEV-199)

**Complexity**: Simple (7 new files, 0 modified, ~450 LOC)
**Critical Path**: T1 -> T2 -> T3 -> T4 -> T7 (verify)

## Wave 1: Config

- [x] T1 [AC-3] Create `internal/orchestrator/config.go`
  - Config struct with 10 fields (LinearAPIKey, GitHubToken, CallbackPort, WorktreesRoot, DBPath, ConcurrencyLimit, PollInterval, LogLevel, LogJSON, ShutdownTimeout)
  - `NewConfig(c *cli.Context) (*Config, error)` — parses CLI context, calls Validate
  - `Validate() error` — collects all errors into multi-error string, creates worktrees dir on success
  - See technical-spec.md Section 1 for complete implementation

## Wave 2: Stubs (depends on T1)

- [x] T2 [P] [AC-9] Create `internal/orchestrator/watchers.go`
  - Constants: `WatcherLinearPoller`, `WatcherPRWatcher`, `WatcherCommentWatcher`
  - `NewWatcherStub(name string, pollInterval time.Duration) shutdown.ServiceFunc`
  - Returns closure: log started -> block on ctx.Done -> log stopped -> return nil
  - See technical-spec.md Section 2 for complete implementation

## Wave 3: Orchestrator (depends on T1, T2)

- [x] T3 [AC-1, AC-5, AC-7] Create `internal/orchestrator/orchestrator.go`
  - `Orchestrator` struct with `cfg *Config` and `store state.StateStore`
  - `New(cfg *Config, store state.StateStore) *Orchestrator`
  - `Run() error`: reconciliation -> build callback server + watchers -> shutdown.Manager.Run()
  - See technical-spec.md Section 3 for complete implementation

## Wave 4: Entry point (depends on T1, T3)

- [x] T4 [AC-3, AC-4, AC-8, AC-10] Create `cmd/orchestrator/main.go`
  - urfave/cli app with 10 flags (EnvVars with ORCH_ prefix, DurationFlag for durations)
  - `run(c *cli.Context) error`: NewConfig -> InitLogger -> NewSQLiteStore (defer Close) -> New + Run
  - Version: `version.Get().Short()`
  - See technical-spec.md Section 4 for complete implementation

## Wave 5: Tests (parallel, depend on T1-T4)

- [x] T5 [P] [AC-3] Create `internal/orchestrator/config_test.go`
  - `TestValidate_ValidConfig` — valid config passes
  - `TestValidate_MissingLinearAPIKey` — error contains flag/env name
  - `TestValidate_MissingGitHubToken` — error contains flag/env name
  - `TestValidate_InvalidCallbackPort` — 0 and 65536 both fail
  - `TestValidate_MissingRequiredStrings` — worktrees root, db path
  - `TestValidate_InvalidConcurrencyLimit` — 0 and -1 fail
  - `TestValidate_InvalidDurations` — zero poll interval and shutdown timeout fail
  - `TestValidate_InvalidLogLevel` — "banana" fails
  - `TestValidate_MultipleErrors` — empty config shows all errors at once
  - `TestValidate_WorktreesDirCreated` — valid config creates the directory
  - Use `t.TempDir()` for worktrees root paths
  - See technical-spec.md Section 5 for patterns

- [x] T6 [P] [AC-1, AC-5] Create `internal/orchestrator/orchestrator_test.go`
  - Mock `state.StateStore` with call recording
  - `TestRun_ReconciliationRunsBeforeServices` — verify ListJobsByStatus called (reconciliation ran)
  - `TestRun_ReconciliationFailure_PreventsServices` — mock returns error, verify Run returns it, no port bound
  - Logger setup: `logging.ResetLogger()` + `logging.InitLogger("debug", false, nil)` with `t.Cleanup`
  - Do NOT use `t.Parallel()` (logging singleton)
  - See technical-spec.md Section 5 for mock pattern

- [x] T7 [P] [AC-9] Create `internal/orchestrator/watchers_test.go`
  - `TestWatcherStub_ReturnsNilOnCancel` — cancel context before calling, verify nil return
  - `TestWatcherStub_BlocksUntilCancel` — verify stub blocks, then cancel and verify nil return
  - Logger setup same as T6
  - See technical-spec.md Section 5 for implementation

## Dependency Graph

```
T1 ─────────────┬──── T5 [P]
                │
T2 (needs T1) ──┼──── T7 [P]
                │
T3 (needs T1,T2)┼──── T6 [P]
                │
T4 (needs T1,T3)┘
```

## Validation

| AC | Task(s) |
|----|---------|
| AC-1 | T3, T6 |
| AC-2 | (existing shutdown.Manager) |
| AC-3 | T1, T4, T5 |
| AC-4 | T4 |
| AC-5 | T3, T6 |
| AC-6 | (existing shutdown.Manager) |
| AC-7 | T3 |
| AC-8 | T4 |
| AC-9 | T2, T7 |
| AC-10 | T4 |

All 10 ACs covered. AC-2 and AC-6 are handled by the existing `shutdown.Manager` (already tested).

## Execution Order

Sequential: T1 -> T2 -> T3 -> T4
Then parallel: T5, T6, T7

**Suggested next**: `/brains.next` to begin implementation
