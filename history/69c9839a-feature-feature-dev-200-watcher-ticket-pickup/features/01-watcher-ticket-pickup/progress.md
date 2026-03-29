# Progress Log

## T001 - Add ProjectID and RepoDir to Config
- Status: Complete
- Files: `internal/orchestrator/config.go`, `internal/orchestrator/config_test.go`
- Notes: Added fields, validation (including .git dir check), and 3 new test cases

## T002 - Add CLI flags
- Status: Complete
- Files: `cmd/orchestrator/main.go`
- Notes: Added `--project-id`/`ORCH_PROJECT_ID` and `--repo-dir`/`ORCH_REPO_DIR`

## T003 - Extend Orchestrator struct
- Status: Complete
- Files: `internal/orchestrator/orchestrator.go`
- Notes: Added `linear`, `worktrees`, `sessions` fields + updated constructor

## T004 - Fix existing tests
- Status: Complete
- Files: `internal/orchestrator/orchestrator_test.go`
- Notes: Updated New() calls, switched from nil to stubs for real poller compatibility

## T005 - Implement NewLinearPoller
- Status: Complete
- Files: `internal/orchestrator/watcher_linear.go` (NEW)
- Notes: Poll loop, processTicket pipeline, rollback, shortTitle helper, markNeedsAttention

## T006 - Create test doubles
- Status: Complete
- Files: `internal/orchestrator/watcher_linear_test.go` (NEW, top section)
- Notes: stubLinear, stubWorktree, stubSession, stubState, capturingSessionManager

## T007 - Happy path and concurrency tests
- Status: Complete
- Files: `internal/orchestrator/watcher_linear_test.go`
- Notes: 6 tests (SingleTicket, TicketFileWritten, CallbackURL, ConcurrencyLimit, ConcurrencyMultiPoll, SkipExistingJob)

## T008 - Rollback and failure tests
- Status: Complete
- Files: `internal/orchestrator/watcher_linear_test.go`
- Notes: 6 tests (RollbackOnSpawnFailure, RollbackOnCreateJobFailure, RollbackOnWorktreeFailure, NeedsAttentionOnFailure, LinearFailureAfterJob, RemoveLabelFailureAfterJob)

## T009 - Edge case and shutdown tests
- Status: Complete
- Files: `internal/orchestrator/watcher_linear_test.go`
- Notes: 5 tests (GracefulShutdown, ShutdownBetweenPolls, EmptyPoll, PollError, EmptyDescription) + shortTitle tests

## T010 - Wire real clients in main.go
- Status: Complete
- Files: `cmd/orchestrator/main.go`, `internal/orchestrator/orchestrator.go`
- Notes: Created linear.NewClient, worktree.New, cmux.New. Replaced linear poller stub with real watcher.
