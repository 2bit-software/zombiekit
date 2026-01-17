# Feature Specification: Core Repository Setup

**Feature Branch**: `001-core-repo-setup`
**Created**: 2025-12-21
**Status**: Draft
**Input**: User description: "Create core repository structure with Taskfile, Docker, and test harnesses for ZombieKit"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Developer Clones and Builds Project (Priority: P1)

A developer clones the ZombieKit repository and wants to build the `brains` CLI binary from source. They should be able to initialize the project, download dependencies, and build a working binary with a single command.

**Why this priority**: Without a buildable project, no other development work can proceed. This is the absolute minimum viable product.

**Independent Test**: Can be fully tested by running `task build` after clone and verifying a binary is produced in `./bin/brains` that executes and displays help.

**Acceptance Scenarios**:

1. **Given** a fresh clone of the repository, **When** the developer runs `task init && task build`, **Then** a working binary is produced at `./bin/brains`
2. **Given** the built binary, **When** the developer runs `./bin/brains --help`, **Then** usage information is displayed
3. **Given** the built binary, **When** the developer runs `./bin/brains version`, **Then** version information including git commit hash is displayed

---

### User Story 2 - Developer Runs Tests (Priority: P2)

A developer wants to run the test suite to verify changes haven't broken existing functionality. They should be able to run all tests with coverage reporting.

**Why this priority**: Test infrastructure enables quality assurance and is foundational for all future development.

**Independent Test**: Can be fully tested by running `task test` and verifying tests execute and coverage report is generated.

**Acceptance Scenarios**:

1. **Given** the project is initialized, **When** the developer runs `task test`, **Then** all tests execute and results are displayed
2. **Given** the test run completes, **When** tests pass, **Then** a coverage report file is generated
3. **Given** test infrastructure exists, **When** a test harness file is created for a new package, **Then** it follows the established testing patterns

---

### User Story 3 - Developer Starts Database Services (Priority: P3)

A developer needs PostgreSQL running locally to work on Tier 2 features (conversation import, MCP server with sticky-memory). They should be able to start required services with a single command.

**Why this priority**: Database services enable Tier 2 feature development but are not required for core CLI functionality.

**Independent Test**: Can be fully tested by running `task db:up` and verifying PostgreSQL container is running and accepting connections.

**Acceptance Scenarios**:

1. **Given** Docker is running, **When** the developer runs `task db:up`, **Then** PostgreSQL container starts with pgvector extension
2. **Given** PostgreSQL is running, **When** the developer runs `task db:migrate`, **Then** database schema is created/updated
3. **Given** services are running, **When** the developer runs `task db:down`, **Then** services are stopped and cleaned up

---

### User Story 4 - Developer Runs Code Quality Checks (Priority: P3)

A developer wants to ensure their code meets project standards before committing. They should be able to run linting, formatting, and security scans.

**Why this priority**: Code quality tooling enables consistent codebase but is not strictly required for basic functionality.

**Independent Test**: Can be fully tested by running `task lint` on existing code and verifying linter output.

**Acceptance Scenarios**:

1. **Given** Go code exists, **When** the developer runs `task fmt`, **Then** code is formatted according to project standards
2. **Given** Go code exists, **When** the developer runs `task lint`, **Then** golangci-lint runs with project configuration
3. **Given** Go code exists, **When** the developer runs `task vet`, **Then** go vet identifies potential issues

---

### User Story 5 - CI Pipeline Runs All Checks (Priority: P4)

The CI system needs to run a comprehensive validation suite on every pull request. A single command should execute all quality gates.

**Why this priority**: CI automation ensures consistent quality but builds on individual quality check tasks.

**Independent Test**: Can be fully tested by running `task ci` and verifying all checks execute in sequence.

**Acceptance Scenarios**:

1. **Given** the project is checked out, **When** CI runs `task ci`, **Then** formatting, linting, vetting, tests, and build all execute
2. **Given** any check fails, **When** `task ci` runs, **Then** the task fails with appropriate error message
3. **Given** all checks pass, **When** `task ci` completes, **Then** exit code is 0

---

### Edge Cases

- What happens when Docker is not installed? Commands requiring Docker should fail gracefully with helpful error message.
- What happens when Go version is incompatible? Build should check Go version and warn if below minimum required.
- How does the system handle missing development tools (golangci-lint, etc)? `task init` should install required tools or provide installation instructions.
- What happens when database port is already in use? Docker compose should use non-conflicting ports (e.g., 9432 instead of 5432).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST build a single statically-linked binary named `brains` for the host platform
- **FR-002**: System MUST support Go 1.22+ as the minimum Go version
- **FR-003**: System MUST use Taskfile for all build, test, and development automation
- **FR-004**: System MUST provide `task init` to download dependencies and install development tools
- **FR-005**: System MUST provide `task build` to compile the CLI binary with version/commit information embedded
- **FR-006**: System MUST provide `task test` to run all tests with coverage reporting
- **FR-007**: System MUST provide `task lint` to run golangci-lint with project configuration
- **FR-008**: System MUST provide `task fmt` to format all Go code
- **FR-009**: System MUST provide `task vet` to run go vet on all packages
- **FR-010**: System MUST provide `task ci` to run all quality checks in sequence
- **FR-011**: System MUST provide `task db:up` to start PostgreSQL with pgvector extension via Docker Compose
- **FR-012**: System MUST provide `task db:down` to stop database services
- **FR-013**: System MUST provide `task db:migrate` to run database migrations
- **FR-014**: System MUST include a `.gitignore` appropriate for Go projects
- **FR-015**: System MUST include a `docker-compose.yml` for development services
- **FR-016**: System MUST include golangci-lint configuration for consistent code quality
- **FR-017**: System MUST include test harness files for each planned package (profile, spec, mcp, web, conversation)
- **FR-018**: System MUST use urfave/cli/v2 for CLI command structure
- **FR-019**: System MUST use pgx/v5 for PostgreSQL database access
- **FR-020**: System MUST use mark3labs/mcp-go for MCP protocol implementation

### Key Entities

- **CLI Binary**: The compiled `brains` executable with embedded version information
- **Taskfile**: YAML configuration defining all build, test, and development tasks
- **Docker Compose**: Service definitions for PostgreSQL and future optional services
- **Test Harness**: Placeholder test files establishing testing patterns for each package
- **Golangci Config**: Linter configuration defining code quality standards

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can clone and build the project in under 2 minutes (excluding dependency download)
- **SC-002**: `task --list` displays all available tasks with descriptions
- **SC-003**: `task build` produces a working binary that responds to `--help` and `version` commands
- **SC-004**: `task test` runs at least 5 test harness files and generates coverage output
- **SC-005**: `task ci` completes successfully on a fresh clone with properly configured environment
- **SC-006**: PostgreSQL container starts within 30 seconds of running `task db:up`
- **SC-007**: All test harnesses compile and run (even if tests are minimal/skipped initially)
- **SC-008**: Project structure matches the directory layout defined in MASTER-DESIGN.md

## Assumptions

- Docker is available for database services (following tiered architecture where Tier 1 works without Docker)
- Go 1.22+ is installed and `go` is in PATH
- `task` (go-task) is installed for task automation
- Developer has basic familiarity with Go project structure
- The golangci-lint configuration follows patterns from mcp-genie reference project
- PostgreSQL uses non-standard port (9432) to avoid conflicts with local installations
