# Feature Specification: Business-Requirement Function Comments

**Feature Branch**: `feat/business-comment-requirement-function-comments`
**Created**: 2026-04-03
**Status**: Draft
**Input**: Update all zombiekit workflows so that every function created or updated (except tests) must have an attached comment describing its business requirement in plain language.

## Description

When zombiekit workflows create or update Go functions or methods, the resulting code must include a doc comment on every non-test function/method that describes the function's business purpose — what it does for users or the system — not a technical rephrasing of the implementation.

This extends the existing test-comment business-language requirement (commit 807ef78) to cover all production code.

### What "business-language" means

Describe the **outcome** the function produces from a user's or system's perspective. Ask: "If I removed all code from this function, what would the caller lose?" The answer is the comment.

- Outcome-oriented: `// CreateAccount provisions a new user account.`
- Technical rephrasing: `// CreateAccount inserts a user row and hashes the password.`

The test: if the comment would still be true after a complete reimplementation of the internals, it's business-language. If it would become false, it's a technical description.

## User Scenarios & Testing

### User Story 1 - Workflow-generated functions have business comments (Priority: P1)

A developer uses `/brains.next` to implement a feature. The workflow generates Go functions. Each non-test function has a doc comment that describes what the function accomplishes in business terms, reusing spec verbiage where appropriate.

**Why this priority**: This is the core behavior change — without it, nothing else matters.

**Independent Test**: Generate code via any workflow; inspect that every non-test function has a business-language doc comment.

**Acceptance Scenarios**:

1. **Given** a feature workflow is in the implement step, **When** the agent creates a new exported function, **Then** the function has a doc comment describing its business purpose (not a technical rephrasing).
2. **Given** a feature workflow is in the implement step, **When** the agent creates a new unexported function, **Then** the function has a doc comment describing its business purpose.
3. **Given** a bug workflow is in the fix step, **When** the agent modifies or creates a function, **Then** the function has a business-purpose doc comment.
4. **Given** a refactor workflow is in the implement step, **When** the agent creates or moves a function, **Then** the function retains or gains a business-purpose doc comment.

---

### User Story 2 - Test code defers to existing comment conventions (Priority: P1)

Test code already has its own comment style requirements defined in the spec and task templates ("Test Comment Style"). This feature does not override, duplicate, or conflict with those rules.

**Why this priority**: Avoiding conflict with the existing test comment rules.

**Independent Test**: Generate test code; verify the agent follows the existing test comment conventions, not this new requirement.

**Acceptance Scenarios**:

1. **Given** a workflow creates test code, **When** the existing test comment conventions apply, **Then** the agent follows those conventions and this requirement does not interfere.

---

### User Story 3 - Comments reuse spec verbiage (Priority: P2)

When a function directly implements a functional requirement from the spec, the doc comment should reuse the spec's language rather than inventing new phrasing.

**Why this priority**: Maintains traceability between spec and code without embedding spec IDs.

**Independent Test**: Compare generated doc comments against spec FR descriptions; verify language alignment.

**Acceptance Scenarios**:

1. **Given** a function implements FR-001 ("System MUST allow users to create accounts"), **When** the doc comment is written, **Then** it reads something like `// CreateAccount lets a user create a new account.` — not `// CreateAccount inserts a row into the users table.`

---

### Edge Cases

- What happens when a function is purely infrastructural (e.g., `main`, `init`, simple constructors)? Comment should still describe business purpose, even if brief (e.g., `// NewService sets up the account management service.`).
- What happens when an existing function already has a good doc comment? Leave it alone — only add/update if missing or purely technical.
- What about generated code (`_gen.go`)? Excluded — generated code should not be hand-edited.

## Requirements

### Functional Requirements

- **FR-001**: All workflow templates (feature, feature-light, bug, refactor) MUST instruct the implementing agent to add a business-language doc comment to every non-test function and method it creates or updates.
- **FR-002**: Business-language doc comments MUST describe the outcome the function produces from the caller's perspective — not how it works internally. Litmus test: would the comment still be true after a complete reimplementation? If yes, it's correct.
- **FR-003**: Doc comments SHOULD reuse verbiage from the feature spec's functional requirements where applicable.
- **FR-004**: Test code follows its own existing comment conventions (see "Test Comment Style" in spec and task templates). This requirement does not override or restate those rules.
- **FR-005**: Generated code files MUST be excluded. A file is generated if its name ends in `_gen.go`, `.pb.go`, `_string.go`, or it contains the header `// Code generated ... DO NOT EDIT.`
- **FR-006**: The requirement MUST be added to the implement profile (`embed/profiles/implement.md`) as the single enforcement point, and echoed in spec/task templates for visibility.
- **FR-007**: "Creates or updates" means: if the agent writes or materially modifies a function signature or body, it must ensure a business-language doc comment exists. Trivial changes (fixing a typo, adjusting whitespace) do not trigger this requirement.
- **FR-008**: Interface method declarations MUST also have business-language doc comments describing the contract from the caller's perspective.

### Scope

The change targets these files:
- `embed/workflows/feature.md` or its implement profile
- `embed/workflows/feature-light.md`
- `embed/workflows/bug.md`
- `embed/workflows/refactor.md`
- `embed/profiles/implement.md` (if this is the shared enforcement point)
- `.brains/templates/spec-template.md`
- `.brains/templates/tasks-template.md`

### Out of Scope

- Retroactively adding comments to existing functions (separate initiative)
- Automated linting/CI enforcement of comment quality
- Changing the existing test-comment business-language requirement

## Success Criteria

- **SC-001**: Every workflow's implement step includes explicit instructions requiring business-language doc comments on non-test functions.
- **SC-002**: The instruction uses consistent language across all workflows (ideally from a single shared location).
- **SC-003**: Examples of good and bad comments are included in the instructions.

## Testing Requirements

### Test Strategy

This is a documentation/template change. Verification is by inspection: read each workflow and confirm the requirement is present and correctly scoped.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Inspection | Each workflow template contains the comment requirement |
| FR-002 | Inspection | Examples in templates show business vs. technical phrasing |
| FR-003 | Inspection | Templates mention reusing spec verbiage |
| FR-004 | Inspection | Templates defer test code to existing test comment conventions |
| FR-005 | Inspection | Templates explicitly exclude generated code |
| FR-006 | Inspection | Requirement lives in a shared location referenced by all workflows |

Function Comment Style:
- Function doc comments in code MUST use short business-language
  descriptions of what the function accomplishes — NOT technical rephrasings.
- Good:  `// ImportMessages brings external conversation history into the system.`
- Good:  `// ResolveConflict picks the most recent version when two edits collide.`
- Bad:   `// ImportMessages iterates over the input slice and calls db.Insert for each.`
- Bad:   `// ResolveConflict compares timestamps and returns the newer struct.`
- Spec language should be reused where it fits naturally.
- Test code follows its own existing comment conventions.
- Generated code is excluded from this requirement.
