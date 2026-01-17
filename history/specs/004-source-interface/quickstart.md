# Quickstart: Profile Source Abstraction

**Feature**: 004-source-interface
**Date**: 2025-12-22

## Overview

This feature adds a `--source` flag to profile commands, allowing users to work with either brains profiles (`.brains/profiles/`) or Claude agents (`.claude/agents/`).

## Quick Usage

### List profiles from default source (brains)
```bash
brains profile list
```

### List Claude agents
```bash
brains profile list --source claude
# or
brains profile list -s claude
```

### Show a Claude agent
```bash
brains profile show systems-architect --source claude
```

### Compose Claude agents
```bash
brains profile compose architect,reviewer --source claude
```

### Create a new Claude agent
```bash
brains profile create my-agent --source claude
```

### Create in global location
```bash
brains profile create my-agent --source claude --global
```

### Validate Claude agents
```bash
brains profile validate --source claude
```

### Initialize Claude agents directory
```bash
brains init --source claude
```

## Key Differences Between Sources

| Aspect | Brains | Claude |
|--------|--------|--------|
| Directory | `.brains/profiles/` | `.claude/agents/` |
| Resolution | local > parent > global | local > global only |
| Default inherits | `true` | `false` |
| Extra fields | - | `model`, `color` |

## Example Claude Agent

```markdown
---
name: code-reviewer
description: Reviews code for best practices
model: sonnet
color: blue
includes: []
inherits: false
---

You are a code reviewer. Focus on:
- Code clarity and readability
- Performance implications
- Security considerations
```

## JSON Output

All list/show commands support JSON output:

```bash
brains profile list --source claude --format json
```

## File Locations

### Brains Profiles
- Local: `{project}/.brains/profiles/`
- Parent: `{ancestor}/.brains/profiles/` (up to git root)
- Global: `~/.brains/profiles/`

### Claude Agents
- Local: `{project}/.claude/agents/`
- Global: `~/.claude/agents/`
