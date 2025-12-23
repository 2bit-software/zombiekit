# Quickstart: Profile Import Subcommand

**Feature**: 005-profile-import
**Date**: 2025-12-22

## What This Feature Does

The `brains profiles import` command converts Claude agents from `.claude/agents/` directories into brains profiles in `.brains/profiles/` directories.

## Usage

### Basic Import

```bash
# Import all Claude agents to brains profiles
brains profiles import claude
```

### Preview Before Import (Dry Run)

```bash
# See what would be imported without making changes
brains profiles import claude --dry-run
```

### JSON Output

```bash
# Get structured output for scripting
brains profiles import claude --format json
```

## What Gets Converted

| Claude Agent Field | Brains Profile Field | Notes |
|--------------------|----------------------|-------|
| name | name | Preserved |
| description | description | Preserved |
| includes | includes | Preserved |
| model | - | Discarded |
| color | - | Discarded |
| body content | body content | Preserved unchanged |

All imported profiles have `inherits: false` set.

## Scope Preservation

- Local agents (`.claude/agents/`) → Local profiles (`.brains/profiles/`)
- Global agents (`~/.claude/agents/`) → Global profiles (`~/.brains/profiles/`)

## Collision Handling

If a brains profile already exists with the same name as a Claude agent, **it will be overwritten** without prompting.

Use `--dry-run` first to see what would be affected.

## Example Output

### Text Output

```
Imported 5 profiles from claude source:
  Created:     3 profiles
  Overwritten: 2 profiles

Created:
  .brains/profiles/reviewer.md
  .brains/profiles/architect.md
  ~/.brains/profiles/global-helper.md

Overwritten:
  .brains/profiles/researcher.md
  ~/.brains/profiles/coder.md
```

### JSON Output

```json
{
  "created": 3,
  "overwritten": 2,
  "failed": 0,
  "dry_run": false,
  "created_paths": [
    ".brains/profiles/reviewer.md",
    ".brains/profiles/architect.md",
    "~/.brains/profiles/global-helper.md"
  ],
  "overwritten_paths": [
    ".brains/profiles/researcher.md",
    "~/.brains/profiles/coder.md"
  ],
  "failed_agents": []
}
```

## Error Handling

If some agents fail to import (e.g., invalid frontmatter), the import continues with valid agents. Failures are reported in the summary:

```
Imported 4 profiles from claude source:
  Created:     3 profiles
  Overwritten: 1 profiles
  Failed:      1 agents

Failed:
  broken-agent.md: parsing frontmatter: yaml: line 3: could not find expected ':'
```

## Prerequisites

- The brains directory structure (`.brains/profiles/`) will be created automatically if it doesn't exist
- Claude agents must be in standard locations (`.claude/agents/` or `~/.claude/agents/`)
