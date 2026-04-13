# Initiative: bash-hook-warnings

**Type**: feature
**Status**: in_progress
**Created**: 2026-04-12
**ID**: 69dc634f-feature-bash-hook-warnings

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-04-12 20:55 |
| plan | completed | 2026-04-12 21:10 |
| tasks | completed | 2026-04-12 21:20 |
| implement | completed | 2026-04-12 21:45 |

## Description

Extend the PreToolUse hook to inspect Bash tool invocations and inject
non-blocking warnings when the command matches an author-defined rule
(e.g. `go test` → use `task dev -- test`). Rules live in the existing
`.brains/rules/` markdown format with new `commands:` and optional
`requires_files:` frontmatter fields.

## Goals

- Ship command-matching as a first-class path alongside existing
  file-glob matching.
- Support a Taskfile-presence gate so rules only nudge when there is
  actually a better alternative on disk.
- No regression in existing file-rule behavior or tests.
- Keep authoring surface unified — one rules system, one frontmatter
  schema.

## Progress

- Spec and research drafts landed (2026-04-12).
- Plan and technical-spec drafted; no spikes needed.
- 17-task plan executed end-to-end; all tasks complete.

## Completion

**Completed**: 2026-04-12 21:50
**Duration**: ~80 minutes

### Outcomes

- Feature: bash-hook-warnings — Complete
  - `commands:` frontmatter field on rules
  - `requires_files:` and `requires_files_absent:` gates
  - `handlePreBash` branch in PreToolUse handler
  - Per-trigger dedup with legacy-state migration
  - `HandleResult` and `AuditRecord` carry `MatchedRule{ID, Trigger}` pairs
  - INFRASTRUCTURE.md rule-authoring docs

### Test Results

- `internal/rules` — ok (new command_matcher, gate, service tests)
- `internal/hook` — ok (8 new Bash handler tests + migration test)
- `internal/cli` — ok
- `internal/server` protobuf init panic is pre-existing, unrelated

### Notes

Minor scope additions beyond the original spec:
- `AuditRecord.Command` field so `hook log` surfaces the matched bash command.
- `formatMatchedRules` pretty-print helper rendering `id(trigger)` for command rules.
