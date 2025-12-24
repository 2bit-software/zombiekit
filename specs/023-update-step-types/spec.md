# Feature Specification: Update Step Types

**Feature Branch**: `023-update-step-types`
**Created**: 2025-12-23
**Status**: Draft
**Input**: User description: "Update the steps available to be feature, bug, refactor, plan, tasks, eat (implement), audit, clarify, complete. The feature step should match the specify step from spec-kit."

## Clarifications

### Session 2025-12-23

- Q: Should legacy steps (init, specify, implement) be removed or kept as deprecated aliases? → A: Remove completely (breaking change acceptable)
- Q: How should step prerequisites be enforced? → A: Hard block with guidance (refuse execution and show required next step)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start New Feature Workflow (Priority: P1)

A developer initiates a new feature specification workflow by running the feature step. The system creates a new initiative with the appropriate folder structure and provides guidance on the specification creation process.

**Why this priority**: The feature step is the primary entry point for the workflow. Without it, users cannot begin specifying new features, which is the core use case of ZombieKit.

**Independent Test**: Can be fully tested by running `/brains.feature "user-auth"` and verifying that an initiative folder is created with proper templates, and the step directive guides the user through specification creation.

**Acceptance Scenarios**:

1. **Given** no active initiative, **When** user runs the feature step with a name, **Then** a new initiative folder is created with spec.md and research.md templates
2. **Given** the feature step is executed, **When** the step completes, **Then** the system provides a directive matching the specify workflow from spec-kit (research, create, audit, highlight phases)

---

### User Story 2 - Create Bug Investigation (Priority: P2)

A developer initiates a bug investigation workflow to analyze and document a bug before fixing it. The system creates an initiative focused on root cause analysis and fix specification.

**Why this priority**: Bug fixes are a frequent workflow type. Having a dedicated step ensures bugs are properly investigated before implementation, reducing fix-fail cycles.

**Independent Test**: Can be fully tested by running `/brains.bug "payment-timeout"` and verifying that the bug investigation template is created with appropriate sections for reproduction steps, root cause analysis, and fix specification.

**Acceptance Scenarios**:

1. **Given** no active initiative, **When** user runs the bug step with a name, **Then** a new bug-type initiative is created with bug-specific templates
2. **Given** the bug step is executed, **When** the step completes, **Then** the directive guides the user through bug analysis workflow

---

### User Story 3 - Create Refactor Specification (Priority: P2)

A developer initiates a refactoring workflow to plan and document code restructuring without changing behavior. The system creates an initiative focused on before/after comparison and behavior preservation.

**Why this priority**: Refactoring is a common workflow that needs structure to ensure behavior is preserved. A dedicated step prevents ad-hoc refactoring.

**Independent Test**: Can be fully tested by running `/brains.refactor "extract-auth-service"` and verifying the refactor-specific templates are created.

**Acceptance Scenarios**:

1. **Given** no active initiative, **When** user runs the refactor step with a name, **Then** a new refactor-type initiative is created
2. **Given** the refactor step is executed, **When** the step completes, **Then** the directive focuses on behavior preservation and code structure changes

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

A developer runs the eat (implement) step to execute tasks from the task list, implementing the feature according to the plan.

**Why this priority**: Implementation is where value is delivered. The eat step provides structure for systematic task execution.

**Independent Test**: Can be fully tested by running `/brains.eat` on an initiative with tasks and verifying the system guides through task-by-task implementation.

**Acceptance Scenarios**:

1. **Given** an initiative with tasks.md, **When** user runs the eat step, **Then** the system presents the next task and guides implementation
2. **Given** eat step is active, **When** a task is completed, **Then** the task is marked complete and next task is presented

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

A developer runs the clarify step to identify underspecified areas and generate targeted clarification questions.

**Why this priority**: Clarification prevents ambiguity from propagating into implementation. Early clarification saves rework.

**Independent Test**: Can be fully tested by running `/brains.clarify` on an initiative and verifying questions are generated for ambiguous areas.

**Acceptance Scenarios**:

1. **Given** an initiative with artifacts, **When** user runs clarify step, **Then** the system identifies underspecified areas
2. **Given** clarification questions are generated, **When** user provides answers, **Then** answers are encoded back into relevant artifacts

---

### User Story 9 - Complete Initiative (Priority: P1)

A developer runs the complete step to mark an initiative as finished and clear the active state.

**Why this priority**: Proper completion ensures clean state management and prevents initiative sprawl.

**Independent Test**: Can be fully tested by running `/brains.complete` on an active initiative and verifying the initiative is marked complete and active state is cleared.

**Acceptance Scenarios**:

1. **Given** an active initiative, **When** user runs complete step, **Then** initiative status is changed to completed
2. **Given** initiative is completed, **When** checking active state, **Then** no active initiative is set

---

### Edge Cases

- What happens when user runs plan before spec is approved? System should reject with guidance to complete specification first.
- What happens when user runs tasks before plan is approved? System should reject with guidance.
- What happens when user runs eat without tasks? System should reject with guidance to generate tasks first.
- How does system handle multiple active initiatives? Only one initiative can be active at a time.
- What happens when user tries to complete an already-completed initiative? System should inform user it's already complete.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support nine step types: feature, bug, refactor, plan, tasks, eat, audit, clarify, complete
- **FR-002**: The feature step MUST execute the specify workflow (research, create, audit, highlight phases)
- **FR-003**: The bug step MUST create bug-type initiatives focused on investigation and fix specification
- **FR-004**: The refactor step MUST create refactor-type initiatives focused on behavior preservation
- **FR-005**: The plan step MUST generate implementation design artifacts from approved specifications
- **FR-006**: The tasks step MUST generate ordered, dependency-tracked task lists from approved plans
- **FR-007**: The eat step MUST guide systematic execution of tasks from the task list
- **FR-008**: The audit step MUST verify cross-artifact alignment (spec, plan, tasks, implementation)
- **FR-009**: The clarify step MUST identify underspecified areas and generate targeted questions
- **FR-010**: The complete step MUST mark initiatives as finished and clear active state
- **FR-011**: System MUST enforce step prerequisites via hard block (refuse execution and display required prerequisite step): plan requires approved spec, tasks requires approved plan, eat requires tasks
- **FR-012**: System MUST remove the init, specify, and implement steps entirely (no backwards compatibility aliases)
- **FR-013**: Each step MUST have a corresponding step template in templates/steps/
- **FR-014**: The feature step directive MUST match the current specify step from spec-kit

### Key Entities

- **Step**: A workflow step definition with name, description, profiles, files, and directive. Now includes: feature, bug, refactor, plan, tasks, eat, audit, clarify, complete.
- **Initiative**: A unit of work created by feature, bug, or refactor steps. Has type, name, status, and associated artifacts.
- **Cycle**: A workflow pass within an initiative (for iterative refinement).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can start new feature workflows with `/brains.feature` and receive specify-equivalent guidance
- **SC-002**: Users can start bug investigations with `/brains.bug` and receive bug-specific guidance
- **SC-003**: Users can start refactoring with `/brains.refactor` and receive refactor-specific guidance
- **SC-004**: Step prerequisites are enforced (users cannot skip workflow phases)
- **SC-005**: All nine step types are available via the step MCP tool
- **SC-006**: Legacy step names (init, specify, implement) are deprecated or removed
- **SC-007**: Users complete feature workflows from start to finish using the new step sequence
