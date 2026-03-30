# Technical Preferences: Orchestrator Admin CLI

## Implementation Hints (from user)

- Commands will later be exposed as HTTP endpoints, then a webgui
- Focus on CLI subcommands first — HTTP and GUI are future phases
- The orchestrator binary already uses `urfave/cli/v2`
- State is in SQLite via `state.StateStore` interface

## Architecture Decision: Admin Service Layer

User chose to create an `internal/admin` package as the reuse boundary between CLI (now) and HTTP (next). This is a deliberate choice over the simpler "extend StateStore only" approach.

Rationale: two known consumers (CLI + HTTP) are on the roadmap, and compound operations like "delete job + release slot" don't belong on the store.

**Pattern:**
- `internal/admin/service.go` — `Service` struct with typed methods
- CLI handlers (`cmd/orchestrator/`) parse args, call Service, format output
- Future HTTP handlers will call the same Service methods, marshal to JSON

The Service takes `state.StateStore` via constructor injection. No other dependencies needed for the admin operations.

## Architecture Decision: Schema Migration for project_id

User chose to add `project_id` to the `jobs` table via a new migration (over requiring `--project-id` flag on delete, or separating delete + slot reset).

Rationale: the data model should be correct. The HTTP/GUI layer will need this association anyway. Existing rows get empty string; the daemon populates it for new jobs.

## Architecture Decision: Daemon moves to `run` subcommand

Bare `orchestrator` currently starts the daemon. After subcommands, it shows help. Daemon moves to `orchestrator run`. This is a breaking change — acceptable because the system is pre-production.

**Flag scoping:**
- `--db-path` is global (shared by admin and daemon)
- All other daemon flags (`--linear-api-key`, `--poll-interval`, etc.) move to the `run` subcommand

## Existing Patterns to Follow

- `state.ApplyReconciliation()` — standalone function that takes a store, does its work, returns. Admin service methods follow the same pattern.
- `state.ErrJobNotFound` — sentinel errors for not-found cases. Reuse for `DeleteJob`.
- Status constants in `state/store.go` — reuse for validation in the admin service.
