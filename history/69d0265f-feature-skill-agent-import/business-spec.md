# Business Spec: Skill/Agent Import

## Description

Import existing Claude Code skills and agents into the zombiekit profile system. This allows users to migrate their Claude-native skills and agents into zombiekit's composable profile infrastructure, gaining inheritance, composition, and centralized management.

## Functional Requirements

### Discovery

- FR-1: List all Claude Code skills available for import (from `~/.claude/skills/` and `.claude/skills/`)
- FR-2: List all Claude Code agents available for import (from `~/.claude/agents/` and `.claude/agents/`)
- FR-3: Resolve symlinks when discovering source files
- FR-4: Exclude skills that are already zombiekit shims (body contains `mcp__zombiekit__profile-compose`)
- FR-5: Show the user which items are available, with name and description

### Import Destination

- FR-6: User chooses destination scope: **local** (`.brains/profiles/`) or **global** (`~/.brains/profiles/`)
- FR-7: If a profile with the same name already exists at the destination, warn and ask to overwrite or rename

### Skill Import

- FR-8: Copy the skill's `SKILL.md` content into a new zombiekit profile directory
- FR-9: Copy all files and subdirectories from the skill directory (except `SKILL.md` itself) into the profile directory. This includes scripts/, assets, and any nested content.
- FR-10: Transform frontmatter: strip `allowed-tools`, preserve `name` and `description`
- FR-11: Preserve the body/prompt content verbatim

### Agent Import

- FR-12: Convert the agent's `.md` file into a zombiekit profile directory with `SKILL.md`
- FR-13: Transform frontmatter: preserve `name` and `description`. Strip `model`, `skills`, `memory`, `color`.
- FR-14: Preserve the body/system-prompt content verbatim
- FR-15: If the agent references skills by name (in `skills:` frontmatter), include a comment at the top of the body listing them as potential `includes:` candidates (e.g., `<!-- Referenced skills: skill1, skill2 — consider adding as includes -->`). Do not auto-resolve.

### Shim Generation (Optional)

- FR-16: After import, ask the user if they want to keep a shim in the original Claude location
- FR-17: If yes, replace the original file with a shim that delegates to `mcp__zombiekit__profile-compose` (same pattern as `skill-install`)
- FR-18: For skills: write shim to `~/.claude/skills/<name>/SKILL.md` (or local equivalent)
- FR-19: For agents: write shim to `~/.claude/agents/<name>.md` with the full original frontmatter preserved (`model`, `color`, `skills`, `memory`) plus `allowed-tools: mcp__zombiekit__profile-compose`, body replaced with compose delegation
- FR-20: If no shim requested, leave the original file untouched (user manages removal themselves)

### Batch Operations

- FR-21: Support importing multiple skills/agents in a single operation
- FR-22: Batch imports use a single scope for all items (passed as a tool argument). Individual overrides are not supported.

### Error Handling

- FR-23: If a source file has invalid/missing frontmatter, skip it with a warning in the response (do not fail the entire import)
- FR-24: If a source path is a broken symlink, skip it with a warning
- FR-25: If a skill and agent share the same name, warn and ask the user to provide an alternate name for one of them

## Acceptance Criteria

- A Claude skill with supporting scripts is imported into `~/.brains/profiles/<name>/` with all files intact
- A Claude agent is imported into a profile directory with correct frontmatter transformation
- An existing zombiekit shim skill is excluded from the import list
- A shim replaces the original skill file and correctly delegates to `profile-compose`
- An agent shim preserves the original `model` field so Claude Code routes to the correct model
- Importing to a scope where the profile already exists prompts for overwrite/rename
- The imported profile loads successfully via `profile-compose` (returns content without error)
- Batch import of 3 skills produces 3 profile directories
- A source file with missing frontmatter is skipped with a warning, other items still import

## MCP Tool Interface

**Tool name**: `skill-import`

**Arguments**:
- `names` (required, array of strings): Names of skills/agents to import
- `scope` (required, `"local"` | `"global"`): Destination scope
- `shim` (optional, boolean, default `false`): Whether to write shims in the original Claude locations
- `working_directory` (optional, string): Working directory for local scope resolution

**Returns**: JSON object with:
- `imported` (array): Successfully imported items with `{name, type, path}`
- `skipped` (array): Skipped items with `{name, reason}`
- `shims` (array): Created shims with `{name, path}` (if `shim: true`)

**Discovery tool**: A separate `skill-import-list` tool lists available items for import (skills + agents with name, type, description, source path).

## Out of Scope

- Automatic migration of `allowed-tools` into zombiekit permission system
- Importing Claude Code commands (these are workflow entry points, not portable skills)
- Importing Claude Code rules (these are project-scoped, not profile-shaped)
- Two-way sync between Claude files and zombiekit profiles
- Automatic detection of skill dependencies beyond `skills:` frontmatter

## Resolved Questions

1. **Agent shim format**: Agent shims preserve the full original frontmatter (`model`, `color`, `skills`, `memory`) plus `allowed-tools: mcp__zombiekit__profile-compose`. Body is replaced with compose delegation.
2. **Naming conflicts**: Warn and ask the user to rename one of them. No auto-prefixing.
3. **CLI vs MCP**: MCP tool only. Matches existing `skill-install` pattern.
4. **Agent model in profile**: Model is NOT preserved in the zombiekit profile — only in the shim (which is what Claude Code reads for routing).
