# Progress Log

## Phase 0: Prerequisites

### T001 - Add GetJobByPR to StateStore
- Status: Complete
- Files: `internal/state/store.go`

### T002 - Add BotUsername to Config
- Status: Complete
- Files: `internal/orchestrator/config.go`, `cmd/orchestrator/main.go`

### T003 - Add ReleaseSlot to handleCommentResolved
- Status: Complete
- Files: `internal/orchestrator/router.go`

## Phase 1: Core Types

### T004 - Create CommentDispatcher
- Status: Complete
- Files: `internal/orchestrator/comment_dispatcher.go` (new)

## Phase 2: Comment Watcher

### T005 - Implement polling loop
- Status: Complete
- Files: `internal/orchestrator/watcher_comment.go` (new)

### T006 - Implement per-PR goroutine
- Status: Complete
- Files: `internal/orchestrator/watcher_comment.go` (same file)

## Phase 3: Integration Wiring

### T007 - Wire CommentDispatcher into Router
- Status: Complete
- Files: `internal/orchestrator/router.go`
- Notes: Added dispatcher field, updated NewRouter signature, added NotifyResult calls

### T008 - Wire into Orchestrator.Run()
- Status: Complete
- Files: `internal/orchestrator/orchestrator.go`
- Notes: Replaced comment watcher stub with real implementation

## Phase 4: Tests

### T009 - Update existing tests for new signatures
- Status: Complete
- Files: `internal/orchestrator/router_test.go`, `internal/orchestrator/orchestrator_test.go`, `internal/orchestrator/watcher_linear_test.go`, `internal/orchestrator/config_test.go`
- Notes: Added GetJobByPR to all mock StateStore impls, added nil dispatcher to NewRouter calls, added BotUsername to test configs, added MockClient with ListOpenPRsFn to orchestrator test

### T010 - Unit tests for CommentDispatcher
- Status: In Progress (agent)

### T011 - Integration tests for comment watcher
- Status: In Progress (agent)

### T012 - Add GetJobByPR test to state store
- Status: In Progress (agent)
