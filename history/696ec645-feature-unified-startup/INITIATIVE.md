# Initiative: unified-startup

**Type**: feature
**Status**: complete
**Created**: 2026-01-19T16:03:17-08:00
**ID**: 696ec645-feature-unified-startup

## Description

Implement `brains start` command that orchestrates GUI and recall services with unified lifecycle management, replacing the shell-based background process management in `task up`.

## Goals

- Single command to start all configured services
- Graceful shutdown with signal handling (SIGINT/SIGTERM)
- YAML-based configuration with local/global discovery
- Service-prefixed logging for clear output
- Fail-fast behavior when services error

## Completion

**Completed**: 2026-01-19T16:40:00-08:00
**Duration**: ~37 minutes

### Outcomes

| Phase | Status | Description |
|-------|--------|-------------|
| Shutdown Manager | Complete | Signal handling, timeout, force exit |
| Configuration System | Complete | YAML config with env overrides |
| Service Runners | Complete | GUI and Recall service wrappers |
| Start Command CLI | Complete | Integrated command with shutdown manager |
| Integration Tests | Complete | Basic test coverage |
| Taskfile Update | Complete | `task up` now uses `brains start` |

### Files Added (11)
- `internal/shutdown/manager.go`
- `internal/shutdown/manager_test.go`
- `internal/config/startup.go`
- `internal/config/startup_test.go`
- `internal/startup/service.go`
- `internal/startup/gui_service.go`
- `internal/startup/recall_service.go`
- `internal/startup/service_test.go`
- `internal/cli/start.go`
- `internal/cli/start_test.go`
- `history/696ec645-feature-unified-startup/696ec645-feat-unified-startup/progress.md`

### Files Modified (2)
- `internal/cli/root.go` - Added start command registration
- `Taskfile.yml` - Simplified `up` task to use `brains start`

### Tests Added
- 16 new tests across 4 packages, all passing

### Notes

The implementation follows the validated spike pattern for signal handling with a two-phase shutdown approach. The Taskfile `up` task was simplified from ~15 lines of shell script to a single `brains start` command.
