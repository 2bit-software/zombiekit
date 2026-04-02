---
status: complete
updated: 2026-04-01
---

# Research: Session-Aware Rules Injection

## Executive Summary

Claude Code and Gemini CLI have converged on nearly identical hook protocols (JSON via stdin, exit codes, matcher patterns). Both provide direct compaction signals (`PreCompact`, `PostCompact`, `SessionStart` with `source: "compact"`). Claude Code rules use a single frontmatter field (`paths`) with gitignore-style globs. A shared JSON file under `/tmp` is the optimal session state backend — sub-1ms reads, zero daemon overhead, well within the generous hook timeout budgets (60-600s).

## Findings

### Codebase Context

**Existing hook system**: None in zombiekit. Current hooks are external shell scripts in `~/.claude/scripts/` (prek-check, sentrux-gate, grep-mcp-hint, codeforge-hook). All follow the same pattern: read JSON stdin → extract file_path → process → stdout/exit code.

**Session management**: `SessionManager` exists in `/internal/mcp/tools/codereasoning/manager.go` with in-memory session map and timestamp tracking. Session ID is currently hardcoded to "default" — needs enhancement.

**Profile composition system**: Proven pattern in `/internal/profile/` — DAG-based composition with multi-source resolution (local → global → embedded), frontmatter + body structure, cycle detection, deduplication. This is the closest existing analog to rules composition.

**MCP server architecture**: Modular tool registration in `/internal/mcp/server.go`. Service + Tool handler pairs. Tools: profile-compose, workflow-compose, code-reasoning, stickymemory, git, gh-pr, initiative.

**CLI framework**: urfave/cli. Entry points at `/cmd/zk-server/main.go` (server) and `/cmd/brains/main.go` (CLI).

### Domain Knowledge

**Claude Code hooks**: 26 events. Key ones: `SessionStart` (startup|resume|compact), `PreCompact`, `PostCompact`, `PreToolUse` (can block), `PostToolUse`. Input via stdin JSON with `session_id`, `transcript_path`, `cwd`, `hook_event_name`. Output: exit 0 + stdout text → injected as context. Also supports JSON output with `additionalContext` field. Environment variables: `$CLAUDE_SESSION_ID`, `$CLAUDE_PROJECT_DIR`, `$CLAUDE_ENV_FILE`.

**Claude Code rules**: `.claude/rules/*.md` with YAML frontmatter. Only one field: `paths` (string or string[]). Gitignore-style globs via `ignore` npm package. Brace expansion supported. Rules without `paths` load unconditionally. Path-scoped rules load when Claude reads matching files. Content wrapped in `<system-reminder>` tags.

**Gemini CLI hooks**: 11 events, nearly identical protocol. Key extras: `BeforeAgent` (inject context per-turn), `BeforeModel` (modify LLM request). Environment: `$GEMINI_SESSION_ID`, `$GEMINI_PROJECT_DIR` (+ `CLAUDE_PROJECT_DIR` compat alias). Same JSON stdin/stdout, exit code contract.

**Gemini CLI rules**: `GEMINI.md` hierarchy (global → workspace → JIT). Configurable filename via `context.fileName`. Import syntax `@file.md`. JIT loading matches Claude Code's subdirectory lazy loading. `GEMINI_SYSTEM_MD` for full system prompt override.

**Compaction lifecycle (both agents)**:
1. `PreCompact` fires (source: auto|manual) — save state
2. LLM summarizes conversation
3. `PostCompact` fires — cleanup
4. `SessionStart` fires with source: "compact" — re-inject rules
5. CLAUDE.md/GEMINI.md re-loaded from disk

**Session state patterns**:

| Pattern | Latency | Complexity | Best For |
|---------|---------|-----------|----------|
| Shared JSON file | ~0.2ms | Minimal | Hook state (recommended) |
| SQLite + WAL | ~1-5ms | Low | Queryable history |
| Unix socket daemon | ~0.05ms | High | Complex state |

**Tool input shapes for file path extraction**:
- Read: `tool_input.file_path`
- Write: `tool_input.file_path`
- Edit: `tool_input.file_path`
- Response: `tool_response.filePath` (note: different casing)

**Frontmatter audit**: Claude Code rules support exactly one field: `paths`. The YAML parser is generic (accepts anything) but the rules loader only reads `paths`. Skills (`.claude/skills/`) have a separate 15-field schema. Writing unsupported fields in rules files is silently ignored by Claude Code — making our superset approach safe.

## Decision Points

- [x] **D1**: Rules frontmatter format — Use Claude Code's `paths` field as-is. Our format is a superset: same `paths` semantics, reserving right to add fields later.
- [ ] **D2**: Session state backend — JSON file recommended. Alternatives: SQLite (if queryable history needed), daemon (if cross-session state needed).
- [ ] **D3**: Rules file location — `.brains/rules/` (project), `~/.brains/rules/` (global), or reuse existing `.claude/rules/` directly?
- [ ] **D4**: Injection format — `<system-reminder>` tags for Claude, plain markdown for Gemini. Or agent-agnostic format?
- [ ] **D5**: Hook binary — Standalone `zk-hook` binary, or subcommand of existing `brains` CLI (`brains hook`)?
- [ ] **D6**: Which hooks to register — Read-time injection (new behavior) vs write-time (Claude Code default) vs both?
- [ ] **D7**: Compaction reset strategy — Full re-inject on `SessionStart source=compact`, or selective re-inject based on what files are in the compacted summary?

## Recommendations

1. **Single Go binary hook** (`brains hook` subcommand) that handles both Claude Code and Gemini CLI events. Detect agent via env vars.
2. **Session state via JSON file**: `/tmp/zk-session-${SESSION_ID}.json`. Store: injected rule sets, files touched, compaction count.
3. **Hook registration**: `SessionStart` (startup|resume|compact) for global rules, `PostToolUse` (Read|Write|Edit) for path-specific rules, `PreCompact` for state snapshot, `SessionEnd` for cleanup.
4. **Extend profile system**: Add `type: rule` and `paths` to profile frontmatter. Reuse composition engine for rules resolution.
5. **Rules stored as markdown** with `paths` frontmatter (Claude Code compatible). Wrap in `<system-reminder>` for Claude, plain markdown for Gemini.
6. **Read-time injection**: Inject rules when agent reads a file, not just on write. This front-loads guidance before the agent makes decisions.

## Sources

- https://code.claude.com/docs/en/hooks — Claude Code hooks reference
- https://code.claude.com/docs/en/memory — CLAUDE.md and rules system
- https://geminicli.com/docs/hooks/reference/ — Gemini CLI hooks
- https://geminicli.com/docs/cli/gemini-md/ — GEMINI.md system
- https://geminicli.com/docs/cli/system-prompt/ — Gemini system prompt override
- Claude Code v2.1.90 binary analysis — frontmatter parser source
- zombiekit codebase — profile system, MCP server, session management
