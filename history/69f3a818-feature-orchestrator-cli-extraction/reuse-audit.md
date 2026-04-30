# Reuse Audit

## Summary
- Duplicates: 1
- Overlaps: 3 (all extend)
- Related: 4
- No match: 2

## Findings

### DUPLICATE
#### setupWorktree function
- Existing: `internal/orchestrator/watcher_linear.go:151`
- Decision: Lift and reuse as `internal/workspace.Prep`
- Plan change: None—already planned for Phase 0. Single caller (`runTicketPipeline`) confirmed via graph traversal; safe to extract.

### OVERLAP
#### JSON file marker read/write helpers
- Existing: `os.WriteFile/ReadFile` patterns throughout codebase (e.g., `internal/initiative/state.go`, `internal/step/service_test.go`)
- Similarity: All follow the same TOML/JSON marshal + `os.WriteFile(path, data, 0644)` pattern
- Decision: Extend
- Rationale: Plan calls for `internal/workspace/marker.go` with `ReadMarker` and implicit write helpers (via `Prep`). Reuse existing `json.Marshal` + `os.WriteFile` idiom; no new abstraction needed.
- Plan change: None—already specified as small helper module.

#### Config loading pattern
- Existing: `cmd/orchestrator/main.go:70` calls `orchestrator.LoadOrchestratorConfig(c.String("config"))` to load TOML config with `ProjectConfig[]` struct
- Similarity: Plan's `loadProjectConfig` does the same TOML load + project selection logic; orchestrator already exports `ProjectConfig` type
- Decision: Extend
- Rationale: Plan's `internal/cli/config.go` should wrap the existing `orchestrator.LoadOrchestratorConfig`, not reimplement it. Add project-selection logic (cwd match, `--project` flag) on top. Orchestrator already handles TOML parsing; no duplication needed.
- Plan change: Clarify in Phase 1 CLI subcommands: `loadProjectConfig` calls `orchestrator.LoadOrchestratorConfig` internally.

#### --config flag pattern (urfave/cli)
- Existing: `cmd/orchestrator/main.go:41-46` defines `--config` flag with usage, aliases, env vars
- Similarity: Plan uses same `--config` pattern on all `brains {worktree,sandbox,workspace}` subcommands
- Decision: Extend
- Rationale: Reuse same flag definition (copy struct verbatim); no new logic. Both orchestrator and brains CLI follow the same convention.
- Plan change: None—flag definitions are boilerplate.

### RELATED
#### SpawnSession signature
- Related: `internal/cmux/manager.go:58` already defines `SpawnSession(ctx context.Context, ticketID, title, worktreePath string, env map[string]string, prompt string) (string, error)`
- Note: Plan's `workspace.Spawner` interface mirrors this signature exactly (with prompt as final param). No compatibility issue—the interface is a thin wrapper around `cmux.CmuxManager`.

#### worktree.Manager interface
- Related: `internal/worktree/manager.go` (not read, but confirmed via search results and orchestrator usage)
- Note: `workspace.Manager` depends on `worktree.Manager`, which is already an interface in the codebase. Plan correctly identifies this as a dependency, not a new abstraction.

#### sandbox package-level functions
- Related: `internal/sandbox/sandbox.go` exports `Create(ctx, name, worktreePath, cfg)`, `Cleanup(ctx, name)`, `Available()`, `Name(ticketID)`
- Note: Plan introduces `workspace.Sandbox` interface to enable testing without Docker. Correct approach; internal/sandbox lacks an interface, so wrapping is necessary and mentioned in Risk Notes.

#### shortTitle function
- Related: `internal/orchestrator/watcher_linear.go:195` (private function `shortTitle`)
- Note: Plan lifts this to `internal/workspace.ShortTitle` as public. Confirmed private to orchestrator; safe to extract.

### NONE
- `internal/workspace` package — confirmed no existing equivalent; new package needed.
- `.ai/workspace.json` marker file — confirmed no existing equivalent; new marker pattern needed (differs from `.ai/ticket.md` which plan already writes).

## Plan Changes

1. **Phase 0**: Clarify that `workspace.Marker` struct read/write uses standard `json.Marshal` + `os.WriteFile` idiom (no new abstraction).

2. **Phase 1**: In `internal/cli/config.go`, `loadProjectConfig` should call `orchestrator.LoadOrchestratorConfig(path)` internally, not reimplement TOML parsing. Only add cwd-based project inference and `--project` flag handling on top.

3. **Phase 0 + 1**: Both `internal/workspace` and CLI commands can reuse the existing orchestrator's `ProjectConfig` type and `LoadOrchestratorConfig` function directly. No duplication.

4. **Testing**: `internal/workspace/workspace_test.go` must stub `Sandbox` interface to avoid Docker dependency. The plan already calls this out in Risk Notes; no change needed.

**No blocking overlaps found.** All overlaps are resolved by extending existing patterns (config loader, JSON marshaling, flag definitions). The extract is safe and follows established patterns.
