# Business Specification: Taskfile Two-File Refactor

## Overview

Refactor the ZombieKit Taskfile from a single file into a two-file architecture following established Taskfile patterns. This improves developer ergonomics by separating stable user-facing commands from internal development tools.

## Goals

1. **Improve discoverability** - Users see only relevant tasks
2. **Reduce cognitive load** - Daily tasks vs developer internals separated
3. **Enable future patterns** - CI-aware execution, Docker-based commands
4. **Follow conventions** - Match patterns used in production codebases

## User Stories

### US-1: Run common tasks easily
**As a** user of the brains CLI,
**I want** short, memorable task names (`up`, `down`, `test`, `build`),
**So that** I can execute common workflows quickly without looking up commands.

### US-2: Access development tools when needed
**As a** contributor to ZombieKit,
**I want** to access internal tools via `task dev -- <args>`,
**So that** I can run migrations, linting, and demos without cluttering the main task list.

### US-3: See only relevant tasks
**As a** user running `task --list`,
**I want** to see only the tasks I need daily,
**So that** I'm not overwhelmed by internal implementation details.

## Functional Requirements

### FR-1: Two-File Structure
- **FR-1.1**: Create `Taskfile.yml` with user-facing tasks
- **FR-1.2**: Create `Taskfile.dev.yml` with development tasks
- **FR-1.3**: Main file delegates to dev file via `--taskfile` flag

### FR-2: Task Distribution

**Taskfile.yml** (user-facing):
| Task | Description | Behavior |
|------|-------------|----------|
| `default` | List available tasks | `task --list` with `silent: true` |
| `dev` | Run development tasks | `task --taskfile Taskfile.dev.yml {{.CLI_ARGS}}` |
| `init` | Install dependencies | Use `status:` for idempotency |
| `build` | Build CLI binary | No change from current |
| `install` | Install to GOBIN | Depends on `build` |
| `test` | Run tests | Delegate: `task --taskfile Taskfile.dev.yml test` |
| `up` | Start services | Rename from `db:up` |
| `down` | Stop services | Rename from `db:down` |
| `ci` | Run CI checks | Delegate: `task --taskfile Taskfile.dev.yml ci` |

**Delegation Syntax:**
```yaml
# Main file delegates to dev file
test:
  desc: Run tests with coverage
  cmds:
    - task --taskfile Taskfile.dev.yml test

# Dev entry point passes args
dev:
  desc: Run development tasks. Use like `task dev -- <args>`
  cmds:
    - task --taskfile Taskfile.dev.yml {{.CLI_ARGS}}
```

**Taskfile.dev.yml** (development):
| Task | Description |
|------|-------------|
| `default` | List dev tasks |
| `test` | Run tests with coverage |
| `ci` | Run fmt, vet, lint, test, build sequence |
| `fmt` | Format Go code |
| `vet` | Run go vet |
| `lint` | Run golangci-lint |
| `db:migrate` | Run all migrations |
| `db:migrate:memory` | Memory table migration |
| `db:migrate:recall` | Recall chunks migration |
| `ollama:pull` | Pull embedding model |
| `recall:demo` | Demo recall feature |
| `webgui:dev` | WebGUI development mode |

**Dev File CI Task:**
```yaml
# Dev file ci task calls back to main file for build
ci:
  desc: Run all CI checks (fmt, vet, lint, test, build)
  cmds:
    - task: fmt
    - task: vet
    - task: lint
    - task: test
    - task --taskfile Taskfile.yml build  # Cross-file call for build (needs LDFLAGS)
```

### FR-3: Idempotent Init
- **FR-3.1**: Convert inline `command -v` check to `status:` pattern
- **FR-3.2**: Skip installation if golangci-lint already exists

**Example:**
```yaml
init:
  desc: Download dependencies and install development tools
  cmds:
    - go mod download
    - go mod tidy
    - task: init:golangci-lint

init:golangci-lint:
  desc: Install golangci-lint if not present
  status:
    - command -v golangci-lint
  cmds:
    - echo "Installing golangci-lint..."
    - go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### FR-4: Silent Default
- **FR-4.1**: Add `silent: true` to `default` task
- **Note**: `silent: true` suppresses the task name output, not the command output. `task --list` will still display.

## Non-Functional Requirements

### NFR-1: Breaking Change - Old Names Removed
- `db:up` and `db:down` are **removed** from the main Taskfile
- Users must use `up` and `down` instead
- This is a **clean break**, not backwards compatible
- Migration tasks remain in dev file as `db:migrate`, `db:migrate:*`

### NFR-2: Documentation
- Each task must have a `desc` field
- Dev file default task explains how to access it

### NFR-3: Variable Scope
- Each Taskfile is standalone; variables are not shared
- `build` task stays in main file (has `LDFLAGS` variable)
- Dev file's `ci` task calls main file's `build` via cross-file call

## Out of Scope

- CI-aware execution branching (future enhancement)
- Docker-based command wrapping (future enhancement)
- Variable sharing via includes (adds complexity)

## Acceptance Criteria

1. `task` shows only user-facing tasks (9 tasks)
2. `task dev` (no args) shows development tasks via `task --list`
3. `task dev -- fmt` formats code
4. `task up` starts PostgreSQL
5. `task down` stops PostgreSQL
6. `task test` runs tests (via delegation to dev file)
7. `task ci` runs full CI pipeline (via delegation to dev file)
8. `task init` skips golangci-lint if already installed (status check)
9. `task dev -- db:migrate` runs all migrations
10. `task dev -- recall:demo` runs the recall demo
11. `task dev -- webgui:dev` starts WebGUI in dev mode
12. `db:up` and `db:down` no longer exist (breaking change accepted)

## Decision Log

| Decision | Rationale |
|----------|-----------|
| Keep variables in main file | Avoids include complexity; build vars only needed for main file |
| Rename `db:up`→`up`, `db:down`→`down` | Shorter, matches common patterns |
| Delegate `test` and `ci` | Allows future CI-aware branching without changing user interface |
| No backwards-compat aliases | Clean break; `db:up` users can use `up` |
| Dev file `ci` calls main `build` | Build needs LDFLAGS; cross-file call avoids variable duplication |
| `task dev` shows list when no args | Matches pattern: entry point is discoverable |
