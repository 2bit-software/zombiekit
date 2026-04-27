# Tasks: Multi-Project Orchestrator

## Complexity
- **Files**: ~22 (17 source + 5+ test)
- **Classification**: Complex
- **Critical path**: T001 → T004 → T005 → T006 → T007 → T008 → T009 → T010

## Task List

### Foundation (parallelizable — no dependencies between them)

- [ ] T001 [P] **Config layer: TOML types, Duration wrapper, loader, validator, example file**
  - Files: `internal/orchestrator/config.go`, `internal/orchestrator/config_test.go`, `orchestrator.example.toml`
  - Create `Duration` wrapper type with `UnmarshalText` for BurntSushi/toml compatibility
  - Define `OrchestratorConfig`, `GlobalConfig`, `ProjectConfig` with `toml:"..."` tags
  - `ProjectConfig` has `toml:"-"` fields: `PollInterval`, `CallbackPort`, `BotUsername`, `SandboxMode` (copied from global in `applyDefaults()`)
  - `LoadOrchestratorConfig(path)`: DecodeFile → applyDefaults → inheritCredentials → Validate
  - Validator uses existing `configRule` pattern from `config.go:78-98`: project ID regex `[a-z0-9][a-z0-9-]*`, no duplicate IDs, no duplicate `(owner, repo)`, repo_dir has `.git`, create worktrees_root
  - Defaults: `BaseBranch`="main", `TrackingLabel`="ai-managed", `ConcurrencyLimit`=1
  - Keep current `Config` struct temporarily as `LegacyConfig` (deleted in T010)
  - Tests: valid config, missing required fields, duplicate project IDs, invalid ID format, credential inheritance
  - Write `orchestrator.example.toml` with all fields documented
  - **Acceptance**: `LoadOrchestratorConfig` parses example TOML correctly; validator rejects bad configs
  - **Traces to**: Plan Phase 1, Spec: Configuration, Config Validation

- [ ] T002 [P] **Schema migration 003: composite primary keys**
  - Files: `internal/state/migrations/003_composite_pks.sql`
  - Clean break: `DROP TABLE IF EXISTS jobs` + `CREATE TABLE jobs (... PRIMARY KEY (project_id, ticket_id))`
  - Same for `comment_watermarks`: `DROP TABLE IF EXISTS` + `CREATE TABLE (... PRIMARY KEY (project_id, pr_number))`
  - Indexes on jobs: `(status)`, `(pr_number)`, `(project_id, status)`
  - Existing `//go:embed migrations/*.sql` picks up the file automatically
  - **Acceptance**: Migration runs without error on fresh DB and on DB with existing tables
  - **Traces to**: Plan Phase 2, Spec: Migration

- [ ] T003 [P] **Callback server: new URL routes, Event.ProjectID, EventDemuxer**
  - Files: `internal/callback/event.go`, `internal/callback/server.go`, `internal/callback/demuxer.go` (new), `internal/callback/demuxer_test.go` (new), `internal/callback/server_test.go`
  - Add `ProjectID string` field to `Event` struct
  - Replace routes: `POST /{ticketID}/complete` → `POST /project/{projectID}/{ticketID}/complete` (same for comment-resolved, failed). Extract both via `r.PathValue()`
  - Remove store dependency from callback server (no legacy lookup)
  - Create `EventDemuxer`: `Register(projectID) <-chan Event`, `Deregister(projectID)`, `Run(ctx, <-chan Event) error`
  - Borrow mutex+map pattern from `CommentDispatcher` but with static registration, 64-buffer channels, drop-on-full semantics
  - Tests: registration, correct routing by projectID, unknown project → warn+drop, full channel → warn+drop, shutdown closes all channels
  - Update server tests for new URL format
  - **Acceptance**: Events routed to correct project channel; unknown projects don't crash
  - **Traces to**: Plan Phase 4, Spec: Callback Server Changes, EventDemuxer

### State Store (depends on T002)

- [ ] T004 **StateStore interface + SQLiteStore implementation update**
  - Files: `internal/state/store.go`
  - Add `projectID string` as first param after `ctx` on: `CreateJob`, `GetJob`, `GetJobByPR`, `ListJobsByStatus`, `DeleteJob`, `SetJobStatus`, `SetPR`, `GetCommentWatermark`, `SetCommentWatermark`
  - `ListAllJobs` unchanged (admin use). `TryAcquireSlot`/`ReleaseSlot` unchanged (already take projectID)
  - Every `WHERE ticket_id = ?` → `WHERE project_id = ? AND ticket_id = ?`
  - `ListJobsByStatus` gains `WHERE project_id = ? AND status IN (?)`
  - `GetJobByPR` gains `WHERE project_id = ? AND pr_number = ?`
  - Ensure `Job` struct has `ProjectID string` field included in scan
  - NOTE: This task intentionally breaks compilation of callers in `internal/orchestrator/` and `internal/admin/`. Those are fixed in T006 and T007. Compile check the `internal/state/` package only.
  - **Acceptance**: `internal/state/` compiles; SQL queries are project-scoped
  - **Traces to**: Plan Phase 3.1-3.3, Spec: Updated StateStore Interface

### ProjectRunner (depends on T001, T003, T004)

- [ ] T005 **ProjectRunner type, RunSupervised, runWithRestart, health tracking**
  - Files: `internal/orchestrator/runner.go` (new), `internal/orchestrator/runner_test.go` (new)
  - Define `ProjectRunner` struct: `id`, `cfg ProjectConfig`, `store`, `linear`, `github`, `worktrees`, `sessions`, `events <-chan callback.Event`, `dispatcher *CommentDispatcher`, `logger`, health tracking fields
  - Constructor: `NewProjectRunner(cfg ProjectConfig, store, linear, github, worktrees, sessions, events, logger)`
  - `RunSupervised(ctx) error`: spawns 4 goroutines (linear-poller, pr-watcher, comment-watcher, event-router) via `sync.WaitGroup`, each wrapped in `runWithRestart`. **Never returns error** — only returns nil on ctx cancellation.
  - `runWithRestart(ctx, name, fn)`: exponential backoff 1s→2min cap, resets after running > 1 poll interval
  - Health tracking: `watcherHealth` per watcher (last success, last error, consecutive fails, current backoff). `Health() ProjectHealth` method.
  - Tests: `runWithRestart` backoff reset logic, `RunSupervised` never-returns-error contract
  - NOTE: Watcher method stubs (linearPoller, prWatcher, commentWatcher, eventRouter) can return `ctx.Err()` initially — real implementations moved in T006
  - **Acceptance**: RunSupervised starts/stops goroutines cleanly; backoff resets correctly
  - **Traces to**: Plan Phase 5.1-5.2, 5.7, Spec: ProjectRunner

- [ ] T006 **Move all watchers + router from Orchestrator to ProjectRunner**
  - Files: `internal/orchestrator/watcher_linear.go`, `internal/orchestrator/watcher_pr.go`, `internal/orchestrator/watcher_comment.go`, `internal/orchestrator/comment_dispatcher.go`, `internal/orchestrator/router.go`
  - Change receiver on all methods: `*Orchestrator` → `*ProjectRunner`
  - Replace `o.cfg.ProjectID` → `p.id`, `o.cfg.*` → `p.cfg.*`, `o.store.*` calls gain `p.id`
  - LinearPoller: callback URL → `fmt.Sprintf("http://localhost:%d/project/%s/%s", p.cfg.CallbackPort, p.id, ticket.Identifier)`
  - LinearPoller: sandbox references → accessed through shared session manager (sandbox config baked into cmux options)
  - PRWatcher: `ListJobsByStatus(ctx, p.id, statuses...)`
  - CommentWatcher: each ProjectRunner gets own `CommentDispatcher`. `BotUsername` via `p.cfg.BotUsername`
  - Router: reads from `p.events` channel. All store calls gain `p.id`. `handleComplete` slot hold is intentional — document, don't change.
  - Return types: watchers return `func(ctx) error` (no longer `shutdown.ServiceFunc`)
  - `ClosedPRTicketStatus` → `ClosedPRStatus` (field rename)
  - **Acceptance**: All watcher/router methods compile on `*ProjectRunner`; store calls include projectID
  - **Traces to**: Plan Phase 5.3-5.6

- [ ] T007 **Admin service + all test mock updates**
  - Files: `internal/admin/service.go`, `internal/state/store_test.go`, `internal/orchestrator/*_test.go` (5+ files)
  - Admin service: add `projectID` to `GetJob`, `DeleteJob`, `ListJobsByStatus`, `SetJobStatus` calls. Accept `--project` filter in admin commands.
  - Store tests: all calls pass `projectID`. Add cross-project isolation test (same ticket_id, different project).
  - Mock StateStores in orchestrator tests: add `projectID` param to every changed interface method. Files: `orchestrator_test.go`, `watcher_pr_test.go`, `watcher_comment_test.go`, `watcher_linear_test.go`, `router_test.go`
  - **Acceptance**: All tests compile and pass; cross-project isolation verified
  - **Traces to**: Plan Phase 3.4, 5.8, 5.9

### Composition Root (depends on T005, T006, T007)

- [ ] T008 **Delete Orchestrator + rewrite composition root**
  - Files: `internal/orchestrator/orchestrator.go`, `cmd/orchestrator/main.go`, `cmd/orchestrator/run.go`
  - Delete `Orchestrator` struct and `Run()` method from `orchestrator.go`
  - Rewrite `main.go`: replace all single-project CLI flags with `--config` (required) + global overrides (`--log-level`, `--callback-port`, `--db-path`). Admin `jobs list` and `slots` show project_id column with optional `--project` filter.
  - Rewrite `run.go` composition root:
    1. Load TOML config + CLI overrides
    2. Init logger
    3. Open store + migrate
    4. Resolve sandbox mode (same `resolveSandboxMode` logic)
    5. Run reconciliation (global, once)
    6. Create Linear client(s) — shared when same API key, separate when project overrides
    7. Create callback server + EventDemuxer
    8. For each project: register demuxer, create GitHub client, worktree manager (`worktree.New(repoDir, worktree.WithWorktreesRoot(...), worktree.WithCopyFiles(...))`), ProjectRunner
    9. Create shared session manager (`cmux.New(cmuxOpts...)`)
    10. Run all via `shutdown.Manager`
  - **Acceptance**: `orchestrator run --config orchestrator.toml` starts N projects; process exits cleanly on SIGINT
  - **Traces to**: Plan Phase 5.10, 6.1, 6.2

- [ ] T009 **Reconciliation update + health endpoint**
  - Files: `internal/state/reconcile.go`, `internal/callback/server.go`
  - `ApplyReconciliation` gains `configuredProjects []string` param
  - Orphan detection: `ListAllJobs` → warn + release slots for jobs with unconfigured project_id
  - Per-project stale detection: iterate configured projects → `ListJobsByStatus(ctx, projectID, StatusInProgress)` → `PlanReconciliation` → mark stale
  - `ResetAllSlots` at the end (unchanged)
  - Health endpoint: `/healthz` returns JSON with global status ("healthy"/"degraded") + per-project watcher states. CallbackServer takes `HealthProvider` interface. Composition root wires to collect `Health()` from all ProjectRunners.
  - **Acceptance**: Orphaned jobs detected; `/healthz` returns per-project JSON
  - **Traces to**: Plan Phase 6.3, 6.4, Spec: Reconciliation, Health Endpoint

### Final (depends on T008, T009)

- [ ] T010 **Integration test + cleanup**
  - Files: `internal/orchestrator/integration_test.go` (new), `internal/orchestrator/config.go` (cleanup), `orchestrator.example.toml`
  - Delete `LegacyConfig` struct, `NewConfig(c *cli.Context)`, old `configRules`
  - Integration test: start with 2-project TOML config, verify both projects' watchers start, verify cross-project store isolation
  - Finalize `orchestrator.example.toml` with production-ready comments
  - Verify all acceptance criteria from business spec
  - **Acceptance**: Integration test passes; `go build ./cmd/orchestrator/` succeeds; no dead code
  - **Traces to**: Plan Phase 7, all business spec acceptance criteria

## Dependency Graph

```
T001 (Config) ────────┐
                       │
T002 (Migration) ──→ T004 (Store) ──┐
                       │             │
T003 (Callback) ───────┤             │
                       │             ▼
                       └──────→ T005 (Runner type)
                                  │
                                  ▼
                               T006 (Move watchers)
                                  │
                                  ▼
                               T007 (Admin + mocks)
                                  │
                                  ▼
                               T008 (Composition root)
                                  │
                                  ▼
                               T009 (Reconciliation + health)
                                  │
                                  ▼
                               T010 (Integration + cleanup)
```

**Parallel opportunities**: T001, T002, T003 can run simultaneously (3 agents).

## Traceability Matrix

| Business Spec Criterion | Task(s) |
|---|---|
| Starts with `--config orchestrator.toml` | T001, T008 |
| Per-project watcher goroutines | T005, T006 |
| Composite PK jobs | T002, T004 |
| Composite PK watermarks | T002, T004 |
| Callback URLs `/project/{projectID}/...` | T003 |
| EventDemuxer routing | T003 |
| Watcher failure → restart with backoff | T005 |
| Infrastructure failure → shutdown | T008 |
| Migration 003 | T002 |
| Config validation | T001 |
| `jobs`/`slots` show project ID | T007, T008 |
| `/healthz` per-project health | T009 |
| Reconciliation orphan detection | T009 |

## Summary

- **Total tasks**: 10
- **Parallel opportunities**: 3 tasks (T001, T002, T003)
- **Critical path length**: 8 tasks (T001 → T004 → T005 → T006 → T007 → T008 → T009 → T010)
- **Complexity**: Complex (22+ files, 4 packages)
