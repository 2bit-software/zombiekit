# Tasks

Complexity: **Medium** (9 Go files touched, 1 new TS file, 1 README
section). Single task list.

## Parallel Opportunity Map

- T001 and T004 can run in parallel (both are tiny type-level additions
  with no overlap).
- T009 (README) is independent of everything and can run any time.
- T007 (shim) must wait for T003 (CLI accepts `session-inject` /
  `compact`) and T005 (editor registered) — otherwise the shim would
  talk to a binary that rejects its events.
- T008 (embed) must wait for T007 (file to embed must exist).
- T010 (manual E2E) is the final gate and is user-driven.

## Tasks

- [ ] **T001 [P]** Add `AgentOpenCode Agent = "opencode"` to the const
  block in `internal/hook/types.go` (between `AgentGemini` and the
  closing paren). Verify `go build ./...` still compiles.
  **Acceptance:** `AgentOpenCode` is referenced from `editor_opencode.go`
  in T005 without import cycles or redeclaration errors.

- [ ] **T002 [P]** Add `GraphiteInjected bool` field to the
  `rules.SessionState` struct. First locate the struct:
  `grep -rn "type SessionState" internal/rules/`. Add the field with
  JSON tag `graphite_injected`, mirroring the existing struct's JSON
  casing convention. No other changes in this task.
  **Acceptance:** `go build ./...` passes; existing
  `internal/hook/session_test.go` tests still pass (legacy state files
  load with the field defaulting to `false`).

- [ ] **T003** Extend `ResetInjectedRules` in
  `internal/hook/session.go:95-98` to also zero `state.GraphiteInjected`.
  Depends on T002.
  **Acceptance:** Unit test in T006a (handler tests) will assert
  graphite re-fires after a compaction reset.

- [ ] **T004** In `internal/hook/handler.go::handleSessionStart`
  (lines 52-92), wrap the `ResetInjectedRules(state)` call AND the
  subsequent `if event.Source != "compact"` compaction-count
  decrement in a single outer `if event.Source != "inject"` block.
  Then wrap the existing graphite append at lines 84-86 in
  `if !state.GraphiteInjected { ...; state.GraphiteInjected = true }`.
  Depends on T002, T003.
  **Acceptance:** Existing Claude/Gemini SessionStart tests in
  `internal/hook/handler_test.go` still pass (these cover the
  `startup`, `resume`, `compact` source cases; `inject` is new).

- [ ] **T005** Extend `internal/cli/hook.go::runHook` `--event` switch
  (lines 60-71) with two new cases:
  - `case "session-inject":` sets `HookEventName = "SessionStart"` and
    `event.Source = "inject"`.
  - `case "compact":` sets `HookEventName = "SessionStart"` and
    `event.Source = "compact"`.
  Also update the `--event` flag `Usage` string (line 22) to list all
  six canonical event values. Depends on T004 (handler must understand
  `Source="inject"` before the CLI starts passing it).
  **Acceptance:** `brains hook --editor claude --event session-inject`
  (with a minimal JSON on stdin) exits 0; same for `--event compact`.

- [ ] **T006** Create `internal/hook/editor_opencode.go` with the
  `opencodeFormatter` type as specified in `technical-spec.md`
  ("`internal/hook/editor_opencode.go` (new)" section). Include
  `init()` registration via `RegisterEditor(AgentOpenCode, ...)`,
  all four Format* methods (SessionStart and PostToolUse emit the
  envelope; PreToolUse and SessionEnd return `""`),
  `ExtractFilePaths` switching on `write`/`edit`/`multi-edit`,
  `IsShellTool` returning true for `bash`, plus the envelope struct
  types and `marshalOpencodeEnvelope` helper. Depends on T001.
  **Acceptance:** `go build ./...` passes; the editor registers
  without panic at package init.

- [ ] **T007 [P]** Create `internal/hook/editor_opencode_test.go`
  mirroring `editor_gemini_test.go`'s structure. Cases:
  - `FormatSessionStart` with one body → valid JSON with
    `hookSpecificOutput.additionalContext` set; without bodies → `{}`.
  - `FormatPostToolUse` same.
  - `FormatPreToolUse([...])` returns `""`.
  - `FormatSessionEnd([...])` returns `""`.
  - `ExtractFilePaths` for `write`, `edit`, `multi-edit` with
    `ToolInput.FilePath` set returns that path; with unknown tool
    name returns nil.
  - `IsShellTool("bash")` true, `IsShellTool("Bash")` false,
    `IsShellTool("run_shell_command")` false.
  Depends on T006.
  **Acceptance:** `go test ./internal/hook/...` passes the new cases.

- [ ] **T008** Add handler regression tests to
  `internal/hook/handler_test.go`:
  - `TestHandleSessionInject_InjectsOnFirstCall`: `Source="inject"`,
    unconditional rule in the rules dir → body appears in
    `HandleResult.Bodies`; `GraphiteInjected` flips to true.
  - `TestHandleSessionInject_DedupsOnSecondCall`: call twice,
    second call's `Bodies` is empty.
  - `TestHandleCompact_AfterInject_ReInjects`: call
    `session-inject`, then call with `Source="compact"`, verify
    unconditional rules fire again and `GraphiteInjected` resets
    then re-fires.
  - `TestHandleSessionInject_GraphiteOnce`: graphite appears on
    first `inject` call, not on second.
  Depends on T004.
  **Acceptance:** all four tests pass; existing handler tests unchanged.

- [ ] **T009 [P]** Add `internal/hook/agent_test.go` case asserting
  `ResolveEditor("opencode")` returns `(AgentOpenCode,
  EditorSourceFlag, nil)`. Depends on T001 and T006 (registration must
  happen before `ResolveEditor` can validate the flag value).
  **Acceptance:** test passes.

- [ ] **T010** Create `embed/integrations/opencode/brains.ts` with the
  shim code from `technical-spec.md` ("The Shim" section). Use
  `process.env.BRAINS_BIN ?? "brains"` for the binary path, wrap all
  spawn logic in try/catch with `console.error` on failure, and
  use append-only mutation on `output.system` and `output.context`.
  Subscribes to three hooks: `experimental.chat.system.transform`,
  `experimental.session.compacting`, `tool.execute.after`.
  Depends on T005, T006 (brains must accept the new events and the
  `opencode` editor must be registered).
  **Acceptance:** file typechecks with `bunx tsc --noEmit
  embed/integrations/opencode/brains.ts` if a Bun toolchain is
  available; otherwise manual syntax review.

- [ ] **T011** Extend `embed.go` at repo root with a new
  `//go:embed embed/integrations/opencode/brains.ts` directive,
  `embeddedOpencodeShim embed.FS` variable, `EmbeddedOpencodeShim
  fs.FS` exported variable, and the corresponding `fs.Sub` wiring in
  `init()`. Follow the exact pattern of `embeddedClaudeCommands`
  (embed.go:12-14, 31, 44-47). Depends on T010 (file must exist at
  build time).
  **Acceptance:** `go build ./...` passes; a trivial test that
  `fs.ReadFile(EmbeddedOpencodeShim, "integrations/opencode/brains.ts")`
  succeeds (add to an existing embed test or create a new
  `embed_test.go` in the repo root).

- [ ] **T012 [P]** Update `README.md` with an "OpenCode" section
  parallel to the existing Claude Code and Gemini CLI sections.
  Cover: (a) path to the shim in the zombiekit repo, (b) how to copy
  it to `.opencode/plugins/brains.ts`, (c) registration in
  `opencode.json` if auto-discovery is not in play, (d) the
  `BRAINS_BIN` environment variable and its role in dev/swap
  workflows, (e) the `OPENCODE_PURE=1` caveat, (f) a stability
  warning about the `experimental.*` hook names potentially
  renaming upstream. Independent of all Go tasks.
  **Acceptance:** section is present and internally consistent;
  no code block references a path that doesn't exist after T010.

- [ ] **T013** Manual E2E with the user. Steps (documented in
  `technical-spec.md` "Manual E2E"):
  1. Agent runs `go build -o $GOPATH/bin/brains-test
     ./cmd/brains` after the user confirms OpenCode is stopped.
  2. User copies `embed/integrations/opencode/brains.ts` into
     `.opencode/plugins/brains.ts` in a test project.
  3. User starts OpenCode with `BRAINS_BIN=brains-test` in its
     environment.
  4. User creates a `.brains/rules/test-unconditional.md` with no
     `paths` field; starts a new session; verifies the rule body
     appears in the first-turn system prompt.
  5. User creates a `.brains/rules/test-go.md` with
     `paths: ["**/*.go"]`; has the agent edit a `.go` file;
     verifies the rule body appears appended to the tool result
     the model sees next.
  6. User triggers compaction; verifies the unconditional rule
     still appears in the compacted context.
  7. User edits a non-matching file; verifies no injection.
  8. User reports pass/fail for each step.

  On failure: agent diagnoses, fixes the Go or shim code, rebuilds
  `brains-test`, user restarts OpenCode, re-runs the failed step.
  Depends on T001-T012.
  **Acceptance:** all seven verification steps pass in real
  OpenCode.

## Traceability

| Spec criterion | Task(s) |
|----------------|---------|
| `--event post-tool-use` reads payload, writes JSON | T005, T006 |
| Matching rule → response body | T006, T008 |
| No matching rule → `{}` no-op | T006, T007 |
| Session dedup, no double-injection | T008 |
| Compaction preserves unconditional rules | T008, T013 |
| Shim reads `BRAINS_BIN` | T010 |
| Existing Claude/Gemini tests still pass | T004, T008 (regression) |
| OpenCode editor unit tests | T007 |
| README documents setup | T012 |
| Real-OpenCode E2E | T013 |

## Critical Path

T001 → T006 → T007
T002 → T003 → T004 → T005 → T010 → T011 → T013
                       ↘ T008 ↗

Longest chain: T002 → T003 → T004 → T005 → T010 → T011 → T013 (7 steps).

## Parallelization

Round 1 (no dependencies): T001, T002, T009 (docs can start any time),
T012 (README).
Round 2: T003 (needs T002).
Round 3: T004 (needs T003), T006 (needs T001).
Round 4: T005 (needs T004), T007 (needs T006), T008 (needs T004).
Round 5: T010 (needs T005, T006).
Round 6: T011 (needs T010).
Round 7: T013 (needs everything) — user-driven.
