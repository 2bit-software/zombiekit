# Research Summary: README Documentation

## Executive Summary

ZombieKit is a prompt composition and artifact management system for Claude Code. It provides structured workflows for AI-assisted development with persistent memory, composable prompts, and artifact version control.

**Core Value Proposition:** Transforms ad-hoc AI conversations into repeatable, structured development workflows.

---

## Key Findings

### What ZombieKit Is

- **NOT an AI orchestrator** - Claude Code handles all orchestration
- **A memory layer** - Persistent storage, prompt composition, artifact tracking
- **A workflow system** - Structured steps from spec to implementation
- **CLI + MCP server** - Works via Claude Code skills

### User Journey

```
brains init → /brains.feature → approve spec → /brains.plan → approve plan → /brains.tasks → /brains.implement → /brains.complete
```

### Core Components

1. **MCP Server** - Exposes tools to Claude Code (profile-compose, stickymemory, step, initiative)
2. **Profile System** - Composable prompt units with inheritance
3. **Initiative System** - Tracks multi-step workflows with artifacts
4. **Skills** - `/brains.*` commands that orchestrate the workflow

### The ZombieKit Cycle

Every stage follows: **Research → Create → Audit → Highlight**

- Research: Parallel agents explore codebase/domain
- Create: Single agent synthesizes findings
- Audit: Parallel agents check completeness
- Highlight: Present key decisions for user approval

### Setup Requirements

1. Go 1.24+ for building
2. Task (taskfile.dev) for build commands
3. Claude Code for the IDE/CLI
4. MCP configuration pointing to `brains serve`

### Available Skills

| Skill | Purpose |
|-------|---------|
| `/brains.feature` | Create feature specification |
| `/brains.bug` | Bug investigation |
| `/brains.refactor` | Refactoring specification |
| `/brains.plan` | Implementation planning |
| `/brains.tasks` | Task breakdown |
| `/brains.implement` | Execute tasks |
| `/brains.status` | Check initiative status |
| `/brains.complete` | Mark done |

---

## User Needs Assessment

### Primary Users

- Developers using Claude Code who want structured AI workflows
- Teams wanting repeatable AI-assisted development processes
- Solo developers wanting persistent context across sessions

### Key User Questions

1. "How do I install and set up ZombieKit?" → Installation section
2. "What can it do?" → Capabilities overview
3. "How do I use it?" → Quick start + workflow guide
4. "What are the commands?" → Skills reference

### Documentation Gaps

- No README exists currently
- DESIGN.md is developer/contributor focused, not user focused
- No quick start guide for new users

---

## Technical Constraints (from user request)

- No CI badges (not running in CI yet)
- User-focused, not contributor-focused
- Must include Claude Code setup instructions
- Must explain the workflow cycle
