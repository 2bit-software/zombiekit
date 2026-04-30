# Constraints

## Behavior That MUST NOT Change

1. **User-facing commands**: `/brains.new`, `/brains.next`, `/brains.complete` behave identically from the user's perspective
2. **Initiative lifecycle**: Create → step progression → complete flow is preserved
3. **Profile composition**: `profile-compose` continues to work with any profile name
4. **Resolution precedence**: Local > global > embedded for both workflows and profiles
5. **Branch management**: Initiative creation still creates branches
6. **Template copying**: spec.md and research.md templates still get copied on create
7. **Idempotent creation**: Creating with same name+type returns existing initiative

## Public Interfaces That Must Remain Stable

### MCP Tools (external API)
- `mcp__zombiekit__workflow-load` — parameters: name, type, working_directory
- `mcp__zombiekit__profile-compose` — parameters: profiles, working_directory
- `mcp__zombiekit__initiative` — parameters: action, dir, name, type, description

### Go Interfaces
- `step.Executor` — `Execute(stepName string, opts *ExecuteOptions) (*StepResponse, error)`
- `step.Resolver` — `GetStep(name string)`, `ListSteps()`
- `profile.ProfileSourceInterface` — Load/list/resolve profiles

### File Formats
- Existing INITIATIVE.md files must remain parseable (backwards compat)
- Profile files with `steps:` frontmatter must not break (graceful ignore)

## Scope Exclusions

- The `new` command markdown (`embed/commands/new.md`) — orchestration boilerplate stays in the workflow LLM prompt for now. Extracting it into the `new` command is a future optimization.
- The `feature-light` and `unmanaged` workflow types — these can adopt the new format but don't need special handling
- Removing the workflow body content — workflows keep their LLM instruction body. The new `steps:` frontmatter is additive.
