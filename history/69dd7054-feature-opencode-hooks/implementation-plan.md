# Implementation Plan

Ordered step list. Each step is independently testable. Steps 1–6 are
pure Go and should land as a single cohesive set before step 7 (the
shim) since the shim depends on the CLI accepting `session-inject` /
`compact` events and the `opencode` editor being registered.

## Step 1 — Session state: add `GraphiteInjected`

**File:** `internal/rules/service.go` (or wherever `SessionState` is
defined — verify during task generation).

- Add `GraphiteInjected bool` field with JSON tag `graphite_injected`.
- Backward compatible: missing field defaults to false, which matches
  existing behavior on first load.

**File:** `internal/hook/session.go`

- Update `ResetInjectedRules` to also zero `GraphiteInjected`.

**Tests:** `internal/hook/session_test.go` — verify a legacy state file
without the field loads cleanly and the field defaults to false.

## Step 2 — Handler: `inject` source branch + graphite gate

**File:** `internal/hook/handler.go::handleSessionStart`

- Wrap the existing `ResetInjectedRules(state)` call in
  `if event.Source != "inject"`, and move the CompactionCount decrement
  inside the same block.
- After the unconditional-rules loop, wrap the graphite append in
  `if !state.GraphiteInjected` and set the flag on append.

**Tests:** `internal/hook/handler_test.go`

- `Source="inject"` on first call: injects unconditional rules + graphite.
- `Source="inject"` on second call: empty result (dedup).
- `Source="compact"` after two inject calls: re-fires everything.
- Existing Claude SessionStart tests still pass (no source change).

## Step 3 — CLI: `session-inject` and `compact` events

**File:** `internal/cli/hook.go`

- Extend `--event` usage string.
- Add `case "session-inject"` and `case "compact"` to the switch that
  sets `event.HookEventName` and `event.Source`.

**Tests:** if `internal/cli/hook_test.go` exists, add coverage; if not,
defer to integration coverage via handler_test.

## Step 4 — Types: register `AgentOpenCode`

**File:** `internal/hook/types.go`

- Add `AgentOpenCode Agent = "opencode"` to the const block.

## Step 5 — Editor: `internal/hook/editor_opencode.go`

- Create the file with the formatter, `ExtractFilePaths`, and
  `IsShellTool` as shown in the technical spec.
- Register in `init()`.

**Tests:** `internal/hook/editor_opencode_test.go` mirroring
`editor_gemini_test.go` structure.

## Step 6 — Agent resolution test

**File:** `internal/hook/agent_test.go`

- Add a case asserting `ResolveEditor("opencode")` returns
  `AgentOpenCode` with `EditorSourceFlag`.

## Step 7 — Shim: `embed/integrations/opencode/brains.ts`

- Create the directory and the `.ts` file as shown in the technical spec.
- Keep it minimal: three hooks, one helper, one tool-name dispatch.
- Add a one-line stderr log on first invocation of each hook so the
  user can confirm registration in OpenCode's plugin log.

No automated tests — coverage is via the Go side + manual E2E.

## Step 8 — Embed the shim

**File:** `embed.go`

- Add `//go:embed embed/integrations/opencode/brains.ts` directive and
  `embeddedOpencodeShim` variable.
- Add `EmbeddedOpencodeShim fs.FS` and wire it in `init()`.

**Tests:** add an assertion in an existing embed test (or create a
trivial one) that `EmbeddedOpencodeShim` can be stat'd and read.

## Step 9 — README

- Add an OpenCode section parallel to Claude Code and Gemini CLI
  sections, covering: where the shim lives in the repo, how to copy
  it into `.opencode/plugins/brains.ts`, how to configure
  `opencode.json` if auto-discovery isn't in play, the `BRAINS_BIN`
  environment variable, the `OPENCODE_PURE=1` caveat, and the
  `experimental.*` hook stability warning.

## Step 10 — Manual E2E (user-driven)

The user performs the manual test flow from the technical spec:

1. Install `brains-test` binary.
2. Stop OpenCode.
3. Copy shim into `.opencode/plugins/`.
4. Restart OpenCode with `BRAINS_BIN=brains-test`.
5. Validate unconditional rule injection on first turn.
6. Validate file-glob rule injection on edit.
7. Validate unconditional rule survives compaction.
8. Report results.

On any failure, iterate: fix the code, rebuild `brains-test`, have the
user stop/restart OpenCode, re-test.

## Traceability to Spec

| Acceptance criterion | Covered by |
|----------------------|------------|
| `--event post-tool-use` reads payload and writes JSON | Step 3, 5 |
| Matching rule → response contains body | Step 5, handler_test |
| No matching rule → no-op shape | Step 5 |
| Session dedup (no double-injection) | Step 2 handler test |
| Compaction preserves unconditional rules | Step 2 handler test + Step 10 manual |
| Shim reads `BRAINS_BIN` | Step 7 |
| Existing Claude/Gemini tests pass | Step 2 regression check |
| OpenCode editor tests exist | Step 5 |
| README documents OpenCode setup | Step 9 |

## Uncertainties Flagged for Tasks

1. Verify exact `output.args` field name (`filePath` vs `file_path`) in
   the OpenCode tool schema at the commit we're targeting. If wrong,
   only the shim's `extractFilePath` changes.
2. Verify `SessionState` struct location (likely `internal/rules/`).
3. Confirm `EditEntry` + multi-edit path extraction matches OpenCode's
   actual payload shape — if OpenCode supplies one `filePath` per
   multi-edit rather than an `edits[]` array, the editor's
   `ExtractFilePaths` is simpler than the claude equivalent.
