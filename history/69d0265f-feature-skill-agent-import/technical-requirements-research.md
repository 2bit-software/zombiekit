# Technical Requirements & Research

## User's Technical Preferences

- Reuse existing `skill-install` shim generation pattern
- Support both skills and agents
- Import into local or global `.brains/profiles/` based on user choice
- Optionally keep a shim in the Claude location pointing to `profile-compose`

## Implementation Constraints

### Existing Code to Reuse

- `internal/skill/install.go` — `GenerateContent()` for shim generation, `WriteSkill()` for file writing, `ValidateName()` for name validation
- `internal/profile/service.go` — `Write()` for saving profiles with atomic writes
- `internal/profile/resolver.go` — `FindProfileDirs()` for discovering existing profiles (collision detection)
- `internal/profile/skill_loader.go` — `ParseFrontmatter()` for reading source files

### Claude File Locations

```
Skills:
  Global: ~/.claude/skills/<name>/SKILL.md  (may be symlink)
  Local:  .claude/skills/<name>/SKILL.md

Agents:
  Global: ~/.claude/agents/<name>.md  (may be symlink)
  Local:  .claude/agents/<name>.md  (no project-level agents observed)
```

### Frontmatter Transformation Rules

**Skill → Profile:**
```yaml
# Source (Claude skill)          # Target (zombiekit profile)
name: commit                  →  name: commit
description: ...              →  description: ...
allowed-tools: Bash(...)      →  (dropped)
                                 type: action  (default)
```

**Agent → Profile:**
```yaml
# Source (Claude agent)          # Target (zombiekit profile)
name: repo-architect          →  name: repo-architect
description: ...              →  description: ...
model: sonnet                 →  (dropped)
color: cyan                   →  (dropped)
skills: skill1,skill2         →  (HTML comment in body listing as includes candidates)
memory: user                  →  (dropped)
```

### Shim Templates

**Skill shim** (same as existing `skill-install`):
```yaml
---
name: {name}
description: >
  {description}
allowed-tools: mcp__zombiekit__profile-compose
---

Call `mcp__zombiekit__profile-compose` with `profiles: ["{name}"]` and follow the returned instructions exactly.
```

**Agent shim** (preserves model for Claude Code routing):
```yaml
---
name: {name}
description: >
  {description}
model: {original-model}
allowed-tools: mcp__zombiekit__profile-compose
---

Call `mcp__zombiekit__profile-compose` with `profiles: ["{name}"]` and follow the returned instructions exactly.
```

### Exposure

The feature should be an **MCP tool** (`skill-import`) to match the existing `skill-install` pattern. A CLI command can wrap it later.
