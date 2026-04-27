# Reuse Audit

## Summary
- Duplicates: 0
- Overlaps: 2 (2 extend existing patterns)
- Related: 2 (noted for consistency)
- No match: 4

## Findings

### OVERLAP

#### TOML config loader
- **Existing**: `internal/config/loader.go:49-60` — `LoadFile(path)` using `toml.DecodeFile`
- **Similarity**: Same library (BurntSushi/toml), same decode-into-struct pattern
- **Decision**: Create new (different package, different struct shape, explicit path vs discovery)
- **Rationale**: Orchestrator config loads a single explicit file, not local+global merge. Reuse the pattern, not the function.

#### Config validation (configRule pattern)
- **Existing**: `internal/orchestrator/config.go:78-98` — `configRule` struct with `msg` + `check func(*Config) bool`
- **Similarity**: Identical validation pattern needed for new config types
- **Decision**: Extend — keep `configRule` idiom, write new rules for `OrchestratorConfig`/`ProjectConfig`
- **Rationale**: Same package, same pattern, same filesystem checks (`.git` validation, `MkdirAll`). Direct reuse of the pattern.

### RELATED

#### EventDemuxer (CommentDispatcher)
- **Related code**: `internal/orchestrator/comment_dispatcher.go:37-42` — mutex-guarded map of buffered channels
- **Note**: Validates the concurrent map-of-channels pattern works in this codebase. EventDemuxer needs different semantics (static registration, continuous streaming, drop-on-full) so create new, but borrow the structural pattern.

#### Migration SQL (existing migration infrastructure)
- **Related code**: `internal/state/migrations.go:14` — `//go:embed migrations/*.sql` + sequential runner
- **Note**: Just add `003_composite_pks.sql` to the directory. The embed directive and runner handle it automatically. No runner changes needed.

### NONE

| Planned Item | Notes |
|---|---|
| Duration wrapper type | No existing Duration wrapper anywhere in codebase |
| Supervisor/restart pattern | No retry/backoff/restart patterns exist. shutdown.Manager is fail-fast only. |
| Health tracking | All 3 existing `/healthz` endpoints are static "ok" stubs — no state tracking |
| Per-project scoping (ProjectRunner) | No multi-instance pattern exists in the codebase |

## Plan Changes

No plan items need to be replaced with existing code. Two items should note the existing patterns to extend:

1. **Phase 1.3 (Config validator)**: Reference `configRule` pattern at `config.go:78-98` — extend with new rules rather than inventing a new validation approach
2. **Phase 4.3 (EventDemuxer)**: Note structural similarity to `CommentDispatcher` — borrow mutex+map pattern
