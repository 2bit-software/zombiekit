---
status: in-progress
---
# Feature Specification: Simplified Command Structure

**Feature Branch**: `696edb15-feature-simplified-commands`
**Created**: 2026-01-19
**Status**: In Progress
**Input**: Simplify ZombieKit commands from 10+ to 5 intuitive commands

## Problem Statement

Developers using ZombieKit must currently memorize multiple commands (`/brains.feature`, `/brains.bug`, `/brains.refactor`, `/brains.plan`, `/brains.tasks`, etc.) and know which one applies to their situation. This creates cognitive overhead and friction, especially for new users or when the work type is ambiguous.

## User Scenarios & Testing

### User Story 1 - Starting New Work (Priority: P1)

As a developer beginning new work, I need to describe what I want to accomplish in natural language, so that I can start productive work without deciding upfront whether it's a "feature" or "refactor".

**Why this priority**: This is the primary entry point for all ZombieKit workflows. Every user must start work somehow, and this determines the first impression and adoption rate.

**Independent Test**: User invokes `/brains.new "add rate limiting to the API"` and system routes to appropriate workflow without errors.

**Acceptance Scenarios**:

1. **Given** I invoke `/brains.new` with "add rate limiting to the API", **When** the system analyzes my request, **Then** it determines the appropriate workflow type (feature) and begins that workflow
2. **Given** the system detects ambiguity between feature and bug fix, **When** it asks for clarification, **Then** I can quickly confirm and proceed without starting over
3. **Given** I want to create a new agent profile, **When** I invoke `/brains.new` with that intent, **Then** the profile creation workflow begins
4. **Given** I invoke `/brains.new bug` explicitly, **When** the system processes my request, **Then** it skips intent detection and starts the bug workflow directly

---

### User Story 2 - Navigating Within a Workflow (Priority: P1)

As a developer mid-workflow, I need to jump to a specific phase or advance to the next logical step, so that I can control my progress without memorizing workflow internals.

**Why this priority**: Once work starts, navigation is the most frequent operation. Poor navigation causes frustration and abandonment.

**Independent Test**: User can move forward with `/brains.next` and backward with `/brains.step spec` without losing work.

**Acceptance Scenarios**:

1. **Given** I'm in the planning phase, **When** I invoke `/brains.next`, **Then** I proceed to task generation
2. **Given** I want to revisit specification after seeing the plan, **When** I invoke `/brains.step spec`, **Then** I return to the specification phase with artifacts editable
3. **Given** I invoke `/brains.next audit`, **When** the system processes it, **Then** I proceed to audit instead of the default next step
4. **Given** I invoke `/brains.step invalid-name`, **When** the system validates, **Then** it returns an error listing valid steps from the registry

---

### User Story 3 - Completing Work (Priority: P1)

As a developer finishing an initiative, I need to mark my work complete through a single command, so that artifacts are finalized and the workflow concludes cleanly.

**Why this priority**: Proper completion prevents orphaned initiatives and maintains clean state for future work.

**Independent Test**: User invokes `/brains.complete` and initiative is marked done with clear confirmation.

**Acceptance Scenarios**:

1. **Given** I've implemented all tasks, **When** I invoke `/brains.complete`, **Then** the initiative is marked done and artifacts are archived
2. **Given** I have incomplete tasks, **When** I invoke `/brains.complete`, **Then** I'm informed of what remains and asked to confirm early completion
3. **Given** no initiative is active, **When** I invoke `/brains.complete`, **Then** I receive a clear error message

---

### User Story 4 - Getting Help (Priority: P2)

As a new or occasional user, I need to see what commands are available, what they do, and where I am in a workflow, so that I can learn the system and orient myself without external documentation.

**Why this priority**: Help is critical for onboarding but not blocking for experienced users.

**Independent Test**: User invokes `/brains.help` and receives actionable guidance for their current context.

**Acceptance Scenarios**:

1. **Given** I'm unsure what command to use, **When** I invoke `/brains.help`, **Then** I see the available commands with brief descriptions
2. **Given** I'm mid-workflow, **When** I invoke `/brains.help`, **Then** I see my current position, completed steps, and available next actions
3. **Given** I want to know what workflows exist, **When** I invoke `/brains.help`, **Then** I see the list of workflow types (feature, bug, refactor, profile)

---

### User Story 5 - Sub-task Creation (Priority: P2)

As a developer who discovers a bug while implementing a feature, I need to start a bug sub-task without abandoning my feature work, so that I can address issues as they arise within context.

**Why this priority**: Important for real-world workflow flexibility but not the primary path.

**Independent Test**: User invokes `/brains.new bug` during active feature and bug is linked as sub-task.

**Acceptance Scenarios**:

1. **Given** I have an active feature initiative, **When** I invoke `/brains.new bug "found null pointer in auth"`, **Then** the bug becomes a sub-task within the current initiative
2. **Given** I complete the bug sub-task, **When** I invoke `/brains.complete`, **Then** only the sub-task is marked complete, not the parent feature

---

### Edge Cases

- What happens when the system can't confidently classify the work type? → Ask the user to choose from available workflow types returned by the registry
- How should users be informed when they try to navigate to a step that doesn't exist? → System returns error listing valid steps from the registry for the current workflow
- What happens when a user invokes "new" while already in an active workflow? → New work becomes a sub-task within the current initiative
- What happens if MCP registry fetch fails or times out? → Use cached/embedded fallback registry with clear warning
- What if user provides both explicit type (`/brains.new bug`) AND description? → Explicit type wins, description used for context only

## Requirements

### Functional Requirements

- **FR-001**: System MUST provide a unified `/brains.new` command that routes to appropriate workflows (feature, bug, refactor, profile)
- **FR-002**: System MUST analyze user descriptions to determine appropriate workflow type using LLM-based classification in the skill profile (not MCP tool) with keyword pattern hints. Classification runs on Claude Code side, not in MCP server.
- **FR-003**: System MUST allow users to override detected workflow type (e.g., `/brains.new bug`)
- **FR-004**: System MUST provide `/brains.next [alt]` command to advance to next step with optional alternative path
- **FR-005**: System MUST provide `/brains.step [name]` command to jump to any named workflow phase
- **FR-006**: System MUST allow backward navigation to earlier workflow phases without losing artifacts
- **FR-007**: System MUST provide `/brains.complete` command to finish current initiative
- **FR-008**: System MUST provide `/brains.help` command showing commands, current state, and valid actions
- **FR-009**: System MUST query MCP workflow registry on each command to validate against available workflows/steps
- **FR-010**: System MUST return errors with valid options when user requests invalid workflow or step
- **FR-011**: System MUST prompt user to select workflow type when intent detection confidence is low (< 0.7 threshold, configurable via ZOMBIEKIT_INTENT_THRESHOLD environment variable)
- **FR-012**: System MUST support sub-task creation within active initiatives (new work becomes child task)
- **FR-013**: System MUST preserve existing artifact editing capabilities when navigating backwards
- **FR-014**: System MUST maintain backwards compatibility during migration (old commands work for 2 release cycles with deprecation warning, then removed)

### Key Entities

- **Initiative**: A tracked work unit with lifecycle (active, completed) stored in `history/` folder. Contains one or more cycles and optional sub-tasks.
- **Artifact**: Generated document (spec.md, plan.md, tasks.md, research.md) within an initiative cycle.
- **Workflow Registry**: MCP endpoint returning available workflow types and their steps. Commands fetch fresh data on every invocation.
- **Intent Classifier**: Component in skill profile that analyzes user description and returns workflow type with confidence score (0-1).
- **Workflow State**: Current initiative context including position in workflow and available transitions. Stored in `.brains/active.json`.
- **Sub-task**: Child work item (bug, feature, refactor) within a parent initiative. Limited to 2 levels of nesting.
- **Step/Phase**: Synonymous terms for a workflow stage (e.g., feature, plan, tasks, eat). Steps have prerequisites and transitions.

## Success Criteria

### Measurable Outcomes

- **SC-001**: New users successfully start their first workflow without consulting documentation
- **SC-002**: Average number of commands to complete a full workflow decreases vs. explicit command approach
- **SC-003**: Users can recall the 5 commands after single explanation (easy to remember)
- **SC-004**: Intent detection correctly classifies work type on first attempt >= 80% of the time
- **SC-005**: Users complete workflows without getting "stuck" on navigation commands
- **SC-006**: Migration from old commands to new commands causes zero workflow disruptions

## Testing Requirements

### Test Strategy

- Integration tests for MCP tool registration and execution
- Unit tests for intent classification logic (pure functions with edge cases)
- E2E tests for complete workflow scenarios (new → step → complete)
- Mock registry for testing error handling and fallback behavior

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | /brains.new routes to correct workflow tool |
| FR-002 | Unit | Intent classifier returns correct type for clear descriptions |
| FR-002 | Unit | Intent classifier returns low confidence for ambiguous input |
| FR-003 | Integration | Explicit type override bypasses intent detection |
| FR-004 | Integration | /brains.next advances to correct next step |
| FR-005 | Integration | /brains.step navigates to named phase |
| FR-006 | Integration | Backward navigation preserves artifacts |
| FR-009 | Integration | Registry fetch provides valid workflow/step lists |
| FR-010 | Unit | Invalid step/workflow returns helpful error with options |
| FR-011 | Integration | Low confidence triggers user prompt |
| FR-012 | Integration | Sub-task creation links to parent initiative |
| FR-007 | Integration | /brains.complete marks initiative done |
| FR-008 | Integration | /brains.help returns context-aware guidance |
| FR-013 | Integration | Backward navigation preserves artifacts unchanged |
| FR-014 | Integration | Old commands emit deprecation warning and function |

### Edge Case Coverage

- Ambiguous input (feature vs refactor) → Tests confirm prompt presented
- Invalid step name → Tests confirm error with valid options listed
- Registry timeout → Tests confirm fallback to embedded registry
- Concurrent initiative conflict → Tests confirm clear error message
- Empty description to /brains.new → Tests confirm prompt for description or type

## Command Summary

| Command | Purpose | MCP Tool |
|---------|---------|----------|
| `/brains.new [type] [desc]` | Start new work (auto-detects type) or add sub-task | workflow `new` action |
| `/brains.step [name]` | Jump to specific step (forwards or backwards) | workflow `step` action |
| `/brains.next [alt]` | Advance to next step (optionally specify alternative) | workflow `next` action |
| `/brains.complete` | Finish current initiative | workflow `complete` action |
| `/brains.help` | Show commands, current state, valid actions | workflow `help` action |

## Assumptions

- Five commands (`new`, `step`, `next`, `complete`, `help`) provide sufficient coverage for typical workflows
- Intent detection through natural language is reliable enough to be helpful rather than frustrating
- Users prefer intelligent defaults with override capability over explicit upfront choices
- Existing workflow phases (research, spec, plan, tasks, implement) remain stable
- MCP registry endpoint latency is acceptable (< 500ms)

## Out of Scope

- Voice or conversational command interfaces
- Automated workflow selection based on code analysis (e.g., auto-detecting a bug from test failures)
- Multi-user workflow coordination
- Command aliases or customization
- Changes to underlying workflow phases (feature, bug, refactor, plan, tasks, eat)

## Open Questions

None at this time - spec transferred from Linear ticket DEV-83.

---

## Business Logic Decision Trees

This section illustrates how the simplified commands route through the workflow system. Each command has an intent detection layer that routes to the appropriate underlying workflow.

### Overview: Command-to-Workflow Mapping

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SIMPLIFIED COMMAND LAYER                            │
│                                                                             │
│   /brains.new    /brains.step    /brains.next    /brains.complete   /brains.help │
└───────┬─────────────┬─────────────┬─────────────────┬───────────────────┬───┘
        │             │             │                 │                   │
        ▼             ▼             ▼                 ▼                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INTENT DETECTION LAYER                              │
│   (Runs in skill profile on Claude side, not in MCP tool)                   │
│                                                                             │
│   • Analyzes user description for work type                                 │
│   • Returns confidence score (0-1)                                          │
│   • Falls back to user prompt if confidence < 0.7                           │
└─────────────────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         WORKFLOW REGISTRY (MCP)                             │
│                                                                             │
│   Available workflows: feature, bug, refactor, profile                      │
│   Available steps: feature, bug, refactor, plan, tasks, eat, audit, clarify │
│   Prerequisites: spec.md→plan, plan.md→tasks, tasks.md→eat                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

### Decision Tree: `/brains.new`

**Purpose**: Start new work with automatic type detection or explicit override.

```
/brains.new [type?] [description?]
                │
                ▼
        ┌───────────────────┐
        │ Has explicit type │
        │ (feature/bug/     │
        │  refactor/profile)│
        └─────────┬─────────┘
                  │
        ┌─────────┴─────────┐
        │                   │
        ▼                   ▼
    ┌───────┐           ┌───────┐
    │  YES  │           │  NO   │
    └───┬───┘           └───┬───┘
        │                   │
        │                   ▼
        │           ┌───────────────────┐
        │           │ Has description?  │
        │           └─────────┬─────────┘
        │                     │
        │           ┌─────────┴─────────┐
        │           │                   │
        │           ▼                   ▼
        │       ┌───────┐           ┌───────┐
        │       │  YES  │           │  NO   │
        │       └───┬───┘           └───┬───┘
        │           │                   │
        │           ▼                   ▼
        │   ┌─────────────────┐   ┌─────────────────┐
        │   │ Intent Detection│   │ Prompt user for │
        │   │ (LLM classifies │   │ type or         │
        │   │  description)   │   │ description     │
        │   └────────┬────────┘   └─────────────────┘
        │            │
        │            ▼
        │   ┌─────────────────┐
        │   │ Confidence ≥0.7?│
        │   └────────┬────────┘
        │            │
        │   ┌────────┴────────┐
        │   │                 │
        │   ▼                 ▼
        │ ┌───────┐       ┌───────┐
        │ │  YES  │       │  NO   │
        │ └───┬───┘       └───┬───┘
        │     │               │
        │     │               ▼
        │     │       ┌─────────────────┐
        │     │       │ Prompt user to  │
        │     │       │ choose from:    │
        │     │       │ • feature       │
        │     │       │ • bug           │
        │     │       │ • refactor      │
        │     │       │ • profile       │
        │     │       └────────┬────────┘
        │     │                │
        │     └────────┬───────┘
        │              │
        ▼              ▼
┌───────────────────────────────┐
│ Check for active initiative   │
└───────────────┬───────────────┘
                │
        ┌───────┴───────┐
        │               │
        ▼               ▼
    ┌───────┐       ┌───────┐
    │ NONE  │       │ACTIVE │
    └───┬───┘       └───┬───┘
        │               │
        ▼               ▼
┌──────────────┐  ┌──────────────────┐
│Create new    │  │Create sub-task   │
│initiative    │  │within current    │
│              │  │initiative        │
│history/      │  │(max 2 levels)    │
│{id}-{type}-  │  │                  │
│{slug}/       │  │history/{parent}/ │
└──────┬───────┘  │ {id}-{type}/     │
       │          └────────┬─────────┘
       │                   │
       └─────────┬─────────┘
                 │
                 ▼
┌────────────────────────────────────────────┐
│ Route to workflow step based on type:      │
│                                            │
│ feature  → step("feature")                 │
│ bug      → step("bug")                     │
│ refactor → step("refactor")                │
│ profile  → profile creation workflow       │
└────────────────────────────────────────────┘
```

---

### Decision Tree: `/brains.step`

**Purpose**: Jump to a specific named step (forward or backward).

```
/brains.step [name]
        │
        ▼
┌───────────────────┐
│ Has active        │
│ initiative?       │
└─────────┬─────────┘
          │
  ┌───────┴───────┐
  │               │
  ▼               ▼
┌───────┐     ┌───────┐
│  NO   │     │  YES  │
└───┬───┘     └───┬───┘
    │             │
    ▼             ▼
┌─────────┐   ┌───────────────────┐
│ ERROR:  │   │ Validate step     │
│ "No     │   │ name against      │
│ active  │   │ registry          │
│ init"   │   └─────────┬─────────┘
└─────────┘             │
                ┌───────┴───────┐
                │               │
                ▼               ▼
            ┌───────┐       ┌───────┐
            │INVALID│       │ VALID │
            └───┬───┘       └───┬───┘
                │               │
                ▼               ▼
        ┌─────────────┐   ┌───────────────────┐
        │ ERROR:      │   │ Check direction   │
        │ "Invalid    │   └─────────┬─────────┘
        │ step. Valid │             │
        │ steps are:  │     ┌───────┴───────┐
        │ {list}"     │     │               │
        └─────────────┘     ▼               ▼
                        ┌───────┐       ┌───────┐
                        │FORWARD│       │BACKWARD│
                        └───┬───┘       └───┬───┘
                            │               │
                            ▼               ▼
                ┌─────────────────┐   ┌─────────────────┐
                │ Check           │   │ ALLOWED         │
                │ prerequisites   │   │ (preserves      │
                └────────┬────────┘   │ artifacts)      │
                         │            └────────┬────────┘
                ┌────────┴────────┐            │
                │                 │            │
                ▼                 ▼            │
            ┌───────┐         ┌───────┐        │
            │ MET   │         │NOT MET│        │
            └───┬───┘         └───┬───┘        │
                │                 │            │
                │                 ▼            │
                │         ┌─────────────┐      │
                │         │ ERROR:      │      │
                │         │ "Requires   │      │
                │         │ {artifact}  │      │
                │         │ with status │      │
                │         │ {status}"   │      │
                │         └─────────────┘      │
                │                              │
                └──────────────┬───────────────┘
                               │
                               ▼
                ┌──────────────────────────────┐
                │ Execute step                 │
                │                              │
                │ 1. Load step definition      │
                │ 2. Resolve file patterns     │
                │ 3. Compose profiles          │
                │ 4. Return StepResponse       │
                └──────────────────────────────┘
```

**Step Registry**:

| Step | Prerequisites | Workflow Type |
|------|--------------|---------------|
| feature | none | spec creation |
| bug | none | spec creation |
| refactor | none | spec creation |
| plan | spec.md (approved) | planning |
| tasks | plan.md (approved) | task generation |
| eat | tasks.md (exists) | implementation |
| audit | varies | validation |
| clarify | varies | disambiguation |

---

### Decision Tree: `/brains.next`

**Purpose**: Advance to the next logical step in the workflow.

```
/brains.next [alt?]
        │
        ▼
┌───────────────────┐
│ Has active        │
│ initiative?       │
└─────────┬─────────┘
          │
  ┌───────┴───────┐
  │               │
  ▼               ▼
┌───────┐     ┌───────┐
│  NO   │     │  YES  │
└───┬───┘     └───┬───┘
    │             │
    ▼             ▼
┌─────────┐   ┌───────────────────┐
│ ERROR:  │   │ Get current step  │
│ "No     │   │ from state        │
│ active  │   └─────────┬─────────┘
│ init"   │             │
└─────────┘             ▼
                ┌───────────────────┐
                │ Has alternative   │
                │ step specified?   │
                └─────────┬─────────┘
                          │
                  ┌───────┴───────┐
                  │               │
                  ▼               ▼
              ┌───────┐       ┌───────┐
              │  YES  │       │  NO   │
              └───┬───┘       └───┬───┘
                  │               │
                  ▼               ▼
          ┌─────────────┐   ┌─────────────────┐
          │ Validate    │   │ Look up default │
          │ alt is      │   │ next step for   │
          │ valid       │   │ current step    │
          │ transition  │   └────────┬────────┘
          └──────┬──────┘            │
                 │                   │
                 └─────────┬─────────┘
                           │
                           ▼
                ┌──────────────────────────────┐
                │ DEFAULT TRANSITIONS:         │
                │                              │
                │ (no init)  → feature/bug/ref │
                │ feature    → plan            │
                │ bug        → plan            │
                │ refactor   → plan            │
                │ plan       → tasks           │
                │ tasks      → eat             │
                │ eat        → eat (next task) │
                │                              │
                │ ALTERNATIVE TRANSITIONS:     │
                │                              │
                │ feature    → audit (verify)  │
                │ plan       → clarify         │
                │ eat        → complete        │
                └──────────────┬───────────────┘
                               │
                               ▼
                ┌───────────────────────────────┐
                │ Route to /brains.step {next}  │
                │ (inherits prerequisite check) │
                └───────────────────────────────┘
```

---

### Decision Tree: `/brains.complete`

**Purpose**: Mark the current initiative as complete.

```
/brains.complete
        │
        ▼
┌───────────────────┐
│ Has active        │
│ initiative?       │
└─────────┬─────────┘
          │
  ┌───────┴───────┐
  │               │
  ▼               ▼
┌───────┐     ┌───────┐
│  NO   │     │  YES  │
└───┬───┘     └───┬───┘
    │             │
    ▼             ▼
┌─────────┐   ┌───────────────────┐
│ ERROR:  │   │ Is this a         │
│ "No     │   │ sub-task?         │
│ active  │   └─────────┬─────────┘
│ init"   │             │
└─────────┘     ┌───────┴───────┐
                │               │
                ▼               ▼
            ┌───────┐       ┌───────┐
            │  YES  │       │  NO   │
            └───┬───┘       └───┬───┘
                │               │
                ▼               ▼
    ┌───────────────────┐   ┌───────────────────┐
    │ Complete sub-task │   │ Check for         │
    │ Return to parent  │   │ incomplete tasks  │
    │ initiative        │   └─────────┬─────────┘
    └───────────────────┘             │
                            ┌─────────┴─────────┐
                            │                   │
                            ▼                   ▼
                        ┌───────┐           ┌───────┐
                        │ SOME  │           │ NONE  │
                        │REMAIN │           │       │
                        └───┬───┘           └───┬───┘
                            │                   │
                            ▼                   │
                ┌───────────────────────┐       │
                │ WARN: "X tasks remain │       │
                │ incomplete:           │       │
                │ • T005: desc          │       │
                │ • T006: desc          │       │
                │                       │       │
                │ Complete anyway?"     │       │
                └───────────┬───────────┘       │
                            │                   │
                    ┌───────┴───────┐           │
                    │               │           │
                    ▼               ▼           │
                ┌───────┐       ┌───────┐       │
                │ USER: │       │ USER: │       │
                │  YES  │       │  NO   │       │
                └───┬───┘       └───┬───┘       │
                    │               │           │
                    │               ▼           │
                    │       ┌─────────────┐     │
                    │       │ ABORT       │     │
                    │       │ (no change) │     │
                    │       └─────────────┘     │
                    │                           │
                    └─────────────┬─────────────┘
                                  │
                                  ▼
                ┌──────────────────────────────────┐
                │ Mark initiative complete:        │
                │                                  │
                │ 1. Update INITIATIVE.md status   │
                │ 2. Clear active.json             │
                │ 3. Archive artifacts (optional)  │
                │ 4. Report completion summary     │
                └──────────────────────────────────┘
```

---

### Decision Tree: `/brains.help`

**Purpose**: Show available commands, current state, and valid actions.

```
/brains.help
        │
        ▼
┌───────────────────┐
│ Has active        │
│ initiative?       │
└─────────┬─────────┘
          │
  ┌───────┴───────┐
  │               │
  ▼               ▼
┌───────┐     ┌───────┐
│  NO   │     │  YES  │
└───┬───┘     └───┬───┘
    │             │
    ▼             ▼
┌─────────────┐   ┌───────────────────────────────┐
│ SHOW:       │   │ SHOW:                         │
│             │   │                               │
│ ## Commands │   │ ## Current Initiative         │
│             │   │ Name: {name}                  │
│ /brains.new │   │ Type: {feature|bug|refactor}  │
│   Start new │   │ Step: {current step}          │
│   work      │   │                               │
│             │   │ ## Artifacts                  │
│ /brains.help│   │ • spec.md: {status}           │
│   Show this │   │ • plan.md: {status}           │
│   help      │   │ • tasks.md: {status}          │
│             │   │                               │
│ ## Workflow │   │ ## Available Actions          │
│ Types       │   │                               │
│             │   │ /brains.next                  │
│ • feature   │   │   → {default next step}       │
│ • bug       │   │                               │
│ • refactor  │   │ /brains.step {name}           │
│ • profile   │   │   Valid steps:                │
│             │   │   • {step1}                   │
└─────────────┘   │   • {step2}                   │
                  │   • ...                       │
                  │                               │
                  │ /brains.complete              │
                  │   Finish this initiative      │
                  │                               │
                  │ ## Sub-tasks                  │
                  │ {list if any}                 │
                  └───────────────────────────────┘
```

---

### Complete Workflow: Feature Development Example

This shows a complete path through the system using simplified commands:

```
User: /brains.new "add rate limiting to API endpoints"
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ INTENT DETECTION                                              │
│ Input: "add rate limiting to API endpoints"                   │
│ Classification: feature (confidence: 0.92)                    │
│ Reason: "add" keyword + new capability                        │
└───────────────────────────────────────────────────────────────┘
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ CREATE INITIATIVE                                             │
│ ID: 696edb15-feature-rate-limiting                            │
│ Type: feature                                                 │
│ Folder: history/696edb15-feature-rate-limiting/               │
└───────────────────────────────────────────────────────────────┘
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ STEP: feature                                                 │
│                                                               │
│ Phase 0: Initialize INITIATIVE.md                             │
│     │                                                         │
│     ▼                                                         │
│ Phase I: Research (parallel agents)                           │
│     │ ├─ research-codebase                                    │
│     │ └─ research-domain                                      │
│     ▼                                                         │
│ Phase II: Create spec.md                                      │
│     │                                                         │
│     ▼                                                         │
│ Phase III: Audit (parallel agents)                            │
│     │ ├─ audit-completeness                                   │
│     │ └─ audit-ai-readiness                                   │
│     ▼                                                         │
│ Phase IV: Highlight → User approves spec                      │
│     │                                                         │
│     └─► spec.md status: approved                              │
└───────────────────────────────────────────────────────────────┘
                │
User: /brains.next
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ STEP: plan                                                    │
│ (prerequisite: spec.md approved ✓)                            │
│                                                               │
│ Step 1: Constitution check                                    │
│ Step 2: Technical analysis                                    │
│ Step 3: Project structure                                     │
│ Step 4: Implementation phases                                 │
│ Step 5: Testing strategy                                      │
│     │                                                         │
│     └─► plan.md status: approved                              │
└───────────────────────────────────────────────────────────────┘
                │
User: /brains.next
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ STEP: tasks                                                   │
│ (prerequisite: plan.md approved ✓)                            │
│                                                               │
│ Step 1: Analyze plan                                          │
│ Step 2: Define task format                                    │
│ Step 3: Organize by phase                                     │
│ Step 4: Apply TDD ordering                                    │
│ Step 5: Mark dependencies                                     │
│ Step 6: Include validation tasks                              │
│     │                                                         │
│     └─► tasks.md created                                      │
└───────────────────────────────────────────────────────────────┘
                │
User: /brains.next
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ STEP: eat (task T001)                                         │
│ (prerequisite: tasks.md exists ✓)                             │
│                                                               │
│ Execute task T001 → Mark [x]                                  │
└───────────────────────────────────────────────────────────────┘
                │
User: /brains.next (repeat until all tasks done)
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ STEP: eat (task T00N)                                         │
│                                                               │
│ next_task: null (all complete)                                │
│ Output: "All tasks complete!"                                 │
└───────────────────────────────────────────────────────────────┘
                │
User: /brains.complete
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ COMPLETE INITIATIVE                                           │
│                                                               │
│ ✓ All tasks complete                                          │
│ ✓ Initiative marked done                                      │
│ ✓ Active state cleared                                        │
│                                                               │
│ Summary:                                                      │
│   Feature: Rate Limiting                                      │
│   Tasks: 12/12 complete                                       │
│   Artifacts: spec.md, plan.md, tasks.md                       │
└───────────────────────────────────────────────────────────────┘
```

---

### Complete Workflow: Bug Fix Example

```
User: /brains.new "users getting 500 errors on login"
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ INTENT DETECTION                                              │
│ Input: "users getting 500 errors on login"                    │
│ Classification: bug (confidence: 0.88)                        │
│ Reason: "errors" + problem description                        │
└───────────────────────────────────────────────────────────────┘
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ CREATE INITIATIVE                                             │
│ ID: 696edb16-bug-login-500-errors                             │
│ Type: bug                                                     │
└───────────────────────────────────────────────────────────────┘
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ STEP: bug                                                     │
│                                                               │
│ Phase I: Investigation (parallel agents)                      │
│     │ ├─ investigate-codebase                                 │
│     │ ├─ investigate-history                                  │
│     │ └─ investigate-dependencies                             │
│     ▼                                                         │
│ Phase II: Classification                                      │
│     │ → IMPLEMENTATION_BUG (not spec gap)                     │
│     ▼                                                         │
│ Phase III: Fix Specification                                  │
│     │                                                         │
│     ▼                                                         │
│ Phase IV: Audit & Highlight → User approves                   │
└───────────────────────────────────────────────────────────────┘
                │
User: /brains.next  →  plan  →  tasks  →  eat  →  complete
```

---

### Complete Workflow: Sub-task Creation

```
User: /brains.new "add user dashboard"
      │
      └─► Initiative created, working on feature...

User: /brains.new bug "found null pointer in auth during testing"
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ ACTIVE INITIATIVE DETECTED                                    │
│ Parent: 696edb15-feature-user-dashboard                       │
│                                                               │
│ Creating sub-task within parent initiative                    │
│ Sub-task: 696edb16-bug-auth-null-pointer                      │
│ Location: history/696edb15-feature-user-dashboard/            │
│           696edb16-bug-auth-null-pointer/                     │
└───────────────────────────────────────────────────────────────┘
                │
User: /brains.next → plan → tasks → eat
                │
User: /brains.complete
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│ SUB-TASK COMPLETE                                             │
│                                                               │
│ Bug fix complete. Returning to parent initiative.             │
│ Current: 696edb15-feature-user-dashboard                      │
│ Step: (resumed from previous position)                        │
└───────────────────────────────────────────────────────────────┘
```

---

### Error Handling Flows

```
┌─────────────────────────────────────────────────────────────────┐
│ ERROR: Invalid Step Name                                        │
│                                                                 │
│ User: /brains.step foobar                                       │
│                                                                 │
│ Response:                                                       │
│   Error: Invalid step 'foobar'                                  │
│   Valid steps for current workflow (feature):                   │
│     • feature (current)                                         │
│     • plan (requires spec.md approved)                          │
│     • tasks (requires plan.md approved)                         │
│     • eat (requires tasks.md)                                   │
│     • audit                                                     │
│     • clarify                                                   │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ ERROR: Prerequisite Not Met                                     │
│                                                                 │
│ User: /brains.step plan                                         │
│ (but spec.md is not approved)                                   │
│                                                                 │
│ Response:                                                       │
│   Error: Cannot proceed to 'plan'                               │
│   Required: spec.md with status 'approved'                      │
│   Current: spec.md with status 'in-progress'                    │
│   Hint: Complete the feature step and approve the spec first    │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ ERROR: No Active Initiative                                     │
│                                                                 │
│ User: /brains.next                                              │
│ (but no active initiative)                                      │
│                                                                 │
│ Response:                                                       │
│   Error: No active initiative                                   │
│   Hint: Use /brains.new to start a new initiative               │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ ERROR: Registry Timeout                                         │
│                                                                 │
│ (MCP registry fails to respond within 500ms)                    │
│                                                                 │
│ Response:                                                       │
│   Warning: Registry unavailable, using embedded fallback        │
│   (continues with embedded step definitions)                    │
└─────────────────────────────────────────────────────────────────┘
```
