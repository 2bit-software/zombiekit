# Technical Requirements & Research: Contextual /brains.help

## Implementation Surface

The entire change is in one file: `embed/commands/help.md`. This is a markdown command file that instructs the AI how to render help output. No Go code changes needed — the `initiative status` MCP tool already provides all necessary data.

## Architecture

The help command is a **prompt template** — it tells the AI agent what to do when `/brains.help` is invoked. The AI executes the instructions (calling MCP tools, formatting output). The "implementation" is writing better instructions in the markdown file.

### Execution Flow

1. User runs `/brains.help`
2. Claude loads `embed/integrations/claude/commands/brains.help.md` (thin wrapper)
3. Wrapper calls `mcp__zombiekit__workflow-load` with `name: "help"`, `type: "command"`
4. Server returns `embed/commands/help.md` content
5. AI follows the instructions in help.md:
   a. Call `mcp__zombiekit__initiative` with `action: "status"`
   b. Branch on `active` field
   c. Render appropriate output format

### Data Sources

All data comes from the `initiative status` MCP call. No new data sources needed.

Key fields to use:
- `active` → determines output mode
- `initiative_type` → determines step context and artifact expectations
- `current_step` / `step_status` → determines progress visualization
- `steps_completed` / `steps_total` → progress fraction
- `available_docs` → artifact status
- `suggested_next` → primary recommended action
- `history_path` → artifact directory path
- `files` → full artifact paths

### Step Context Data

The help command needs a mapping of workflow type → step → description + expected artifacts. This data lives in the workflow markdown files but needs to be summarized in help.md as a reference table.

| Workflow | Step | Description | Expected Artifacts |
|----------|------|-------------|-------------------|
| feature | spec | Research and write business specification | business-spec.md, technical-requirements-research.md, research-summary.md |
| feature | plan | Create implementation plan from spec | implementation-plan.md |
| feature | tasks | Break plan into discrete tasks | tasks.md |
| feature | implement | Execute tasks | (code changes) |
| bug | report | Document the bug and expected behavior | report.md |
| bug | reproduce | Create reliable reproduction | reproduction.md |
| bug | investigate | Root cause analysis | investigation.md |
| bug | classify | Categorize and assess impact | classification.md |
| bug | fix-plan | Plan the fix | fix-plan.md |
| refactor | goal | Define refactoring objectives | goal.md |
| refactor | constraints | Identify constraints and invariants | constraints.md |
| refactor | dependency-analysis | Analyze dependency impact | dependency-analysis.md |
| refactor | safety | Assess safety and create safety net | safety-net.md |
| refactor | plan | Create refactoring plan | refactor-plan.md |
| feature-light | spec | Quick spec and plan (single pass) | notes.md |
| feature-light | implement | Execute implementation | tasks.md, (code changes) |
| unmanaged | (user-driven) | Manual implementation | INITIATIVE.md only |

### Command Validity Matrix

| Command | No Initiative | spec/report/goal | plan/investigate/constraints | tasks/classify/safety | implement/fix-plan |
|---------|--------------|------------------|-----------------------------|-----------------------|-------------------|
| `/brains.new` | **primary** | available (warn: will close current) | available (warn) | available (warn) | available (warn) |
| `/brains.next` | hidden | **primary** | **primary** | **primary** | **primary** |
| `/brains.complete` | hidden | available | available | available | **primary** (alongside next) |
| `/brains.help` | always | always | always | always | always |

## Research Findings Applied

### Git State Machine Pattern → Command Instructions

Instead of static output templates, the help.md should contain conditional rendering instructions organized by state. Each state "owns" its output — no giant switch logic.

### Progressive Disclosure → Arguments Handling

The `$ARGUMENTS` field in help.md is already wired. If arguments contain "full" or "reference", render verbose mode. Default is concise.

### Dual Audience → Output Structure

Use consistent `##` headers that both humans and AI agents can parse:
- `## Initiative: {name}` — header block
- `## Progress` — step visualization
- `## Artifacts` — file status
- `## Available Actions` — filtered commands
- `## Step Context` — what current step does

### `/brains.step` Question

Need to verify: search for `brains.step` in the codebase to determine if this command exists. If not, remove from help output.
