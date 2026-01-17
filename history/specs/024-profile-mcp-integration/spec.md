# Feature Specification: Profile-MCP Integration

**Feature Branch**: `024-profile-mcp-integration`
**Created**: 2025-12-24
**Status**: Draft
**Input**: User description: "Update step profiles to work with MCP zombiekit tool, handle Go backend return data, and maintain separate /spec workflow (research, spec, audit, highlight)"

## Clarifications

### Session 2025-12-24

- Q: Should MCP endpoints be created to replace/supplement spec-kit bash scripts? → A: ~~Yes, MCP provides equivalent endpoints~~ **REVISED**: No separate `spec` tool. All functionality integrated into `initiative` + `step` tools.
- Q: Should the bash scripts be deprecated, or kept as fallback? → A: Bash scripts belong to spec-kit (separate framework) and remain untouched.
- Q: Do we need a separate `spec` tool or should spec-kit actions integrate into steps? → A: **Integrate into steps.** The `initiative create` handles scaffolding; steps handle workflow execution. No separate tool needed.
- Q: Should git operations be exposed to the agent via MCP? → A: **No.** Git operations (branch creation, checkout) are handled internally by Go code. The agent receives branch name as informational context but never executes git commands via MCP. This provides reliability and separation of concerns.
- Q: What layout should initiatives use? → A: **Always use `history/` layout** (timestamped folders). The `specs/NNN-name` pattern is spec-kit's domain and not replicated.

## Overview

This feature enhances the brains/zombiekit MCP system with a unified **initiative-based workflow** using `initiative` + `step` tools:

- **Initiative tool**: Lifecycle management (create, status, complete, list) with automatic git branch creation
- **Step tool**: Workflow execution with prerequisites, multi-phase steps, and structured responses

Note: spec-kit (`.claude/commands/speckit.*.md` + bash scripts) is a **separate, unmodified framework**. This feature does not touch spec-kit. Users who want spec-kit's `specs/NNN-name` layout continue using spec-kit; zombiekit uses `history/` layout.

The goal is to update the embedded step profiles (`templates/steps/*.md`) so they:
- Work correctly with the `initiative` and `step` MCP tools
- Handle the structured JSON response from the Go backend
- Provide clear directives that guide agent behavior
- Keep git operations internal (agent never executes git commands via MCP)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Execute Feature Specification via MCP Step Tool (Priority: P1)

A developer invokes `/brains.feature` (or calls the `step` MCP tool with `step=feature`). The system returns a structured response containing the directive, files to read, and composed profile prompt. The agent uses this response to execute the feature specification workflow (research → create → audit → highlight).

**Why this priority**: This is the primary entry point for the brains workflow. The profile must provide actionable guidance that an agent can follow without additional context.

**Independent Test**: Can be tested by calling `mcp__zombiekit__step(step="feature", dir=".")` and verifying the response contains directive text that guides the agent through the 4-phase workflow.

**Acceptance Scenarios**:

1. **Given** an active initiative, **When** agent calls step tool with `step=feature`, **Then** response contains `directive` with research→create→audit→highlight phase instructions
2. **Given** the step response, **When** agent reads `files_to_read` and `composed_prompt`, **Then** agent has sufficient context to execute the feature workflow
3. **Given** the feature step directive, **When** agent follows it, **Then** agent spawns parallel research agents, synthesizes spec, runs audits, and presents highlights

---

### User Story 2 - Handle MCP Tool Response Structure (Priority: P1)

When a step is executed via MCP, the Go backend returns a structured JSON response. The agent receives: `directive` (instructions), `files_to_read` (absolute paths to context files), `composed_prompt` (merged profile content), `initiative_folder`, `cycle_folder`, and `workflow_phases`. The agent must correctly interpret and use each field.

**Why this priority**: If agents don't correctly handle the response structure, the entire workflow fails. This is foundational to all step execution.

**Independent Test**: Can be tested by parsing a sample step response and verifying all expected fields are present and usable.

**Acceptance Scenarios**:

1. **Given** MCP step response, **When** parsing JSON, **Then** `directive` field contains non-empty instruction text
2. **Given** MCP step response with `files_to_read` array, **When** agent reads files, **Then** each path exists and contains relevant context
3. **Given** MCP step response with `workflow_phases` array, **When** agent reads phases, **Then** each phase has `name`, `description`, `agents`, `outputs`, and `parallel` fields

---

### User Story 3 - Execute Plan Step with Prerequisites Check (Priority: P1)

A developer invokes the plan step. The system checks prerequisites (approved spec.md) and returns planning guidance. If prerequisites are not met, the system returns an error with guidance on what's needed.

**Why this priority**: The plan step is gated by spec approval. Demonstrating prerequisite enforcement validates the entire step sequencing model.

**Independent Test**: Can be tested by calling plan step with approved spec (success) and without approved spec (failure with guidance).

**Acceptance Scenarios**:

1. **Given** spec.md with `status: approved` in frontmatter, **When** agent calls plan step, **Then** response contains planning directive
2. **Given** spec.md without approval status, **When** agent calls plan step, **Then** response is error with hint to approve spec first
3. **Given** plan step response, **When** agent reads directive, **Then** instructions guide creation of implementation plan artifacts

---

### User Story 4 - Execute Tasks Step with Dependency Tracking (Priority: P2)

A developer invokes the tasks step to generate task breakdown from an approved plan. The profile directive guides creation of ordered, parallelizable tasks with dependencies.

**Why this priority**: Tasks step bridges planning to implementation. Clear task generation guidance is essential for execution phase.

**Independent Test**: Can be tested by calling tasks step on approved plan and verifying directive includes task generation methodology.

**Acceptance Scenarios**:

1. **Given** approved plan.md, **When** agent calls tasks step, **Then** response contains task generation directive
2. **Given** tasks step directive, **When** agent follows it, **Then** generated tasks.md has ID format (T001), checkboxes, and dependency markers [P]

---

### User Story 5 - Execute Eat Step with Next Task Identification (Priority: P2)

A developer invokes the eat step to implement tasks. The Go backend identifies the next incomplete task from tasks.md and includes it in the response. The agent uses this to focus implementation.

**Why this priority**: The eat step is where implementation happens. Automatic next-task identification reduces cognitive load.

**Independent Test**: Can be tested by calling eat step with tasks.md containing incomplete tasks, verifying `next_task` field in response.

**Acceptance Scenarios**:

1. **Given** tasks.md with unchecked items, **When** agent calls eat step, **Then** response includes `next_task` with ID, description, and phase
2. **Given** tasks.md with all tasks complete, **When** agent calls eat step, **Then** directive indicates all tasks complete
3. **Given** `next_task` in response, **When** agent implements it, **Then** agent can mark task complete in tasks.md

---

### User Story 6 - Execute Audit Step for Cross-Artifact Alignment (Priority: P2)

A developer invokes the audit step to verify consistency between spec, plan, and tasks. The profile provides audit methodology and severity classification guidance.

**Why this priority**: Audits prevent drift between artifacts. The profile must guide systematic alignment checking.

**Independent Test**: Can be tested by calling audit step and verifying directive includes alignment check methodology and severity classification.

**Acceptance Scenarios**:

1. **Given** an initiative with spec, plan, and tasks, **When** agent calls audit step, **Then** directive guides cross-artifact alignment checking
2. **Given** audit directive, **When** agent follows it, **Then** issues are classified as CRITICAL/MAJOR/MINOR with fix suggestions

---

### User Story 7 - Execute Clarify Step for Ambiguity Detection (Priority: P2)

A developer invokes the clarify step to surface underspecified areas. The profile provides ambiguity scanning methodology and question generation guidance.

**Why this priority**: Early clarification prevents downstream rework. The profile must guide targeted question generation.

**Independent Test**: Can be tested by calling clarify step and verifying directive includes ambiguity categories and question format.

**Acceptance Scenarios**:

1. **Given** an initiative with artifacts, **When** agent calls clarify step, **Then** directive guides ambiguity scanning with taxonomy
2. **Given** clarify directive, **When** agent follows it, **Then** questions are generated with options and recommendations

---

### User Story 8 - Initiative Create with Git Branch (Priority: P2)

A developer creates an initiative via MCP. The Go backend automatically creates a git branch (if in a git repo) and returns the branch name as informational context. Git failures are reported as warnings, not errors.

**Why this priority**: Git branch creation is a convenience feature. The workflow should proceed even if git operations fail.

**Independent Test**: Can be tested by calling `initiative create` in a git repo and verifying the response includes `branch` and optionally `git_warning`.

**Acceptance Scenarios**:

1. **Given** a git repository, **When** agent calls `initiative create`, **Then** git branch is created automatically and `branch` field contains the branch name
2. **Given** a non-git directory, **When** agent calls `initiative create`, **Then** initiative is created successfully with empty `branch` field (graceful degradation)
3. **Given** git branch creation fails (dirty tree, branch exists), **When** agent calls `initiative create`, **Then** initiative is created and `git_warning` contains the error message

---

### Edge Cases

- What happens when step is called without active initiative? System returns `NO_ACTIVE_INITIATIVE` error with hint to create one.
- What happens when profile composition fails? Step executes with empty `composed_prompt` (graceful degradation).
- What happens when files_to_read patterns match no files? Empty array is returned; agent proceeds with directive only.
- How does system handle corrupted frontmatter in artifacts? Prerequisite check fails with guidance to fix artifact.
- What happens when directive content is empty? Agent receives empty string; should fall back to profile content or error.
- What happens when git is not installed? Initiative creates successfully; `branch` is empty, `git_warning` explains git is unavailable.
- What happens when working directory has uncommitted changes? Git checkout may fail; `git_warning` populated, initiative still created.

## Requirements *(mandatory)*

### Functional Requirements

#### Profile Structure

- **FR-001**: Each step profile MUST have YAML frontmatter with `name`, `description`, `profiles`, `files`, and `type` fields
- **FR-002**: The `directive` (markdown body) MUST provide phase-by-phase execution guidance for multi-phase steps
- **FR-003**: Profiles MUST be loadable from three sources: embedded (binary), global (~/.brains/steps/), and local (.brains/steps/)
- **FR-004**: Local profiles MUST override global, which MUST override embedded
- **FR-005**: The `files` field MUST contain glob patterns relative to the cycle folder

#### Step Execution

- **FR-010**: The step tool MUST return structured JSON with: directive, initiative_folder, cycle_folder, files_to_read, composed_prompt, prerequisites, workflow_phases (if applicable), next_task (if eat step)
- **FR-011**: Resolved files_to_read MUST be absolute paths to existing files
- **FR-012**: The composed_prompt MUST be the concatenated content from all profiles listed in the step's `profiles` field
- **FR-013**: For feature/bug/refactor steps, workflow_phases MUST include phase definitions with agents and outputs

#### Prerequisite Enforcement

- **FR-020**: The plan step MUST require spec.md with `status: approved` in frontmatter
- **FR-021**: The tasks step MUST require plan.md with `status: approved` in frontmatter
- **FR-022**: The eat step MUST require tasks.md to exist (no status check)
- **FR-023**: When prerequisites fail, error response MUST include code, message, and hint

#### Multi-Phase Workflow Steps

- **FR-030**: The feature step directive MUST describe 4 phases: research, create, audit, highlight
- **FR-031**: Each phase description MUST include: purpose, input, actions, output, success criteria
- **FR-032**: The phase flow MUST include conditional transitions (audit loop on CRITICAL/MAJOR issues)
- **FR-033**: Maximum iteration count (3) MUST be documented in directive

#### Task Tracking (Eat Step)

- **FR-040**: The eat step MUST identify the next incomplete task by finding first `- [ ]` checkbox in tasks.md
- **FR-041**: The next_task response MUST include task ID, description, and phase
- **FR-042**: When all tasks are complete, directive MUST indicate completion and suggest initiative complete

#### Git Operations (Internal)

- **FR-050**: Git operations MUST be handled internally by Go code, never exposed to agents via MCP
- **FR-051**: The `initiative create` response MUST include `branch` field (branch name or empty string)
- **FR-052**: The `initiative create` response MUST include `git_warning` field when git operations fail (empty string on success)
- **FR-053**: Git failures MUST NOT block initiative creation (graceful degradation)
- **FR-054**: Branch naming MUST follow pattern: `{prefix}/{slug}` where prefix maps from initiative type (feature→feat, bug→fix, refactor→ref)

### Key Entities

- **Initiative**: A workflow container with type (feature/bug/refactor), name, and folder in `history/`. Manages lifecycle and git branch.
- **Step**: A workflow step definition with name, directive, profiles, and files. Loaded from markdown with YAML frontmatter.
- **StepResponse**: Structured output from step execution containing directive, paths, files, prompt, phases, and task info.
- **Phase**: A stage within a multi-phase step (research, create, audit, highlight) with agents, outputs, and parallel flag.
- **Profile**: A composable prompt unit with frontmatter and markdown body. Used to build composed_prompt.
- **Prerequisite**: A requirement for step execution (artifact existence, status value).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All 8 step types (feature, bug, refactor, plan, tasks, eat, audit, clarify) return valid responses via MCP tool
- **SC-002**: Prerequisite enforcement blocks out-of-order step execution 100% of the time
- **SC-003**: Feature step directive contains all 4 phases with clear phase boundaries
- **SC-004**: Eat step correctly identifies next incomplete task from tasks.md in all test cases
- **SC-005**: Composed prompts include content from all listed profiles in correct order
- **SC-006**: Files_to_read patterns resolve to actual files when they exist
- **SC-007**: Agents can execute complete workflows (feature → plan → tasks → eat → complete) using only MCP responses
- **SC-008**: Initiative create in git repo returns non-empty `branch` field
- **SC-009**: Initiative create in non-git directory succeeds with empty `branch` field
- **SC-010**: Git failures during initiative create populate `git_warning` field without blocking creation
