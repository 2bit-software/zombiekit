# Constraints

## Behavior That MUST NOT Change

1. **Template structure**: The templates must retain their existing structure, sections, and placeholder patterns
2. **Template purpose**: Each template still serves the same role (plan, tasks, checklist, commit message)
3. **Embedded file paths**: Templates are embedded Go files — the file paths in `embed/` must not change
4. **History files untouched**: The ~50 `history/specs/` files are historical artifacts and must not be modified

## Public Interfaces That Must Remain Stable

- `embed/templates/plan-template.md` — consumed by Go embed
- `embed/templates/checklist-template.md` — consumed by Go embed
- `embed/templates/tasks-template.md` — consumed by Go embed
- `embed/profiles/commit-message.md` — consumed by Go embed

## Scope Exclusions

- `spec-creator` references in commit-message.md are a separate tool and not in scope
- No changes to Go source code — this is markdown-only
