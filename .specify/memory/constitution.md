<!--
SYNC IMPACT REPORT
==================
Version change: 0.0.0 (template) → 1.0.0
Bump rationale: MAJOR - Initial constitution adoption, replacing template with concrete principles

Modified principles: N/A (initial adoption)
Added sections:
  - Principle I: General Best Practices
  - Principle II: Go Development Standards
  - Principle III: Testing Discipline
  - Principle IV: Database Standards (PostgreSQL)
  - Section: Technology Stack
  - Section: Development Workflow
  - Section: Governance
Removed sections: All placeholder sections replaced

Templates requiring updates:
  - .specify/templates/plan-template.md: ✅ Compatible (Constitution Check section exists)
  - .specify/templates/spec-template.md: ✅ Compatible (no constitution references needed)
  - .specify/templates/tasks-template.md: ✅ Compatible (test-first guidance aligns with Principle III)

Follow-up TODOs: None
-->

# ZombieKit Constitution

## Core Principles

### I. General Best Practices

All code in this project MUST adhere to these universal standards:

**Tool Usage**
- Query mcp-actions for available tools and prefer those over OS utilities/functions
- Query docdocdev for latest documentation and symbol/function signatures for dependencies

**Code Quality**
- Use meaningful variable names that clearly indicate purpose
- Write clear, concise comments explaining WHY code exists, not HOW it works
- Follow consistent formatting, indentation, and naming conventions
- Keep functions small and focused on a single responsibility
- Prioritize readability over cleverness
- Optimize for maintainability first, performance second

**Architecture**
- Compare alternative approaches with pros and cons before implementation
- Prefer composition over inheritance where applicable
- Do NOT add features or enhancements without explicit user request
- Avoid nested scoping beyond 3 levels; extract to separate functions if deeper nesting required
- Avoid if/else blocks exceeding 4 lines; break branches into smaller functions

**Error Handling & Testing**
- Handle errors with proper context and propagation
- Write comprehensive unit tests for all functionality
- Document public interfaces with usage examples

### II. Go Development Standards

All Go code MUST follow these standards:

**Language & Style**
- Use `any` instead of `interface{}` (Go 1.18+)
- Always check errors and return them with additional context
- Document function definitions with purpose and usage instructions
- Use context for cancellation and timeouts
- Prefer slices over arrays when length might change
- Use meaningful struct tags for serialization
- Implement interfaces implicitly (no explicit interface declarations)
- Use named return values for clarity when appropriate

**Pointers & Concurrency**
- Avoid using pointers unless mutation in a closure-style context is necessary
- Leverage Go's concurrency primitives (goroutines, channels) appropriately

**Project Structure**
- Leverage Go modules for dependency management
- Follow standard Go project layout
- main.go files MUST be minimal: gather environment variables/config, pass to app "run function"

**Safety**
- When writing CLI applications, use urfave/cli
- NEVER use "must" functions or methods known to panic in non-test code; use safe versions and check for success
- When comparing errors, ALWAYS use errors.Is or errors.As
- NEVER edit files ending in `_gen.go` or with generated file headers; use the required generation command

### III. Testing Discipline

All testing MUST adhere to these standards:

**Test Execution**
- Pay attention to build tags to determine which tags to include when running tests
- Use `-run` flag to specify tests, but always run against a folder:
  ```bash
  go test -run "TestFoo|TestBar" ./pkg/
  ```
- Disable test cache by using `-count=1`

**Test Tooling**
- Use testify/assert and testify/suite libraries for unit testing

**Test-First Mindset**
- Tests MUST be written before implementation in user story phases
- Verify tests FAIL before implementing the feature

### IV. Database Standards (PostgreSQL)

All PostgreSQL database code MUST follow these standards:

**Schema Safety**
- Always be defensive: use `CREATE TABLE IF NOT EXISTS` for tables and models
- Use `CREATE OR REPLACE` for views and analogous operations for objects that don't support IF NOT EXISTS
- Be smart about creating new objects that might already exist

**Naming Conventions**
- For datetime values, name the column `thing_date` (e.g., `updated_date`)
- For time-only values, name the column `thing_time` (e.g., `updated_time`)

**Primary Keys**
- When creating primary keys that may be exposed to users, use `UUID PRIMARY KEY`
- In Go storage implementations, generate UUID v7 and assign it to the primary key field

## Technology Stack

**Required Technologies**
- **Language**: Go 1.24.0 (per go.mod)
- **CLI Framework**: urfave/cli/v2
- **MCP Server**: mark3labs/mcp-go
- **YAML Parsing**: gopkg.in/yaml.v3, adrg/frontmatter
- **Testing**: testify/assert, testify/suite
- **Databases**: PostgreSQL 16 (production), SQLite with WAL mode (development/single-user)

**Logging & Observability**
- Use slog for structured logging
- Structured logging required for all significant operations

## Development Workflow

**Code Review Requirements**
- All PRs/reviews MUST verify compliance with this constitution
- Complexity MUST be justified in the Complexity Tracking section of implementation plans

**Change Process**
1. Constitution check MUST pass before Phase 0 research
2. Re-check constitution after Phase 1 design
3. Tests written and failing before implementation (user story phases)
4. Commit after each task or logical group

**Documentation**
- Document public interfaces with usage examples
- Write comments explaining WHY, not HOW

## Governance

**Amendment Procedure**
1. Proposed changes MUST be documented with rationale
2. Changes require explicit approval from project maintainers
3. Breaking changes (principle removal/redefinition) require migration plan
4. All amendments MUST update the version according to semantic versioning:
   - MAJOR: Backward incompatible governance/principle removals or redefinitions
   - MINOR: New principle/section added or materially expanded guidance
   - PATCH: Clarifications, wording, typo fixes, non-semantic refinements

**Compliance Review**
- All PRs/reviews MUST verify constitution compliance
- Complexity deviations require explicit justification in implementation plans
- Violations discovered post-merge require immediate remediation plan

**Precedence**
- This constitution supersedes all other practices
- Conflicts between documents resolve in favor of this constitution

**Version**: 1.0.0 | **Ratified**: 2025-12-21 | **Last Amended**: 2025-12-23
