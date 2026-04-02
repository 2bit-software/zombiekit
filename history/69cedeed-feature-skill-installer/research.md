---
status: complete
updated: 2026-04-02
---

# Research: Skill Installer

## Executive Summary

The zombiekit codebase uses urfave/cli/v2 for CLI commands and a custom MCP server with a consistent tool pattern. Both extension points are well-understood. The Claude Code skills system expects a directory-per-skill layout with a `SKILL.md` frontmatter file; the body of that file is the minimal delegating pattern already in use (call profile-compose, follow instructions). The new feature plugs neatly into both extension points with no architectural surprises.

## Findings

### Codebase Context

**CLI framework**: `github.com/urfave/cli/v2` — NOT Cobra. Commands are defined as `*cli.Command` structs with `Name`, `Flags`, `Subcommands`, and `Action func(*cli.Context) error`. New commands are registered in `internal/cli/root.go` via the `Commands` slice in `NewApp()`.

**Existing skill-adjacent CLI command**: `internal/cli/profile.go` — `newProfileCommand()` is a parent with subcommands (compose, list, show, create, validate, import). The skill installer should follow the same pattern: a `skill` parent command with an `install` subcommand.

**MCP tool pattern**:
- Tool files live in `internal/mcp/tools/{toolname}/tool.go`
- Each tool implements `Definition()` returning a `ToolDefinition` and `Execute(ctx, args map[string]any) (string, error)`
- Registered in `internal/mcp/server.go` via `s.mcpServer.AddTool()`
- Known tools registry: `internal/config/tools.go` — add `"skill-install"` here

**Profile discovery**: `internal/profile/resolver.go` — `FindProfileDirs()` walks ancestor directories for `.brains/profiles/` and appends `~/.brains/profiles/`. The skill installer can use the same service to enumerate available profiles for validation and error messages.

**SKILL.md format** (from `~/.claude/skills/create-pr/SKILL.md`):
```yaml
---
name: {skill-name}
description: >
  {what the skill does, trigger phrases}
allowed-tools: mcp__zombiekit__profile-compose, ...
---

Call `mcp__zombiekit__profile-compose` with `profiles: ["{skill-name}"]` and follow the returned instructions exactly.
```
The body is always this one-liner — it's a live pointer to the profile, not embedded instructions.

**Skill directory locations**:
- Global: `~/.claude/skills/{name}/SKILL.md`
- Local: `{working_dir}/.claude/skills/{name}/SKILL.md`
- Claude Code loads both; local takes precedence over global.

### Domain Knowledge

**Why the delegating body pattern matters**: If the SKILL.md contained the actual instructions, updating the skill would require reinstalling it on every machine. By delegating to profile-compose, the profile file is the single source of truth. Update the profile, the skill is instantly updated everywhere it's installed — no reinstall needed.

**Profile metadata for SKILL.md generation**: The profile's `name` and `description` fields can be read from the profile file to populate the SKILL.md frontmatter. The `allowed-tools` field needs to include at minimum `mcp__zombiekit__profile-compose`. Additional allowed tools may be listed in the profile itself (a profile metadata extension, or derived from known patterns).

**NEEDS CLARIFICATION**: Where does the `allowed-tools` list come from for the generated SKILL.md? Options:
- A. Hardcode `mcp__zombiekit__profile-compose` only (minimal, always correct)
- B. Read from a metadata field in the profile file (e.g., a `skill-allowed-tools:` frontmatter key)
- C. Let the user pass allowed-tools as a CLI flag

Option B is most consistent with the existing profile system. Option A is viable MVP.

## Decision Points

- [ ] **D1**: Source of `allowed-tools` for generated SKILL.md — Options: A (hardcode compose only), B (read from profile metadata), C (CLI flag). Recommend B as target, A as MVP.
- [ ] **D2**: Default scope when no flag given — local or global? Recommend local (safer default, matches `--local` being explicit vs `--global`).
- [ ] **D3**: Should the CLI command validate that the profile exists before installing, or install blindly? Recommend: validate against known profiles, error with list if not found.
- [ ] **D4**: Should the MCP tool description be fetched from the profile's description field, or require it as a parameter? Recommend: read from profile if available, fallback to a generic description.

## Recommendations

1. **Start with MVP scope**: Hardcode `allowed-tools: mcp__zombiekit__profile-compose` in generated SKILL.md (D1 option A). Can add profile metadata later without breaking existing installed skills.
2. **Default to local scope** (`brains skill install <name>` installs to `.claude/skills/`), require explicit `--global` flag for global install.
3. **Reuse `profile.Service`** for profile discovery — enumerate available profiles for validation and error messages. Avoids duplicating resolution logic.
4. **New subcommand under `brains skill`** — mirrors the existing `brains profile` command structure. Register `newSkillCommand()` in `root.go`.
5. **MCP tool named `skill-install`** — add to `KnownTools` in `internal/config/tools.go`, implement in `internal/mcp/tools/skillinstall/tool.go`.

## Sources

- `internal/cli/root.go` — command registration
- `internal/cli/profile.go` — reference for CLI command pattern
- `internal/mcp/server.go` — MCP tool registration
- `internal/mcp/tools/stickymemory/tool.go` — reference MCP tool implementation
- `internal/profile/service.go`, `resolver.go` — profile discovery/enumeration
- `internal/config/tools.go` — known tools registry
- `~/.claude/skills/create-pr/SKILL.md` — canonical skill file example
