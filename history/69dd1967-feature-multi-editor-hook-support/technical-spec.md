# Technical Spec: Multi-Editor Hook Support

## Scope

Implements FR1–FR9 from `business-spec.md`. Adds `--editor <claude|gemini>` to `zk hook`, fixes the broken Gemini output path, introduces a small editor registry, and records how the editor was chosen in the audit log.

## Architecture

### Flow diagram

```
stdin JSON ──▶ cli/hook.go
                  │
                  ├─ parse --editor flag
                  ├─ resolveEditor()  ──▶ (editor, source)
                  │                        source ∈ {flag, env, default}
                  ├─ decode HookEvent (existing)
                  ├─ normalize event name from --event flag (existing)
                  ├─ NewHandler(cwd, home, editor)
                  ├─ Handle() ──▶ HandleResult{bodies, matched, skipped}
                  ├─ formatter := editors.Lookup(editor)
                  ├─ output := formatter.Format(eventName, bodies)
                  └─ stdout.Write(output)
```

Handler logic is unchanged. Formatters become per-editor and live in a registry.

### New file: `internal/hook/editors.go`

```go
package hook

// Formatter renders rule bodies for a specific editor's hook response.
// Implementations are stateless.
type Formatter interface {
    // FormatSessionStart returns the stdout payload for a SessionStart event.
    // An empty bodies slice should still produce a valid, consumable response
    // (empty string for Claude, "{}" for Gemini).
    FormatSessionStart(bodies []string) string

    // FormatPreToolUse returns the stdout payload for a PreToolUse event.
    FormatPreToolUse(bodies []string) string

    // FormatSessionEnd returns the stdout payload for a SessionEnd event.
    // Currently a no-op across all editors.
    FormatSessionEnd(bodies []string) string
}

var editors = map[Agent]Formatter{}

// RegisterEditor adds a formatter to the registry. Called from init() in
// each editor's file. Panics on duplicate registration (programmer error).
func RegisterEditor(id Agent, f Formatter) {
    if _, exists := editors[id]; exists {
        panic("hook: editor already registered: " + string(id))
    }
    editors[id] = f
}

// LookupEditor returns the formatter for id. Returns false if unregistered.
func LookupEditor(id Agent) (Formatter, bool) {
    f, ok := editors[id]
    return f, ok
}

// KnownEditors returns the registered editor IDs, sorted, for error messages.
func KnownEditors() []string { ... }
```

### New file: `internal/hook/editor_claude.go`

Claude formatter. Extracts the current logic from `agent.go` (`<system-reminder>` wrapping for SessionStart, `hookSpecificOutput` envelope for PreToolUse). `init()` calls `RegisterEditor(AgentClaude, claudeFormatter{})`.

### New file: `internal/hook/editor_gemini.go`

Gemini formatter. Both `SessionStart` and `PreToolUse` return:

```json
{"hookSpecificOutput":{"additionalContext":"<joined bodies>"}}
```

Empty bodies return `{}` (empty JSON object). `SessionEnd` returns empty string (no-op). `init()` calls `RegisterEditor(AgentGemini, geminiFormatter{})`.

Uses `encoding/json` to build the envelope (not string concat) so any rule body containing quote characters is escaped correctly. This is a correctness fix beyond what the old Claude path did — the old path used `json.Marshal` already, so no regression.

### Modified: `internal/hook/agent.go`

- Delete `FormatOutput` and `FormatPreToolOutput` free functions. Handler no longer calls them directly.
- Keep `Agent` type and `AgentClaude`/`AgentGemini` constants.
- Replace `DetectAgent()` with `ResolveEditor(flagValue string) (Agent, EditorSource, error)`:
  - If `flagValue != ""`: validate against `LookupEditor`; return `(Agent(flagValue), EditorSourceFlag, nil)` or error `unknown editor: <value> (valid: claude, gemini)`.
  - Else if `CLAUDE_CODE_ENTRYPOINT` set: return `(AgentClaude, EditorSourceEnv, nil)`.
  - Else: return `(AgentClaude, EditorSourceDefault, nil)`. *(Behavior change: today defaults to Gemini.)*
- Remove the `GEMINI_SESSION_ID` check entirely. No documented Gemini env var exists; keeping the check would be cargo-culted.

```go
type EditorSource string

const (
    EditorSourceFlag    EditorSource = "flag"
    EditorSourceEnv     EditorSource = "env"
    EditorSourceDefault EditorSource = "default"
)
```

### Modified: `internal/hook/handler.go`

`Handle()` stops calling `FormatOutput`/`FormatPreToolOutput`. It returns raw bodies in `HandleResult.Bodies []string`. CLI layer picks the formatter.

Rationale: keeping formatting out of the handler means adding an editor never touches handler code, satisfying the FR6 extension-point acceptance criterion.

`HandleResult` gains a `Bodies []string` field. The existing `Output string` field is retained for one step during migration, then removed. To minimize churn, we can drop `Output` in the same PR — the only non-test caller is `hook.go:91`.

Unrecognized `HookEventName` in the handler switch: return an error `hook: unrecognized event: <name>`. Today it falls through silently — this is a fail-loud fix per FR8.

### Modified: `internal/cli/hook.go`

```go
Flags: []cli.Flag{
    &cli.StringFlag{
        Name:  "event",
        Usage: "Hook event type: session-start, pre-tool-use, session-end",
    },
    &cli.StringFlag{
        Name:  "editor",
        Usage: "Target coding editor: claude, gemini (default: auto-detect via env, fallback claude)",
    },
},
```

`runHook` changes:

1. Parse `--editor` string.
2. `editor, source, err := hook.ResolveEditor(c.String("editor"))` — return error fast, before stdin read.
3. Decode stdin as today. `json.Decode` errors now surface with context `reading hook event from stdin: …`.
4. Normalize event name from `--event` (today's switch).
5. `handler := hook.NewHandler(event.CWD, homeDir, editor)`.
6. `result, err := handler.Handle(&event)` — error propagates (fail loud on unrecognized event).
7. `formatter, _ := hook.LookupEditor(editor)` — registry lookup is safe because `ResolveEditor` already validated.
8. Dispatch on `event.HookEventName` to call the right formatter method. This three-line switch stays at the CLI layer since `HookEventName` is already the canonical identifier.
9. `fmt.Print(output)` unchanged.
10. Audit sink writes `EditorSource: string(source)` in addition to `Agent`.

### Modified: `internal/hook/audit.go`

```go
type AuditRecord struct {
    // ... existing fields ...
    Agent        string `json:"agent"`
    EditorSource string `json:"editor_source,omitempty"` // NEW: flag|env|default
    // ...
}
```

`omitempty` keeps JSON backward-compatible for any offline reader that doesn't know the field.

### Modified: `README.md`

Find the existing Claude Code hooks section. Add a parallel subsection for Gemini CLI with a minimal `.gemini/settings.json` example:

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [{ "type": "command", "command": "zk hook --editor gemini --event session-start" }] }
    ],
    "BeforeTool": [
      {
        "matcher": ".*",
        "hooks": [{ "type": "command", "command": "zk hook --editor gemini --event pre-tool-use" }]
      }
    ]
  }
}
```

Note that Gemini's event name is `BeforeTool` in `settings.json` but the `zk hook --event` flag value stays `pre-tool-use` — the flag is zombiekit's canonical form, not the editor's.

## Test plan

Updates to `internal/hook/agent_test.go`:

- Rename detection tests for `ResolveEditor`:
  - `TestResolveEditor_Flag_Claude`
  - `TestResolveEditor_Flag_Gemini`
  - `TestResolveEditor_Flag_Unknown` (asserts error + unknown-editor message)
  - `TestResolveEditor_Env_Claude` (sets `CLAUDE_CODE_ENTRYPOINT`)
  - `TestResolveEditor_NoEnv_DefaultsClaude` (behavior change assertion)
  - Delete `TestDetectAgent_*Gemini*` tests that relied on `GEMINI_SESSION_ID`.
- Move formatter tests to new `editor_claude_test.go` and `editor_gemini_test.go`:
  - Claude tests assert `<system-reminder>` wrapping (SessionStart) and the existing envelope bytes (PreToolUse) — golden byte equivalence via existing fixtures.
  - Gemini tests assert `hookSpecificOutput.additionalContext` shape, empty-bodies → `{}`, SessionEnd → empty.
  - Use `encoding/json` Unmarshal in Gemini tests to assert structure (not string matching on serialization order).

Updates to `internal/hook/handler_test.go`:

- Change default test editor from `AgentGemini` to `AgentClaude`. Existing Claude-format assertions already use `AgentClaude`; the other tests only care about which rules matched, not formatting, so most tests just move to the new default.
- Add `TestHandler_UnrecognizedEvent_Errors`.
- Delete the two tests that relied on the old formatting living in the handler (they migrate to the editor_*_test.go files).

New `internal/cli/hook_test.go` (if it doesn't exist — if it does, extend it):

- End-to-end: feed stdin, assert stdout bytes for each (editor × event) combination.
- Assert that bad `--editor` exits before stdin is read (use a stdin that would panic on read).
- Assert audit record contains `EditorSource`.

## Ordered implementation steps

1. **Scaffolding** — Create `internal/hook/editors.go` with `Formatter` interface, registry, `LookupEditor`, `KnownEditors`. Add `EditorSource` type and constants.
2. **Claude formatter** — Create `editor_claude.go` by lifting `FormatOutput`/`FormatPreToolOutput` Claude branches verbatim into methods on `claudeFormatter`. Keep `<system-reminder>` wrapping and the existing JSON envelope byte-identical. `init()` registers it.
3. **Gemini formatter** — Create `editor_gemini.go` with correct JSON envelopes. `init()` registers it.
4. **ResolveEditor** — Replace `DetectAgent` in `agent.go`. Delete `GEMINI_SESSION_ID` check. Delete free-function formatters.
5. **Handler change** — `Handle()` returns `Bodies []string` on `HandleResult`; drop `Output`. Unrecognized event returns error.
6. **CLI wiring** — Add `--editor` flag. Call `ResolveEditor` before stdin read. Call `LookupEditor` + dispatch on event name after `Handle()`. Pass `editorSource` to audit sink.
7. **Audit record** — Add `EditorSource` field to `AuditRecord` (omitempty).
8. **Test migration** — Update existing tests, add new ones per the test plan above.
9. **README** — Add Gemini CLI hooks subsection with the minimal `settings.json` example and the editor-flag note.
10. **Manual smoke test** — Run `task dev -- lint:check` and `task dev -- test`. Manually invoke each of the six (editor × event) combinations with a crafted stdin payload and diff the stdout against the expected envelopes.

## Flagged uncertainties

- **Exact README anchor** for the new Gemini section depends on where Claude hooks are documented today. Recent commit `220b9cf docs(readme): document Claude Code hooks and rule injection` suggests the section already exists. Confirm during step 9; if it doesn't exist, we add a new top-level "Hooks" section covering both.
- **`AgentGemini` default in existing handler tests** — if any handler test secretly depends on the old Gemini-style plain-markdown output (not on matching logic), it will break during step 5. Not expected, but watch for it during the test migration.
- **`hook_log` subcommand** (`internal/cli/hook_log.go`) — presumably reads the audit log. If it pretty-prints fields, it may need a minor update to surface `EditorSource`. Out of scope for acceptance criteria; note and defer unless trivial.
