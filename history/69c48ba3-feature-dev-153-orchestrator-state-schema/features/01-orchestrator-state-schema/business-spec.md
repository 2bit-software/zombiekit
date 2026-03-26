# Business Spec: Orchestrator State Schema

## Overview

The orchestrator daemon needs persistent state to survive process restarts without losing in-flight work. This ticket defines the schema for that state and ensures the database initializes correctly on first run and subsequent restarts.

## Actors

- **Orchestrator daemon** — the only reader/writer of this state store
- **Human operator** — configures the DB file path, observes startup success/failure

## Entities

### Job

A job represents a single Linear ticket being processed through the autonomous pipeline.

| Field | Description |
|-------|-------------|
| Ticket ID | The Linear ticket identifier (e.g., `DEV-153`). Unique per job. |
| Worktree Path | Absolute filesystem path to the git worktree created for this ticket. |
| Cmux Session | Name of the cmux terminal session running the Claude Code agent. |
| PR Number | GitHub pull request number, if one has been created. Optional. |
| Status | Current state of the job (see Status Lifecycle below). |
| Created At | Automatic timestamp for when the job was first recorded. |
| Updated At | Automatic timestamp, refreshed on every state change. |

**Status Lifecycle:**

```
queued → in_progress → pr_open → completed
                    ↘ failed
         pr_open → needs_attention
```

- `queued` — ticket detected, worktree not yet created
- `in_progress` — agent session spawned, work underway
- `pr_open` — PR created, awaiting review
- `completed` — PR merged, cleanup done
- `failed` — agent reported failure, human triage required
- `needs_attention` — PR review comments need agent attention

### Comment Watermark

Tracks the last-processed PR review comment to avoid reprocessing.

| Field | Description |
|-------|-------------|
| PR Number | The GitHub PR being tracked. Unique per watermark. |
| Last Processed Comment ID | The ID of the most recently processed review comment. |
| Updated At | Automatic timestamp, refreshed on every watermark update. |

### Concurrency Slot

Controls how many jobs can run simultaneously per project.

| Field | Description |
|-------|-------------|
| Project ID | The Linear project being throttled. Unique per slot record. |
| Active Count | Number of currently running jobs for this project. |
| Limit | Maximum concurrent jobs allowed for this project. |

## Behaviors

### B1: First-time Startup

**Given** a configured DB path pointing to a non-existent file,
**When** the orchestrator starts,
**Then** the SQLite file is created, all tables exist with correct columns and constraints, and the process continues normally.

### B2: Idempotent Restart

**Given** a DB file that already exists with all tables,
**When** the orchestrator starts,
**Then** no duplicate tables or errors occur. Existing data is preserved.

### B3: Schema Migration

**Given** a DB file from a previous version missing newer tables/columns,
**When** the orchestrator starts,
**Then** pending migrations apply in order without manual intervention. Existing data is preserved.

### B4: Invalid DB Path

**Given** an invalid or unwritable DB path (e.g., `/root/forbidden/db.sqlite`),
**When** the orchestrator starts,
**Then** the process fails fast with a clear error message indicating the path problem. No partial state is created.

### B5: DB Path Configuration

**Given** the user wants to control where state is stored,
**When** they set `ORCHESTRATOR_DB_PATH` environment variable,
**Then** the orchestrator uses that path instead of the default.

Default path: `~/.zombiekit/orchestrator.db`. Parent directories are created automatically. "Invalid path" (B4) means the path is fundamentally unusable (permission denied, empty string), not merely non-existent.

## Constraints

- **SQLite only** — no PostgreSQL backend needed. The orchestrator runs on a single machine.
- **No CRUD operations in this ticket** — schema and initialization only. Read/write operations are DEV-154.
- **No crash-recovery logic** — reconciliation of stale in-progress jobs is a separate concern.
- **No speculative columns** — only columns that the three watchers (Linear poller, callback server, PR comment poller) will actually read/write.

## Relationships

- **Blocks**: DEV-154 (StateStore CRUD operations) — cannot implement reads/writes without the schema
- **Parent**: DEV-146 (Autonomous Dev Pipeline initiative)
