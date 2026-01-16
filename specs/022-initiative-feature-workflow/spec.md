# Feature Specification: Initiative Feature Workflow

**Feature Branch**: `022-initiative-feature-workflow`
**Created**: 2025-12-23
**Status**: Draft
**Input**: User description: "New initiative/feature workflow that replicates speckit.specify with research-create-audit cycle. Builds initiative folders in ./history, creates state files, copies templates, and guides LLM through the full specification workflow via MCP step endpoint."

## Overview

The "feature" step implements the complete ZombieKit workflow cycle for creating a new feature specification.

### End-to-End Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ 1. USER types: /brains.feature "user authentication"                        │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ 2. CLAUDE CODE SKILL (.claude/commands/brains.feature.md)                   │
│    - Calls MCP tool: mcp_zombiekit__step(step="feature", name="user-auth")  │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ 3. MCP STEP TOOL (brains CLI)                                               │
│    - Creates initiative folder: ./history/{hex}-user-auth/                  │
│    - Creates cycle folder: ./history/{hex}-user-auth/{hex}-feat-user-auth/  │
│    - Creates/switches git branch: feat/user-auth                            │
│    - Copies templates to cycle folder                                       │
│    - Updates .brains/active.json                                            │
│    - Returns: { directive, initiative_folder, cycle_folder, files_to_read,  │
│                 composed_prompt, workflow_phases }                          │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ 4. CLAUDE CODE SKILL uses returned prompts to orchestrate:                  │
│    RESEARCH → CREATE → AUDIT → (loop if issues) → HIGHLIGHT → USER APPROVAL │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Separation of Concerns

| Component | Responsibility |
|-----------|----------------|
| `/brains.feature` skill | Orchestration, agent spawning, user interaction |
| MCP `step` tool | Folder/file creation, git handling, prompt composition |
| Brains CLI | Stateless utility, no LLM calls, no orchestration |

This workflow replicates and extends the `speckit.specify` process, adding:
- Parallel research phase before specification writing
- Audit phase with AI-readiness checks
- Loop-back mechanism for critical/major issues
- User approval gate before proceeding

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start a New Feature Initiative (Priority: P1)

A developer invokes the "feature" step via MCP with a feature name and description. The system creates the initiative structure, copies templates, and returns the directive that guides the LLM through the complete research-specify-audit cycle.

**Why this priority**: This is the core entry point that orchestrates the entire "new feature" experience. The returned directive must enable autonomous LLM operation through all specification phases.

**Independent Test**: Call `mcp_zombiekit__step` with `step="feature"`, `name="user-auth"`, and `description="Add user authentication with OAuth2"`. Verify folder creation, template copying, state update, and that the returned directive contains instructions for the research-create-audit cycle.

**Acceptance Scenarios**:

1. **Given** a project with a `.brains` folder, **When** the developer calls the step endpoint with `step="feature"`, `name="user-auth"`, and `description="..."`, **Then**:
   - An initiative folder is created at `./history/{hex}-user-auth/`
   - A cycle folder is created at `./history/{hex}-user-auth/{hex}-feat-user-auth/`
   - `INITIATIVE.md` is created at initiative root with metadata and cycle list
   - Templates are copied to cycle folder: `spec.md`, `research.md`, `audit/`
   - `.brains/active.json` is updated with initiative and cycle paths

2. **Given** the feature step completes successfully, **When** the response is returned, **Then** it includes:
   - `directive`: Multi-phase instructions for research→create→audit cycle
   - `initiative_folder`: Absolute path to the initiative folder
   - `cycle_folder`: Absolute path to the active cycle folder
   - `files_to_read`: List of template files now available
   - `composed_prompt`: Profiles for the specification workflow
   - `workflow_phases`: Structured description of each phase

3. **Given** the LLM receives the directive, **When** it begins execution, **Then** it can autonomously:
   - Conduct parallel research using multiple agents
   - Create the spec.md using the research findings
   - Run audit checks on the specification
   - Loop back if critical/major issues are found
   - Present highlights for user review

---

### User Story 2 - Research Phase Execution (Priority: P1)

The LLM, guided by the directive, executes the research phase before writing the specification. Multiple research agents run in parallel to gather context, and findings are collated into research.md.

**Why this priority**: Research before specification prevents incomplete or incorrect specs. This is a core differentiator from simple template-based approaches.

**Independent Test**: With an active initiative, verify the directive instructs spawning of parallel research agents and specifies how findings should be collated into research.md.

**Acceptance Scenarios**:

1. **Given** the feature step returns a directive, **When** the LLM reads the research phase instructions, **Then** it finds clear guidance on:
   - What to research (codebase context, related patterns, dependencies)
   - How to spawn parallel research agents
   - How to collate and deduplicate findings
   - Where to store research output (research.md)

2. **Given** research is complete, **When** research.md is written, **Then** it contains:
   - Summary of findings organized by category
   - Key insights relevant to the feature
   - Identified constraints and dependencies
   - Questions or ambiguities discovered

---

### User Story 3 - Specification Creation Phase (Priority: P1)

After research, a single agent creates the specification using the template and research findings.

**Why this priority**: The spec is the primary output artifact. It must be structured, AI-readable, and complete.

**Independent Test**: With research.md populated, verify the directive guides creation of spec.md following the template structure.

**Acceptance Scenarios**:

1. **Given** research.md contains findings, **When** the LLM executes the create phase, **Then** spec.md is written with:
   - All mandatory sections from the template
   - Content derived from research findings
   - Testable requirements
   - Measurable success criteria
   - No implementation details

2. **Given** spec.md is created, **When** the file state is checked, **Then** it is marked as "draft" in frontmatter or metadata.

---

### User Story 4 - Audit Phase Execution (Priority: P1)

After specification creation, audit agents check for completeness, AI-readiness, and quality.

**Why this priority**: Audit prevents low-quality specs from proceeding. It catches issues before implementation planning.

**Independent Test**: With spec.md created, verify the directive instructs audit checks and defines pass/fail criteria.

**Acceptance Scenarios**:

1. **Given** spec.md exists, **When** the LLM executes the audit phase, **Then** it checks:
   - Completeness (all sections filled, no placeholders)
   - AI-readiness (unambiguous, testable requirements)
   - No implementation details
   - Success criteria measurable
   - Edge cases identified

2. **Given** audit finds CRITICAL or MAJOR issues, **When** the directive is followed, **Then** the LLM loops back to fix issues before proceeding.

3. **Given** audit passes (no critical/major issues), **When** the directive is followed, **Then** the LLM proceeds to highlight phase.

---

### User Story 5 - Highlight and User Approval (Priority: P2)

After audit passes, key decisions and findings are highlighted for user review. User must approve before the initiative can proceed to the next stage.

**Why this priority**: User approval ensures alignment before investing in implementation planning.

**Independent Test**: With audit passed, verify the directive instructs presenting highlights and waiting for user approval.

**Acceptance Scenarios**:

1. **Given** audit passes, **When** highlight phase executes, **Then** the user sees:
   - Summary of the feature specification
   - Key decisions made during specification
   - Any minor issues or assumptions noted
   - Clear prompt for approval

2. **Given** user approves, **When** approval is recorded, **Then** the initiative is ready for `/brains.plan` or next phase.

3. **Given** user rejects approval with feedback, **When** the directive is followed, **Then** the LLM returns to audit phase with user feedback incorporated, addresses the issues, and attempts approval again.

---

### User Story 6 - Add New Cycle to Existing Initiative (Priority: P2)

A developer has completed a feature cycle but needs to refactor or fix bugs within the same initiative. They invoke a new cycle (e.g., `/brains.refactor`) which creates a new cycle folder within the existing initiative.

**Why this priority**: Multiple cycles per initiative enable iterative development without losing history, but the basic single-cycle workflow must work first.

**Independent Test**: Create a feature initiative, complete it, then invoke `/brains.refactor` and verify a new cycle folder is created within the same initiative.

**Acceptance Scenarios**:

1. **Given** an active initiative at `./history/{hex}-user-auth/` with a completed feature cycle, **When** `/brains.refactor` is invoked with the same initiative, **Then** a new cycle folder is created at `./history/{hex}-user-auth/{new-hex}-ref-user-auth/`.

2. **Given** a new cycle is created, **When** `INITIATIVE.md` is updated, **Then** it lists both the original feature cycle and the new refactor cycle.

3. **Given** a new cycle is created, **When** the state file is checked, **Then** it tracks the same initiative but points to the new active cycle.

---

### Edge Cases

- What happens when `./history` folder does not exist? The system creates it automatically.
- What happens when a feature with the same name already exists? A new unique ID is generated (timestamp-based), so naming collisions are impossible.
- What happens if research fails to find relevant context? The directive includes fallback guidance for minimal research scenarios.
- What happens if audit keeps finding issues after 3 loops? The directive instructs surfacing to user for manual intervention.
- What happens if the step is called without a name? Return a clear error requiring the `name` parameter.

## Requirements *(mandatory)*

### Functional Requirements

**MCP Tool Interface:**
- **FR-000**: The MCP step tool MUST be callable from Claude Code skills via `mcp_zombiekit__step(step, name, description, ...)`.
- **FR-000a**: The MCP step tool MUST perform all file/folder operations synchronously before returning.
- **FR-000b**: The MCP step tool MUST NOT make LLM API calls or spawn agents (orchestration is the skill's responsibility).

**Initiative Creation:**
- **FR-001**: System MUST provide a "feature" step accessible via `mcp_zombiekit__step` endpoint with `step="feature"`.
- **FR-002**: System MUST accept `name` (required) and `description` (optional) parameters.
- **FR-003**: System MUST create an initiative folder in `./history/` with naming format `{hex}-{normalized-name}` (type-agnostic container).
- **FR-003a**: System MUST create a cycle folder within the initiative with naming format `{hex}-feat-{normalized-name}` for the first feature cycle.
- **FR-004**: System MUST create `INITIATIVE.md` at the initiative level with: name, status, created timestamp, ID, description, and list of cycles.

**Template and File Management:**
- **FR-005**: System MUST copy template files to the cycle folder (not initiative folder): `spec.md`, `research.md`, and `audit/` directory.
- **FR-006**: Templates MUST be sourced from: (1) `.brains/templates/` if present, or (2) embedded defaults.
- **FR-007**: Template files MUST be copied with placeholder content ready for LLM population.

**State Management:**
- **FR-008**: System MUST update `.brains/active.json` to track ONLY the path to the currently active initiative (no status duplication).
- **FR-008a**: Initiative status (active, blocked, completed) MUST be stored in `INITIATIVE.md` frontmatter as the single source of truth.
- **FR-008b**: To determine initiative status, system MUST read and parse `INITIATIVE.md` frontmatter (not rely on state file).
- **FR-009**: When creating a new cycle within an existing initiative, system MUST reuse the existing initiative folder and create only a new cycle subfolder.
- **FR-009a**: System MUST auto-detect whether to create a new initiative or add a cycle based on: (1) presence of active initiative, (2) step type invoked.
- **FR-009b**: System MUST support `--new-initiative` flag to explicitly create a new initiative regardless of current state.
- **FR-010**: System MUST validate that the working directory contains a `.brains` folder.

**Response Content:**
- **FR-011**: System MUST return a StepResponse containing: directive, initiative_folder, cycle_folder, files_to_read, and composed_prompt.
- **FR-012**: The directive MUST include multi-phase instructions for: Research → Create → Audit → Highlight.
- **FR-012a**: All cycle types (feature, refactor, bug fix) MUST follow the same workflow structure with cycle-type-specific agents/prompts.
- **FR-013**: The directive MUST specify how to spawn parallel research agents.
- **FR-014**: The directive MUST define audit criteria and loop-back conditions.
- **FR-014a**: The directive MUST enforce a maximum of 3 audit/approval retry loops before requiring user intervention.
- **FR-015**: The directive MUST include user approval gate instructions.
- **FR-015a**: When creating a new cycle, `files_to_read` MUST include artifacts from previous cycles in the same initiative.

**Git Integration:**
- **FR-016**: System MUST create (or reuse) a git branch named `feat/<name>` for new feature initiatives.
- **FR-017**: When adding a new cycle to an existing initiative, system MUST NOT change the git branch name.

**Error Handling:**
- **FR-018**: System MUST return clear error messages for missing/invalid parameters.
- **FR-019**: System MUST normalize feature names to slug format.

### Key Entities

- **Initiative**: A top-level container for a unit of work, located at `./history/{hex}-{name}/`. Type-agnostic; can contain multiple cycles.
- **Cycle**: A single workflow pass (feature, refactor, or bug fix) within an initiative. Located at `./history/{hex}-{name}/{hex}-{type}-{name}/`. Contains all artifacts for one specification workflow.
- **Initiative Metadata** (`INITIATIVE.md`): Located at initiative root, contains name, status, description, and list of cycles.
- **Research Document** (`research.md`): Collated findings from the research phase, located in cycle folder.
- **Specification Document** (`spec.md`): The formal specification with user stories, requirements, and success criteria, located in cycle folder.
- **Audit Report** (`audit/{date}.md`): Results of audit checks with severity levels (CRITICAL, MAJOR, MINOR, INFO).
- **State File** (`.brains/active.json`): Tracks ONLY the path to the currently active initiative and cycle. Status is NOT stored here; it lives in `INITIATIVE.md` frontmatter.

## Expected Artifacts

### Folder Structure After Feature Step

```
history/
└── {hex}-{name}/                          # Initiative (top-level container)
    ├── INITIATIVE.md                      # Initiative metadata and description
    └── {hex}-feat-{name}/                 # First cycle (feature)
        ├── research.md                    # [blank template] → [populated after research]
        ├── spec.md                        # [blank template] → [populated after create]
        └── audit/                         # [empty] → [reports added after audit]
```

### Folder Structure After Subsequent Cycles

```
history/
└── {hex}-{name}/                          # Initiative (unchanged)
    ├── INITIATIVE.md                      # Updated with cycle history
    ├── {hex}-feat-{name}/                 # First cycle (complete)
    │   ├── research.md
    │   ├── spec.md
    │   └── audit/
    └── {hex}-ref-{name}/                  # Second cycle (refactor)
        ├── research.md
        ├── spec.md
        └── audit/
```

### Artifact States Through Workflow

| Artifact | After Feature Step | After Research | After Create | After Audit |
|----------|-------------------|----------------|--------------|-------------|
| `INITIATIVE.md` | Populated (metadata) | Unchanged | Unchanged | Updated (status) |
| `research.md` | Blank template | Populated (findings) | Unchanged | Unchanged |
| `spec.md` | Blank template | Unchanged | Populated (spec) | May be updated |
| `audit/*.md` | Empty folder | Empty | Empty | Report(s) added |

### Markdown File States

**`INITIATIVE.md` States:**
- `active`: Initiative is in progress
- `blocked`: Waiting on user input or external dependency
- `completed`: All phases finished and approved

**`spec.md` States (tracked in frontmatter):**
- `template`: Blank template, not yet written
- `draft`: Written but not audited
- `audited`: Passed audit with no critical/major issues
- `approved`: User has approved, ready for planning

**`research.md` States:**
- `template`: Blank, research not started
- `in-progress`: Research agents running
- `complete`: Research finished, findings collated

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new feature initiative can be created and ready for the research phase with a single call completing in under 2 seconds.
- **SC-002**: 100% of feature step invocations with valid parameters result in a complete initiative structure with all required files.
- **SC-003**: The returned directive enables autonomous LLM execution through research, create, and audit phases without additional human guidance.
- **SC-004**: 90% of specifications pass audit on first attempt when research phase is properly executed.
- **SC-005**: User approval is required before any initiative can proceed to planning phase.
- **SC-006**: Error messages for invalid inputs clearly describe the problem and suggest corrective action.

## Assumptions

- The existing initiative service (`internal/initiative`) and step service (`internal/step`) from spec 021 are implemented and available.
- The profile composition system (`internal/profile`) is available for creating the composed prompt.
- Claude Code's Task tool will be used for spawning parallel research agents (not orchestrated by brains CLI).
- The directive returned is a comprehensive prompt that the LLM follows, not procedural code.
- Templates are stored in `templates/templates/` (embedded) and optionally overridden in `.brains/templates/` (local).
- The naming convention uses `feat` as a short prefix (consistent with git conventions).

## Clarifications

### Session 2025-12-23

- Q: What naming conventions apply to git branches vs initiative folders? → A: Git branches use `feat/<title>`, `fix/<title>`, or `ref/<title>` format (no hex ID, no number prefix). When an initiative's type changes (e.g., from feature to refactor), the git branch name remains unchanged.
- Q: How are initiatives and cycles structured? → A: An **initiative** is a top-level container at `./history/{hex}-{name}/`. Within each initiative are **cycles** representing different workflow phases: `./history/{hex}-{name}/{hex}-feat-{name}/` for the first feature cycle, then `./history/{hex}-{name}/{hex}-ref-{name}/` for a subsequent refactor, etc. Multiple cycles can exist within a single initiative as work evolves.
- Q: How does the system decide whether to create a new initiative or add a cycle? → A: Auto-detect based on step type + active initiative state. If there's an active initiative and a different step type is invoked, create a new cycle within the existing initiative. If no active initiative exists, create a new initiative. Users can explicitly pass `--new-initiative` flag to break out and start a fresh initiative regardless of current state.
- Q: What happens when user rejects approval? → A: Return to audit phase with user's feedback incorporated. The audit surfaces specific issues for the LLM to address before attempting another approval cycle.
- Q: How do cycles access artifacts from previous cycles? → A: Previous cycle artifacts are included in `files_to_read` for new cycles. This enables informed refactoring/fixes based on original requirements and prior work.
- Q: Is there a maximum number of audit/approval retry loops? → A: Yes, limit to 3 loops. After 3 failed attempts, require user intervention to prevent infinite loops while giving reasonable chance for self-correction.
- Q: Do all cycle types follow the same workflow? → A: Yes, all cycle types (feature, refactor, bug fix) follow the same Research → Create → Audit → Highlight workflow structure, but with different agents/prompts at each phase tailored to the cycle type.
- Q: What is the MCP tool vs skill responsibility split? → A: The MCP `step` tool handles all synchronous operations (folder/file creation, git branch handling, template copying, state updates) and returns composed prompts. The Claude Code skill (e.g., `/brains.feature`) handles orchestration (spawning agents, workflow execution, user interaction). The MCP tool never makes LLM calls.
- Q: Where is initiative status stored? → A: Initiative status (active, blocked, completed) is stored as YAML frontmatter in `INITIATIVE.md`, making it the single source of truth. The state file (`.brains/active.json`) only tracks WHICH initiative is currently active (path reference), not its status. This ensures status is human-readable in the markdown file and avoids duplication/sync issues between files.

## Relationship to Existing Speckit

This feature step replicates the workflow from `.claude/commands/speckit.specify.md` with these enhancements:

| speckit.specify | Feature Step |
|-----------------|--------------|
| Creates branch + spec folder | Creates initiative folder in `./history` |
| Writes spec.md directly | Returns directive for research→create→audit |
| Single-pass specification | Multi-phase with audit loop |
| Manual quality checklist | Automated audit phase |
| Immediate next step | User approval gate |

The key difference: speckit.specify is a Claude Code skill (orchestration), while the feature step provides the structured directive that enables that orchestration.
