# Feature Specification: Update Step Types & MCP Tool Interface

**Feature Branch**: `023-update-step-types`
**Created**: 2025-12-23
**Updated**: 2025-12-24 (analysis issues resolved)
**Status**: Draft
**Input**: User description: "Update the steps available to be feature, bug, refactor, plan, tasks, eat (implement), audit, clarify, complete. The feature step should match the specify step from spec-kit."

## Clarifications

### Session 2025-12-23

- Q: Should legacy steps (init, specify, implement) be removed or kept as deprecated aliases? → A: Remove completely (breaking change acceptable)
- Q: How should step prerequisites be enforced? → A: Hard block with guidance (refuse execution and show required next step)

### Session 2025-12-24

- Q: Should the single `step` MCP tool be split to reduce interface overloading? → A: Yes, split into two tools: `initiative` (lifecycle management) and `step` (workflow execution)
- Q: What actions should the `initiative` tool support? → A: create, status, complete, list
- Q: Should creation steps (feature/bug/refactor) remain as step types? → A: Yes, but initiative creation is separate from step execution. User calls `initiative create` first, then `step feature` to run the specification workflow
- Q: What happens when creating an initiative with a duplicate name? → A: Allowed - initiatives are prefixed with a unique hex ID (e.g., `abc123-user-auth`), so name collisions cannot occur
- Q: Should `initiative create` support a force flag when one is already active? → A: No - user must complete or abandon current initiative first (keep interface simple)
- Q: How is artifact "approval" detected for prerequisites? → A: YAML frontmatter `status: approved` field in the artifact file. The prerequisite checker reads frontmatter and validates the status value.
- Q: How does the `eat` step track task progress? → A: The eat step reads tasks.md, identifies the next incomplete task (first `- [ ]` checkbox), and provides implementation guidance for that task. Task completion is tracked via markdown checkboxes in tasks.md (`- [x]`). The eat step does not mutate tasks.md - the agent marks tasks complete as it works.
- Q: How are clarification answers encoded back into artifacts? → A: The clarify step presents questions and captures answers in the Clarifications section of spec.md (or relevant artifact). The agent appends Q/A pairs to the existing clarifications, preserving session context.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start New Feature Workflow (Priority: P1)

A developer initiates a new feature specification workflow by first creating an initiative via `initiative create`, then running `step feature` to get specification guidance. The system creates the folder structure and templates during initiative creation, then provides workflow guidance during step execution.

**Why this priority**: Feature creation is the primary entry point for the workflow. Separating initiative lifecycle from step execution provides clearer tool interfaces.

**Independent Test**: Can be fully tested by calling `initiative(action="create", type="feature", name="user-auth")` followed by `step(step="feature")`, verifying that initiative folder is created with templates, and the step directive guides specification creation.

**Acceptance Scenarios**:

1. **Given** no active initiative, **When** user calls `initiative create` with type=feature and name, **Then** a new initiative folder is created with spec.md and research.md templates
2. **Given** an active feature initiative, **When** user calls `step feature`, **Then** the system provides a directive matching the specify workflow from spec-kit (research, create, audit, highlight phases)

---

### User Story 2 - Create Bug Investigation (Priority: P2)

A developer initiates a bug investigation workflow by calling `initiative create type=bug`, then `step bug` to get investigation guidance. The system creates an initiative focused on root cause analysis and fix specification.

**Why this priority**: Bug fixes are a frequent workflow type. Having a dedicated step ensures bugs are properly investigated before implementation, reducing fix-fail cycles.

**Independent Test**: Can be fully tested by calling `initiative(action="create", type="bug", name="payment-timeout")` followed by `step(step="bug")`, verifying bug investigation guidance is provided.

**Acceptance Scenarios**:

1. **Given** no active initiative, **When** user calls `initiative create` with type=bug and name, **Then** a new bug-type initiative is created with bug-specific templates
2. **Given** an active bug initiative, **When** user calls `step bug`, **Then** the directive guides the user through bug analysis workflow

---

### User Story 3 - Create Refactor Specification (Priority: P2)

A developer initiates a refactoring workflow by calling `initiative create type=refactor`, then `step refactor` to get refactoring guidance. The system creates an initiative focused on before/after comparison and behavior preservation.

**Why this priority**: Refactoring is a common workflow that needs structure to ensure behavior is preserved. A dedicated step prevents ad-hoc refactoring.

**Independent Test**: Can be fully tested by calling `initiative(action="create", type="refactor", name="extract-auth-service")` followed by `step(step="refactor")`, verifying refactor-specific guidance is provided.

**Acceptance Scenarios**:

1. **Given** no active initiative, **When** user calls `initiative create` with type=refactor and name, **Then** a new refactor-type initiative is created
2. **Given** an active refactor initiative, **When** user calls `step refactor`, **Then** the directive focuses on behavior preservation and code structure changes

---

### User Story 4 - Execute Planning Phase (Priority: P1)

After specification approval, a developer runs the plan step to create implementation artifacts including architecture decisions, component design, and implementation approach.

**Why this priority**: Planning bridges specification to implementation. Without it, developers jump from "what" to "how" without a structured design phase.

**Independent Test**: Can be fully tested by running `/brains.plan` on an approved spec and verifying that plan.md is created with implementation design.

**Acceptance Scenarios**:

1. **Given** an active initiative with approved spec, **When** user runs the plan step, **Then** the system generates implementation plan artifacts
2. **Given** plan step is executed, **When** complete, **Then** plan.md contains architecture decisions, component breakdown, and implementation approach

---

### User Story 5 - Generate Task Breakdown (Priority: P1)

A developer runs the tasks step to convert the implementation plan into an ordered list of actionable tasks with dependencies.

**Why this priority**: Tasks make the plan executable. Without task breakdown, developers lack clear work items and dependency ordering.

**Independent Test**: Can be fully tested by running `/brains.tasks` on an approved plan and verifying tasks.md is created with ordered, dependency-tracked tasks.

**Acceptance Scenarios**:

1. **Given** an active initiative with approved plan, **When** user runs the tasks step, **Then** tasks.md is generated with ordered task list
2. **Given** tasks are generated, **When** viewing tasks.md, **Then** each task shows dependencies and acceptance criteria

---

### User Story 6 - Execute Implementation (Priority: P1)

A developer runs the eat (implement) step to execute tasks from the task list, implementing the feature according to the plan. The eat step reads tasks.md, identifies the next incomplete task (first unchecked `- [ ]` item), and provides implementation guidance for that specific task.

**Why this priority**: Implementation is where value is delivered. The eat step provides structure for systematic task execution.

**Task tracking mechanism**: Task progress is tracked via markdown checkboxes in tasks.md. The eat step reads the file to find the next incomplete task but does not modify tasks.md - the agent marks tasks complete (`- [x]`) as it works.

**Independent Test**: Can be fully tested by running `/brains.eat` on an initiative with tasks.md containing unchecked tasks, verifying the system identifies the next incomplete task and provides implementation guidance for it.

**Acceptance Scenarios**:

1. **Given** an initiative with tasks.md containing unchecked tasks, **When** user runs the eat step, **Then** the system identifies the first `- [ ]` task and provides implementation guidance for it
2. **Given** tasks.md has some completed tasks (`- [x]`), **When** user runs the eat step, **Then** the system skips completed tasks and presents the next incomplete one
3. **Given** all tasks in tasks.md are marked complete (`- [x]`), **When** user runs the eat step, **Then** the system indicates all tasks are complete and suggests running `initiative complete`

---

### User Story 7 - Run Audit Check (Priority: P2)

A developer runs the audit step to verify consistency between specification, plan, and implementation.

**Why this priority**: Audits catch drift between artifacts. Regular audits ensure the implementation matches the specification.

**Independent Test**: Can be fully tested by running `/brains.audit` and verifying cross-artifact alignment is checked.

**Acceptance Scenarios**:

1. **Given** an initiative with spec, plan, and implementation, **When** user runs audit step, **Then** the system checks alignment between artifacts
2. **Given** audit finds misalignment, **When** audit completes, **Then** issues are reported with severity and fix suggestions

---

### User Story 8 - Request Clarification (Priority: P2)

A developer runs the clarify step to identify underspecified areas and generate targeted clarification questions. Answers are captured in the Clarifications section of spec.md (or the relevant artifact).

**Why this priority**: Clarification prevents ambiguity from propagating into implementation. Early clarification saves rework.

**Clarification encoding mechanism**: The clarify step appends Q/A pairs to the Clarifications section of the artifact, preserving session context (e.g., "Session 2025-12-24" header with bullet points).

**Independent Test**: Can be fully tested by running `/brains.clarify` on an initiative and verifying questions are generated for ambiguous areas, then verifying answers are appended to the Clarifications section.

**Acceptance Scenarios**:

1. **Given** an initiative with artifacts, **When** user runs clarify step, **Then** the system identifies underspecified areas and presents questions
2. **Given** clarification questions are generated, **When** user provides answers, **Then** answers are appended to the Clarifications section of the relevant artifact with session date header
3. **Given** a clarification session already exists for today, **When** new answers are provided, **Then** they are appended to the existing session section

---

### User Story 9 - Complete Initiative (Priority: P1)

A developer runs `initiative complete` to mark an initiative as finished and clear the active state.

**Why this priority**: Proper completion ensures clean state management and prevents initiative sprawl.

**Independent Test**: Can be fully tested by calling `initiative(action="complete")` on an active initiative and verifying the initiative is marked complete and active state is cleared.

**Acceptance Scenarios**:

1. **Given** an active initiative, **When** user calls `initiative complete`, **Then** initiative status is changed to completed
2. **Given** initiative is completed, **When** checking active state, **Then** no active initiative is set

---

### User Story 10 - Check Initiative Status (Priority: P2)

A developer calls `initiative status` to see the current state of the active initiative, including which step they're on and what documents are available.

**Why this priority**: Status visibility helps developers understand where they are in the workflow and what's available.

**Independent Test**: Can be fully tested by calling `initiative(action="status")` and verifying it returns current step, available docs, and suggested next step.

**Acceptance Scenarios**:

1. **Given** an active initiative, **When** user calls `initiative status`, **Then** current step and available documents are returned
2. **Given** no active initiative, **When** user calls `initiative status`, **Then** system indicates no active initiative and suggests creating one

---

### Edge Cases

- What happens when user calls `step` without an active initiative? System should return error with guidance to run `initiative create` first.
- What happens when user runs `step plan` before spec is approved? System should reject with guidance to complete specification first.
- What happens when user runs `step tasks` before plan is approved? System should reject with guidance.
- What happens when user runs `step eat` without tasks? System should reject with guidance to generate tasks first.
- How does system handle multiple active initiatives? Only one initiative can be active at a time. `initiative create` when one exists should error with guidance to complete or abandon first.
- What happens when user tries to call `initiative complete` on an already-completed initiative? System should inform user it's already complete.
- What happens when user calls `initiative create` while one is active? System should error with guidance to complete or abandon current initiative first.
- What happens when user calls `initiative status` with no active initiative? System should return empty state with suggestion to create one.

## Requirements *(mandatory)*

### Functional Requirements

#### MCP Tool Interface (New)

- **FR-100**: System MUST provide two MCP tools: `initiative` (lifecycle) and `step` (execution)
- **FR-101**: The `initiative` tool MUST support actions: create, status, complete, list
- **FR-102**: The `initiative create` action MUST require `type` (feature|bug|refactor) and `name` parameters
- **FR-103**: The `initiative create` action MUST create initiative folder, cycle folder, git branch, and copy templates
- **FR-104**: The `initiative status` action MUST return active initiative info, current step, and available documents
- **FR-105**: The `initiative complete` action MUST mark initiative as finished and clear active state
- **FR-106**: The `initiative list` action MUST return all initiatives with their status
- **FR-107**: The `step` tool MUST accept `step` name and optional `initiative` override
- **FR-108**: The `step` tool MUST NOT handle initiative creation (no type/name/description params)
- **FR-109**: The `step` tool MUST return: directive, paths, files_to_read, composed_prompt, prerequisites status

#### Step Types

- **FR-001**: System MUST support eight step types: feature, bug, refactor, plan, tasks, eat, audit, clarify
- **FR-002**: The feature step MUST execute the specify workflow (research, create, audit, highlight phases)
- **FR-003**: The bug step MUST provide bug investigation guidance (not create initiative - that's `initiative create type=bug`)
- **FR-004**: The refactor step MUST provide refactor planning guidance focused on behavior preservation
- **FR-005**: The plan step MUST generate implementation design artifacts from approved specifications
- **FR-006**: The tasks step MUST generate ordered, dependency-tracked task lists from approved plans
- **FR-007**: The eat step MUST guide systematic execution of tasks from the task list
- **FR-008**: The audit step MUST verify cross-artifact alignment (spec, plan, tasks, implementation)
- **FR-009**: The clarify step MUST identify underspecified areas and generate targeted questions
- **FR-010**: The complete step is REMOVED - use `initiative complete` action instead
- **FR-011**: System MUST enforce step prerequisites via hard block: plan requires approved spec, tasks requires approved plan, eat requires tasks
- **FR-012**: System MUST remove the init, specify, and implement steps entirely (no backwards compatibility aliases)
- **FR-013**: Each step MUST have a corresponding step definition with directive and profiles
- **FR-014**: The feature step directive MUST match the current specify step from spec-kit
- **FR-015**: Artifact approval MUST be detected via YAML frontmatter `status: approved` field in the artifact file
- **FR-016**: The eat step MUST identify the next incomplete task by finding the first unchecked checkbox (`- [ ]`) in tasks.md
- **FR-017**: The clarify step MUST append Q/A pairs to the Clarifications section of the relevant artifact with session date headers

### Key Entities

- **Step**: A workflow step definition with name, description, profiles, files, and directive. Now includes: feature, bug, refactor, plan, tasks, eat, audit, clarify. (Note: complete is now an initiative action, not a step)
- **Initiative**: A unit of work created via `initiative create`. Has type (feature|bug|refactor), name, status, and associated artifacts. Identified by hex ID prefix (e.g., `abc123-user-auth`), ensuring name uniqueness without collision handling.
- **Cycle**: A workflow pass within an initiative (for iterative refinement).
- **MCP Tool**: An exposed tool interface. Two tools: `initiative` (lifecycle CRUD) and `step` (workflow execution).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create new initiatives via `initiative create` with type and name
- **SC-002**: Users can run feature/bug/refactor specification workflows via `step feature|bug|refactor`
- **SC-003**: Users can check initiative status via `initiative status` and see current step + available docs
- **SC-004**: Users can complete initiatives via `initiative complete`
- **SC-005**: Step prerequisites are enforced (users cannot skip workflow phases)
- **SC-006**: All eight step types are available via the `step` MCP tool (feature, bug, refactor, plan, tasks, eat, audit, clarify)
- **SC-007**: Legacy step names (init, specify, implement) are removed
- **SC-008**: The `step` tool has a simplified interface (no creation params)
- **SC-009**: The `initiative` tool handles all lifecycle operations (create, status, complete, list)
- **SC-010**: Users complete feature workflows from start to finish: `initiative create` → `step feature` → `step plan` → `step tasks` → `step eat` → `initiative complete`
