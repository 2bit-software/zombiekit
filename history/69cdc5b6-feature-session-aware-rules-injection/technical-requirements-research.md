# Technical Requirements Research

## Resolved Implementation Decisions

1. **Hook binary**: `brains hook` subcommand of existing CLI (not standalone binary)
2. **Hooks on Read/Write/Edit/MultiEdit** — PostToolUse events trigger path-based rules injection
3. **Session state via JSON file**: `/tmp/zk-session-{SESSION_ID}.json`
4. **Rules location**: `.brains/rules/` (project) + `~/.brains/rules/` (global). Not `.claude/rules/` — Claude handles its own.
5. **Conflict resolution**: Composition — both project and global rules with same filename are injected independently. Project appears first.
6. **Output format**: Plain text stdout. `<system-reminder>` wrapping for Claude, plain markdown for Gemini.
7. **SessionStart resume**: Reset tracking. Don't assume rules persist across resume.
8. **Custom API agents**: Out of scope for MVP.
9. **Bash file creation**: Out of scope.
10. **Rules frontmatter**: Superset of Claude Code's `paths` field. v1 is `paths` only.

## Hook Registration

```json
{
  "hooks": {
    "SessionStart": [{
      "matcher": "startup|resume|compact",
      "command": "brains hook --event session-start"
    }],
    "PostToolUse": [{
      "matcher": "Read|Write|Edit|MultiEdit",
      "command": "brains hook --event post-tool-use"
    }],
    "SessionEnd": [{
      "command": "brains hook --event session-end"
    }]
  }
}
```

## Session State File Shape

```json
{
  "session_id": "abc123-def456",
  "agent": "claude",
  "started_at": "2026-04-01T10:00:00Z",
  "compaction_count": 0,
  "injected_rules": {
    "project:go.md": "2026-04-01T10:01:00Z",
    "global:coding-general.md": "2026-04-01T10:00:05Z"
  }
}
```

Rule identity key: `{source}:{filename}` where source is `project` or `global`.

## Rules Frontmatter Schema (v1)

Superset of Claude Code. v1 supports only `paths` (identical semantics). Future fields reserved.

```yaml
---
paths:
  - "**/*.go"
---
# Go Standards
...
```

- `paths`: string or string[] — gitignore-style globs, brace expansion supported
- Absent `paths` → unconditional rule (injected at every SessionStart)
- Additional fields silently ignored by Claude Code's native loader — safe to add later

## Agent Detection

| Check Order | Env Var | Agent | Output Format |
|-------------|---------|-------|---------------|
| 1 | `CLAUDE_SESSION_ID` | Claude Code | `<system-reminder>` wrapped |
| 2 | `GEMINI_SESSION_ID` | Gemini CLI | Plain markdown |

Session ID source: stdin JSON `session_id` field (authoritative). Env var used only for agent detection.

## File Path Extraction by Tool

| Tool | Path Source |
|------|------------|
| Read | `tool_input.file_path` |
| Write | `tool_input.file_path` (fallback: `tool_response.filePath`) |
| Edit | `tool_input.file_path` (fallback: `tool_response.filePath`) |
| MultiEdit | `tool_input.edits[].file_path` |
