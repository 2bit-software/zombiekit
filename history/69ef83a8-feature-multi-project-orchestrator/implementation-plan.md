# Implementation Plan: Multi-Project Orchestrator

## Implementation Phases

Work is ordered by dependency. Phases 1-4 can be worked in parallel. Phases 3+5 are executed atomically (interface change + all callers updated together) to maintain compilability.

### Phase 1: Config Layer

**Goal**: TOML config parsing, validation, and Duration wrapper. No behavioral changes.

#### 1.1 Duration wrapper type
- **File**: `internal/orchestrator/config.go` (new section)
- **What**: Create a `Duration` type wrapping `time.Duration` with `UnmarshalText` for TOML decoding. BurntSushi/toml v1.6.0 does NOT decode `time.Duration` from strings natively.

#### 1.2 TOML config types
- **File**: `internal/orchestrator/config.go` — replace current `Config` struct
- **What**: Define `OrchestratorConfig`, `GlobalConfig`, `ProjectConfig` with `toml:"..."` tags
- **ProjectConfig `toml:"-"` fields** (copied from GlobalConfig during `applyDefaults()`):
  - `PollInterval Duration` — needed by `runWithRestart` backoff reset
  - `CallbackPort int` — needed for callback URL construction
  - `BotUsername string` — needed by `watcher_comment.go` for `filterBotComments`
  - `SandboxMode string` — "auto"/"enabled"/"disabled" from global
- **GlobalConfig fields**: includes `Sandbox string` for the sandbox mode flag
- **Keep**: Current `Config` struct temporarily as `LegacyConfig` (deleted in Phase 7)
- **Defaults**: `BaseBranch` defaults to `"main"`, `TrackingLabel` defaults to `"ai-managed"` (matches existing default), `ConcurrencyLimit` defaults to `1`

#### 1.3 Config loader and validator
- **File**: `internal/orchestrator/config.go`
- **What**: `LoadOrchestratorConfig(path) (*OrchestratorConfig, error)`
  - `toml.DecodeFile` → `applyDefaults()` → `inheritCredentials()` → `Validate()`
- **Validation rules**: per business spec (project ID regex, no duplicate IDs, no duplicate owner/repo, repo_dir has .git, create worktrees_root)
- **Reuse**: Extend the existing `configRule` pattern at `config.go:78-98` — same `msg` + `check` struct, new rules for `OrchestratorConfig`/`ProjectConfig`. Reuse filesystem checks (`.git` stat, `MkdirAll`) from `config.go:113-124`.
- **Test**: `internal/orchestrator/config_test.go` — valid config, missing fields, duplicate IDs, invalid project ID format

#### 1.4 Example config file
- **File**: `orchestrator.example.toml` at repo root
- **What**: Documented example with all fields and comments

### Phase 2: Schema Migration

**Goal**: New DB tables with composite PKs. Clean break — drops existing data.

#### 2.1 Migration 003 SQL
- **File**: `internal/state/migrations/003_composite_pks.sql`
- **What**: `DROP TABLE IF EXISTS jobs; CREATE TABLE jobs (... PRIMARY KEY (project_id, ticket_id)); DROP TABLE IF EXISTS comment_watermarks; CREATE TABLE comment_watermarks (... PRIMARY KEY (project_id, pr_number));`
- **Indexes**: `(status)`, `(pr_number)`, `(project_id, status)` on jobs

#### 2.2 Embed migration
- **File**: `internal/state/migrations.go`
- **What**: Existing `//go:embed migrations/*.sql` directive picks up 003 automatically. No runner changes needed.

### Phase 3+5: State Store + ProjectRunner Extraction (atomic)

**Why merged**: Changing the StateStore interface breaks all callers. The interface change (Phase 3) and caller updates (Phase 5) must land together to maintain compilability. All mock StateStore implementations in test files must also be updated simultaneously.

**Goal**: Updated store interface, ProjectRunner type, all watcher methods moved.

#### 3.1 Update StateStore interface
- **File**: `internal/state/store.go`
- **What**: Add `projectID string` as first param after `ctx` on: `CreateJob`, `GetJob`, `GetJobByPR`, `ListJobsByStatus`, `DeleteJob`, `SetJobStatus`, `SetPR`, `GetCommentWatermark`, `SetCommentWatermark`
- **Keep**: `ListAllJobs` unchanged (admin), `TryAcquireSlot`/`ReleaseSlot` already take `projectID`

#### 3.2 Update SQLiteStore implementation
- **File**: `internal/state/store.go` (implementation methods)
- **What**: Add `project_id` to all SQL queries. Every `WHERE ticket_id = ?` becomes `WHERE project_id = ? AND ticket_id = ?`

#### 3.3 Update Job struct
- **File**: `internal/state/store.go` (or `types.go`)
- **What**: Ensure `Job` struct has `ProjectID string` field and it's included in scan

#### 3.4 Update admin service
- **File**: `internal/admin/service.go`
- **What**: Add `projectID` to all StateStore calls: `GetJob`, `DeleteJob`, `ListJobsByStatus`, `SetJobStatus`. Admin commands accept `--project` filter flag. `ListAllJobs` remains unchanged.

#### 5.1 Create ProjectRunner type
- **File**: `internal/orchestrator/runner.go` (new file)
- **What**: `ProjectRunner` struct with per-project deps (github, worktrees, cfg) and shared deps (store, linear, sessions, events channel)
- **Constructor**: `NewProjectRunner(cfg ProjectConfig, store, linear, github, worktrees, sessions, events, logger)`
- **Sandbox handling**: `SandboxMode` on `ProjectConfig` (copied from global). Sandbox resolution (`resolveSandboxMode`) runs once in the composition root and the result (`SandboxAvailable bool`, `SandboxConfig sandbox.Config`) is passed via the session manager constructor. The session manager (`cmux.SessionManager`) is **shared** across projects since sandbox config is global.
- **Archiver/Auditor**: Pass `NoopArchiver{}` and `NoopAuditor{}` as currently done

#### 5.2 RunSupervised + runWithRestart
- **File**: `internal/orchestrator/runner.go`
- **What**: `RunSupervised(ctx) error` spawns 4 goroutines via WaitGroup, each wrapped in `runWithRestart`. Never returns error.
- **Backoff**: 1s initial, 2min cap, resets after running > 1 poll interval
- **Test**: `internal/orchestrator/runner_test.go` — test `runWithRestart` backoff reset, test `RunSupervised` never-returns-error contract

#### 5.3 Move LinearPoller to ProjectRunner
- **File**: `internal/orchestrator/watcher_linear.go`
- **What**: Change receiver from `*Orchestrator` to `*ProjectRunner`. Replace `o.cfg.ProjectID` with `p.id`, `o.cfg.*` with `p.cfg.*`, all store calls gain `p.id`.
- **Callback URL**: `fmt.Sprintf("http://localhost:%d/project/%s/%s", p.cfg.CallbackPort, p.id, ticket.Identifier)`
- **Sandbox**: References to `o.cfg.SandboxAvailable` and `o.cfg.SandboxConfig` → accessed through the shared session manager (which already has sandbox config baked in via `cmux.WithCommandBuilder`)
- **Return type**: `func(ctx context.Context) error` (called by runWithRestart)

#### 5.4 Move PRWatcher to ProjectRunner
- **File**: `internal/orchestrator/watcher_pr.go`
- **What**: Same receiver change. `ListJobsByStatus(ctx, p.id, statuses...)`. `o.github.*` → `p.github.*`.

#### 5.5 Move CommentWatcher to ProjectRunner
- **File**: `internal/orchestrator/watcher_comment.go`
- **What**: Same receiver change. Each ProjectRunner gets its own `CommentDispatcher`. `BotUsername` accessed via `p.cfg.BotUsername`.

#### 5.6 Move Router to ProjectRunner
- **File**: `internal/orchestrator/router.go`
- **What**: Router becomes a method on `*ProjectRunner`. Reads from `p.events <-chan callback.Event`. All store calls gain `p.id`.
- **Slot release in handleComplete**: Intentionally holds slot (slot = active PR being watched). Document, don't change.

#### 5.7 Health tracking
- **File**: `internal/orchestrator/runner.go`
- **What**: `ProjectRunner` tracks per-watcher health state. Exposed via `Health() ProjectHealth` method.

#### 5.8 Update all mock StateStores in tests
- **Files**: `orchestrator_test.go`, `watcher_pr_test.go`, `watcher_comment_test.go`, `watcher_linear_test.go`, `router_test.go`
- **What**: All mock/stub `StateStore` implementations gain `projectID` param on every changed method. Mechanical but required for compilation.

#### 5.9 Update store tests
- **File**: `internal/state/store_test.go`
- **What**: All test calls pass `projectID`. Test cross-project isolation.

#### 5.10 Delete old Orchestrator
- **File**: `internal/orchestrator/orchestrator.go`
- **What**: Remove `Orchestrator` struct and `Run()` method.

### Phase 4: Callback Server + EventDemuxer

**Goal**: New URL routes, ProjectID in events, EventDemuxer component.

#### 4.1 Add ProjectID to Event
- **File**: `internal/callback/event.go`
- **What**: Add `ProjectID string` field to `Event` struct

#### 4.2 New URL routes
- **File**: `internal/callback/server.go`
- **What**: Replace `POST /{ticketID}/complete` with `POST /project/{projectID}/{ticketID}/complete` (and same for comment-resolved, failed). Extract both from `r.PathValue()`.
- **Remove**: Store dependency from callback server

#### 4.3 EventDemuxer
- **File**: `internal/callback/demuxer.go` (new file)
- **Pattern note**: Borrow mutex-guarded map-of-channels structure from `CommentDispatcher` (`comment_dispatcher.go:37-42`), different semantics (static registration, continuous streaming, drop-on-full).
- **What**: `EventDemuxer` with `Register(projectID) <-chan Event`, `Deregister(projectID)`, `Run(ctx, <-chan Event) error`
- **Behavior**: 64-buffer channels per project, unknown project = warn+drop, full channel = warn+drop
- **Test**: `internal/callback/demuxer_test.go` — registration, routing, unknown project, full channel, shutdown

#### 4.4 Update callback tests
- **File**: `internal/callback/server_test.go`
- **What**: Update route tests for new URL format

### Phase 6: Composition Root + CLI

**Goal**: New entry point wiring everything together.

#### 6.1 Rewrite main.go
- **File**: `cmd/orchestrator/main.go`
- **What**: Replace all single-project CLI flags with `--config` (required) plus global overrides (`--log-level`, `--callback-port`, `--db-path`).
- **Admin commands**: `jobs list` and `slots` show `project_id` column. `jobs list` accepts `--project` filter.

#### 6.2 Rewrite run.go
- **File**: `cmd/orchestrator/run.go`
- **What**: New composition root:
  1. Load TOML config, apply CLI overrides
  2. Init logger
  3. Open store + migrate
  4. Resolve sandbox mode (`resolveSandboxMode` — same logic as current, reads `Sandbox` from global config)
  5. Run reconciliation (global, once)
  6. Create per-credential Linear clients (if all projects share the same `linear_api_key`, one client; if a project overrides it, separate client for that project)
  7. Create callback server + EventDemuxer
  8. For each project: register demuxer channel, create GitHub client, create worktree manager (`worktree.New(repoDir, worktree.WithWorktreesRoot(...), worktree.WithCopyFiles(...))`), create ProjectRunner
  9. Create shared session manager (`cmux.New(cmuxOpts...)` — sandbox config baked in via options)
  10. Run all via shutdown.Manager

#### 6.3 Reconciliation update
- **File**: `internal/state/reconcile.go`
- **What**: `ApplyReconciliation` gains `configuredProjects []string` param.
  - Use `ListAllJobs` for orphan detection (jobs with project_id not in configured set → warn + release slots)
  - Per configured project: `ListJobsByStatus(ctx, projectID, StatusInProgress)` → `PlanReconciliation` → mark stale jobs
  - `ResetAllSlots` at the end (same as current)

#### 6.4 Health endpoint
- **File**: `internal/callback/server.go` (healthz handler)
- **What**: `/healthz` returns JSON with global status + per-project watcher states. CallbackServer takes a `HealthProvider` interface: `func() map[string]ProjectHealth`. Composition root wires this to collect health from all ProjectRunners.

### Phase 7: Cleanup + Tests

#### 7.1 Delete LegacyConfig
- **File**: `internal/orchestrator/config.go`
- **What**: Remove old `Config` struct, `NewConfig(c *cli.Context)`, old validation rules

#### 7.2 Integration test
- **File**: `internal/orchestrator/integration_test.go`
- **What**: Start orchestrator with 2-project TOML config, verify both projects' watchers start, verify cross-project isolation in store

#### 7.3 Example TOML for deployment
- **File**: `orchestrator.example.toml`
- **What**: Production-ready example with comments

## Spec Corrections

### handleComplete slot release is intentional

The audit flagged `handleComplete` not releasing the slot as a bug. Research shows this is **by design**:

1. Linear poller picks up ticket → acquires slot → spawns agent
2. Agent completes → `handleComplete` creates PR → slot stays held
3. Comment watcher monitors PR → each comment acquires a NEW slot → resolved → releases that slot
4. PR watcher detects merge/close → releases the ORIGINAL slot from step 1

The slot represents "this project has an active PR consuming attention." Releasing it after PR creation would let the poller grab a new ticket while the existing PR is still being watched, potentially exceeding the intended concurrency limit.

### ClosedPRTicketStatus renamed to ClosedPRStatus

Current field is `ClosedPRTicketStatus` in `Config`. New field is `ClosedPRStatus` in `ProjectConfig`. Mechanical rename when moving methods.

## Dependency Graph

```
Phase 1 (Config) ─────────┐
                           │
Phase 2 (Migration) ───────┼──→ Phase 3+5 (Store + ProjectRunner) ──→ Phase 6 (Composition Root) ──→ Phase 7 (Cleanup)
                           │
Phase 4 (Callback+Demuxer)─┘
```

Phases 1, 2, 4 can be worked in parallel. Phase 3+5 is atomic and depends on 1, 2, 4. Phase 6 depends on 3+5. Phase 7 depends on 6.

## Files That Need Changes

| File | Phase | Change |
|------|-------|--------|
| `internal/orchestrator/config.go` | 1 | TOML config types, Duration wrapper, loader, validator |
| `internal/state/migrations/003_composite_pks.sql` | 2 | New migration (drop+create) |
| `internal/state/store.go` | 3+5 | Updated interface + implementation |
| `internal/admin/service.go` | 3+5 | Add projectID to store calls |
| `internal/orchestrator/runner.go` | 3+5 | New file: ProjectRunner, RunSupervised, runWithRestart, health |
| `internal/orchestrator/orchestrator.go` | 3+5 | Delete Orchestrator struct |
| `internal/orchestrator/watcher_linear.go` | 3+5 | Methods on *ProjectRunner |
| `internal/orchestrator/watcher_pr.go` | 3+5 | Methods on *ProjectRunner |
| `internal/orchestrator/watcher_comment.go` | 3+5 | Methods on *ProjectRunner |
| `internal/orchestrator/comment_dispatcher.go` | 3+5 | Per-project scoping (no structural change) |
| `internal/orchestrator/router.go` | 3+5 | Methods on *ProjectRunner |
| `internal/callback/event.go` | 4 | Add ProjectID field |
| `internal/callback/server.go` | 4,6 | New URL routes, health endpoint |
| `internal/callback/demuxer.go` | 4 | New file: EventDemuxer |
| `cmd/orchestrator/main.go` | 6 | New CLI flags |
| `cmd/orchestrator/run.go` | 6 | New composition root |
| `internal/state/reconcile.go` | 6 | Orphan detection, per-project scoping |
| 5+ test files | 3+5 | Mock updates, new tests |
