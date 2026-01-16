---
name: plan
description: Create the implementation plan from an approved specification
profiles:
  - research
files:
  - "spec.md"
  - "plan.md"
  - "research.md"
  - "data-model.md"
  - "quickstart.md"
  - "contracts/**/*.md"
type: step
---
# Implementation Planning Workflow

## Context

You are creating an implementation plan from an approved specification. The plan translates the "what" (specification) into the "how" (technical approach). Your plan guides subsequent task generation and implementation.

### Your Responsibilities

- Load and verify the approved specification
- Check the project constitution for architectural constraints
- Define technical approach, dependencies, and project structure
- Break work into implementation phases
- Ensure testability and maintainability

### System Responsibilities (handled by MCP tool)

- Prerequisite verification (spec.md with approved status)
- File path resolution
- Profile composition
- State management

---

## Response Handling

When you receive the MCP response, process fields in this order:

1. **Check `prerequisites.met`**: If false, follow `prerequisites.hint` to unblock
2. **Read `files_to_read`**: Load spec.md, research.md, and any existing plan.md
3. **Follow `directive`**: Execute according to this document
4. **Output to `cycle_folder`**: Save plan.md and related artifacts here
5. **Reference `composed_prompt`**: Additional context from research profile

---

## Prerequisites

| Required | Status | Blocking Step |
|----------|--------|---------------|
| spec.md | approved | feature/bug/refactor |

If prerequisites are not met, the MCP response will contain:
- `prerequisites.met: false`
- `prerequisites.required`: What's needed
- `prerequisites.hint`: How to satisfy the requirement

---

## Workflow

### Step 1: Constitution Check

Before any technical decisions, check for project constitution:

```
IF .specify/memory/constitution.md exists:
    - Load constitution principles
    - Verify plan will adhere to all principles
    - Flag any potential violations
    - Document compliance in plan
ELSE:
    - Note: No constitution defined for this project
    - Apply general best practices
```

### Step 2: Technical Analysis

Analyze the specification to determine:

1. **Language/Stack**: Based on project context and spec requirements
2. **Dependencies**: External libraries, services, APIs
3. **Architecture Pattern**: Based on complexity and requirements
4. **Performance Needs**: Scale, latency, throughput requirements
5. **Constraints**: Platform, deployment, security requirements

### Step 3: Project Structure

Define the file/folder structure:

```text
{project_root}/
├── {source_dir}/
│   ├── {module_1}/
│   └── {module_2}/
├── {test_dir}/
└── {config}/
```

Include rationale for structure decisions.

### Step 4: Implementation Phases

Break work into logical phases:

- **Phase 0**: Research and spike (if needed)
- **Phase 1**: Core functionality
- **Phase 2**: Integration
- **Phase 3**: Polish and testing

Each phase should:
- Be independently verifiable
- Have clear completion criteria
- Build on previous phases

### Step 5: Testing Strategy

Define approach for:
- Unit tests
- Integration tests
- E2E tests (if applicable)
- Test coverage expectations

---

## Output

Create `plan.md` in `cycle_folder` with this structure:

```markdown
# Implementation Plan: {Feature Name}

**Branch**: `{branch}` | **Date**: {YYYY-MM-DD} | **Spec**: [spec.md](./spec.md)

## Summary

{1-2 sentence technical summary}

## Technical Context

**Language/Version**: {e.g., Go 1.24.0}
**Primary Dependencies**: {comma-separated list}
**Storage**: {database/file/memory}
**Testing**: {framework and approach}
**Target Platform**: {OS/environment}
**Constraints**: {key limitations}

## Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| {Principle 1} | PASS/FAIL | {justification} |
| ... | ... | ... |

**Gate Status**: PASS/FAIL

## Project Structure

{ASCII tree structure}

## Implementation Phases

### Phase 0: {Name}
{Description}

### Phase 1: {Name}
{Description}

...

## Artifacts Generated

| Artifact | Location | Status |
|----------|----------|--------|
| data-model.md | {path} | pending |
| contracts/ | {path} | pending |
| quickstart.md | {path} | pending |

## Next Step

Call the `step` MCP tool with `step: "tasks"` to generate the detailed task breakdown.
```

---

## Success Criteria

- [ ] Technical context fully specified
- [ ] Constitution check completed (if constitution exists)
- [ ] Project structure defined with rationale
- [ ] Implementation phases are logical and verifiable
- [ ] Testing strategy defined
- [ ] No implementation code included (plan only)

---

## Behavior Rules

1. **Spec-Driven**: Every technical decision traces to a specification requirement
2. **Constitution First**: Always check constitution before making architecture decisions
3. **No Implementation**: Plan describes approach, not code
4. **Phased Approach**: Break work into verifiable phases
5. **Testability Focus**: Every component must be testable
6. **Document Decisions**: Explain WHY, not just WHAT
7. **Generate Supporting Artifacts**: Create data-model.md, contracts/, quickstart.md as needed
