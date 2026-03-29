# Tasks: GitHubClient Interface and Test Stub

## Complexity: Simple

- Files: 4
- Lines: ~250
- Cross-module deps: 0
- Reference: `internal/linear/` (exact pattern to follow)

## Dependency Graph

```
T001 (client.go) ──┐
                    ├── T003 (mock.go) ── T004 (mock_test.go)
T002 (errors.go) ──┘
```

## Tasks

### Wave 1 (parallel)

- [ ] T001 [P] Create `internal/github/client.go` — Interface and domain types
  - Define `CommentKind` string enum (`CommentKindIssue`, `CommentKindReview`)
  - Define `CreatePRInput` struct (Title, Body, Head, Base)
  - Define `PRComment` struct (ID, Author, Body, CreatedAt, Path, DiffHunk, InReplyToID)
  - Define `PRSummary` struct (Number, Title, Head, Base, Labels)
  - Define `Client` interface with 8 methods: CreatePR, UpdatePRBody, GetCommentsSince, PostCommentReply, ApplyLabel, IsMerged, IsClosed, ListOpenPRs
  - All methods take `context.Context` as first parameter
  - Doc comments on all exported types and methods
  - **Reference**: `internal/linear/client.go`
  - **Verify**: `go vet ./internal/github/`
  - **AC**: Interface compiles, types match business-spec.md exactly

- [ ] T002 [P] Create `internal/github/errors.go` — Error types and predicates
  - Define `ErrorKind` int enum: `ErrNotFound`, `ErrRateLimited`, `ErrAPI`, `ErrNetwork` (iota + 1)
  - Define `Error` struct with `Kind ErrorKind`, `Message string`, `Err error`
  - Implement `Error() string` and `Unwrap() error`
  - Four constructors: `NewNotFoundError`, `NewRateLimitedError`, `NewAPIError`, `NewNetworkError`
  - Four predicates: `IsNotFound`, `IsRateLimited`, `IsAPIError`, `IsNetworkError` (using `errors.As`)
  - **Reference**: `internal/linear/errors.go` (exact copy with `github` package name)
  - **Verify**: `go vet ./internal/github/`
  - **AC**: Error types match linear pattern; predicates use `errors.As`

### Wave 2

- [ ] T003 Create `internal/github/mock.go` — Mock implementation with call recording
  - Compile-time assertion: `var _ Client = (*MockClient)(nil)`
  - Define `Call` struct: `{Method string, Args []any}`
  - Define `MockClient` struct with 8 `*Fn` function fields + `Calls []Call`
  - 8 method implementations: record call (context excluded), delegate to Fn or return error
  - Unconfigured error format: `fmt.Errorf("MockClient.MethodName not configured")`
  - **Depends on**: T001
  - **Reference**: `internal/linear/mock.go` (exact pattern)
  - **Verify**: `go vet ./internal/github/`
  - **AC**: MockClient satisfies Client interface; all calls recorded

### Wave 3

- [ ] T004 Create `internal/github/mock_test.go` — Tests for mock, errors, and consumer wiring
  - `TestMockClient_InterfaceCompliance` — compile check [AC 1]
  - `TestMockClient_ConfiguredResponse_CreatePR` — returns configured PR number [AC 2]
  - `TestMockClient_ConfiguredResponse_GetCommentsSince` — returns 2 comments in order [AC 3]
  - `TestMockClient_UnconfiguredMethod` — returns descriptive error
  - `TestMockClient_CallRecording_AllMethods` — 8 entries with correct method/args [AC 5]
  - `TestMockClient_ErrorPredicates` — each kind matches its predicate only
  - `TestMockClient_ErrorPredicates_NilAndForeign` — all return false for nil/foreign
  - `TestMockClient_ErrorUnwrap` — cause accessible via Unwrap/errors.Is
  - `TestMockClient_ConfiguredError` — error propagates through mock [AC 4]
  - `TestMockClient_ConsumerWiring` — function accepting Client works with mock [AC 1]
  - `TestMockClient_CallAccumulation` — repeated calls accumulate
  - **Depends on**: T001, T002, T003
  - **Reference**: `internal/linear/mock_test.go` (mirror structure)
  - **Verify**: `go test ./internal/github/`
  - **AC**: All tests pass; all 5 business spec acceptance criteria covered

## Traceability

| Spec Requirement | Task |
|-----------------|------|
| Client interface (8 methods) | T001 |
| Domain types (CreatePRInput, PRComment, PRSummary, CommentKind) | T001 |
| Error types (4 kinds + predicates) | T002 |
| Mock (configurable responses) | T003 |
| Mock (call recording) | T003 |
| Mock (compile-time verification) | T003 |
| AC 1: Consumer wiring | T004 |
| AC 2: CreatePR configured response | T004 |
| AC 3: GetCommentsSince configured response | T004 |
| AC 4: Error propagation | T004 |
| AC 5: Call recording verification | T004 |
