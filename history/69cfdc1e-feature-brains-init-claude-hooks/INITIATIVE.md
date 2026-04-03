# Initiative: brains-init-claude-hooks

**Type**: feature
**Status**: completed
**Created**: 2026-04-03
**ID**: 69cfdc1e-feature-brains-init-claude-hooks

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-04-03 08:26 |
| plan | completed | 2026-04-03 08:27 |
| tasks | completed | 2026-04-03 08:27 |
| implement | completed | 2026-04-03 08:30 |

## Description

Add `brains hook --event` entries to Claude's `settings.json` when `brains init --claude` is run, making hook installation part of the standard init flow.

## Completion

**Completed**: 2026-04-03
**Duration**: ~10 minutes

### Outcomes
- Feature: ensureHooks() function — Complete
- Feature: hookEntryExists() idempotency check — Complete
- Feature: Wired into both initLocal and initGlobal — Complete
- Tests: Fresh install, idempotent re-run, preserves existing hooks — Complete
