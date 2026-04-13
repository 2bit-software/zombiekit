# Tasks: Multi-Editor Hook Support

**Complexity**: Medium (~12 files — 6 new, 6 modified)
**Total tasks**: 14
**Parallel opportunities**: 5 tasks flagged `[P]`

## Task List

### Phase 1 — Registry and formatters (foundation)

- [ ] **T001** Create `internal/hook/editors.go` with:
  - `Formatter` interface (`FormatSessionStart`, `FormatPreToolUse`, `FormatSessionEnd`, all `([]string) string`)
  - `editors map[Agent]Formatter` package-level
  - `RegisterEditor(id Agent, f Formatter)` — panics on duplicate
  - `LookupEditor(id Agent) (Formatter, bool)`
  - `KnownEditors() []string` — returns sorted list of registered IDs for error messages
  - `EditorSource` string type with constants `EditorSourceFlag = "flag"`, `EditorSourceEnv = "env"`, `EditorSourceDefault = "default"`
  - **Acceptance**: file compiles; `go vet ./internal/hook/...` clean; no callers yet.

- [ ] **T002** [P] Create `internal/hook/editor_claude.go`:
  - Move `hookResponse` and `hookSpecificOutput` struct types from `agent.go` verbatim (unexported, local to this file)
  - Implement `claudeFormatter` with three methods:
    - `FormatSessionStart(bodies)` — joins bodies with `\n\n`, wraps in `<system-reminder>\n%s\n</system-reminder>`, empty bodies returns empty string (matches current `FormatOutput`)
    - `FormatPreToolUse(bodies)` — builds the existing `hookResponse` envelope, `json.Marshal`, empty bodies returns empty string (matches current `FormatPreToolOutput`)
    - `FormatSessionEnd(bodies)` — returns empty string
  - `init()` calls `RegisterEditor(AgentClaude, claudeFormatter{})`
  - **Acceptance**: compiles; behavior matches existing `FormatOutput`/`FormatPreToolOutput` byte-for-byte (guarded by T008).
  - **Depends on**: T001.

- [ ] **T003** [P] Create `internal/hook/editor_gemini.go`:
  - Define `geminiEnvelope` and `geminiHookOutput` as unexported struct types per `technical-spec.md` (with `omitempty` on `HookSpecificOutput`)
  - Implement `geminiFormatter` with three methods:
    - `FormatSessionStart(bodies)` and `FormatPreToolUse(bodies)`: if `len(bodies) == 0` marshal `geminiEnvelope{}` (yields `{}`); else marshal `geminiEnvelope{HookSpecificOutput: &geminiHookOutput{AdditionalContext: strings.Join(bodies, "\n\n")}}`
    - `FormatSessionEnd(bodies)` — returns empty string
  - `init()` calls `RegisterEditor(AgentGemini, geminiFormatter{})`
  - **Acceptance**: compiles; shape verified by T009.
  - **Depends on**: T001.

### Phase 2 — Glue, handler, CLI

- [ ] **T004** Modify `internal/hook/agent.go`:
  - Delete `DetectAgent`, `FormatOutput`, `FormatPreToolOutput`, `hookResponse`, `hookSpecificOutput` (the last two already moved in T002)
  - Add `ResolveEditor(flagValue string) (Agent, EditorSource, error)` per the flow in `implementation-plan.md` step 4 (registry-driven validation, no hardcoded list, reads `CLAUDE_CODE_ENTRYPOINT`, default `AgentClaude`)
  - File should now contain only `Agent` type, `AgentClaude`/`AgentGemini` constants, and `ResolveEditor`. If `agent.go` becomes thin, contents may be folded into `editors.go` — implementer's call.
  - **Acceptance**: `go build ./internal/hook/...` succeeds; `grep -n "DetectAgent\|FormatOutput\|FormatPreToolOutput" internal/hook/agent.go` returns nothing.
  - **Depends on**: T001, T002, T003.

- [ ] **T005** Modify `internal/hook/handler.go`:
  - Change `HandleResult.Output string` → `HandleResult.Bodies []string`
  - Remove all calls to `FormatOutput` / `FormatPreToolOutput` inside `Handle()`; return raw bodies instead
  - Handler switch `default` case: `return HandleResult{}, fmt.Errorf("hook: unrecognized event: %s", event.HookEventName)`
  - SessionEnd path returns `HandleResult{Bodies: nil}, nil`
  - **Acceptance**: `go build ./...` succeeds (T007 will update the one caller); `grep -n "AgentClaude\|AgentGemini" internal/hook/handler.go` returns zero lines (handler is editor-agnostic).
  - **Depends on**: T004.

- [ ] **T006** [P] Modify `internal/hook/audit.go`:
  - Add field `EditorSource string \`json:"editor_source,omitempty"\`` to `AuditRecord`
  - **Acceptance**: existing audit consumers still build; new field present.
  - **Depends on**: T001 (for `EditorSource` constant type, though the field is a plain string).

- [ ] **T007** Modify `internal/cli/hook.go`:
  - Add `&cli.StringFlag{Name: "editor", Usage: "Target coding editor: claude, gemini (default: auto-detect via env, fallback claude)"}` to the flags list
  - In `runHook`:
    1. Call `hook.ResolveEditor(c.String("editor"))` **before** stdin read; return error on failure
    2. Decode stdin (unchanged)
    3. Normalize event name from `--event` flag (unchanged)
    4. Build handler (unchanged)
    5. `result, err := handler.Handle(&event)` — propagate error
    6. `formatter, _ := hook.LookupEditor(editor)`
    7. Switch on `event.HookEventName` to call `formatter.FormatSessionStart`/`FormatPreToolUse`/`FormatSessionEnd` with `result.Bodies`
    8. `fmt.Print(output)`
    9. Pass `EditorSource: string(source)` into `AuditRecord`
  - Remove `hook.DetectAgent()` call
  - **Acceptance**: `go build ./...` succeeds; `zk hook --editor claude --event session-start < /dev/null` produces a valid error (malformed stdin) rather than panicking.
  - **Depends on**: T004, T005, T006.

### Phase 3 — Tests

- [ ] **T008** [P] Create `internal/hook/editor_claude_test.go`:
  - Migrate `TestFormatOutput_Claude` from `agent_test.go` → rename to `TestClaudeFormatter_SessionStart`. Replace the function call with `claudeFormatter{}.FormatSessionStart(...)`. Assertion strings unchanged.
  - Migrate `TestFormatPreToolOutput_Claude` from `agent_test.go` → rename to `TestClaudeFormatter_PreToolUse`. Same pattern.
  - Add `TestClaudeFormatter_SessionEnd` asserting empty string.
  - **Acceptance**: `go test ./internal/hook/ -run TestClaudeFormatter` passes.
  - **Depends on**: T002.

- [ ] **T009** [P] Create `internal/hook/editor_gemini_test.go`:
  - `TestGeminiFormatter_SessionStart`: call with two bodies, unmarshal output into `map[string]any`, assert `hookSpecificOutput.additionalContext` equals joined bodies; assert map has no `hookEventName` or `permissionDecision` keys.
  - `TestGeminiFormatter_PreToolUse`: same shape assertion.
  - `TestGeminiFormatter_EmptyBodies_SessionStart`: assert output equals exactly `{}` (2 bytes).
  - `TestGeminiFormatter_EmptyBodies_PreToolUse`: same.
  - `TestGeminiFormatter_SessionEnd`: assert empty string.
  - **Acceptance**: `go test ./internal/hook/ -run TestGeminiFormatter` passes.
  - **Depends on**: T003.

- [ ] **T010** Modify `internal/hook/agent_test.go`:
  - Delete the four `TestDetectAgent_*` tests.
  - Delete the four `TestFormatOutput_*` / `TestFormatPreToolOutput_*` tests (they are migrated to T008 for Claude; Gemini versions are obsolete — the new Gemini behavior is covered by T009).
  - Add `TestResolveEditor_Flag_Claude` (flag `claude`, asserts `(AgentClaude, EditorSourceFlag, nil)`).
  - Add `TestResolveEditor_Flag_Gemini`.
  - Add `TestResolveEditor_Flag_Unknown` — flag `frobnitz`, asserts error, asserts error string contains `unknown editor`, `claude`, and `gemini`.
  - Add `TestResolveEditor_Env_Claude` — set `CLAUDE_CODE_ENTRYPOINT=cli` via `t.Setenv`, empty flag, asserts `(AgentClaude, EditorSourceEnv, nil)`.
  - Add `TestResolveEditor_NoEnv_DefaultsClaude` — `t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")`, empty flag, asserts `(AgentClaude, EditorSourceDefault, nil)`.
  - File may become very thin; keep it anyway.
  - **Acceptance**: `go test ./internal/hook/ -run TestResolveEditor` passes; no test with `DetectAgent` in its name remains.
  - **Depends on**: T004, T008 (the Claude format tests must exist elsewhere before deleting here), T009.

- [ ] **T011** Modify `internal/hook/handler_test.go`:
  - Delete `TestHandler_ClaudeFormat_SessionStart` (originally around line 328).
  - Delete `TestHandler_ClaudeFormat_PreToolUse` (originally around line 511).
  - Update any test that asserted on `result.Output` to assert on `result.Bodies` instead. Most handler tests only check `MatchedRules`/`SkippedRules`, so this is usually zero-touch.
  - If any remaining test constructs `NewHandler` with `AgentGemini` for no formatting reason, switch it to `AgentClaude` (aligns with new default). Tests that specifically assert Gemini output no longer exist here — those moved to T009.
  - Add `TestHandler_UnrecognizedEvent_Errors`: construct a handler, pass `HookEvent{HookEventName: "BogusEvent"}`, assert returned error is non-nil and message contains `unrecognized event`.
  - **Acceptance**: `go test ./internal/hook/...` passes; the two deleted tests are gone.
  - **Depends on**: T005, T008.

- [ ] **T012** Create `internal/cli/hook_test.go` (new file):
  - `TestCLIHook_UnknownEditor_FailsBeforeStdinRead`: replace `os.Stdin` with a reader that panics on `Read`; invoke with `--editor frobnitz`; assert error returned, no panic. (Use `t.Cleanup` to restore stdin.)
  - `TestCLIHook_ClaudeEvent_SessionStart`: pipe canned JSON, assert stdout starts with `<system-reminder>`.
  - `TestCLIHook_GeminiEvent_SessionStart`: pipe canned JSON, unmarshal stdout, assert `hookSpecificOutput.additionalContext` present.
  - `TestCLIHook_AuditRecord_EditorSourceFlag`: inject a fake sink (pattern: make `newHookAuditSink` overridable via a package var or pass a sink into `runHook`), invoke with `--editor claude`, assert the captured record has `EditorSource == "flag"`.
  - `TestCLIHook_AuditRecord_EditorSourceDefault`: same pattern, no flag, `t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")`, assert `EditorSource == "default"`.
  - `TestCLIHook_UnrecognizedEvent_FailsLoud`: pipe JSON whose `hook_event_name` is `Bogus`, omit `--event` flag (or route around the CLI normalization), assert error + stderr contains `unrecognized event`.
  - **Acceptance**: `go test ./internal/cli/ -run TestCLIHook` passes.
  - **Note**: If the current `runHook` structure makes sink injection awkward, extract a `runHookWithDeps(c, stdin io.Reader, sink hook.AuditSink) error` helper and have `runHook` call it. This is a mechanical refactor, not a scope expansion.
  - **Depends on**: T007.

### Phase 4 — Docs and verification

- [ ] **T013** [P] Modify `README.md`:
  - Find the existing Claude Code hooks section (added in commit `220b9cf`).
  - Add a parallel Gemini CLI subsection with a minimal `.gemini/settings.json` block per `technical-spec.md`.
  - Include the note: Gemini's `BeforeTool` event maps to `zk hook --event pre-tool-use` at the zombiekit layer (zombiekit's `--event` values are canonical, not editor-specific).
  - **Acceptance**: README renders; both editor configs present side-by-side.
  - **Depends on**: nothing — can land any time.

- [ ] **T014** Verification:
  - Run `task dev -- lint:check` and `task dev -- test`.
  - Execute the manual smoke matrix from `implementation-plan.md` step 10 (8 rows: all editor × event combinations + empty-bodies + unknown-editor).
  - Run `grep -n "AgentClaude\|AgentGemini" internal/hook/handler.go` — must return zero lines (AC13, extension point).
  - Spot-check `internal/cli/hook_log.go`: if it pretty-prints `AuditRecord` fields individually, add one line for `EditorSource`. If it passes the struct to an encoder, no change needed.
  - **Acceptance**: all commands green, smoke matrix all rows match expected output, grep returns zero lines.
  - **Depends on**: T001–T013.

## Dependency graph

```
T001 ──┬─▶ T002 [P] ──┬─▶ T004 ──▶ T005 ──┐
       ├─▶ T003 [P] ──┘                   │
       └─▶ T006 [P] ───────────────────▶  T007 ──▶ T012
                                          │
             T002 ─▶ T008 [P] ────────▶ T010 ─▶ T011 ─▶ T014
             T003 ─▶ T009 [P] ────────▶ T010
                                                  T013 [P] (parallel any time)
```

**Critical path**: T001 → T002/T003 → T004 → T005 → T007 → T012 → T014
**Parallel fan-out after T001**: T002, T003, T006 can run concurrently
**Parallel fan-out after formatters**: T008, T009 can run concurrently with each other, and with T006/T013

## Traceability

| Spec requirement | Tasks |
|---|---|
| FR1 (editor flag) | T007 |
| FR2 (Claude parity) | T002, T008, T011, T014 |
| FR3 (Gemini correct) | T003, T009, T014 |
| FR4 (canonical event) | T005, T007 |
| FR5 (default Claude) | T004, T010 |
| FR6 (extension point) | T001, T002, T003, T004, T014 (grep) |
| FR7 (audit log) | T006, T007, T012 |
| FR8 (fail loud) | T004, T005, T007, T010, T011, T012 |
| FR9 (docs) | T013 |
| AC1–AC2 (Claude bytes) | T008, T011 |
| AC3–AC5 (Gemini shape) | T009, T012 |
| AC6 (Gemini empty bodies) | T009 |
| AC7 (unknown editor) | T010, T012 |
| AC8 (malformed stdin) | T012 |
| AC9 (unrecognized event) | T011, T012 |
| AC10–AC11 (env/default) | T010, T012 |
| AC12 (audit fields) | T006, T012 |
| AC13 (extension point) | T001, T014 |
| AC14 (README) | T013 |

Every FR and AC maps to at least one task. No orphan tasks.

## Execution order

Suggested:

1. **T001** (foundation)
2. **T002, T003, T006** in parallel
3. **T004** (requires formatters registered)
4. **T005** (handler change)
5. **T008, T009** in parallel (formatter tests)
6. **T010, T011** (test migration — T010 depends on formatters existing elsewhere)
7. **T007** (CLI wiring — requires T005 and T006)
8. **T012** (CLI tests)
9. **T013** any time (parallel)
10. **T014** (verification)

Run `/brains.implement` to begin implementation.
