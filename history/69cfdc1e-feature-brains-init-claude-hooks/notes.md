# Notes: brains init --claude hooks

## What
When `brains init --claude` is passed (local or global), also add `brains hook --event` entries to the `hooks` section of the Claude `settings.json`. Three hooks:
- `SessionStart` → `brains hook --event session-start`
- `PreToolUse` (matcher: `Read|Write|Edit|MultiEdit`) → `brains hook --event pre-tool-use`
- `SessionEnd` → `brains hook --event session-end`

## Constraints
- Idempotent: if the hook entries already exist, skip them (noop on second run)
- Preserve all existing hooks — never overwrite user-customized entries
- Works for both `--global` (writes to `~/.claude/settings.json`) and local (writes to `.claude/settings.json`)

## Acceptance Criteria
- [ ] Running `brains init --claude` adds all three hook entries to settings.json
- [ ] Running it again is a noop (no duplicates)
- [ ] Existing hooks in the same event category are preserved
- [ ] Tests cover fresh install, idempotent re-run, and preservation of existing hooks
