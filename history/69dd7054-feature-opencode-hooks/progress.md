# Progress

## T001 — AgentOpenCode const
- Status: Complete
- Files: `internal/hook/types.go`

## T002 — SessionState.GraphiteInjected
- Status: Complete
- Files: `internal/rules/types.go`
- Notes: JSON tag `graphite_injected,omitempty`; omitempty so legacy
  state files round-trip cleanly.

## T003 — ResetInjectedRules clears GraphiteInjected
- Status: Complete
- Files: `internal/hook/session.go`

## T004 — handleSessionStart inject branch + graphite gate
- Status: Complete
- Files: `internal/hook/handler.go`
- Notes: Wrapped `ResetInjectedRules` and compaction-count bookkeeping
  in `if event.Source != "inject"`. Added dedup check to the
  unconditional rules loop (previously unconditional) so the inject
  path dedups correctly on repeat calls. Graphite append is gated by
  `!state.GraphiteInjected`.

## T005 — CLI session-inject / compact events
- Status: Complete
- Files: `internal/cli/hook.go`

## T006 — editor_opencode.go
- Status: Complete
- Files: `internal/hook/editor_opencode.go`

## T007 — editor_opencode_test.go
- Status: Complete
- Files: `internal/hook/editor_opencode_test.go`
- Notes: 11 test functions covering envelope shape, empty-bodies,
  no-op methods, path extraction for all three tools + camelCase +
  multi-edit fallback + cross-editor name isolation, shell tool.

## T008 — handler regression tests
- Status: Complete
- Files: `internal/hook/handler_test.go`
- Notes: 4 new tests. `TestHandler_Compact_AfterInject_ReInjects` is
  the direct regression test for the issue flagged during spec
  review. `TestHandler_SessionInject_DoesNotClobberExistingDedup`
  verifies the inject branch doesn't corrupt PreToolUse dedup.

## T009 — agent resolution test
- Status: Complete
- Files: `internal/hook/agent_test.go`

## T010 — brains.ts shim
- Status: Complete
- Files: `embed/integrations/opencode/brains.ts`
- Notes: Single-file plugin. Three hooks. BRAINS_BIN env var.
  Append-only on `output.system`. Error catching per hook. Startup
  stderr log on first invocation.

## T011 — embed.go
- Status: Complete
- Files: `embed.go`

## T012 — README
- Status: Complete
- Files: `README.md`
- Notes: OpenCode section parallel to Gemini CLI section, including
  shim copy instructions, opencode.json registration snippet,
  BRAINS_BIN dev-loop explanation, and the experimental-hook /
  OPENCODE_PURE caveats.

## T013 — Manual E2E
- Status: PENDING — user-driven
- Notes: Handoff to user. Awaiting stop-OpenCode signal before
  installing `brains-test` binary.

## Side cleanup

- `internal/hook/editor_gemini_test.go:97` had a pre-existing
  compile error (`Success: true` passed to `*bool` field). Fixed as
  an unblock; not part of the OpenCode feature scope.

## Verification

- `task dev -- test` on `internal/hook/...`: 75.7% coverage, all
  tests pass including new ones.
- `task dev -- vet`: clean.
- `task dev -- build`: clean binary.
- `task dev -- lint`: only pre-existing warnings in
  `internal/mcp/tools/initiative/tool.go` and
  `internal/orchestrator/router.go` — no new warnings in
  `internal/hook`, `internal/rules`, `internal/cli`, or `embed.go`.
- Pre-existing `TestRun_ReconciliationRuns` failure in
  `internal/orchestrator` confirmed pre-existing on `main` via git
  stash — not caused by this feature.
