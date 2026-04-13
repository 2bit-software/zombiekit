# Feature Specification: Bash Hook Command Warnings

**Feature Branch**: `69dc634f-feature-bash-hook-warnings`
**Created**: 2026-04-12
**Status**: Draft

## Summary

Extend the PreToolUse hook to inspect Bash tool invocations, match the
command against user-authored rules, and inject non-blocking warnings that
recommend a better alternative (e.g. "run `task dev -- test` instead of
`go test`"). Rules are authored in the existing `.brains/rules/` markdown
format with a new `commands:` frontmatter field and an optional
`requires_files:` gate.

## User Scenarios

### US1 — Warn on `go test`, suggest Taskfile alternative (P1)

The user runs `go test ./...` in a project that has a `Taskfile.yml` with a
`dev -- test` target. The PreToolUse hook detects `go test`, confirms the
Taskfile exists, and injects a non-blocking warning telling the model to use
`task dev -- test` instead. The model sees the warning, aborts the bash
call, and reissues via task.

**Independent test**: author a rule, fire a synthetic PreToolUse event with
`tool_name: "Bash"` and `command: "go test ./..."`, assert the rule body
appears in `additionalContext` and `permissionDecision` stays `allow`.

### US2 — Silent pass-through when command does not match (P1)

The user runs `ls`, `git status`, `cat foo.txt`. No rule matches. The hook
emits empty output and the tool runs normally.

### US3 — Taskfile-gated rule stays silent when no Taskfile exists (P2)

Same rule as US1, but the project has no `Taskfile.yml`. The
`requires_files:` gate fails, the rule does not fire, the command runs
normally. (Optional follow-up: a second rule keyed on the absence of
Taskfile could nudge the user to create one — explicitly out of scope for
v1.)

### US4 — Rule already injected earlier in session is not re-injected (P2)

Dedup via `SessionState.InjectedRules` already covers this for file-glob
rules; command-matched rules follow the same path.

### Edge cases

- Command contains the matcher as a substring of an unrelated word
  (`gopher test-helper`) — must not match. Matching uses whole-token
  prefix semantics, not raw substring.
- Chained commands (`cd foo && go test`) — matcher scans each
  `&&`/`;`/`||`-separated segment independently.
- Command has leading env vars (`CGO_ENABLED=0 go test ./...`) — env
  assignments are stripped before matching.
- Rule file declares both `paths:` and `commands:` — rule fires for
  whichever hook path is active (file-tool or Bash), never both at once.
- Unknown tool name in `PreToolUse` — fall through to no-op.

## Requirements

### Functional

- **FR-001**: The PreToolUse handler MUST branch on `tool_name`. For
  `Bash`, it extracts `tool_input.command` (string) and resolves rules
  via a new command-matching path.
- **FR-002**: `RuleFrontmatter` MUST accept a `commands:` field — list of
  strings, each a command prefix matcher (e.g. `"go test"`,
  `"npm install"`).
- **FR-003**: Matching MUST treat each matcher as a whole-token prefix
  against each segment of the command (split on `&&`, `;`, `||`, `|`).
  Leading `VAR=value` assignments are stripped before comparison.
- **FR-004**: `RuleFrontmatter` MUST accept an optional `requires_files:`
  field — list of file paths resolved by walking from `event.cwd` up to
  the repo root. If set, ALL listed files must exist for the rule to
  fire.
- **FR-004b**: `RuleFrontmatter` MUST accept an optional
  `requires_files_absent:` field — symmetrical inversion of
  `requires_files`. ALL listed files must resolve to NOT existing for
  the rule to fire. Both fields may be set on the same rule; both
  gates must pass.
- **FR-004c**: Gate evaluation MUST run before the command matcher.
  Order per event: (1) requires_files all-exist, (2)
  requires_files_absent all-missing, (3) command matcher.
- **FR-004d**: Per-trigger dedup MUST key on `ruleID + "|" + trigger`
  for command-matched rules. File-glob rules keep bare-`ruleID` dedup.
  `SessionState.InjectedRules` stores the composite key; legacy entries
  load as `ruleID|""` for back-compat.
- **FR-004e**: `HandleResult` MUST carry `(ruleID, trigger)` pairs in
  place of bare rule IDs so the audit sink records which trigger
  fired.
- **FR-005**: A rule with `commands:` but no `paths:` MUST NOT be treated
  as unconditional (i.e. it must not fire at SessionStart).
- **FR-006**: Matched rule bodies are injected via the existing
  `FormatPreToolOutput` envelope with `permissionDecision: "allow"` —
  warnings remain non-blocking.
- **FR-007**: Command-matched rules participate in the existing
  `SessionState.InjectedRules` dedup — a rule fires at most once per
  session.
- **FR-008**: Audit records (the uncommitted `AuditSink` work) MUST log
  matched/skipped command-rule IDs the same way file-rule IDs are
  logged today.

### Out of scope (v1)

- Regex or glob patterns in `commands:` — prefix-only for now.
- Blocking (`permissionDecision: "deny"`) — all warnings stay
  non-blocking.
- Parsing subshells, backticks, `$()` expansions.
- Modifying the command the user ran — we only inject advice text.

## Success Criteria

- **SC-001**: Authoring a command rule and triggering it end-to-end
  (synthetic `PreToolUse` event through the `hook` CLI) produces the
  expected `additionalContext` JSON.
- **SC-002**: A chained command (`cd x && go test`) with a `go test`
  rule fires once.
- **SC-003**: The same session triggering the same rule twice injects
  it only on the first hit (dedup).
- **SC-004**: A rule gated on `Taskfile.yml` fires in repos that have
  one and stays silent in repos that don't.
- **SC-005**: No regression in existing file-glob rule tests.

## Test Strategy

Integration tests in `internal/hook/handler_test.go` mirroring existing
patterns: temp git dir, write a rule under `.brains/rules/`, construct a
`HookEvent` with `ToolName: "Bash"` and `ToolInput.Command`, call
`Handler.Handle`, assert on `HandleResult.Output` and
`MatchedRuleIDs`. Unit tests for the command tokenizer (env-strip +
segment split + prefix match) as a pure function.

## Open Questions

1. Should `commands:` matchers be exact prefix tokens (`"go test"`
   matches `go test ./...` but not `go testthings`) — confirmed yes, but
   should we also support a trailing `*` for explicit wildcarding later?
2. Should `requires_files:` support globs (e.g. `Taskfile*.yml`) or
   exact paths only? Exact is simpler; glob is one-liner via doublestar.
3. Where does `requires_files:` resolve from — event `cwd`, or walk up
   to repo root? Probably `cwd` first, then repo root.
4. Do we ship a default seed rule set (go test, npm test, mix test,
   docker compose) or leave authoring to the user? Recommend: no
   defaults, document examples in INFRASTRUCTURE.md.
