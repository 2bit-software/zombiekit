# Tasks: Watcher 2 — PR Comment Queue

## Complexity: Medium (10 files, ~450 LOC)

## Phase 0: Prerequisites

- [ ] T001 [P] [FR-007] Add `GetJobByPR` to StateStore interface and SQLite implementation
  - **Files**: `internal/state/store.go`
  - **Do**: Add `GetJobByPR(ctx context.Context, prNumber int64) (*Job, error)` to `StateStore` interface. Implement in `SQLiteStore` — query `jobs` table by `pr_number`. Return `nil, nil` on no rows.
  - **Accept**: `go test ./internal/state/...` passes. Test creates a job, sets PR number, then retrieves by PR number.

- [ ] T002 [P] [FR-006] Add `BotUsername` to Config
  - **Files**: `internal/orchestrator/config.go`, CLI flag definition file (find `--tracking-label` flag and add `--bot-username` alongside it)
  - **Do**: Add `BotUsername string` field to `Config`. Add validation (required, non-empty). Add `--bot-username` / `ORCH_BOT_USERNAME` CLI flag.
  - **Accept**: Config validation fails without BotUsername. Passes with it.

- [ ] T003 [P] [FR-005,FR-012] Add `ReleaseSlot` call to `handleCommentResolved` in Router
  - **Files**: `internal/orchestrator/router.go`
  - **Do**: Add `r.store.ReleaseSlot(ctx, r.cfg.ProjectID)` after the watermark/archive block in `handleCommentResolved`. Log error but don't fail.
  - **Accept**: Existing router tests compile and pass. Slot count decrements after comment-resolved event.

## Phase 1: Core Types

- [ ] T004 [FR-003,FR-012] Create CommentDispatcher with SessionResult signaling
  - **Depends on**: None (pure new code)
  - **Files**: `internal/orchestrator/comment_dispatcher.go` (new)
  - **Do**: Implement `SessionResult`, `SessionResultKind`, `prQueue`, `CommentDispatcher` as specified in technical-spec.md. Methods: `NewCommentDispatcher`, `RegisterSession`, `NotifyResult` (debug-level log for unregistered sessions), `CreateQueue` (channel capacity 100), `GetQueue`, `RemoveQueue`, `ActivePRs`.
  - **Accept**: Compiles. Unit tests in T010 will validate behavior.

## Phase 2: Comment Watcher

- [ ] T005 [FR-001,FR-002,FR-006,FR-007,FR-010,FR-011] Implement comment watcher polling loop
  - **Depends on**: T001, T002, T004
  - **Files**: `internal/orchestrator/watcher_comment.go` (new)
  - **Do**: Implement `(o *Orchestrator) NewCommentWatcher(dispatcher *CommentDispatcher) shutdown.ServiceFunc`. Follow the `NewLinearPoller` pattern: ticker + select on ctx.Done/ticker.C. Implement `pollComments` (list PRs, reap stale queues) and `pollPRComments` (get job by PR, skip terminal states, get watermark, fetch review comments, filter bot comments, dispatch to per-PR queue — create lazily). Include `writeCommentJSON` and `acquireSlotBlocking` helpers.
  - **Accept**: Compiles. Integration tests in T011 will validate behavior.

- [ ] T006 [FR-003,FR-004,FR-005,FR-008,FR-009] Implement per-PR goroutine (`runPRQueue`)
  - **Depends on**: T004, T005
  - **Files**: `internal/orchestrator/watcher_comment.go` (same file as T005)
  - **Do**: Implement `runPRQueue` as specified in technical-spec.md. Key behaviors: (1) check IsMerged/IsClosed before each dispatch, (2) acquireSlotBlocking, (3) writeCommentJSON, (4) RegisterSession + SpawnSession, (5) block on SessionResult channel, (6) on failure: drain channel tracking max ID, advance watermark, exit, (7) on merge/close: drain channel, exit. Implement `drainChannel` returning max comment ID.
  - **Accept**: Compiles. Integration tests in T011 will validate behavior.

## Phase 3: Integration Wiring

- [ ] T007 [FR-012] Wire CommentDispatcher into Router
  - **Depends on**: T003, T004
  - **Files**: `internal/orchestrator/router.go`
  - **Do**: Add `dispatcher *CommentDispatcher` field to `Router`. Update `NewRouter` signature to accept `*CommentDispatcher`. Call `dispatcher.NotifyResult` at end of `handleCommentResolved` (with `SessionResolved`) and `handleFailed` (with `SessionFailed`). Guard with `if r.dispatcher != nil` for backward compat during tests.
  - **Accept**: Existing router tests compile (update `NewRouter` calls to pass `nil` for dispatcher). New notification calls are verified in T011.

- [ ] T008 Wire comment watcher into Orchestrator.Run()
  - **Depends on**: T005, T006, T007
  - **Files**: `internal/orchestrator/orchestrator.go`
  - **Do**: Create `CommentDispatcher` in `Run()`. Pass to `NewRouter` and `NewCommentWatcher`. Replace `NewWatcherStub(WatcherCommentWatcher, ...)` with `o.NewCommentWatcher(dispatcher)`.
  - **Accept**: Orchestrator compiles and starts cleanly. Comment watcher logs "comment watcher started" on startup.

## Phase 4: Tests

- [ ] T009 [P] Update existing router tests for new NewRouter signature
  - **Depends on**: T007
  - **Files**: `internal/orchestrator/router_test.go` (if exists, otherwise note as N/A)
  - **Do**: Update all `NewRouter(...)` calls to include the new `*CommentDispatcher` parameter (pass `nil` where dispatcher behavior is not under test).
  - **Accept**: All existing router tests pass.

- [ ] T010 [P] Unit tests for CommentDispatcher
  - **Depends on**: T004
  - **Files**: `internal/orchestrator/comment_dispatcher_test.go` (new)
  - **Do**: Tests: `TestRegisterAndNotify` (register, notify, verify channel receives), `TestNotifyWithoutRegistration` (no panic, debug log), `TestCreateAndRemoveQueue` (lifecycle), `TestActivePRs` (list reflects state).
  - **Accept**: `go test ./internal/orchestrator/... -run TestComment` passes.

- [ ] T011 [P] Integration tests for comment watcher
  - **Depends on**: T005, T006, T007, T008
  - **Files**: `internal/orchestrator/watcher_comment_test.go` (new)
  - **Do**: Tests using mock interfaces: `TestPollDetectsNewComments`, `TestSerialProcessing`, `TestBotCommentFiltered`, `TestTerminalJobSkipped`, `TestMergeDetection`, `TestFailureDrainsQueue`, `TestGracefulShutdown`, `TestPRReaping`, `TestSlotBlocking`, `TestIndependentPRQueues`, `TestFollowUpCommentAfterResolution`.
  - **Accept**: `go test ./internal/orchestrator/... -run TestWatcherComment` passes.

- [ ] T012 Add GetJobByPR test to state store tests
  - **Depends on**: T001
  - **Files**: `internal/state/store_test.go`
  - **Do**: Test: create job, set PR number via `SetPR`, retrieve via `GetJobByPR`, verify fields match. Test nil return when no job matches PR number.
  - **Accept**: `go test ./internal/state/... -run TestGetJobByPR` passes.

## Dependency Graph

```
T001 ──┐
T002 ──┼── T005 ── T006 ──┐
T004 ──┘                   ├── T008
T003 ── T007 ──────────────┘
                           ├── T009
                           ├── T011
T004 ── T010
T001 ── T012
```

## Parallel Opportunities

| Group | Tasks | Can run in parallel |
|-------|-------|-------------------|
| Phase 0 | T001, T002, T003 | All three |
| Phase 1 | T004 | Solo (but parallel with Phase 0) |
| Phase 2 | T005, T006 | Sequential (T006 depends on T005) |
| Phase 3 | T007, T008 | Sequential |
| Phase 4 | T009, T010, T011, T012 | T010 and T012 parallel; T009 and T011 after Phase 3 |

## Critical Path

T004 → T005 → T006 → T008 → T011

## FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T005 |
| FR-002 | T005 |
| FR-003 | T004, T006 |
| FR-004 | T006 |
| FR-005 | T003, T006 |
| FR-006 | T002, T005 |
| FR-007 | T001, T005 |
| FR-008 | T006 |
| FR-009 | T006 |
| FR-010 | T005 |
| FR-011 | T005 |
| FR-012 | T003, T004, T007 |
