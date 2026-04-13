# Business Spec: Multi-Editor Hook Support

## Problem

`zk hook` injects zombiekit rules into AI coding agent sessions. Today it is coupled to Claude Code's hook protocol: the output formatters were written against Claude Code's envelope, and the "Gemini fallback" emits plain markdown on stdout. Research confirms this fallback is broken — Gemini CLI has a full hook system (documented in `google-gemini/gemini-cli`) that requires JSON on stdout for every event, and the plain-markdown path will not be consumed correctly. Additionally, agent detection is implicit (env-var sniffing), which is brittle: there is no documented Gemini env var to detect against, and as more editors adopt hook systems (OpenCode, others) env-var sniffing will not scale.

Users configuring zombiekit in their editor's hooks need a deterministic way to tell `zk hook` which editor is calling it, so the right output envelope is emitted.

## Functional Requirements

### FR1 — Explicit editor selection flag
`zk hook` accepts a single string flag `--editor <value>` that names the calling coding environment. Allowed values: `claude`, `gemini`. When present, the flag is authoritative: zombiekit parses and formats for that editor regardless of environment. The flag is global to `zk hook` and applies to every `--event` value.

### FR2 — Claude Code support (parity with today)
With `--editor claude`, `zk hook` produces output byte-equivalent to today's behavior. Byte-equivalence is defined by the existing tests in `internal/hook/agent_test.go` and `internal/hook/handler_test.go` (the Claude-format tests) — those tests continue to pass without modification beyond reaching the new code path.

### FR3 — Gemini CLI support (correct for the first time)
With `--editor gemini`, `zk hook` produces output Gemini CLI consumes successfully:

- **`SessionStart`**: stdout is a JSON object `{"hookSpecificOutput": {"additionalContext": "<rule bodies>"}}`. No Claude-specific sibling fields (`hookEventName`, `permissionDecision`) are present.
- **`PreToolUse`**: stdout is a JSON object `{"hookSpecificOutput": {"additionalContext": "<rule bodies>"}}`. Same shape as SessionStart.
- **`SessionEnd`**: **no-op** — empty stdout, exit 0. Same as Claude today.
- **Rule bodies empty**: stdout is `{}` (empty JSON object), exit 0. Never plain empty string.
- Exit code: always 0 on the happy path. Exit code 2 (Gemini "hard block") and bash-deny semantics are out of scope for this cycle; existing warning behavior is preserved.

### FR4 — Event-name canonicalization at the CLI layer
The canonical event identifier is the `--event` flag value (`session-start`, `pre-tool-use`, `session-end`). The handler's switch on `HookEventName` is the source of truth. If the stdin `hook_event_name` field contains a Gemini-native name (`BeforeTool`, `AfterTool`), it is **ignored** — the CLI flag wins. This keeps event-name translation out of the handler and out of per-editor parsers.

### FR5 — Backward-compatible default
If `--editor` is omitted:

1. Check env vars: `CLAUDE_CODE_ENTRYPOINT` set → `claude`.
2. Otherwise → **default to `claude`** (behavior change: today's default is Gemini).

Existing Claude users' `settings.json` configurations, which invoke `zk hook --event …` without an editor flag, continue to work unchanged. No env detection for Gemini is attempted (no documented env var). Gemini users must pass `--editor gemini` explicitly.

### FR6 — Extension point for future editors
A registry in `internal/hook/editors.go` maps an editor ID (`claude`, `gemini`, later `opencode`) to a formatter. Adding a new editor requires defining a formatter and calling `RegisterEditor(id, formatter)`. No parser-per-editor abstraction is introduced — the existing `HookEvent` struct covers both Claude and Gemini base schemas per research. The `Agent` type name is retained for this cycle (no rename refactor). OpenCode is **not** implemented; the registry only demonstrates that a third ID could be added without touching the handler.

### FR7 — Audit log records editor selection
The `AuditRecord` gains a field identifying how the editor was chosen. The field is an enum-typed string with values `flag`, `env`, `default`. The existing `Agent` field (the editor ID itself) remains unchanged. This is a backward-compatible addition (new optional field); existing log readers that don't know about it will ignore it.

### FR8 — Fail loudly on misuse
Error handling follows the principle "I want to know when things go wrong":

- Unknown `--editor` value: fail at CLI parse time, non-zero exit, stderr message like `unknown editor: <value> (valid: claude, gemini)`. Stdin is not read.
- Empty stdin or malformed JSON: fail with non-zero exit and stderr message describing the parse error. Current Go error behavior is preserved.
- Unrecognized `hook_event_name` (after the CLI `--event` override): fail with non-zero exit and stderr message naming the unknown event.

No silent fallbacks, no empty envelopes masking failures.

### FR9 — Configuration documentation
README is extended with a Gemini CLI `settings.json` example parallel to the existing Claude Code example. Both examples show the minimal config needed to wire zombiekit rules into the editor's `SessionStart` and tool-use hooks. If a `zk init`-style command auto-wires Claude hooks today, a follow-up task to add a Gemini equivalent is noted in the spec's Out of Scope section, but not required in this cycle.

## Acceptance Criteria

- [ ] `zk hook --editor claude --event session-start` produces output byte-equivalent to the current Claude path; existing Claude tests pass unchanged.
- [ ] `zk hook --editor claude --event pre-tool-use` produces the existing `hookSpecificOutput` envelope with `hookEventName` and `permissionDecision`; existing tests pass.
- [ ] `zk hook --editor gemini --event session-start` produces `{"hookSpecificOutput": {"additionalContext": "<bodies>"}}` with no Claude-specific sibling fields. New test.
- [ ] `zk hook --editor gemini --event pre-tool-use` produces `{"hookSpecificOutput": {"additionalContext": "<bodies>"}}`. New test.
- [ ] `zk hook --editor gemini --event session-end` produces empty stdout, exit 0. New test.
- [ ] `zk hook --editor gemini --event session-start` with zero matched rules produces `{}` (valid empty JSON), exit 0. New test.
- [ ] `zk hook --editor frobnitz …` exits non-zero with stderr containing `unknown editor`, does not read stdin. New test.
- [ ] `zk hook --editor gemini …` with malformed stdin exits non-zero with stderr describing the parse error. New test.
- [ ] `zk hook --editor gemini …` with a stdin event whose `hook_event_name` is unrecognized (after CLI override) exits non-zero with stderr naming the event. New test.
- [ ] `zk hook --event session-start` (no flag) with `CLAUDE_CODE_ENTRYPOINT` set behaves as Claude (parity with today).
- [ ] `zk hook --event session-start` (no flag) with no relevant env vars defaults to Claude (behavior change from today's Gemini default).
- [ ] Audit records include both `Agent` (editor ID) and a new `EditorSource` string field with value `flag`, `env`, or `default`. Existing audit log consumers are unaffected.
- [ ] A new file `internal/hook/editors.go` hosts a registry with `RegisterEditor(id, formatter)`; `claude` and `gemini` are registered; handler code calls the registry rather than switching on an `Agent` constant directly.
- [ ] Adding a hypothetical third editor (documented only — no implementation) is a localized change: one new formatter + one `RegisterEditor` call, no handler edits. Verified by code review, not by a test.
- [ ] README contains a working Gemini CLI `settings.json` example alongside the existing hook documentation.

## Out of Scope

- **OpenCode implementation.** Only the registry extension point is required.
- **New event types.** `BeforeAgent`, `AfterTool`, `Notification`, `PreCompress`, etc. are not added. The spec covers `SessionStart`, `PreToolUse`, `SessionEnd` only.
- **Gemini `exit 2` (hard block) semantics** for bash-deny rules. The existing warning-with-exit-0 behavior is preserved across editors.
- **Renaming `Agent` to `Editor`** in the codebase. Nice-to-have, deferred.
- **Gemini extension bundling** (`gemini-extension.json` + `hooks/hooks.json`) as a zombiekit distribution channel.
- **Auto-wiring Gemini `settings.json` via `zk init`** or a parallel bootstrap command. Follow-up task if/when a user requests it.
- **Expanding `HookEvent` struct with Gemini-only fields** (`transcript_path`, `timestamp`, `source`, `reason`, `mcp_context`). Not load-bearing for rule injection; added later if needed.
- **Changing the rule file format or rule-matching engine.**
