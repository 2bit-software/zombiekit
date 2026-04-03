# Feature Specification: Graphite Stack Branching

**Feature Branch**: `feat/graphite-stack-branching`
**Created**: 2026-04-03
**Status**: Draft
**Input**: User description: "Allow using graphite for stacking when branching during MCP initiative creation. Ask user if they want to stack before handing off to create_initiative, and use gt create under the hood when stacking."

## Decisions

These decisions were resolved during the spec phase and inform the requirements below.

| ID | Decision | Choice | Rationale |
|----|----------|--------|-----------|
| D1 | Branch creation method | `gt create <branch-name> --no-interactive` | Verified: creates empty branch when no staged changes. Single command, auto-parents to current branch. |
| D2 | Auto-initialization | Ask user to `gt init` only when they explicitly request stacking and repo isn't initialized. Otherwise, don't offer graphite. | Balances convenience with not being opinionated. |
| D3 | Trunk branch behavior | Support graphite stacking from any branch, including trunk | Graphite tracking has value even from trunk (enables `gt submit`, `gt modify`). |
| D4 | How user requests stacking | Keyword detection in prompt (like automode), not an interactive question | User signals intent in their `/brains.new` prompt text. No extra questions. |
| D5 | Graphite status reporting | Startup hook reports graphite availability, repo initialization, and current branch stack status | Provides context upfront so workflows can make informed decisions without runtime detection. |
| D6 | Data flow mechanism | `new.md` detects keyword OR "stacked" signal, appends `USE_GRAPHITE: true` metadata to arguments before dispatching to workflow | Follows existing Linear ticket metadata pattern. |
| D7 | Implicit stacking | If current branch is already graphite-tracked (in a stack), auto-enable stacking without requiring keyword | Continuing a stack is the natural default when you're already in one. |

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Stack with Graphite via Prompt Keyword (Priority: P1)

A developer starts new work via `/brains.new` and includes a stacking keyword in their prompt (e.g., "stack", "graphite", "gt stack"). The startup hook has already reported that graphite is available and initialized. The system detects the keyword, appends `USE_GRAPHITE: true` to the workflow arguments, and the initiative is created with a graphite-tracked branch.

**Why this priority**: This is the core feature — enabling graphite stacking in the initiative creation flow.

**Independent Test**: Run `/brains.new stack: add user notifications` in a graphite-initialized repo, verify the branch appears in `gt log short` as a child of the current branch.

**Acceptance Scenarios**:

1. **Given** the user is on branch `feat/auth-api` in a graphite-initialized repo and the startup hook reported `graphite: available, initialized`, **When** they run `/brains.new stack: add rate limiting`, **Then** the new initiative branch is created via `gt create feat/add-rate-limiting --no-interactive` as a graphite-tracked child of `feat/auth-api`.
2. **Given** the user is on `main` and the startup hook reported graphite ready, **When** they run `/brains.new stack: add dashboard`, **Then** the branch `feat/add-dashboard` is created as a graphite-tracked child of `main`.
3. **Given** graphite stacking is triggered, **When** the initiative is created, **Then** the `CreateResponse` includes `branching_method: "graphite"`.

---

### User Story 2 - Startup Hook Reports Graphite Status (Priority: P1)

When a conversation starts in a git repo, the startup hook checks whether `gt` is in PATH and whether the repo has a `.graphite/` directory. It prints this status alongside the existing conversation ID output.

**Why this priority**: Prerequisite for Story 1 — the workflow needs this information to know whether graphite is an option.

**Independent Test**: Start a new conversation in a graphite-initialized repo, verify the startup hook output includes graphite status.

**Acceptance Scenarios**:

1. **Given** `gt` is installed and the repo has `.graphite/`, **When** the startup hook runs, **Then** it outputs: `graphite: available, initialized`.
2. **Given** `gt` is installed but no `.graphite/` directory exists, **When** the startup hook runs, **Then** it outputs: `graphite: available, not initialized`.
3. **Given** `gt` is not in PATH, **When** the startup hook runs, **Then** it outputs: `graphite: not available`.

---

### User Story 3 - Auto-Init When User Requests Stacking in Uninitialized Repo (Priority: P2)

The user includes a stacking keyword in their prompt, but the startup hook reported graphite as available but not initialized. The system asks the user if they'd like to run `gt init` before proceeding.

**Why this priority**: Graceful handling of the "graphite installed but repo not set up" case. Secondary because most graphite users will have already initialized.

**Independent Test**: Run `/brains.new stack: add feature` in a repo where `gt` is installed but `.graphite/` doesn't exist.

**Acceptance Scenarios**:

1. **Given** the startup hook reported `graphite: available, not initialized` and user prompt contains "stack", **When** `new.md` processes the prompt, **Then** it asks the user: "Graphite is installed but this repo isn't initialized. Run `gt init`?"
2. **Given** the user confirms `gt init`, **When** initialization succeeds, **Then** the flow continues with graphite stacking as normal.
3. **Given** the user declines `gt init`, **When** the flow continues, **Then** the branch is created with regular `git checkout -b` and a note is shown that stacking was skipped.

---

### User Story 4 - Implicit Stacking When Already in a Stack (Priority: P1)

A developer is working on a graphite-tracked branch (already part of a stack). They start new work via `/brains.new` without mentioning stacking. The startup hook reported `graphite: available, initialized, stacked`. The system auto-enables stacking — the new branch is created as a graphite child of the current branch.

**Why this priority**: This is the most ergonomic path — developers already in a stack shouldn't have to say "stack" every time. The default should match the context.

**Independent Test**: From a graphite-tracked branch, run `/brains.new add rate limiting` (no stack keyword), verify the new branch is graphite-tracked as a child.

**Acceptance Scenarios**:

1. **Given** the startup hook reported `graphite: available, initialized, stacked`, **When** the user runs `/brains.new add rate limiting` (no stacking keyword), **Then** `new.md` auto-appends `USE_GRAPHITE: true` and the branch is created with graphite.
2. **Given** the user is on a graphite-tracked branch, **When** they don't use a stacking keyword, **Then** no extra questions are asked — stacking is silently implied.
3. **Given** the user is on a graphite-tracked branch but explicitly says "no stack" or "git branch", **When** the initiative is created, **Then** regular `git checkout -b` is used (opt-out override).

---

### User Story 5 - Regular Branching Unchanged (Priority: P1)

Developers who don't mention stacking in their prompt AND aren't on a graphite-tracked branch experience no change in behavior. The existing `git checkout -b` flow works exactly as before.

**Why this priority**: Regression prevention. The existing flow must remain intact.

**Independent Test**: Run `/brains.new add user notifications` (no stacking keyword, not on a graphite-tracked branch), verify branch is created with `git checkout -b` as before.

**Acceptance Scenarios**:

1. **Given** graphite is not installed, **When** the user starts new work with any prompt, **Then** the flow is identical to current behavior.
2. **Given** graphite is installed and initialized but user is NOT on a graphite-tracked branch, **When** the user starts new work WITHOUT a stacking keyword, **Then** the branch is created with `git checkout -b` as before. `CreateResponse` includes `branching_method: "git"`.

---

### User Story 6 - Graphite Failure Fallback (Priority: P2)

When the user requests stacking but `gt create` fails (corrupted state, incompatible version, etc.), the system falls back to `git checkout -b` with a warning.

**Why this priority**: Robustness. Graphite issues should never block initiative creation.

**Independent Test**: Mock `gt create` failure, verify fallback to `git checkout -b` with warning in response.

**Acceptance Scenarios**:

1. **Given** graphite stacking is requested and `gt create` fails, **When** the initiative is being created, **Then** `git checkout -b` is used instead and a warning is included in the response.
2. **Given** graphite fallback occurred, **When** the `CreateResponse` is returned, **Then** `branching_method` is `"git"` and a `branching_warning` field contains the graphite error message.

---

### Edge Cases

- `gt create` fails mid-execution (e.g., corrupted `.graphite/` state) — fallback to `git checkout -b`, include warning in response.
- User on detached HEAD and requests stacking — graphite stacking not attempted (same as current: `EnsureBranch` returns nil for non-repository states).
- Branch name already exists as a graphite-tracked branch — switch to existing branch via `git checkout`, skip `gt create`.
- Branch name already exists but is NOT graphite-tracked — switch to existing branch via `git checkout`, then run `gt track --parent <current-branch> --no-interactive` to add it to the stack.
- `gt` binary exists but is not executable — `isGraphiteAvailable()` returns false (same as not installed).
- User cancels/interrupts during `gt init` prompt (Story 3) — proceed with regular git branching.
- Stacking keyword appears in feature description but wasn't meant as a directive (e.g., "implement stack trace logging") — the keyword detection should use specific patterns like `stack:` prefix or explicit "use graphite" / "gt stack" to minimize false positives.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Startup hook MUST detect whether `gt` CLI is available in PATH and report status.
- **FR-002**: Startup hook MUST detect whether the current repo is graphite-initialized (`.graphite/` directory exists) and report status.
- **FR-003**: Startup hook MUST detect whether the current branch is graphite-tracked by running `gt info` and checking exit code (0 = tracked, non-zero = untracked).
- **FR-003a**: Startup hook MUST output graphite status in the format `graphite: {available|not available}[, {initialized|not initialized}][, {stacked|not stacked}]` alongside existing conversation ID output.
- **FR-004**: `new.md` MUST detect stacking keywords in user prompt. Keywords: `stack:` prefix, `use graphite`, `gt stack`, `graphite stack`. Detection MUST be case-insensitive.
- **FR-004a**: `new.md` MUST detect anti-stacking keywords in user prompt: `no stack`, `no graphite`, `git branch`. When present, these override both explicit stacking keywords and implicit stacking signals.
- **FR-005**: When stacking keyword is detected and graphite is available + initialized (per startup hook output), `new.md` MUST append `USE_GRAPHITE: true` to the arguments before dispatching to the workflow.
- **FR-005a**: When startup hook reported `stacked` (current branch is graphite-tracked) and no anti-stacking keyword is present, `new.md` MUST auto-append `USE_GRAPHITE: true` even without an explicit stacking keyword.
- **FR-006**: When stacking keyword is detected but graphite is available and NOT initialized, `new.md` MUST ask the user if they want to run `gt init` before proceeding.
- **FR-007**: The `initiative create` MCP tool MUST accept an optional `use_graphite` boolean parameter.
- **FR-008**: When `use_graphite` is true, `GitService` MUST create the branch using `gt create <branch-name> --no-interactive`. The command auto-parents the branch to the current branch.
- **FR-009**: If `gt create` fails when `use_graphite` is true, `GitService` MUST fall back to `git checkout -b` and return both the fallback method used and the graphite error.
- **FR-010**: `CreateResponse` MUST include a `BranchingMethod` field (`json:"branching_method,omitempty"`) with value `"graphite"` or `"git"`. Present on all create responses (both new and idempotent).
- **FR-011**: `CreateResponse` MUST include a `BranchingWarning` field (`json:"branching_warning,omitempty"`) populated when graphite fallback occurred.
- **FR-012**: System MUST NOT alter the existing branching flow when `use_graphite` is false or omitted. All existing behavior preserved.
- **FR-013**: When returning an idempotent (already-existed) initiative, `branching_method` MUST be empty string (no branching occurs on idempotent path).

### Key Entities

- **Startup hook** (`.claude/settings.json` or equivalent): Extended to detect and report graphite status.
- **GitService** (`internal/step/git.go`): New method `EnsureBranchGraphite(initType, name string) (method string, err error)` — returns `"graphite"` on success, `"git"` on fallback. Separate from existing `EnsureBranch()`.
- **Initiative Tool** (`internal/mcp/tools/initiative/tool.go`): New `use_graphite` input parameter. `createNewInitiative()` calls `EnsureBranchGraphite()` when true, `EnsureBranch()` when false.
- **CreateResponse** (`internal/mcp/tools/initiative/types.go`): New fields `BranchingMethod string` and `BranchingWarning string`.
- **new.md command** (`embed/commands/new.md`): Stacking keyword detection section (between branch check and classification). Appends `USE_GRAPHITE: true` metadata.
- **feature.md workflow** (`embed/workflows/feature.md`): Step 3 reads `USE_GRAPHITE` from arguments, passes `use_graphite: true` to `initiative create`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Graphite stacking path adds zero additional interactive prompts compared to non-graphite flow (when graphite is available + initialized). Stacking intent is expressed in the initial prompt.
- **SC-002**: Startup hook correctly reports graphite status — verified by checking output in repos with/without graphite.
- **SC-003**: Existing non-graphite workflow has zero behavioral changes — all current tests pass without modification.
- **SC-004**: Failed graphite operations fall back gracefully — initiative creation never fails due to graphite issues.

## Testing Requirements *(mandatory)*

### Test Strategy

Integration tests for the `GitService` graphite path. Unit tests for graphite detection logic (availability check, repo initialization check). Startup hook tested via shell execution. Workflow markdown changes tested indirectly through E2E usage.

Go test framework (`testing` package). Tests use temporary directories with git repos. Graphite detection tests check PATH and filesystem state.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Unit | `isGraphiteAvailable()` returns correct bool based on PATH |
| FR-002 | Unit | `isGraphiteInitialized()` checks `.graphite/` directory existence |
| FR-003 | Unit | `isGraphiteTracked()` returns true when `gt info` exits 0, false otherwise |
| FR-003a | Integration | Startup hook script outputs correct graphite status string (all 3 states) |
| FR-005a | Integration | When startup hook reports `stacked`, `new.md` auto-appends `USE_GRAPHITE: true` without keyword |
| FR-007 | Integration | `initiative create` with `use_graphite=true` passes flag to GitService |
| FR-008 | Integration | `EnsureBranchGraphite()` creates branch via `gt create` in a graphite-initialized test repo |
| FR-009 | Integration | When `gt create` fails, `EnsureBranchGraphite()` falls back to `git checkout -b` and returns `"git"` + error |
| FR-010 | Unit | `CreateResponse` contains `branching_method: "graphite"` when graphite used, `"git"` otherwise |
| FR-011 | Unit | `CreateResponse` contains `branching_warning` only when fallback occurred |
| FR-012 | Integration | Existing `EnsureBranch()` behavior unchanged when `use_graphite` is false/omitted |
| FR-013 | Unit | Idempotent create response has empty `branching_method` |

### Edge Case Coverage

- `gt` binary exists but is not executable → `isGraphiteAvailable()` returns false
- `.graphite/` exists but is corrupted → `gt create` fails, fallback to git
- Branch already exists and is graphite-tracked → switch to existing branch, skip `gt create`
- Branch already exists but not graphite-tracked → switch + `gt track --parent <current> --no-interactive`
- User on detached HEAD → graphite stacking not attempted
- Stacking keyword false positive (e.g., "stack trace") → mitigated by requiring specific patterns (`stack:` prefix, `use graphite`, `gt stack`)

## Out of Scope

- Graphite submit/push integration (separate feature — this is branching only)
- Graphite merge/conflict resolution
- Persistent "always use graphite" preference in config (follow-up)
- Integration with `gt modify`, `gt absorb`, or other stack management commands
- Graphite version detection or minimum version enforcement
