# Tasks: --env-file Flag

## Overview

| Metric | Value |
|--------|-------|
| Total tasks | 7 |
| Parallelizable | 0 |
| Complexity | Simple |
| Critical path | T001 → T002 → T003 → T004 → T005 → T006 → T007 |

## Task List

- [ ] T001 [US1] Add godotenv dependency (`go.mod`)
- [ ] T002 [US1] Add `--env-file` flag to serve command (`internal/cli/serve.go`)
- [ ] T003 [US2] Implement `loadEnvFile()` function (`internal/cli/serve.go`)
- [ ] T004 [US1] Call `loadEnvFile()` at start of `runServe()` (`internal/cli/serve.go`)
- [ ] T005 [US1,US2,US3] Write integration tests (`internal/cli/serve_test.go`)
- [ ] T006 Verify build and lint pass
- [ ] T007 Manual smoke test with actual `.env` file

## Task Details

### T001: Add godotenv dependency
**File:** `go.mod`
**Action:** `go get github.com/joho/godotenv`
**Acceptance:** `go mod tidy` succeeds, import available

### T002: Add --env-file flag to serve command
**File:** `internal/cli/serve.go`
**Action:** Add `StringFlag` in `newServeCommand()`:
```go
&cli.StringFlag{
    Name:    "env-file",
    Usage:   "Path to environment file to load",
    EnvVars: []string{"BRAINS_ENV_FILE"},
},
```
**Acceptance:** Flag shows in `brains serve --help`

### T003: Implement loadEnvFile() function
**File:** `internal/cli/serve.go`
**Action:** Create function that:
- Validates file exists (`os.Stat`)
- Validates not a directory (`info.IsDir()`)
- Loads with `godotenv.Load(path)`
- Returns descriptive errors

**Acceptance:** Function compiles, handles edge cases

### T004: Call loadEnvFile() at start of runServe()
**File:** `internal/cli/serve.go`
**Action:** Add at very beginning of `runServe()`, before logging init:
```go
if envFile := c.String("env-file"); envFile != "" {
    if err := loadEnvFile(envFile); err != nil {
        return err
    }
}
```
**Acceptance:** Env file loaded before `config.LoadStorageConfig()`

### T005: Write integration tests
**File:** `internal/cli/serve_test.go` (or new file)
**Test cases:**
1. `TestLoadEnvFile_Success` - file loads, vars set
2. `TestLoadEnvFile_NotOverride` - existing env vars preserved
3. `TestLoadEnvFile_MissingFile` - clear error message
4. `TestLoadEnvFile_IsDirectory` - clear error message

**Acceptance:** All tests pass, cover US1, US2, US3 scenarios

### T006: Verify build and lint
**Action:**
- `go build ./...`
- `golangci-lint run`

**Acceptance:** No errors, no warnings

### T007: Manual smoke test
**Action:**
1. Create `.env` with `BRAINS_POSTGRES_URL`
2. Run `brains serve --env-file .env --mode stdio`
3. Verify PostgreSQL connection works

**Acceptance:** Server starts, connects to Postgres

## Traceability Matrix

| Task | Spec FR | User Story |
|------|---------|------------|
| T001 | - | - |
| T002 | FR-001 | US1 |
| T003 | FR-004, FR-006, FR-007 | US2 |
| T004 | FR-002, FR-003 | US1 |
| T005 | FR-001-007 | US1, US2, US3 |
| T006 | - | - |
| T007 | SC-001 | US1 |

## Next Step

After completing all tasks, run `/brains.complete` to mark the initiative done.
