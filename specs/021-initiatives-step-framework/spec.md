# Feature Specification: Initiatives Step Framework

**Feature Branch**: `021-initiatives-step-framework`
**Created**: 2025-12-23
**Status**: Draft
**Input**: User description: "Introduce an initiatives tool for CLI/MCP operations that support spec-kit like workflows. MCP tool endpoint called like 'mcp_zombiekit__step' with arguments for step and dir. New features/bugs/refactors create folders in './history'. Returns: general directive, history folder location, list of files to read, and composed profile prompt for the step."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Execute a Step in an Initiative (Priority: P1)

A developer working with Claude Code needs to execute a specific step (e.g., "specify", "plan", "implement") within their current initiative. They invoke the MCP tool with the step name and working directory, and receive all the context needed to perform that step.

**Why this priority**: This is the core functionality - without step execution, the framework provides no value. This enables the fundamental workflow of guided, context-aware development.

**Independent Test**: Can be fully tested by calling the MCP endpoint with a step name and directory, verifying it returns all four required outputs (directive, history location, files list, composed profile prompt).

**Acceptance Scenarios**:

1. **Given** a project with a `.brains` folder and an existing initiative in `./history/feature-xyz/`, **When** the developer calls `mcp_zombiekit__step` with step="specify" and dir="/path/to/project", **Then** they receive a response containing: (1) the directive for the specify step, (2) the history folder path, (3) a list of relevant files to read, and (4) the pre-composed profile prompt for this step.

2. **Given** a valid step name and directory, **When** the MCP tool is invoked, **Then** the response is returned in a structured format that can be programmatically consumed.

3. **Given** an unknown step name, **When** the developer calls the MCP endpoint, **Then** they receive a clear error message indicating the step is not recognized and listing available steps.

---

### User Story 2 - Start a New Initiative (Priority: P2)

A developer wants to start working on a new feature, bug fix, or refactor. They initiate a new initiative which creates the appropriate folder structure in `./history` and prepares the initial context for the first step.

**Why this priority**: Creating new initiatives is essential for the workflow, but depends on the step execution capability. Without being able to start initiatives, developers cannot begin new work items.

**Independent Test**: Can be tested by calling the MCP endpoint with a "new" or "init" operation, verifying it creates the expected folder structure in `./history`.

**Acceptance Scenarios**:

1. **Given** a project with a `.brains` folder, **When** the developer initiates a new feature initiative with a name like "user-auth", **Then** a new folder is created at `./history/feature-user-auth/` (or similar naming convention) with any required initial files.

2. **Given** a new initiative is created, **When** the first step is requested, **Then** the history folder path returned points to the newly created initiative folder.

3. **Given** an initiative type of "bug" or "refactor", **When** a new initiative is created, **Then** the folder naming reflects the initiative type (e.g., `./history/bug-login-crash/`).

---

### User Story 3 - Define Custom Steps (Priority: P3)

A team wants to customize the available steps or their behavior for their specific workflow. They can define step configurations that specify what each step does, what files it needs, and what profile prompts to compose.

**Why this priority**: Customization enables teams to adapt the framework to their needs, but the core framework must work first with sensible defaults.

**Independent Test**: Can be tested by creating a step definition file and verifying the framework uses the custom configuration when that step is invoked.

**Acceptance Scenarios**:

1. **Given** a step definition file exists in the project's `.brains` folder, **When** that step is invoked, **Then** the framework uses the custom directive, file list, and profile composition defined in that file.

2. **Given** no custom step definition exists, **When** a built-in step is invoked, **Then** the framework uses default step configurations.

---

### Edge Cases

- What happens when the specified directory does not contain a `.brains` folder? The system should return an error indicating the directory is not a valid brains-enabled project.
- How does the system handle when the `./history` folder does not exist? The system should create it automatically when starting a new initiative.
- What happens when a step is invoked but no active initiative exists? The system should either prompt to create one or return an error with guidance.
- How does the system handle concurrent initiatives in the same project? A state file tracks the "current" initiative (similar to git branch); explicit initiative parameter in MCP calls overrides this default.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide an MCP tool endpoint named `mcp_zombiekit__step` (or similar discoverable name) that accepts step name and directory parameters.
- **FR-002**: System MUST return a structured response containing: (1) general directive for the step, (2) history folder/location path, (3) list of files to read, (4) composed profile prompt for the step.
- **FR-003**: System MUST support creating new initiatives with types: feature, bug, refactor.
- **FR-004**: System MUST create initiative folders in `./history` directory with appropriate naming conventions.
- **FR-005**: System MUST load step definitions from configuration to determine behavior for each step.
- **FR-006**: System MUST compose profile prompts by aggregating relevant profile content based on step requirements.
- **FR-007**: System MUST validate that the specified directory contains a `.brains` folder before executing steps.
- **FR-008**: System MUST provide clear error messages when steps fail, including available alternatives.
- **FR-009**: System MUST support a set of default/built-in steps for common workflows (specify, plan, implement, etc.).
- **FR-010**: Step definitions MUST be extensible, allowing projects to define custom steps.
- **FR-011**: System MUST maintain a state file tracking the "current" active initiative, updated when initiatives are created or switched.
- **FR-012**: System MUST accept an optional initiative parameter in MCP calls to override the current initiative from the state file.
- **FR-013**: System MUST support initiative lifecycle states: active and completed.
- **FR-014**: System MUST provide a mechanism to mark an initiative as completed (e.g., via "complete" step).

### Key Entities

- **Initiative**: Represents a unit of work (feature, bug, refactor) with a type, name, creation timestamp, status (active/completed), and folder location. Lives in the `./history` directory.
- **Step**: A defined stage in the initiative workflow with a name, directive text, required file patterns, and profile composition rules.
- **Step Definition**: Configuration that defines how a step behaves - what directive to provide, what files are relevant, and what profiles to compose.
- **Step Response**: The structured output from executing a step, containing directive, history location, file list, and composed prompt.
- **Initiative State**: A state file tracking the currently active initiative, enabling git-branch-like workflow switching.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can execute any defined step and receive all four required outputs within 2 seconds of invocation.
- **SC-002**: New initiatives can be created with a single command/call, with the folder structure ready for use immediately.
- **SC-003**: 100% of step invocations on valid configurations return properly structured responses without errors.
- **SC-004**: Custom step definitions override default behavior without requiring changes to the core framework.
- **SC-005**: Error messages clearly indicate the problem and suggest corrective actions in at least 90% of error scenarios.

## Clarifications

### Session 2025-12-23

- Q: How does the system determine which initiative to use when multiple exist? → A: State file tracks "current" initiative (like git branch), with explicit parameter override capability.
- Q: What lifecycle states do initiatives have? → A: Simple states: active, completed. Tracked via state file (`.brains/active.json`, gitignored). Use `/brains complete` to mark done. (From existing design docs)

## Assumptions

- The existing profile system (`internal/profile`) will be used for composing profile prompts.
- The `./history` folder convention is acceptable for storing initiative data (not in `.brains` to keep project configuration separate from work-in-progress).
- Step definitions will use a file-based format (likely YAML or TOML frontmatter in markdown files) consistent with existing profile patterns.
- The MCP tool infrastructure from `mark3labs/mcp-go` is already in place and this feature extends it.
- Default step names will align with common development workflows: "init", "specify", "plan", "implement", "review", "complete".
