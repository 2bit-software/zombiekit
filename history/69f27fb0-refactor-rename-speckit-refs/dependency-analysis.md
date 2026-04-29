# Dependency Analysis

## Affected Files

| File | Occurrences | Reference Type |
|------|-------------|----------------|
| `embed/templates/plan-template.md` | 7 | `/speckit.plan`, `/speckit.tasks`, `.specify/` path |
| `embed/templates/checklist-template.md` | 2 | `/speckit.checklist` |
| `embed/templates/tasks-template.md` | 1 | `/speckit.tasks` |
| `embed/profiles/commit-message.md` | 4 | `speckit` as tool name |

**Total**: 4 files, 14 occurrences

## How These Files Are Consumed

All four files live under `embed/` and are compiled into the Go binary via `//go:embed`. They are served as template content when workflows create new spec artifacts. The content is documentation/commentary within the templates — not executable references. Changing the text does not affect any code paths.

## Downstream Impact

- None. These are human-readable notes inside template files. No Go code parses or acts on the speckit strings.
- The commit-message profile references `speckit` in its tooling detection guidance. Updating this ensures the LLM looks for `zombiekit`/`brains` instead.

## Command Mapping

| Old (speckit) | New (brains/zombiekit) |
|---------------|----------------------|
| `/speckit.plan` | `/brains.next` (plan step) |
| `/speckit.tasks` | `/brains.next` (tasks step) |
| `/speckit.checklist` | `/brains.next` (checklist step) |
| `.specify/templates/commands/` | `embed/workflows/` (loaded via MCP) |
| `speckit` (tool name) | `zombiekit` / `brains` |
