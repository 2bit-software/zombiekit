# Initiative: env-file-flag

**Type**: feature
**Status**: complete
**Created**: 2026-01-21T19:46:31-08:00
**ID**: 69719d97-feature-env-file-flag

## Description

Add `--env-file` flag to `brains serve` command to load environment variables from a dotenv file before any other configuration processing. This enables MCP clients like Claude Code to pass a .env file path, solving the problem where stdio-mode processes don't inherit shell environment variables or direnv configurations.

## Goals

- Allow `brains serve --env-file .env` to load PostgreSQL credentials and other settings
- Env file values should not override existing environment variables
- CLI flags should still take highest precedence
- Provide clear error messages for missing or invalid files

## Progress

All tasks completed successfully.

## Completion

**Completed**: 2026-01-21T19:56:00-08:00
**Duration**: ~10 minutes

### Outcomes

| Work Item | Status |
|-----------|--------|
| T001: Add godotenv dependency | Complete |
| T002: Add --env-file flag | Complete |
| T003: Implement loadEnvFile() | Complete |
| T004: Call loadEnvFile() in runServe() | Complete |
| T005: Write integration tests | Complete |
| T006: Verify build and lint | Complete |
| T007: Manual smoke test | Complete |

### Files Changed

- `go.mod` - Added github.com/joho/godotenv v1.5.1
- `internal/cli/serve.go` - Added flag and loadEnvFile() function
- `internal/cli/serve_test.go` - New file with 4 test cases

### Notes

- All tests pass
- No new lint issues introduced (pre-existing issues in other files)
- MCP configuration in ~/.claude.json updated to use the new flag
