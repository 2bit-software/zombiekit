# Research Summary

## Headline

Gemini CLI has a **full, documented hook system** closely mirroring Claude Code's JSON-over-stdio pattern. Zombiekit's current "Gemini = plain markdown fallback" path is **incorrect** — Gemini requires JSON on stdout for every event. The existing agent-polymorphic plumbing inside `internal/hook/` is sound; only the CLI-layer detection and the Gemini formatters need work.

## Gemini CLI hook protocol

**Source**: `docs/hooks/reference.md` in `google-gemini/gemini-cli`, `geminicli.com/docs/hooks/`, Google Developers Blog post.

### Events (Gemini name → Claude equivalent)

| Gemini event | Claude event | Notes |
|---|---|---|
| `SessionStart` | `SessionStart` | Same name. Gemini adds `source: startup\|resume\|clear`. |
| `SessionEnd` | `SessionEnd` | Gemini adds `reason: exit\|clear\|logout\|…`. |
| `BeforeTool` | `PreToolUse` | Different name. |
| `AfterTool` | `PostToolUse` | Different name. |
| `BeforeAgent` | — | No Claude equivalent. |
| `AfterAgent` | — | No Claude equivalent. |
| `BeforeModel` / `AfterModel` / `BeforeToolSelection` | — | Not needed for rule injection use case. |
| `Notification` / `PreCompress` | — | Not needed. |

### Input schema (stdin, base fields)

```json
{
  "session_id": "string",
  "transcript_path": "string",
  "cwd": "string",
  "hook_event_name": "string",
  "timestamp": "string"
}
```

All snake_case — same convention as Claude. `BeforeTool` appends `tool_name`, `tool_input`, `mcp_context`. `AfterTool` appends `tool_response` as well.

Zombiekit's existing `HookEvent` struct (`internal/hook/types.go`) already covers `session_id`, `hook_event_name`, `cwd`, `tool_name`, `tool_input`, `tool_response`. It is missing `transcript_path`, `timestamp`, `source`, `reason`, `mcp_context` — but none of those are load-bearing for rule injection today. The struct is effectively compatible with Gemini's base schema already.

### Output schema (stdout) — the breaking difference

Gemini **only accepts JSON on stdout**. Stderr is for logs. Exit 0 = success, exit 2 = hard block, other non-zero = warning.

Common envelope:

```json
{
  "systemMessage": "string",
  "suppressOutput": true,
  "continue": true,
  "decision": "allow" | "deny",
  "reason": "string",
  "hookSpecificOutput": { "additionalContext": "..." }
}
```

**Critical**: `hookSpecificOutput.additionalContext` is the field Gemini uses to inject text into the model's context — identical field name to Claude Code. The *difference* is that Claude also requires sibling fields `hookEventName` and `permissionDecision` in its envelope, while Gemini does not. Also, Claude currently accepts plain stdout for `SessionStart`; Gemini does not — Gemini needs JSON for every event.

### Configuration

`settings.json` at one of three precedence layers (project `.gemini/settings.json`, user `~/.gemini/settings.json`, system `/etc/gemini-cli/settings.json`) or bundled in a Gemini extension's `hooks/hooks.json`. Shape:

```json
{
  "hooks": {
    "BeforeTool": [
      {
        "matcher": "write_file|replace",
        "hooks": [
          { "type": "command", "command": "zk hook --gemini --event pre-tool-use", "timeout": 60000 }
        ]
      }
    ],
    "SessionStart": [
      { "hooks": [ { "type": "command", "command": "zk hook --gemini --event session-start" } ] }
    ]
  }
}
```

### Environment variables

**No documented `GEMINI_CLI_ENTRYPOINT` analogue.** The current detection check for `GEMINI_SESSION_ID` in `agent.go:16` is unverified by the official reference. This means implicit detection of Gemini-vs-Claude from env vars alone is unreliable. An explicit flag is the right call.

## Zombiekit hook subsystem (current state)

**CLI**: `internal/cli/hook.go` has one flag (`--event`) with values `session-start`, `pre-tool-use`, `session-end`. Agent detection via `DetectAgent()` happens at `hook.go:56`, *after* flag parsing, *before* `NewHandler()`. This is the single point where a new `--editor`/`--<coding-env>` flag would override detection.

**Event decode**: `json.NewDecoder(os.Stdin).Decode(&event)` at `hook.go:38`. The `HookEvent` struct is schema-compatible with both Claude Code and Gemini base events.

**Handler**: `internal/hook/handler.go` dispatches on `HookEventName` (`SessionStart`, `PreToolUse`, `SessionEnd`). It calls `FormatOutput` (SessionStart) and `FormatPreToolOutput` (PreToolUse), both of which take `Agent` as a parameter. **No handler logic changes needed to support additional editors** — only the formatters.

**Formatters**: `internal/hook/agent.go`.
- `FormatOutput(agent, bodies)` → Claude wraps in `<system-reminder>`; default (Gemini) returns plain markdown (**broken for Gemini**).
- `FormatPreToolOutput(agent, bodies)` → Claude returns the `hookSpecificOutput` envelope with `hookEventName` + `permissionDecision` + `additionalContext`; default returns plain markdown (**also broken for Gemini**).

Both Gemini formatter paths need to emit JSON: `{"hookSpecificOutput": {"additionalContext": "..."}}`.

**Event-name translation**: Gemini calls tool hooks `BeforeTool`/`AfterTool`, not `PreToolUse`/`PostToolUse`. If the user configures Gemini to call `zk hook --gemini --event pre-tool-use`, the CLI flag already normalizes to `PreToolUse` — so the handler switch doesn't care. But if the user passes Gemini's event name directly via the stdin `hook_event_name` field, the handler's switch will miss. **Simplest design**: keep using the CLI `--event` flag as the canonical event identifier; the hook config maps Gemini's `BeforeTool` → `--event pre-tool-use` in the `settings.json` command string.

**Audit**: `AuditRecord.Agent` is a free-form string, already populated from `DetectAgent()`. No schema change needed.

**Tests**: `agent_test.go` has explicit Claude-vs-Gemini format tests — these will need updates once Gemini's output becomes JSON. `handler_test.go` uses `AgentGemini` as the default for most tests, which will trigger test updates when Gemini output format changes.

**Docs**: README has no hook documentation. Hook configs are currently considered an implementation detail auto-wired by init commands.

## Extension design implications

- One `Agent` type (alias `EditorID` if we want the rename) + a registry of parsers/formatters indexed by ID.
- CLI accepts a single `--editor` string flag with allowed values `claude`, `gemini`, later `opencode`. Bool flags (`--claude`/`--gemini`) get unwieldy past three editors. Fallback to env detection if flag is absent, for backward compatibility.
- Each editor has a formatter for `SessionStart` and `PreToolUse` output. Parsers can be shared while event schemas remain compatible.
- Event-name normalization stays at the CLI layer (`--event` flag → canonical `HookEventName`), not in the stdin payload.

## Open questions resolved by research

- **Does Gemini have hooks?** Yes — full system, documented.
- **Is current Gemini path correct?** No — it emits plain stdout where Gemini requires JSON.
- **Does `GEMINI_SESSION_ID` env var exist?** Not documented in the official hook reference. Existing detection is best-effort; the flag should be the primary signal.
- **Is a unified parser viable?** Yes — Gemini and Claude share base field names (snake_case) and the relevant tool-event fields.

## Open questions remaining

- Should `--editor` default to `claude` (most common today) or `gemini` (current fallback) when flag is absent and env detection fails? Recommendation: `claude`, because current fallback default is `gemini` and that path is broken — flipping the default to Claude reduces the blast radius of unconfigured invocations.
- Is `Agent` the right type name going forward, or should it be renamed `Editor` to match the flag naming? Leaning: rename to `Editor` for clarity, since "agent" is heavily overloaded in this codebase.
- What does `zk init --gemini` (or equivalent) look like? Out of scope for this cycle unless it already exists and needs updating.
