# Tasks: Callback Event Router

**Complexity**: Medium (12 files, 4 cross-module dependencies)
**Total tasks**: 17
**Parallel groups**: 3 independent groups in Phase 1

## Dependency Graph

```
Phase 1 (parallel):
  Group A: T001, T002, T003         (Config)
  Group B: T004, T005, T006         (LinearClient.PostComment)
  Group C: T007, T008               (Archival/Friction stubs)

Phase 2 (sequential, depends on Phase 1):
  T009, T010, T011                  (Orchestrator wiring)

Phase 3 (sequential, depends on Phase 2):
  T012, T013, T014, T015, T016     (Router core + tests)

Phase 4 (sequential, depends on Phase 3):
  T017                              (Wire router into Orchestrator.Run)
```

## Phase 1: Foundation (Parallelizable)

### Group A: Config Extensions

- [ ] T001 [P] Add `GitHubOwner`, `GitHubRepo`, `BaseBranch`, `TrackingLabel` fields to `Config` struct in `internal/orchestrator/config.go`. Add validation for required fields (`GitHubOwner`, `GitHubRepo` non-empty). `BaseBranch` and `TrackingLabel` need no validation (have defaults).
  - **Traces to**: IR-003
  - **Accept**: Config struct compiles with new fields, `Validate()` rejects empty `GitHubOwner`/`GitHubRepo`.

- [ ] T002 [P] Add CLI flags `--github-owner` (`ORCH_GITHUB_OWNER`), `--github-repo` (`ORCH_GITHUB_REPO`), `--base-branch` (`ORCH_BASE_BRANCH`, default `"main"`), `--tracking-label` (`ORCH_TRACKING_LABEL`, default `"ai-managed"`) in `cmd/orchestrator/main.go`. Wire them into `NewConfig()` in `internal/orchestrator/config.go`.
  - **Traces to**: IR-003
  - **Accept**: Flags appear in `--help` output with correct defaults and env vars.

- [ ] T003 [P] Add validation tests for new config fields in `internal/orchestrator/config_test.go`: missing `GitHubOwner`, missing `GitHubRepo`, and defaults for `BaseBranch`/`TrackingLabel`. Follow existing `TestValidate_Missing*` pattern.
  - **Traces to**: IR-003
  - **Accept**: `go test ./internal/orchestrator/ -run TestValidate` passes.

### Group B: LinearClient.PostComment

- [ ] T004 [P] Add `PostComment(ctx context.Context, issueID string, body string) error` to the `Client` interface in `internal/linear/client.go`.
  - **Traces to**: IR-001
  - **Accept**: Interface compiles. Existing implementations fail to compile (expected -- T005 fixes).

- [ ] T005 [P] Implement `PostComment` on `HTTPClient` in `internal/linear/http_client.go`. Define `commentCreateMutation` GraphQL const and `commentCreateResponse` struct. Use `doWithRetry` pattern. Check `resp.CommentCreate.Success`.
  - **Depends on**: T004
  - **Traces to**: IR-001, FR-004
  - **Accept**: `go build ./internal/linear/...` compiles.

- [ ] T006 [P] Add `PostCommentFn` field to `MockClient` in `internal/linear/mock.go`. Implement `PostComment` method with call recording following existing mock pattern.
  - **Depends on**: T004
  - **Traces to**: IR-001
  - **Accept**: `go build ./internal/linear/...` compiles. Mock satisfies `Client` interface.

### Group C: Stub Packages

- [ ] T007 [P] Create `internal/archival/archiver.go`: Define `Archiver` interface with `Archive(ctx context.Context, ticketID string, eventKind callback.EventKind) error`. Provide `NoopArchiver` struct that returns nil.
  - **Traces to**: IR-004, FR-006
  - **Accept**: `go build ./internal/archival/...` compiles.

- [ ] T008 [P] Create `internal/friction/auditor.go`: Define `Auditor` interface with `Audit(ctx context.Context, ticketID string, eventKind callback.EventKind) error`. Provide `NoopAuditor` struct that returns nil.
  - **Traces to**: IR-005, FR-007
  - **Accept**: `go build ./internal/friction/...` compiles.

## Phase 2: Orchestrator Wiring

- [ ] T009 Add `github github.Client` field to `Orchestrator` struct in `internal/orchestrator/orchestrator.go`. Update `New()` constructor signature to accept `github.Client` parameter.
  - **Depends on**: T001
  - **Traces to**: IR-002
  - **Accept**: Struct compiles. Existing callers will fail to compile (expected -- T010 fixes).

- [ ] T010 Update `run()` in `cmd/orchestrator/main.go` to create `github.NewClient(cfg.GitHubToken, cfg.GitHubOwner, cfg.GitHubRepo)` and pass it to `orchestrator.New()`.
  - **Depends on**: T002, T009
  - **Traces to**: IR-002
  - **Accept**: `go build ./cmd/orchestrator/...` compiles.

- [ ] T011 Update existing tests in `internal/orchestrator/orchestrator_test.go` to pass a mock or nil `github.Client` to `New()` so they compile again.
  - **Depends on**: T009
  - **Traces to**: IR-002
  - **Accept**: `go test ./internal/orchestrator/ -run TestOrchestrator` passes.

## Phase 3: Router Core

- [ ] T012 Create `internal/orchestrator/router.go` with `Router` struct, `NewRouter` constructor, and `Run` method (select loop on events channel and ctx.Done). Add `handleEvent` dispatcher that switches on `Event.Kind`. Include `markNeedsAttention` helper that updates both Linear and state (skipping state if job is nil).
  - **Depends on**: T007, T008, T009
  - **Traces to**: FR-001, FR-008, FR-009
  - **Accept**: `go build ./internal/orchestrator/...` compiles. Router struct has all fields from technical spec.

- [ ] T013 Implement `handleComplete` method on Router in `internal/orchestrator/router.go`: GetJob, ReadFile, GetTicket, CreatePR, SetPR, ApplyLabel, Archive, Audit. On failure at any step, call `markNeedsAttention` and return.
  - **Depends on**: T012
  - **Traces to**: FR-002, FR-003, FR-006, FR-007
  - **Accept**: `go build ./internal/orchestrator/...` compiles.

- [ ] T014 Implement `handleFailed` method on Router in `internal/orchestrator/router.go`: GetJob (nil allowed), defer ReleaseSlot, SetTicketStatus, conditional SetJobStatus, PostComment, Archive.
  - **Depends on**: T012
  - **Traces to**: FR-004, FR-006
  - **Accept**: `go build ./internal/orchestrator/...` compiles.

- [ ] T015 Implement `handleCommentResolved` method on Router in `internal/orchestrator/router.go`: GetJob, verify PRNumber, parse CommentID, ReadFile, UpdatePRBody, PostCommentReply, SetCommentWatermark, Archive, Audit.
  - **Depends on**: T012
  - **Traces to**: FR-005, FR-006, FR-007
  - **Accept**: `go build ./internal/orchestrator/...` compiles.

- [ ] T016 Write integration tests in `internal/orchestrator/router_test.go`. Tests: (1) CompletionEvent happy path, (2) CompletionEvent missing pr-description, (3) CompletionEvent unknown ticket, (4) FailureEvent happy path, (5) FailureEvent unknown ticket, (6) FailureEvent Linear API failure -> slot still released, (7) CommentResolvedEvent happy path, (8) CommentResolvedEvent nil PRNumber, (9) CommentResolvedEvent invalid CommentID, (10) Channel closed -> Run returns nil, (11) Context cancelled -> Run returns nil.
  - **Depends on**: T013, T014, T015
  - **Traces to**: All FRs
  - **Accept**: `go test ./internal/orchestrator/ -run TestRouter` passes.

## Phase 4: Final Wiring

- [ ] T017 Update `Orchestrator.Run()` in `internal/orchestrator/orchestrator.go` to create a `Router` with `callbackSrv.Events()`, noop archiver/auditor, and pass `router.Run` to the shutdown manager as a service.
  - **Depends on**: T012
  - **Traces to**: FR-009
  - **Accept**: `go build ./cmd/orchestrator/...` compiles. `go test ./internal/orchestrator/...` passes.

## Traceability Matrix

| Spec Requirement | Tasks |
|-----------------|-------|
| FR-001 (event dispatch) | T012 |
| FR-002 (CompletionEvent) | T013 |
| FR-003 (missing pr-description) | T013, T016 |
| FR-004 (FailureEvent) | T005, T014 |
| FR-005 (CommentResolvedEvent) | T015 |
| FR-006 (Archive all events) | T007, T013, T014, T015 |
| FR-007 (Audit complete+comment) | T008, T013, T015 |
| FR-008 (partial failure) | T012, T016 |
| FR-009 (ServiceFunc) | T012, T017 |
| IR-001 (PostComment) | T004, T005, T006 |
| IR-002 (github.Client) | T009, T010, T011 |
| IR-003 (Config fields) | T001, T002, T003 |
| IR-004 (Archiver) | T007 |
| IR-005 (Auditor) | T008 |
