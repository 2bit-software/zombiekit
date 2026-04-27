# Business Spec: Multi-Project Orchestrator Support

## Problem

The orchestrator currently watches a single project. Running multiple projects requires launching separate orchestrator processes, which is operationally burdensome.

## Goal

A single orchestrator process watches N projects from a unified configuration file.

## Functional Requirements

### Configuration
- Orchestrator accepts a TOML config file via `--config orchestrator.toml`
- Config has two sections: `[global]` for shared settings, `[[project]]` for per-project settings
- Each project defines: unique ID (must match `[a-z0-9][a-z0-9-]*`), repo directory, GitHub owner/repo, Linear project ID, base branch, worktrees root, concurrency limit, tracking label, copy files, closed-PR-ticket status
- Global settings include: DB path, callback port, poll interval, log level, log format, shutdown timeout, bot username, sandbox config
- Credentials (Linear API key, GitHub token) defined in `[global]`; each `[[project]]` may optionally override them (global is the fallback)
- CLI flags override global settings only (not per-project); precedence: CLI flag > env var > config file
- `base_branch` defaults to `"main"` if omitted from a project

### Config Validation (fail-fast at startup)
- All projects validated before any project starts
- Rejects: duplicate project IDs, missing required fields, invalid durations
- Rejects: project IDs containing characters outside `[a-z0-9-]` or not starting with `[a-z0-9]`
- Rejects: duplicate `(github_owner, github_repo)` pairs across projects
- Verifies: each project's `repo_dir` contains a `.git` directory
- Creates: each project's `worktrees_root` directory if it doesn't exist
- Warns: overlapping `worktrees_root` paths across projects

### Runtime Behavior
- Each project gets its own independent set of watchers (LinearPoller, PRWatcher, CommentWatcher) and event router
- A shared callback server (single port) receives webhook events for all projects
- Events are routed to the correct project via project ID in the callback URL path
- Each project has its own concurrency limit, enforced independently
- A failure in one project's watchers does not affect other projects — failed watchers restart with exponential backoff (1s initial, 2min cap, hard-coded)
- Backoff resets to 1s after a watcher runs successfully for at least one full poll interval
- A ProjectRunner never returns an error to the top-level errgroup — it logs failures and retries indefinitely with capped backoff. Total project failure is surfaced via the health endpoint, not via process shutdown.
- Infrastructure failures (callback server crash, EventDemuxer crash) trigger full shutdown

### Data Isolation
- Jobs are scoped by project: composite key `(project_id, ticket_id)` — same ticket ID in different projects does not collide
- Comment watermarks are scoped by project: composite key `(project_id, pr_number)` — same PR number in different GitHub repos does not collide
- Concurrency slots are already per-project (no change needed)
- `ListJobsByStatus` and `GetJobByPR` accept a `projectID` parameter — each project's watchers only see their own jobs
- `ListAllJobs` remains global (admin use)

### Migration
- Migration 003 recreates tables with composite PKs. Existing data is dropped (no backfill). This is a clean break — restart any in-flight agents after upgrade.
- CLI flags for single-project mode are removed. Config file is the only way to run the orchestrator.

### Health Reporting
- Single `/healthz` endpoint returns JSON with global status and per-project watcher states
- A project is "unhealthy" if any of its watchers has been in continuous backoff for >5 minutes

### Reconciliation
- Runs once globally at startup before any project starts (existing pattern)
- Validates that all active jobs belong to a configured project
- Jobs belonging to unconfigured projects are logged as warnings and their slots are released

## Acceptance Criteria

- [ ] Orchestrator starts with `--config orchestrator.toml` and watches all defined projects
- [ ] Each project gets its own set of watcher goroutines (LinearPoller, PRWatcher, CommentWatcher, Router)
- [ ] Jobs are scoped to projects — composite PK `(project_id, ticket_id)`
- [ ] Comment watermarks are scoped to projects — composite PK `(project_id, pr_number)`
- [ ] Callback URLs: `POST /project/{projectID}/{ticketID}/{action}`
- [ ] Events are demuxed to the correct project's router via EventDemuxer
- [ ] A project watcher failure triggers restart with backoff, not process-wide shutdown
- [ ] Infrastructure failure (callback server) triggers clean shutdown of all projects
- [ ] Migration 003 recreates tables with composite PKs (clean break, drops existing data)
- [ ] Config validation rejects invalid configs at startup before any project starts
- [ ] `orchestrator jobs` and `orchestrator slots` subcommands show project ID in output
- [ ] `/healthz` returns per-project watcher health status
- [ ] Reconciliation at startup releases slots for jobs belonging to unconfigured projects

## Out of Scope

- Per-project poll intervals (global only for now)
- Hot-reloading config changes
- Web UI for config management
- Cross-workspace Linear support (assumes single Linear workspace)

## Resolved Questions

1. **Migration**: Clean break. Drop and recreate tables with composite PKs. No backfill, no legacy flag. Restart agents after upgrade.
2. **Credential inheritance**: Per-project `github_token` and `linear_api_key` fields are optional in `[[project]]`. If omitted, inherited from `[global]`.
3. **Callback URLs**: New format only (`/project/{projectID}/{ticketID}/{action}`). No old-format support.
4. **CLI flags**: Config file only. No legacy single-project CLI flag mode.
5. **Health endpoint**: Single `/healthz` returning JSON with per-project status.
6. **Slot in handleComplete**: Research confirmed this is intentional — the slot represents an active PR being watched. Released by PRWatcher on merge/close, not by the router. Not a bug.
