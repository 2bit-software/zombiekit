# Reuse Audit

Performed inline against code already read during planning
(`handler.go`, `editors.go`, `editor_claude.go`, `editor_gemini.go`,
`types.go`, `session.go`, `agent.go`, `embed.go`, `cli/hook.go`). No
dispatched agent — the plan is small and its reuse surface fits
entirely inside one package.

## Summary

- Duplicates: 0
- Overlaps: 2 (0 extend, 2 create new)
- Related: 3 (noted for consistency)
- No match: 6

## Findings

### OVERLAP

#### `marshalOpencodeEnvelope` vs `marshalGeminiEnvelope`

- **Existing**: `internal/hook/editor_gemini.go:66-79`
- **Similarity**: Structurally identical — both build a JSON object,
  place rule bodies at `hookSpecificOutput.additionalContext`, and fall
  back to `{}` on empty/error. They differ in one field: Gemini's
  envelope includes `"decision": "allow"` (required by the Gemini CLI
  protocol); OpenCode's does not (no decision semantic on
  `tool.execute.after` or `experimental.chat.system.transform`).
- **Decision**: Create new. Do **not** extract a shared helper.
- **Rationale**: Parameterizing the Gemini marshaller would mean either
  (a) adding an `includeDecision bool` flag the Gemini call site would
  always set true, or (b) introducing a builder/option struct for a
  15-line function. Both add indirection for no payoff. The envelopes
  belong to their respective editor contracts; the similarity is
  incidental. Keep them separate and accept the minor duplication — the
  same pattern already exists between `editor_claude.go`'s
  `claudeHookResponse` and Gemini's envelope.
- **Plan change**: None — plan already specifies a new
  `marshalOpencodeEnvelope`.

#### `opencodeFormatter.ExtractFilePaths` vs
`claudeFormatter.ExtractFilePaths` / `geminiFormatter.ExtractFilePaths`

- **Existing**: `internal/hook/editor_claude.go:58-86`,
  `internal/hook/editor_gemini.go:43-59`
- **Similarity**: All three iterate tool names and pull a file path off
  `ToolInput` / `ToolResponse`. The *structure* repeats per editor; the
  *vocabulary* (tool names) is editor-specific.
- **Decision**: Create new.
- **Rationale**: The editor abstraction exists precisely to localize
  tool-name vocabulary. Folding all three into a shared helper would
  require passing a tool-name map, which is exactly what the per-editor
  implementation already is. This is intended duplication; the
  `Editor` interface codifies it.
- **Plan change**: None.

### RELATED

#### `AgentOpenCode` const

- **Related code**: `internal/hook/types.go:57-60` (`AgentClaude`,
  `AgentGemini`)
- **Note**: Name follows the existing `Agent{Name}` pattern. String
  value `"opencode"` (lowercase, matching `"claude"` and `"gemini"`).

#### Session-inject handler behavior

- **Related code**: `internal/hook/handler.go:52-92`
  (`handleSessionStart`). The existing dedup-gated loop at
  lines 74-78 already produces the correct "first call injects,
  subsequent dedups" behavior when invoked without a preceding
  `ResetInjectedRules`.
- **Note**: No new handler function is needed — the plan's
  `Source == "inject"` branch reuses the existing loop. This is
  cheaper than my first instinct to add a `handleSessionInject`.

#### `GraphiteInjected` vs existing `CompactionCount`

- **Related code**: `internal/rules/SessionState.CompactionCount` (the
  only existing counter on session state — see
  `internal/hook/handler.go:61-64` and
  `internal/hook/session.go:97`).
- **Note**: New field lives alongside `CompactionCount` in the same
  struct; JSON key `graphite_injected` follows snake_case
  convention of the existing fields. Reset semantics (cleared inside
  `ResetInjectedRules`) parallel how `InjectedRules` is cleared.

### NONE

These items have no existing equivalent and proceed as planned:

1. OpenCode CLI `--event session-inject` / `--event compact` cases in
   `internal/cli/hook.go`.
2. `internal/hook/editor_opencode.go` (new file).
3. `embed/integrations/opencode/brains.ts` (new file).
4. `embed.go` directive for the shim.
5. `editor_opencode_test.go` (new test file).
6. README "OpenCode" section.

## Plan Changes

None. The plan already correctly identifies where existing code is
extended (handler, CLI switch, session state struct, embed.go) versus
where new files are created (opencode editor, shim). The reuse audit
confirms this split and formally records that the envelope and
path-extraction duplication with Gemini/Claude is intentional, not
oversight.
