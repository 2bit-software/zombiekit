# Reuse Audit

## Summary
- Duplicates: 3 (redirected to existing)
- Overlaps: 2 (1 extend, 1 create new with rationale)
- Related: 1 (noted for consistency)
- No match: 4

## Findings

### DUPLICATE

#### Claude `<system-reminder>` wrapping
- **Existing**: `internal/hook/agent.go:FormatOutput` (lines 25–38)
- **Decision**: Lift verbatim into `editor_claude.go:claudeFormatter.FormatSessionStart`. No rewrite, no behavior change.
- **Plan change**: Step 2 explicitly notes "copy format strings verbatim" to preserve byte-equivalence.

#### Claude PreToolUse JSON envelope
- **Existing**: `internal/hook/agent.go:FormatPreToolOutput` + `hookResponse`/`hookSpecificOutput` types (lines 41–49, 54–78)
- **Decision**: Move the types and the `json.Marshal` call into `editor_claude.go`. Type definitions remain unexported and local to the Claude formatter file. No behavior change; the existing `json.Marshal` path is already correct for the Claude envelope.
- **Plan change**: Step 2 notes the `hookResponse` types move with the formatter.

#### HookEvent stdin schema
- **Existing**: `internal/hook/types.go:HookEvent` (lines 5–14)
- **Decision**: Reuse as-is for Gemini. Research confirmed base fields (`session_id`, `hook_event_name`, `cwd`, `tool_name`, `tool_input`, `tool_response`) are compatible between Claude Code and Gemini CLI wire formats. Gemini-only fields (`transcript_path`, `timestamp`, `source`, `reason`, `mcp_context`) are not load-bearing for rule injection and are explicitly out of scope.
- **Plan change**: No new parser type introduced. Confirmed in `technical-spec.md` FR6 paragraph.

### OVERLAP

#### `Agent` type and constants
- **Existing**: `internal/hook/types.go:Agent` (lines 37–42) with `AgentClaude`, `AgentGemini` constants.
- **Similarity**: Exactly the identifier type we need for the registry key.
- **Decision**: **Extend** — reuse `Agent` as the registry key type. Do not introduce a parallel `EditorID` type. The spec already chose to defer the `Agent` → `Editor` rename.
- **Rationale**: Introducing a second name for the same concept would be pure churn. Every caller already takes `Agent`.
- **Plan change**: `technical-spec.md` already uses `Agent` as the registry key; confirming here.

#### `DetectAgent` env-var detection
- **Existing**: `internal/hook/agent.go:DetectAgent` (lines 12–20)
- **Similarity**: Does what `ResolveEditor` needs to do, minus the flag path.
- **Decision**: **Create new** `ResolveEditor` and delete `DetectAgent`.
- **Rationale**: Three concrete changes justify replacement over extension: (1) new `(Agent, EditorSource, error)` return signature vs. current single-value return, (2) removal of the `GEMINI_SESSION_ID` check which is not supported by research, (3) default flip from Gemini to Claude. A wrapper function that extended `DetectAgent` would require changing its signature anyway, which amounts to rewriting it. The only caller of `DetectAgent` is `internal/cli/hook.go:56` — one call site — so replacement is safe.
- **Plan change**: Step 4 in `implementation-plan.md` already calls out deletion + replacement; adding this rationale note here.

### RELATED

#### Domain-specific registries in the repo
- **Related code**: `internal/orchestrator/comment_dispatcher.go:CommentDispatcher.RegisterSession`, `internal/recall/postgres/storage.go` (storage driver registration), `internal/recall/claude/import_test.go`
- **Note**: The repo uses `Register*` naming for domain-specific coordination (session lifecycle, storage drivers). None are generic plugin/formatter registries, so there is no shared pattern to adopt. The planned `RegisterEditor` follows the same naming convention, which keeps the codebase consistent without forcing a false abstraction.

### NONE

The following planned items have no prior art in the repo and proceed as planned:

1. **`Formatter` interface** — no existing abstraction in `internal/hook/`. The old code was free functions branching on `Agent`. Grep confirmed no `type.*Formatter` exists anywhere in `internal/`.
2. **Gemini JSON envelope emitter** — the closest thing is the Claude envelope, and per research the shapes differ (no `hookEventName`/`permissionDecision` siblings, `{}` for empty bodies). Not a reuse opportunity, must be written.
3. **`EditorSource` enum type** — no existing provenance/source-tag pattern on `AuditRecord`. `AuditRecord.Agent` is a free string today.
4. **`--editor` CLI flag** — no existing string-enum flag in `internal/cli/hook.go` to extend. `--event` is the only flag today and it is a different semantic axis.

## Plan Changes

No structural changes to `implementation-plan.md` required. Three confirmations folded back into the plan's intent:

1. **Step 2 (Claude formatter)** — explicit reuse of `FormatOutput`/`FormatPreToolOutput` bodies and the `hookResponse` types. Move, don't rewrite.
2. **Step 4 (ResolveEditor)** — rationale for replacing `DetectAgent` rather than wrapping it documented above; the existing plan already captures the mechanical deletion.
3. **Registry key type** — confirmed as `Agent`, not a new `EditorID`. Already what the plan says.

No planned items removed, no planned items replaced with direct references to existing symbols. The plan remains internally consistent; step ordering and dependency graph unchanged.
