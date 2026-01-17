# Data Model: CLI Init Enhancement

**Feature**: 020-cli-init-here
**Date**: 2025-12-23

## Overview

This feature involves no database entities. The "data model" consists of:
1. Embedded filesystem structures (compile-time)
2. Target directory structures (runtime output)
3. CLI flag configuration

## Embedded Filesystems

### EmbeddedCommands

**Source Path**: `integrations/claude/commands/`
**Go Variable**: `EmbeddedCommands embed.FS`
**Prefix in FS**: `integrations/claude/commands/`

| File | Description |
|------|-------------|
| `brains.audit.md` | Audit command skill |
| `brains.bug.md` | Bug specification skill |
| `brains.clarify.md` | Clarification skill |
| `brains.complete.md` | Completion skill |
| `brains.eat.md` | Fun command skill |
| `brains.feature.md` | Feature specification skill |
| `brains.implement.md` | Implementation skill |
| `brains.init.md` | Init skill (meta) |
| `brains.plan.md` | Planning skill |
| `brains.refactor.md` | Refactoring skill |
| `brains.research.md` | Research skill |
| `brains.revise.md` | Revision skill |
| `brains.status.md` | Status skill |
| `brains.tasks.md` | Task generation skill |
| `brains.update.md` | Update skill |

### EmbeddedTemplates

**Source Path**: `templates/templates/`
**Go Variable**: `EmbeddedTemplates embed.FS`
**Prefix in FS**: `templates/templates/`

| File | Description |
|------|-------------|
| `agent-file-template.md` | Template for agent definitions |
| `checklist-template.md` | Template for quality checklists |
| `plan-template.md` | Template for implementation plans |
| `spec-template.md` | Template for feature specifications |
| `tasks-template.md` | Template for task breakdowns |

## Target Directory Structure

### When `brains init` (default)

```
<current-directory>/
├── .claude/
│   └── commands/
│       ├── brains.audit.md
│       ├── brains.bug.md
│       ├── brains.clarify.md
│       ├── brains.complete.md
│       ├── brains.eat.md
│       ├── brains.feature.md
│       ├── brains.implement.md
│       ├── brains.init.md
│       ├── brains.plan.md
│       ├── brains.refactor.md
│       ├── brains.research.md
│       ├── brains.revise.md
│       ├── brains.status.md
│       ├── brains.tasks.md
│       └── brains.update.md
└── .brains/
    └── templates/
        ├── agent-file-template.md
        ├── checklist-template.md
        ├── plan-template.md
        ├── spec-template.md
        └── tasks-template.md
```

### When `brains init --global`

```
~/.brains/
└── profiles/           # Only profiles directory (existing behavior)
```

## CLI Configuration Model

### InitConfig (conceptual)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Global` | bool | false | If true, init in ~/.brains/ |
| `Force` | bool | false | If true, overwrite existing files |
| `SourceType` | enum | "brains" | Profile source type (existing) |

### InitResult (conceptual)

| Field | Type | Description |
|-------|------|-------------|
| `FilesCopied` | int | Count of files successfully copied |
| `FilesSkipped` | int | Count of files skipped (already exist) |
| `FilesOverwritten` | int | Count of files overwritten (--force) |
| `Errors` | []error | Any errors encountered during copy |
| `TargetDir` | string | Root directory where init occurred |

## State Transitions

```
Not Initialized ──[brains init]──> Initialized
                                      │
                                      │ [brains init] (no --force)
                                      ▼
                                   Unchanged (files skipped)
                                      │
                                      │ [brains init --force]
                                      ▼
                                   Updated (files overwritten)
```

## Validation Rules

1. **Source validation**: Embedded filesystem must contain at least one file in each collection
2. **Target validation**: Current directory must be writable
3. **File conflict**: Existing files are skipped unless `--force` is provided
4. **Permission preservation**: Files written with 0644, directories with 0755
