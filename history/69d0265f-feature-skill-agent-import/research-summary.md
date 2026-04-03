# Research Summary: Skill/Agent Import

## Existing Architecture

### Claude Code File Formats

| Type | Location | Frontmatter Fields | Body |
|------|----------|-------------------|------|
| **Skill** | `~/.claude/skills/<name>/SKILL.md` | `name`, `description`, `allowed-tools` | Instructions |
| **Agent** | `~/.claude/agents/<name>.md` | `name`, `description`, `model`, `skills`, `memory`, `color` | System prompt |
| **Command** | `~/.claude/commands/<name>.md` | `description` | Invocation instructions |

### Zombiekit Profile System

- **Profile locations** (precedence order): local `.brains/profiles/` > parent dirs > global `~/.brains/profiles/` > embedded
- **Profile format**: `SKILL.md` in a named directory, or `<name>.md` flat file
- **Frontmatter fields**: `name`, `description`, `includes`, `inherits`, `type`, `model`, `color`
- **Skill directories** can contain supporting files (scripts/, etc.)

### Skill-Install Flow (existing)

1. `skill-install` MCP tool receives `name` and `scope` (local/global)
2. Loads the named profile via `profile.NewService().Show(name)`
3. Generates a shim `SKILL.md` with:
   - `allowed-tools: mcp__zombiekit__profile-compose`
   - Body: `Call mcp__zombiekit__profile-compose with profiles: ["<name>"]`
4. Writes to `~/.claude/skills/<name>/SKILL.md` (global) or `.claude/skills/<name>/SKILL.md` (local)

### Key Code Locations

| Component | Path |
|-----------|------|
| Skill install logic | `internal/skill/install.go` |
| Skill install MCP tool | `internal/mcp/tools/skillinstall/tool.go` |
| Profile service | `internal/profile/service.go` |
| Profile resolver | `internal/profile/resolver.go` |
| Skill loader | `internal/profile/skill_loader.go` |
| Profile-compose MCP tool | `internal/mcp/tools/profile/tool.go` |
| Profile-save MCP tool | `internal/mcp/tools/profile/tool.go` |
| MCP server registration | `internal/mcp/server.go` |

## Key Findings

### Skills vs Agents: Structural Differences

- **Skills** live in directories with `SKILL.md` + optional supporting files
- **Agents** are single `.md` files with richer frontmatter (`model`, `skills`, `memory`, `color`)
- Agents have no directory structure — just a flat file

### Import Mapping

| Claude Source | Zombiekit Target |
|---------------|-----------------|
| Skill (`SKILL.md` in dir) | Profile directory with `SKILL.md` |
| Agent (`.md` file) | Profile directory with `SKILL.md` |

Both map to the same target format — a zombiekit profile. The key transformation is:
- Strip Claude-specific frontmatter (`allowed-tools`, `model`, `skills`, `memory`)
- Add/preserve zombiekit frontmatter (`name`, `description`, `type`, `includes`)
- Copy supporting files (scripts, etc.) for skills

### Shim Generation (reverse of skill-install)

The existing `skill-install` creates shims that point Claude skills → zombiekit profiles. The import feature needs:
1. Copy content INTO a zombiekit profile
2. Optionally create a shim in the original Claude location pointing back to the new profile

### Symlink Consideration

Both `~/.claude/skills` and `~/.claude/agents` may be symlinks (e.g., to `~/Projects/personal/ai/claude/skills/`). The import tool should resolve symlinks to find actual source files.

### Profile Name Validation

Profile names must match: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$` (lowercase alphanumeric with hyphens).
