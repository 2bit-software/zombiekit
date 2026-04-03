---
name: help
description: Show available commands, current state, and valid actions
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Help Workflow

Goal: Provide clear visibility into available commands and current state.

### Execution Steps

1. **Load Active State**
   - Read `.brains/active.json`
   - Determine if there's an active initiative

2. **Show Available Commands**
   - Always display the command reference

3. **Show Current State** (if active initiative)
   - Initiative name and start date
   - Current work item and phase
   - Recent activity

4. **Show Valid Next Actions**
   - Based on current state, show what makes sense

### Command Reference

```markdown
# ZombieKit Commands

| Command | Purpose |
|---------|---------|
| `/brains.new [type] [desc]` | Start new work (auto-detects feature/bug/refactor) |
| `/brains.step <name>` | Jump to specific step (spec, plan, tasks, implement, etc.) |
| `/brains.next [alt]` | Advance to next step in workflow |
| `/brains.complete` | Finish current initiative |
| `/brains.help` | Show this help (you are here) |
```

### Output Format (No Active Initiative)

```markdown
# ZombieKit Help

## Commands

| Command | Purpose |
|---------|---------|
| `/brains.new [type] [desc]` | Start new work |
| `/brains.step <name>` | Jump to specific step |
| `/brains.next` | Advance to next step |
| `/brains.complete` | Finish current initiative |
| `/brains.help` | Show this help |

## Current State

No active initiative.

## Getting Started

Start new work with:
```
/brains.new add user authentication
/brains.new fix login not working
/brains.new refactor auth module
```

The system auto-detects whether this is a feature, bug, or refactor.
```

### Output Format (With Active Initiative)

```markdown
# ZombieKit Help

## Commands

| Command | Purpose |
|---------|---------|
| `/brains.new [type] [desc]` | Start new work |
| `/brains.step <name>` | Jump to specific step |
| `/brains.next` | Advance to next step |
| `/brains.complete` | Finish current initiative |
| `/brains.help` | Show this help |

## Current State

**Initiative**: {name}
**Started**: {date} ({X days ago})
**Location**: history/{id}/

### Active Work Item
**{type}/{name}** - {phase} phase

Artifacts:
- business-spec.md (complete)
- implementation-plan.md (in progress)

### All Work Items

| Type | Name | Status | Phase |
|------|------|--------|-------|
| Feature | auth-api | Complete | - |
| Feature | session-mgmt | In Progress | planning |

## Suggested Actions

Based on current state:
1. `/brains.next` - Continue to tasks phase
2. `/brains.step implement` - Skip to implementation
3. `/brains.complete` - Mark initiative done
```

### Workflow Phase Reference

**Feature phases**: spec -> plan -> tasks -> implement
**Bug phases**: report -> investigate -> fix-plan -> implement
**Refactor phases**: goal -> analysis -> plan -> tasks -> implement

**Common steps** (available anytime):
- `audit` - Cross-check artifact alignment
- `clarify` - Identify ambiguities
- `research` - Standalone research
- `update` - Modify existing artifacts
- `revise` - Re-enter workflow cycle
