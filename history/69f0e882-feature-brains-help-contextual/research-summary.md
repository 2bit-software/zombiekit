# Research Summary: Contextual /brains.help

## Current State

### What Exists

The help command lives at `embed/commands/help.md` and is loaded via `mcp__zombiekit__workflow-load` with `name: "help"` and `type: "command"`. The integration shim is at `embed/integrations/claude/commands/brains.help.md`.

**Current behavior**: The command is a static markdown template with two output formats (active initiative / no initiative). It lists all commands in a table and shows placeholder data. It does NOT:
- Call `initiative status` to get real state
- Conditionally render based on workflow type or step
- Validate artifacts or prerequisites
- Filter commands by relevance to current state

### What Data Is Available

The `initiative status` MCP tool returns rich state:

| Field | Type | Description |
|-------|------|-------------|
| `active` | bool | Whether an initiative exists |
| `initiative_id` | string | Full ID (e.g., `69f0e882-feature-brains-help-contextual`) |
| `initiative_type` | string | `feature`, `bug`, `refactor` |
| `current_step` | string | Parsed from step table in INITIATIVE.md |
| `step_status` | string | `pending`, `in_progress`, `completed`, `skipped` |
| `steps_completed` | int | Number of completed steps |
| `steps_total` | int | Total steps in workflow |
| `available_docs` | []string | Artifact filenames found in initiative dir |
| `suggested_next` | string | Deterministic next action recommendation |
| `history_path` | string | Relative path to initiative dir |
| `initiative_file` | string | Path to INITIATIVE.md |
| `files` | []string | All artifact paths |

Implementation: `internal/initiative/service.go` (`Status()` method), `internal/initiative/markdown.go` (INITIATIVE.md parsing), `internal/mcp/tools/initiative/tool.go` (MCP handler).

### All Commands

| Command | File | Description |
|---------|------|-------------|
| `/brains.new` | `embed/commands/new.md` | Start new work — classifies and loads workflow |
| `/brains.next` | `embed/commands/next.md` | Advance to next workflow step |
| `/brains.complete` | `embed/commands/complete.md` | Mark initiative complete |
| `/brains.help` | `embed/commands/help.md` | This command — currently static |

### All Workflows

| Workflow | Steps | Artifacts |
|----------|-------|-----------|
| **feature** | spec → plan → tasks → implement | business-spec.md, technical-requirements-research.md, research-summary.md, audit-reports/ |
| **feature-light** | spec+plan → implement | notes.md, tasks.md |
| **bug** | report → reproduce → investigate → classify → fix-plan | report.md, reproduction.md, investigation.md, classification.md, fix-plan.md |
| **refactor** | goal → constraints → dependency-analysis → safety → plan | goal.md, constraints.md, dependency-analysis.md, refactor-plan.md, safety-net.md |
| **unmanaged** | (empty step table, user drives) | INITIATIVE.md only |

### Profiles Available

| Profile | Purpose |
|---------|---------|
| `status` | Display initiative status and suggested next steps |
| `complete` | Handle completion workflow |
| `implement` | Implementation phase logic |
| `automode` | Auto-execution mode overrides |

## Domain Research Findings

### Git's State Machine Pattern (Primary Model)

Git's `wt-status.c` is the gold standard for contextual help:
- State determines hints (rebase shows rebase commands, merge shows merge commands)
- Consistent grammar: "use `command` to `verb`"
- Functions grouped by scenario, not one giant switch
- Dual output modes: human-friendly (default) vs machine-friendly (`--porcelain`)

### Progressive Disclosure (Two-Tier Model)

Recommended structure:
1. **Default**: State-aware summary (~40 lines) — where you are, what's next, what's relevant
2. **Verbose** (`/brains.help full`): Complete command reference, all workflow phases, all step types

### State Visualization

- Progress fraction: "Step 2/4 (50%)" — simple, universal
- Visual marker for current step (arrow or `>>>`)
- Relative time ("3h ago") over raw timestamps in help output
- Available actions listed per state, not globally

### Dual Audience (Human + AI Agent)

- Structured markdown with consistent `##` headers for agent parsing
- Exact command strings (not prose) for all suggested actions
- Machine-parseable metadata (frontmatter or fenced block) at top
- The MCP tool's JSON output already serves the machine audience; help.md serves the human audience

## Key Gaps Identified

1. **No dynamic state rendering** — help.md is a static template, doesn't call `initiative status`
2. **No step-specific guidance** — doesn't explain what the current step does or what artifacts are expected
3. **No command filtering** — shows all commands always, regardless of relevance
4. **No artifact validation** — doesn't check if prerequisites are met for next step
5. **No workflow-type awareness** — same output for feature, bug, refactor
6. **No progressive disclosure** — single output format, no brief vs verbose mode
7. **No escape hatch guidance** — doesn't tell users how to recover from stuck states
8. **References `/brains.step`** — a command that may not be fully implemented
9. **No Linear/source ticket integration** — ignores source section from INITIATIVE.md
10. **No initiative listing** — when no initiative is active, doesn't show recent/available initiatives
