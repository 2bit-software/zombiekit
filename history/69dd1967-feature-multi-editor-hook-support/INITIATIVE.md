# Initiative: multi-editor-hook-support

**Type**: feature
**Status**: in_progress
**Created**: 2026-04-13
**ID**: 69dd1967-feature-multi-editor-hook-support

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-04-13 10:15 |
| plan | completed | 2026-04-13 10:45 |
| tasks | completed | 2026-04-13 10:55 |
| implement | completed | 2026-04-13 11:30 |

## Description

<!-- Add a description of this initiative -->

## Goals

<!-- Define the goals for this initiative -->

## Progress

<!-- Track progress here -->

## Completion

**Completed**: 2026-04-13 11:45
**Duration**: same-day

### Outcomes

- Feature: multi-editor-hook-support — Complete
  - `--editor <claude|gemini>` flag on `zk hook`
  - New `internal/hook/editors.go` formatter registry; `editor_claude.go` lifts existing Claude behavior verbatim; `editor_gemini.go` fixes the previously-broken Gemini output path (was emitting plain markdown; now emits `{"hookSpecificOutput":{"additionalContext":"..."}}`)
  - Handler is now editor-agnostic (`HandleResult.Bodies []string`); unrecognized events fail loud
  - `AuditRecord.EditorSource` enum field (`flag|env|default`, `omitempty` for backward compatibility)
  - Default editor flipped from Gemini to Claude; `GEMINI_SESSION_ID` env sniffing removed (no documented signal)
  - README extended with a parallel Gemini `settings.json` example

### Verification

- Build + vet clean
- `internal/hook` and `internal/cli` tests pass uncached
- Smoke matrix: Claude + Gemini × SessionStart / SessionEnd / unknown-editor all behave per spec
- Extension-point check: `grep AgentClaude\|AgentGemini internal/hook/handler.go` returns zero lines

### Notes

Pre-existing test failures in `internal/orchestrator` and `internal/server` confirmed unrelated (reproduced on main via `git stash`).

