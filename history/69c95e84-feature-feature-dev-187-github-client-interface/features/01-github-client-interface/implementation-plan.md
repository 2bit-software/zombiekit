# Implementation Plan: GitHubClient Interface and Test Stub

## Overview

Three files, zero external dependencies, one existing pattern to replicate. This is a mechanical translation of the business spec into Go code following the `internal/linear/` package conventions exactly.

## Phase 1: Foundation (types + errors)

### Task 1: Create `internal/github/client.go`

**What**: Interface definition + all domain types.

**Files**: `internal/github/client.go`

**Contents**:
- `CommentKind` string enum (`CommentKindIssue`, `CommentKindReview`)
- `CreatePRInput` struct
- `PRComment` struct
- `PRSummary` struct
- `Client` interface with 8 methods

**Dependency**: None. This is the root artifact.

**Verify**: `go vet ./internal/github/`

### Task 2: Create `internal/github/errors.go`

**What**: Error type, constructors, and predicates. Direct copy of `internal/linear/errors.go` structure with `github` package name.

**Files**: `internal/github/errors.go`

**Contents**:
- `ErrorKind` int enum: `ErrNotFound`, `ErrRateLimited`, `ErrAPI`, `ErrNetwork`
- `Error` struct with `Kind`, `Message`, `Err`
- `Error()` and `Unwrap()` methods
- Four `New*Error(msg, cause)` constructors
- Four `Is*(err)` predicate functions

**Dependency**: None. Independent of client.go.

**Verify**: `go vet ./internal/github/`

## Phase 2: Mock

### Task 3: Create `internal/github/mock.go`

**What**: `MockClient` struct implementing `Client` interface with per-method function fields and call recording.

**Files**: `internal/github/mock.go`

**Contents**:
- Compile-time assertion: `var _ Client = (*MockClient)(nil)`
- `Call` struct (reuse same shape as `linear.Call`)
- `MockClient` struct with 8 `*Fn` fields + `Calls []Call`
- 8 method implementations that record calls then delegate to `*Fn` or return error

**Dependency**: Task 1 (needs `Client` interface and domain types).

**Verify**: `go vet ./internal/github/`

## Phase 3: Tests

### Task 4: Create `internal/github/mock_test.go`

**What**: Tests validating mock behavior, error predicates, and consumer wiring. Mirror `internal/linear/mock_test.go` structure.

**Test cases**:
1. Interface compliance (compile check)
2. Configured response for `CreatePR` returns expected PR number
3. Configured response for `GetCommentsSince` returns ordered comments
4. Unconfigured method returns descriptive error
5. Call recording for all 8 methods (method name + args, context excluded)
6. Error predicates: each kind returns true only for its predicate
7. Error predicates: nil and foreign errors return false
8. Error unwrap chain
9. Configured error propagation through mock
10. Consumer wiring: function accepting `Client` interface works with mock

**Dependency**: Tasks 1-3.

**Verify**: `go test ./internal/github/`

## Execution Order

```
Task 1 (client.go) ──┐
                      ├── Task 3 (mock.go) ── Task 4 (mock_test.go)
Task 2 (errors.go) ──┘
```

Tasks 1 and 2 are independent and can be implemented in parallel. Task 3 depends on both. Task 4 depends on Task 3.

## Verification Checklist

- [ ] `go vet ./internal/github/` passes after each task
- [ ] `go test ./internal/github/` passes after Task 4
- [ ] All 5 acceptance criteria from the spec are covered by tests
- [ ] No imports of external libraries beyond stdlib + testify
