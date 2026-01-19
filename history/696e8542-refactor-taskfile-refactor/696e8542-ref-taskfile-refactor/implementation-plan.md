# Implementation Plan: Taskfile Two-File Refactor

## Overview

Refactor ZombieKit's single `Taskfile.yml` into a two-file architecture:
- `Taskfile.yml` - User-facing tasks (9 tasks)
- `Taskfile.dev.yml` - Development tasks (12 tasks)

## Prerequisites

- [ ] Current `Taskfile.yml` backed up (git handles this)
- [ ] No pending changes to Taskfile.yml

## Implementation Steps

### Phase 1: Create Taskfile.dev.yml

**Step 1.1**: Create new `Taskfile.dev.yml` with development tasks

Extract these tasks from current Taskfile.yml:
- `fmt`, `vet`, `lint` (code quality)
- `db:migrate`, `db:migrate:memory`, `db:migrate:recall` (migrations)
- `ollama:pull`, `recall:demo` (ML/demo)
- `webgui:dev` (frontend dev)

Add new tasks:
- `default` - List dev tasks
- `test` - Test implementation (moved from main)
- `ci` - CI implementation (cross-file call to main `build`)

**Verification**: `task --taskfile Taskfile.dev.yml --list` shows 12 tasks

### Phase 2: Refactor Taskfile.yml

**Step 2.1**: Add `dev` entry point task

```yaml
dev:
  desc: Run development tasks. Use like `task dev -- <args>`
  cmds:
    - task --taskfile Taskfile.dev.yml {{.CLI_ARGS}}
```

**Step 2.2**: Convert `default` to silent with list

```yaml
default:
  desc: List available tasks
  silent: true
  cmds:
    - task --list
```

**Step 2.3**: Rename `db:up` → `up` and `db:down` → `down`

Keep implementation, change task names only.

**Step 2.4**: Convert `test` to delegation

```yaml
test:
  desc: Run tests with coverage
  cmds:
    - task --taskfile Taskfile.dev.yml test
```

**Step 2.5**: Convert `ci` to delegation

```yaml
ci:
  desc: Run all CI checks (fmt, vet, lint, test, build)
  cmds:
    - task --taskfile Taskfile.dev.yml ci
```

**Step 2.6**: Convert `init` to use `status:` pattern

```yaml
init:
  desc: Download dependencies and install development tools
  cmds:
    - go mod download
    - go mod tidy
    - task: init:golangci-lint

init:golangci-lint:
  internal: true
  desc: Install golangci-lint if not present
  status:
    - command -v golangci-lint
  cmds:
    - echo "Installing golangci-lint..."
    - go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Note: `internal: true` hides the subtask from `task --list` output.

**Step 2.7**: Remove migrated tasks

Remove from main file:
- `fmt`, `vet`, `lint`
- `db:migrate`, `db:migrate:memory`, `db:migrate:recall`
- `ollama:pull`, `recall:demo`
- `webgui:dev`

**Verification**: `task --list` shows exactly 9 tasks

### Phase 3: Verification

**Step 3.1**: Run acceptance criteria tests

| Test | Command | Expected |
|------|---------|----------|
| AC-1 | `task --list` | 9 visible tasks (init:golangci-lint hidden via internal: true) |
| AC-2 | `task dev` | Shows dev task list |
| AC-3 | `task dev -- fmt` | Runs go fmt |
| AC-4 | `task up` | Starts PostgreSQL |
| AC-5 | `task down` | Stops PostgreSQL |
| AC-6 | `task test` | Runs tests |
| AC-7 | `task ci` | Runs full CI pipeline |
| AC-8 | `task init` (with golangci-lint installed) | Shows "up to date" |
| AC-9 | `task dev -- db:migrate` | Runs migrations |
| AC-10 | `task dev -- recall:demo` | Runs demo |
| AC-11 | `task dev -- webgui:dev` | Starts WebGUI |
| AC-12 | `task db:up` | Error: task not found |

**Step 3.2**: Verify build still works

```bash
task build && ./bin/brains version
```

## Rollback Plan

If issues discovered:
1. `git checkout Taskfile.yml`
2. `rm Taskfile.dev.yml`

## File Changes Summary

| File | Action |
|------|--------|
| `Taskfile.yml` | Modify (remove 8 tasks, add 2, rename 2, update 3) |
| `Taskfile.dev.yml` | Create (12 tasks) |

## Dependencies

```
Phase 1 (Create dev file)
    ↓
Phase 2 (Refactor main file)
    ↓
Phase 3 (Verify)
```

Phases must be sequential - main file references dev file.

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| CI pipeline breaks | Test `task ci` before committing |
| Build variables missing in dev file | Dev file's `ci` calls main file's `build` |
| Users confused by breaking change | Update README if exists |
