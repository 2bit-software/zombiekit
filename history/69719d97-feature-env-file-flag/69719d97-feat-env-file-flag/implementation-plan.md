# Implementation Plan: --env-file Flag

## Overview

Add `--env-file` flag to the `brains serve` command to load environment variables from a dotenv file before any other configuration processing.

## Tasks

### T001: Add godotenv dependency
**File:** `go.mod`
**Action:** Run `go get github.com/joho/godotenv`

### T002: Add --env-file flag to serve command
**File:** `internal/cli/serve.go`
**Action:** Add StringFlag for `--env-file` with `BRAINS_ENV_FILE` env var support

### T003: Implement loadEnvFile function
**File:** `internal/cli/serve.go`
**Action:** Create function that:
- Validates file exists and is not a directory
- Uses `godotenv.Load()` to set env vars (does not override existing)
- Returns descriptive errors for common failure cases

### T004: Call loadEnvFile at start of runServe
**File:** `internal/cli/serve.go`
**Action:** Add env file loading as the first operation in `runServe()`, before logging initialization

### T005: Write integration tests
**File:** `internal/cli/serve_test.go`
**Action:** Test cases:
- Env file loads variables correctly
- Existing env vars are not overwritten
- Missing file returns clear error
- Directory path returns clear error

### T006: Verify build and lint
**Action:** Run `go build ./...` and `golangci-lint run`

### T007: Manual smoke test
**Action:** Test with actual `.env` file and verify PostgreSQL connection works

## Dependency Order

```
T001 (dependency)
  → T002 (flag)
    → T003 (function)
      → T004 (integration)
        → T005 (tests)
          → T006 (verify)
            → T007 (smoke test)
```

## Estimated Scope

- **Files changed:** 2 (`go.mod`, `internal/cli/serve.go`)
- **Files created:** 0 (tests go in existing `serve_test.go` or new file)
- **Lines added:** ~60
- **Complexity:** Low - straightforward flag addition with well-tested library

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| godotenv behavior differs from expectation | `Load()` is documented to not override; verify with test |
| Path resolution issues | Use os.Stat before godotenv.Load for clear errors |
| Affects other commands | Only serve command modified; other commands unchanged |
