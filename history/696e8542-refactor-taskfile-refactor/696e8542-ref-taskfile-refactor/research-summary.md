# Research Summary: Taskfile Refactor

## Current State Analysis

### Existing Tasks (14 total)

| Task | Description | Proposed Location |
|------|-------------|-------------------|
| `default` | List available tasks | `Taskfile.yml` |
| `init` | Download dependencies and install dev tools | `Taskfile.yml` |
| `build` | Build the brains CLI binary | `Taskfile.yml` |
| `install` | Build and install brains to GOBIN | `Taskfile.yml` |
| `test` | Run tests with coverage | `Taskfile.yml` (delegates to dev) |
| `db:up` | Start PostgreSQL container | `Taskfile.yml` (rename to `up`) |
| `db:down` | Stop PostgreSQL container | `Taskfile.yml` (rename to `down`) |
| `db:migrate` | Run all database migrations | `Taskfile.dev.yml` |
| `db:migrate:memory` | Run memory table migration | `Taskfile.dev.yml` |
| `db:migrate:recall` | Run recall chunks migration | `Taskfile.dev.yml` |
| `fmt` | Format Go code | `Taskfile.dev.yml` |
| `vet` | Run go vet | `Taskfile.dev.yml` |
| `lint` | Run golangci-lint | `Taskfile.dev.yml` |
| `ci` | Run all CI checks | `Taskfile.yml` (delegates to dev) |
| `ollama:pull` | Pull embedding model | `Taskfile.dev.yml` |
| `recall:demo` | Demo recall feature | `Taskfile.dev.yml` |
| `webgui:dev` | Start WebGUI dev mode | `Taskfile.dev.yml` |

### Variables (shared)

The current Taskfile defines version/build variables that should remain in the main Taskfile:
- `VERSION`, `COMMIT`, `BUILD_DATE`, `VERSION_PKG`, `LDFLAGS`

## Gap Analysis

| Pattern | Current | Target |
|---------|---------|--------|
| Two-file architecture | Single file | `Taskfile.yml` + `Taskfile.dev.yml` |
| `dev` entry point | Missing | `task dev -- <args>` |
| `silent: true` on default | Missing | Add to `default` |
| `status:` for idempotency | Inline check in `init` | Convert to `status:` pattern |
| Short lifecycle names | `db:up`, `db:down` | `up`, `down` |
| Task delegation | Not used | Delegate `test`, `ci` to dev file |

## Proposed Split

### `Taskfile.yml` (User-Facing)

Tasks users need daily:
- `default` - List tasks (with `silent: true`)
- `dev` - **NEW** - Delegate to dev Taskfile
- `init` - Install dependencies (with `status:` pattern)
- `build` - Build CLI
- `install` - Install CLI
- `test` - Run tests (delegate to dev)
- `up` - Start services (rename from `db:up`)
- `down` - Stop services (rename from `db:down`)
- `ci` - CI checks (delegate to dev)

### `Taskfile.dev.yml` (Development)

Internal tasks:
- `default` - List dev tasks
- `test` - Implementation of test runner
- `ci` - Implementation of CI pipeline
- `fmt` - Format code
- `vet` - Run go vet
- `lint` - Run linter
- `db:migrate` - Run migrations
- `db:migrate:*` - Individual migrations
- `ollama:pull` - Pull ML models
- `recall:demo` - Demo features
- `webgui:dev` - WebGUI development

## Technical Decisions

1. **Keep variables in main Taskfile** - Build metadata shared across both files via includes would add complexity; simpler to duplicate or reference
2. **Rename `db:up`/`db:down` to `up`/`down`** - Matches common pattern, shorter commands
3. **`test` delegation** - Main file calls dev file for implementation, allows CI-aware branching later
4. **`ci` delegation** - Main file provides entry point, dev file has full pipeline
