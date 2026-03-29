## Progress Log

### Phase 1A - Config Extensions (T001-T003)
- Status: Complete
- Files: `internal/orchestrator/config.go`, `cmd/orchestrator/main.go`, `internal/orchestrator/config_test.go`
- Notes: Added GitHubOwner, GitHubRepo (required), BaseBranch (default "main"), TrackingLabel (default "ai-managed")

### Phase 1B - LinearClient.PostComment (T004-T006)
- Status: Complete
- Files: `internal/linear/client.go`, `internal/linear/http_client.go`, `internal/linear/mock.go`
- Notes: Added PostComment via commentCreate GraphQL mutation. Updated mock and existing test stubs.

### Phase 1C - Archival/Friction Stubs (T007-T008)
- Status: Complete
- Files: `internal/archival/archiver.go`, `internal/friction/auditor.go`
- Notes: New packages with interfaces + NoopArchiver/NoopAuditor.

### Phase 2 - Orchestrator Wiring (T009-T011)
- Status: Complete
- Files: `internal/orchestrator/orchestrator.go`, `cmd/orchestrator/main.go`, `internal/orchestrator/orchestrator_test.go`, `internal/orchestrator/watcher_linear_test.go`
- Notes: Added github.Client to Orchestrator struct and constructor. Updated all callers and tests.

### Phase 3 - Router Core + Tests (T012-T016)
- Status: Complete
- Files: `internal/orchestrator/router.go`, `internal/orchestrator/router_test.go`
- Notes: Router with Run (ServiceFunc), handleComplete, handleFailed, handleCommentResolved, markNeedsAttention. 11 integration tests all passing.

### Phase 4 - Wire Router (T017)
- Status: Complete
- Files: `internal/orchestrator/orchestrator.go`
- Notes: Router wired into Orchestrator.Run() with NoopArchiver and NoopAuditor. Passed to shutdown manager alongside other services.
