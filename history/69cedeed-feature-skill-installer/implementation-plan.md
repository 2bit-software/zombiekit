# Implementation Plan: Skill Installer

**Initiative**: 69cedeed-feature-skill-installer
**Date**: 2026-04-02
**Status**: Ready

## Overview

Add `brains skill install <name> [--global]` CLI command and `skill-install` MCP tool. Both entry points share core install logic in a new `internal/skill/` package. The installer validates the profile exists, reads its description, and writes a SKILL.md that delegates to `mcp__zombiekit__profile-compose`.

## Step Sequence

### Step 1 — Core install logic (`internal/skill/`)

**Files created:**
- `internal/skill/install.go`
- `internal/skill/install_test.go`

**What it does:**
- `ValidateName(name string) error` — validates against `^[a-z0-9][a-z0-9-]*[a-z0-9]$` or single-char `^[a-z0-9]$`; rejects `.`, `/`, `\`, spaces, uppercase, underscores
- `TargetDir(global bool, workingDir string) (string, error)` — returns `~/.claude/skills/` or `{workingDir}/.claude/skills/`
- `GenerateContent(name, description string) string` — produces SKILL.md content from the exact template in FR-002
- `WriteSkill(targetDir, name, content string) (string, error)` — creates `{targetDir}/{name}/SKILL.md`, creates dirs if missing, overwrites existing SKILL.md, errors if `{name}` exists as a plain file; returns full path

This package has no I/O imports beyond `os` and `path/filepath`. No profile service dependency here — that belongs to the caller.

**Tests (real temp dirs, no mocking):**
- `ValidateName`: valid and invalid names from FR-007 test table
- `GenerateContent`: assert output matches exact template with and without description
- `WriteSkill`: creates dirs, overwrites idempotently, errors on plain-file collision

### Step 2 — CLI command (`internal/cli/skill.go`)

**Files created:**
- `internal/cli/skill.go`

**Files modified:**
- `internal/cli/root.go` — add `newSkillCommand()` to Commands slice

**Command structure:**
```
brains skill
  install <name> [--global]
```

**`newSkillInstallCommand()` action:**
1. Read `name` from `c.Args().First()`; error if empty
2. Call `skill.ValidateName(name)` — error on invalid
3. Create profile service: `profile.NewServiceWithSource("brains", "")`
4. Call `svc.Show(name, false)` to get profile metadata
   - On not-found error: call `svc.List()`, build error message with available names
   - On other errors: wrap and return
5. Extract description from `ShowResult` (see technical spec for field access)
6. Call `skill.TargetDir(c.Bool("global"), "")` for target directory
7. Generate content: `skill.GenerateContent(name, description)`
8. Write: `skill.WriteSkill(targetDir, name, content)`
9. Print: `fmt.Printf("Installed skill '%s' to %s\n", name, fullPath)`

**Flags:**
- `--global` (`BoolFlag`) — install to `~/.claude/skills/`

### Step 3 — MCP tool (`internal/mcp/tools/skillinstall/`)

**Files created:**
- `internal/mcp/tools/skillinstall/tool.go`

**Files modified:**
- `internal/config/tools.go` — add `"skill-install"` to `KnownTools`
- `internal/mcp/server.go` — add `skillInstallTool *skillinstalltool.Tool` field, instantiate in `NewServer()`, register in `registerTools()`

**Tool parameters:**
- `name` (string, required)
- `scope` (string, required, enum: `local`, `global`)
- `working_directory` (string, optional)

**`Execute()` logic:**
1. Extract and validate `name`, `scope`, `working_directory` from args
2. Call `skill.ValidateName(name)` — return error on invalid
3. Resolve working dir: `working_directory` arg if provided, else `""`
4. Create profile service: `profile.NewService(workingDir)`
5. Call `svc.Show(name, false)` — on not-found: call `svc.List()`, return formatted error
6. Extract description
7. Call `skill.TargetDir(scope == "global", workingDir)`
8. Generate + write skill
9. Return success string: `"Installed skill '{name}' to {fullPath}"`

### Step 4 — Wire up

**`internal/mcp/server.go` changes:**
```go
// In Server struct:
skillInstallTool *skillinstalltool.Tool

// In NewServer():
skillInstallTool: skillinstalltool.NewTool(),

// In registerTools():
if s.config.IsToolEnabled("skill-install") {
    // ... mcp.NewTool("skill-install", ...) + s.mcpServer.AddTool(...)
}
```

**`internal/cli/root.go` change:**
Add `newSkillCommand()` to the Commands slice.

## Dependencies Between Steps

```
Step 1 (core) → Step 2 (CLI) → Step 4 (wire)
Step 1 (core) → Step 3 (MCP) → Step 4 (wire)
```

Steps 2 and 3 can be done in parallel after Step 1.

## What's NOT in scope

- Reading `allowed-tools` from profile metadata (hardcoded to `mcp__zombiekit__profile-compose`)
- `brains skill list` or `brains skill remove` commands
- Skill update notifications
- Marketplace/registry integration

## Verified Interfaces

- **`ShowResult.Description`**: Direct field on `ShowResult`, not nested — use `result.Description`
- **`ListEntry.Name`**: Confirmed `Name string` field exists on `ListEntry`
- **`profile.SourceTypeBrains`**: Use the typed constant, not the string `"brains"`
