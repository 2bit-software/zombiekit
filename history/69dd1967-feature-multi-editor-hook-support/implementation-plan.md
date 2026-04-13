# Implementation Plan

Traces directly to `business-spec.md` FRs and ACs. Detailed design in `technical-spec.md`.

## Steps

### 1. Editor registry scaffolding
**Files**: `internal/hook/editors.go` (new)
**Spec**: FR6
**Changes**: Define `Formatter` interface, `RegisterEditor`, `LookupEditor`, `KnownEditors`, `EditorSource` type with constants `EditorSourceFlag`, `EditorSourceEnv`, `EditorSourceDefault`.
**Depends on**: nothing.

### 2. Claude formatter (lift existing behavior)
**Files**: `internal/hook/editor_claude.go` (new)
**Spec**: FR2
**Changes**: Implement `claudeFormatter` with three methods. Bodies joined with `\n\n`. SessionStart wraps in `<system-reminder>`. PreToolUse emits the existing `hookResponse`/`hookSpecificOutput` envelope using `json.Marshal` to preserve byte output. SessionEnd returns empty string (explicit: `func (claudeFormatter) FormatSessionEnd([]string) string { return "" }`). `init()` calls `RegisterEditor(AgentClaude, claudeFormatter{})`. Move the `hookResponse` and `hookSpecificOutput` types into this file as unexported locals; they are not used anywhere else.
**Depends on**: step 1.
**Byte-equivalence guard**: the existing tests `TestFormatOutput_Claude` and `TestFormatPreToolOutput_Claude` in `internal/hook/agent_test.go` assert exact byte output of the Claude format. These tests are the contract — they migrate to `editor_claude_test.go` verbatim (renamed to `TestClaudeFormatter_SessionStart` / `TestClaudeFormatter_PreToolUse`) and must continue to pass without modifying any assertion strings. If either test fails during migration, step 2 is not done.

### 3. Gemini formatter (new, correct)
**Files**: `internal/hook/editor_gemini.go` (new)
**Spec**: FR3
**Changes**: Implement `geminiFormatter` with three methods. Bodies joined with `\n\n`. SessionStart and PreToolUse both marshal the exact same struct shape:

```go
type geminiEnvelope struct {
    HookSpecificOutput *geminiHookOutput `json:"hookSpecificOutput,omitempty"`
}
type geminiHookOutput struct {
    AdditionalContext string `json:"additionalContext"`
}
```

With bodies present: marshal `geminiEnvelope{HookSpecificOutput: &geminiHookOutput{AdditionalContext: joined}}` — produces `{"hookSpecificOutput":{"additionalContext":"..."}}`. With empty bodies: marshal `geminiEnvelope{}` — produces `{}` (exactly two bytes). SessionEnd returns empty string (no-op). `init()` calls `RegisterEditor(AgentGemini, geminiFormatter{})`.
**Depends on**: step 1.

### 4. Replace DetectAgent with ResolveEditor
**Files**: `internal/hook/agent.go` (modified)
**Spec**: FR1, FR5, FR8
**Changes**:
- New `ResolveEditor(flagValue string) (Agent, EditorSource, error)` in package `hook` (same package as `editors.go`, so call sites use `LookupEditor` and `KnownEditors` unqualified — no `hook.` prefix).
- Validation flow:
  1. If `flagValue != ""`: call `_, ok := LookupEditor(Agent(flagValue))`. If `!ok`: return `Agent(""), "", fmt.Errorf("unknown editor: %s (valid: %s)", flagValue, strings.Join(KnownEditors(), ", "))`. If `ok`: return `Agent(flagValue), EditorSourceFlag, nil`.
  2. Else if `os.Getenv("CLAUDE_CODE_ENTRYPOINT") != ""`: return `AgentClaude, EditorSourceEnv, nil`.
  3. Else: return `AgentClaude, EditorSourceDefault, nil`.
- **Validation is registry-driven** — no hardcoded `{claude, gemini}` list. Adding a new editor in the future is one `RegisterEditor` call; `ResolveEditor` automatically accepts it.
- Delete `DetectAgent`, `FormatOutput`, `FormatPreToolOutput`, and the `hookResponse`/`hookSpecificOutput` types (which have moved to `editor_claude.go` in step 2).
- Delete the `GEMINI_SESSION_ID` env-var check entirely — no documented signal exists.
**Depends on**: steps 1–3 (registry populated before `ResolveEditor` is callable; init-order within package `hook` ensures `init()` in `editor_claude.go`/`editor_gemini.go` runs before any test or caller invokes `ResolveEditor`).

### 5. Handler returns bodies, fails loud on unknown event
**Files**: `internal/hook/handler.go`, `internal/hook/handler_test.go`
**Spec**: FR4, FR8
**Changes**:
- `HandleResult.Output string` → `HandleResult.Bodies []string`.
- Handler no longer calls formatters. Returns `Bodies` from rule resolution.
- Switch `default` case returns `fmt.Errorf("hook: unrecognized event: %s", event.HookEventName)`.
- SessionEnd path returns `Bodies: nil, nil`.
**Depends on**: steps 1–4.
**Migration note**: only caller of `HandleResult.Output` is `internal/cli/hook.go:91` — updated in step 6.

### 6. CLI flag + formatter dispatch
**Files**: `internal/cli/hook.go`
**Spec**: FR1, FR4, FR5, FR8
**Changes**:
- Add `--editor` string flag.
- Call `ResolveEditor(c.String("editor"))` before stdin read; propagate error.
- Decode stdin (as today).
- Normalize event name from `--event` flag (as today).
- Build handler (as today).
- `handler.Handle(&event)` → propagate error.
- `formatter, _ := hook.LookupEditor(editor)`.
- Three-line switch on canonical event name → `formatter.FormatSessionStart`/`FormatPreToolUse`/`FormatSessionEnd`.
- Pass `editorSource` into audit record.
**Depends on**: steps 1–5.

### 7. Audit record field
**Files**: `internal/hook/audit.go` (modify), `internal/hook/filesink.go` (inspect — update only if it enumerates fields rather than passing `AuditRecord` to an encoder)
**Spec**: FR7
**Changes**: Add `EditorSource string \`json:"editor_source,omitempty"\`` to `AuditRecord`.
**Must land before or with step 6**, because step 6 writes `EditorSource` into the audit record. Re-check dependency graph — step 7 is a prerequisite of step 6, not a successor.
**Backward compatibility**: `omitempty` means older consumers see unchanged output when the feature is inactive; new consumers see the new field. No version bump needed.

### 8. Test migration
**Files**: `internal/hook/agent_test.go` (modify), `internal/hook/editor_claude_test.go` (new), `internal/hook/editor_gemini_test.go` (new), `internal/hook/handler_test.go` (modify), `internal/cli/hook_test.go` (new — file does not exist today)
**Spec**: AC1–AC14
**Changes**:

In `internal/hook/agent_test.go`:
- Delete `TestDetectAgent_Claude`, `TestDetectAgent_Gemini`, `TestDetectAgent_BothSet_ClaudeWins`, `TestDetectAgent_NeitherSet_DefaultsGemini`.
- Move `TestFormatOutput_Claude`, `TestFormatOutput_Gemini`, `TestFormatPreToolOutput_Claude`, `TestFormatPreToolOutput_Gemini` out of this file (see below).
- Add `TestResolveEditor_Flag_Claude`, `TestResolveEditor_Flag_Gemini`, `TestResolveEditor_Flag_Unknown` (asserts error message contains `unknown editor` and both valid editor IDs), `TestResolveEditor_Env_Claude` (sets `CLAUDE_CODE_ENTRYPOINT`), `TestResolveEditor_NoEnv_DefaultsClaude` (unsets env, asserts source is `EditorSourceDefault`).

In `internal/hook/editor_claude_test.go` (new):
- Migrate `TestFormatOutput_Claude` → `TestClaudeFormatter_SessionStart` with identical assertion bytes.
- Migrate `TestFormatPreToolOutput_Claude` → `TestClaudeFormatter_PreToolUse` with identical assertion bytes.
- Add `TestClaudeFormatter_SessionEnd` asserting empty string.

In `internal/hook/editor_gemini_test.go` (new):
- `TestGeminiFormatter_SessionStart`: marshal bodies, unmarshal stdout, assert `hookSpecificOutput.additionalContext == joined bodies` and no `hookEventName` or `permissionDecision` fields present (unmarshal into a map and assert key absence).
- `TestGeminiFormatter_PreToolUse`: same shape assertion.
- `TestGeminiFormatter_EmptyBodies_SessionStart`: golden bytes `{}` (exact two-byte comparison).
- `TestGeminiFormatter_EmptyBodies_PreToolUse`: golden bytes `{}`.
- `TestGeminiFormatter_SessionEnd`: empty string.

In `internal/hook/handler_test.go`:
- Delete `TestHandler_ClaudeFormat_SessionStart` (line 328) — the behavior it asserts now lives in `editor_claude_test.go`.
- Delete `TestHandler_ClaudeFormat_PreToolUse` (line 511) — same reason.
- Change any remaining test that constructs a handler with `AgentGemini` to use `AgentClaude` if the test does not assert on output format. Handler tests should not care about editor after this refactor; they care about matched/skipped rules on `HandleResult.Bodies`.
- Add `TestHandler_UnrecognizedEvent_Errors` asserting that `Handle()` returns a non-nil error for a made-up `HookEventName`.

In `internal/cli/hook_test.go` (new file — create):
- `TestCLIHook_UnknownEditor_FailsBeforeStdinRead`: invoke `runHook` with `--editor frobnitz` and a `os.Stdin` replacement that panics on `Read`. Assert error returned, no panic.
- `TestCLIHook_ClaudeEventMapping_SessionStart`: pipe a canned JSON event, assert stdout matches the expected Claude SessionStart bytes end-to-end.
- `TestCLIHook_GeminiEventMapping_SessionStart`: same, for Gemini — assert shape via unmarshal.
- `TestCLIHook_AuditRecord_EditorSourceFlag`: invoke with `--editor claude`, intercept the audit sink, assert `EditorSource == "flag"`.
- `TestCLIHook_AuditRecord_EditorSourceDefault`: invoke without flag and with no env var, assert `EditorSource == "default"`.
- `TestCLIHook_UnrecognizedEvent_FailsLoud`: pipe a JSON event whose `hook_event_name` is unknown (use `--event` to bypass the CLI-side normalization, or drop `--event` so the stdin `hook_event_name` reaches the handler switch), assert non-zero error + stderr contains `unrecognized event`.

**Depends on**: steps 1–7.

### 9. README update
**Files**: `README.md`
**Spec**: FR9, AC14
**Changes**: Locate the existing Claude Code hooks section (per commit `220b9cf`). Add a sibling Gemini CLI subsection with the minimal `.gemini/settings.json` example from `technical-spec.md`. Include the note that Gemini's `BeforeTool` hook event maps to `zk hook --event pre-tool-use` at the zombiekit layer.
**Depends on**: nothing (doc change can land in parallel).

### 10. Verification
**Commands**:
- `task dev -- lint:check`
- `task dev -- test`
- **Manual smoke matrix** — pipe a minimal JSON event into `go run ./cmd/zk hook --editor <e> --event <ev>` and diff stdout against expected bytes for each row:

  | # | editor | event | expected |
  |---|--------|-------|----------|
  | 1 | claude | session-start | `<system-reminder>`-wrapped bodies |
  | 2 | claude | pre-tool-use | Claude envelope with `hookEventName`+`permissionDecision`+`additionalContext` |
  | 3 | claude | session-end | empty stdout, exit 0 |
  | 4 | gemini | session-start | `{"hookSpecificOutput":{"additionalContext":"..."}}` |
  | 5 | gemini | pre-tool-use | same Gemini envelope |
  | 6 | gemini | session-end | empty stdout, exit 0 |
  | 7 | gemini | session-start with zero matching rules | `{}` (two bytes) |
  | 8 | frobnitz | session-start | non-zero exit, stderr contains `unknown editor` |

- **Extension-point sanity check** (covers AC13): after implementation, run a grep to confirm no handler code references editor IDs directly:
  `grep -n "AgentClaude\|AgentGemini" internal/hook/handler.go` should return zero lines. The handler must be editor-agnostic after this refactor.

**Spec**: task-completion criteria in CLAUDE.md; AC3–AC9 and AC13.

## Dependency graph

```
1 registry ──┬─▶ 2 claude formatter ──┐
             ├─▶ 3 gemini formatter ──┤
             ├─▶ 4 resolve editor  ◀──┘
             └─▶ 7 audit field ──────┐
                                     ▼
                 4 ──▶ 5 handler ──▶ 6 CLI ──▶ 8 tests ──▶ 10 verify
9 README (parallel, any time)
```

Step 7 is an explicit prerequisite of step 6 because step 6 writes `EditorSource` into the audit record.

## Risk / scope callouts

- **Byte-equivalence for Claude** — the only safe way to lift is verbatim copy of the format strings and `json.Marshal` call. Existing Claude-format tests serve as the guard; they must pass without modification (except file-path updates where the tests move).
- **Test file moves** — moving assertions from `agent_test.go` to per-editor test files is mechanical but noisy in the diff. Worth keeping in one commit for reviewability.
- **`hook_log` subcommand** — if `internal/cli/hook_log.go` pretty-prints `AuditRecord` fields, surfacing `EditorSource` is a trivial add. Check during step 7 and include if a one-line change; otherwise defer.

## Spec → step traceability

| Spec | Steps |
|------|-------|
| FR1 (editor flag) | 4, 6 |
| FR2 (Claude parity) | 2, 8 |
| FR3 (Gemini correct) | 3, 6, 8 |
| FR4 (canonical event) | 5, 6 |
| FR5 (default Claude) | 4, 8 |
| FR6 (extension point) | 1, 2, 3, 4 |
| FR7 (audit log) | 7, 6, 8 |
| FR8 (fail loud) | 4, 5, 6, 8 |
| FR9 (docs) | 9 |

## Remaining uncertainties

None material. Flagged low-risk items above. Ready for reuse-audit → final audit.
