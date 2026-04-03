---
status: complete
updated: 2026-04-02
---

# Research: Active Initiative Detection in /brains.new

## Executive Summary

The `/brains.new` command currently returns a hard error (`INITIATIVE_ALREADY_ACTIVE`) when a different initiative is active. The feature request is to replace this with an interactive prompt offering three resolution options. The change spans two layers: the workflow markdown (where the initiative check logic is described) and the Go MCP tool (where the error is raised).

## Findings

### Codebase Context

- **Initiative lifecycle**: Managed by `internal/initiative/service.go` (Create, Status, Complete, List) and `internal/mcp/tools/initiative/tool.go` (MCP adapter)
- **State file**: `.brains/active.json` — minimal pointer with `initiative` path, `started` timestamp, `status`
- **Conflict detection**: `tool.go:181-186` returns `INITIATIVE_ALREADY_ACTIVE` error when a different initiative is active
- **Idempotency**: Same name+type creation is safe — returns existing with `AlreadyExisted: true`
- **Complete action**: Calls `stateManager.Clear()` (removes `.brains/active.json`), preserves `history/` folder
- **No delete/abandon action exists** — only `complete` clears the active pointer
- **History storage**: Self-contained folders under `history/{id}/`, no cross-references

### Domain Knowledge

- The conflict handling happens in two places:
  1. **Workflow markdown** (e.g., `embed/workflows/feature.md` step 1) — describes the initiative check as instructions to the LLM
  2. **Go MCP tool** (`tool.go:createNewInitiative`) — enforces the constraint in code
- The workflow markdown is the primary control surface — it's what the LLM follows. The Go tool is the enforcement backstop.
- Since `/brains.new` is an LLM-driven command (not a direct CLI), the interactive prompt is implemented by giving the LLM instructions in the workflow markdown to detect the error and present options.

## Decision Points

- [x] **D1**: Where to implement the prompt — in the workflow markdown (LLM instructions) vs Go tool
  - **Decision**: Both. The workflow markdown should detect the active initiative *before* calling create, present options, and act. The Go tool keeps its error as a safety net.
- [x] **D2**: Whether "delete history" should be a new MCP tool action or use bash
  - **Decision**: Add an `abandon` action to the initiative MCP tool — it clears state AND removes the history folder. Cleaner than bash `rm -rf`.
- [x] **D3**: Whether to change the `new.md` command or the individual workflow files
  - **Decision**: The `new.md` command dispatches to workflow files. The initiative check is in the workflow files (step 1). The check-and-prompt logic belongs in the workflow files, but since all workflows share the same step 1, we could also put it in `new.md` before dispatch.

## Recommendations

1. **Add `abandon` action to initiative MCP tool** — clears state AND removes history folder, with confirmation metadata in the response
2. **Update workflow step 1 instructions** — before calling `initiative create`, call `initiative status` first. If active, present the three options to the user via `AskUserQuestion` or direct prompt
3. **Keep the Go-level `INITIATIVE_ALREADY_ACTIVE` error** as a safety net — it should rarely fire if the workflow handles it properly

## Sources

- `internal/mcp/tools/initiative/tool.go` — MCP tool implementation
- `internal/initiative/service.go` — initiative CRUD service
- `internal/initiative/state.go` — FileStateManager
- `embed/workflows/feature.md` — feature workflow with initiative check
- `embed/commands/new.md` — new command classification
