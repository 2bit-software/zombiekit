# Initiative: skill-installer

**Type**: feature
**Status**: completed
**Created**: 2026-04-02
**ID**: 69cedeed-feature-skill-installer

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-04-02 14:26 |
| plan | completed | 2026-04-02 15:30 |
| tasks | completed | 2026-04-02 15:35 |
| implement | completed | 2026-04-02 16:10 |

## Description

Add CLI and MCP endpoints to install Claude Code skills from existing zombiekit profiles.

## Goals

- `brains skill install <name> [--global]` CLI command
- `skill-install` MCP tool usable from within Claude conversations
- Skills delegate to `profile-compose` at runtime (live updates without reinstall)
- Full test coverage including stdio protocol-level integration test

## Completion

**Completed**: 2026-04-02
**Duration**: ~2 hours

### Outcomes

- `internal/skill/install.go` — core logic (ValidateName, TargetDir, GenerateContent, WriteSkill)
- `internal/cli/skill.go` — `brains skill install` CLI command
- `internal/mcp/tools/skillinstall/tool.go` — `skill-install` MCP tool
- `internal/mcp/server.go` — tool wired into MCP server
- `internal/config/tools.go` — tool registered in KnownTools
- `internal/cli/root.go` — CLI command registered
- Tests: unit (skill package), CLI, MCP tool, MCP stdio protocol end-to-end

### Notes

All tests pass. The stdio protocol test (`TestServer_StdioProtocol_SkillInstall`) exercises the full JSON-RPC path using `io.Pipe()` — no subprocess needed.
