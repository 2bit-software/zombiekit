# Research Summary: Orchestrator Admin CLI

## Codebase Findings

### State Store Interface

The `state.StateStore` interface already exposes most operations needed:
- `GetJob(ctx, ticketID)` -> single job lookup
- `ListJobsByStatus(ctx, ...statuses)` -> filtered list (but no "list all" method)
- `SetJobStatus(ctx, ticketID, status)` -> status update with validation
- `ReleaseSlot(ctx, projectID)` -> single slot release
- `ResetAllSlots(ctx)` -> bulk slot reset

**Missing from the interface:**
- `ListAllJobs()` — `ListJobsByStatus` requires at least one status. A new method or a "list all" variant is needed.
- `DeleteJob(ticketID)` — no delete operation exists. Needs to be added to the interface and SQLite implementation.
- `ListSlots()` — no way to query slot state. Needs a new method.

### CLI Framework

The binary uses `urfave/cli/v2`. Currently has zero subcommands — the `Action` on the app itself runs the daemon. Adding subcommands is straightforward with urfave: define `Commands` on the app, move the daemon into a `run` (or `start`) subcommand.

**Breaking change consideration**: Currently `orchestrator` (no args) starts the daemon. After adding subcommands, bare `orchestrator` would show help. The daemon would move to `orchestrator run` or `orchestrator start`. This is a deliberate UX improvement but changes the invocation. Since this is pre-production, the break is acceptable.

### Reconciliation Pattern

`state.ApplyReconciliation()` is a good pattern to follow — standalone function that takes a store, does its work, returns. Admin commands should follow the same pattern: standalone functions in a dedicated package (or in `state/`) that accept a store and return results.

### Database Concurrency

SQLite is configured with WAL mode and 5-second busy timeout. Read operations (list, get) will work fine even with the daemon running. Write operations (delete, set-status, reset) may conflict with daemon writes but should succeed within the busy timeout window in practice.

### Job Lifecycle

```
queued -> in-progress -> complete -> closed
                      \-> needs-attention (manual recovery)
```

Status constants are defined in `state/store.go`: `StatusQueued`, `StatusInProgress`, `StatusNeedsAttention`, `StatusComplete`, `StatusClosed`.

### Concurrency Slots Schema

```sql
concurrency_slots (
    project_id   TEXT PRIMARY KEY,
    active_count INTEGER NOT NULL DEFAULT 0,
    slot_limit   INTEGER NOT NULL DEFAULT 1
)
```

No existing method to SELECT from this table for display purposes.

## Design Decisions

### Service Layer Placement

Admin operations should live in `internal/state/` (for CRUD) or a new `internal/admin/` package (for compound operations like "delete job + release slot"). Given that most operations map 1:1 to store methods, extending the `StateStore` interface is simpler and avoids an unnecessary abstraction layer.

### Output Format

Plain text with aligned columns for lists. No JSON flag for now — add it when the HTTP layer is built. Keep it simple.

### Subcommand Structure

```
orchestrator run              # start the daemon (moved from root action)
orchestrator jobs list        # list all jobs
orchestrator jobs get <id>    # show single job
orchestrator jobs delete <id> # remove job + release slot
orchestrator jobs set-status <id> <status>  # update status
orchestrator slots list       # show slot state
orchestrator slots reset      # reset all slots
```
