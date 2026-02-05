# Implementation Plan: Workflow Step Tracking

**Spec**: [spec.md](./spec.md)
**Created**: 2026-01-31

## Overview

This plan implements workflow step tracking with three main changes:
1. Simplify `active.json` to minimal pointer
2. Add step definitions to workflow frontmatter
3. Track cycle/step state in INITIATIVE.md

## Implementation Phases

### Phase 1: Data Model Changes

**Goal**: Update types and simplify active.json schema.

#### Task 1.1: Simplify InitiativeState

**File**: `internal/initiative/types.go`

Remove fields from `InitiativeState`, keeping only:
```go
type InitiativeState struct {
    Initiative string    `json:"initiative"` // path to initiative folder
    Status     string    `json:"status"`     // "in_progress" | "complete"
    Started    time.Time `json:"started"`    // when initiative became active
}
```

Remove:
- `Cycle` - no longer tracked here
- `LastActivity` - tracked in INITIATIVE.md
- `CurrentStep` - tracked in INITIATIVE.md

#### Task 1.2: Add InitiativeStatus type

**File**: `internal/initiative/types.go`

```go
type InitiativeStatus string

const (
    InitiativeStatusInProgress InitiativeStatus = "in_progress"
    InitiativeStatusComplete   InitiativeStatus = "complete"
)
```

#### Task 1.3: Update StateManager.Save()

**File**: `internal/initiative/state.go`

Remove the automatic `LastActivity` update since we're simplifying the struct.

---

### Phase 2: INITIATIVE.md Parser

**Goal**: Parse cycles and step tables from markdown.

#### Task 2.1: Create initiativemd package

**File**: `internal/initiative/markdown.go` (new)

Types:
```go
type ParsedInitiative struct {
    Name      string
    Type      string
    Status    string
    Created   string
    Cycles    []ParsedCycle
}

type ParsedCycle struct {
    Number int
    Type   string      // "feat", "ref", "fix"
    Name   string
    Status string      // "active", "completed"
    Steps  []ParsedStep
    Notes  string      // freeform text after table
}

type ParsedStep struct {
    Name    string
    Status  string // "pending", "in_progress", "completed", "skipped"
    Updated string
}
```

Functions:
```go
func ParseInitiativeMD(path string) (*ParsedInitiative, error)
func (p *ParsedInitiative) ActiveCycle() *ParsedCycle
func (c *ParsedCycle) CurrentStep() *ParsedStep
func (c *ParsedCycle) NextStep() *ParsedStep
```

#### Task 2.2: Implement markdown table parser

Parse the cycle sections:
- Header pattern: `### N. type/name (status)`
- Table pattern: `| Step | Status | Updated |`
- Extract rows until next section or EOF

#### Task 2.3: Implement markdown writer

**File**: `internal/initiative/markdown.go`

```go
func (p *ParsedInitiative) UpdateStepStatus(cycleNum int, stepName, status, timestamp string) error
func (p *ParsedInitiative) AddStep(cycleNum int, afterStep string, newStep ParsedStep) error
func (p *ParsedInitiative) WriteTo(path string) error
```

The writer must preserve:
- Non-cycle sections (Description, Goals, etc.)
- Notes after cycle tables
- Formatting/whitespace

---

### Phase 3: Workflow Frontmatter

**Goal**: Add step definitions to workflow profiles.

#### Task 3.1: Define step schema

**File**: `internal/step/types.go`

```go
type WorkflowStep struct {
    Name    string `yaml:"name"`
    Profile string `yaml:"profile"`
}

type WorkflowMeta struct {
    Name        string         `yaml:"name"`
    Description string         `yaml:"description"`
    Steps       []WorkflowStep `yaml:"steps,omitempty"`
}
```

#### Task 3.2: Update feature.md profile

**File**: `embed/profiles/feature.md`

Add to frontmatter:
```yaml
---
name: feature
description: Feature development workflow
steps:
  - name: spec
    profile: feature
  - name: plan
    profile: plan
  - name: tasks
    profile: tasks
  - name: implement
    profile: implement
---
```

#### Task 3.3: Update bug.md profile

**File**: `embed/profiles/bug.md`

```yaml
steps:
  - name: investigate
    profile: bug
  - name: fix
    profile: implement
  - name: verify
    profile: audit
```

#### Task 3.4: Update refactor.md profile

**File**: `embed/profiles/refactor.md`

```yaml
steps:
  - name: analyze
    profile: refactor
  - name: plan
    profile: plan
  - name: implement
    profile: implement
  - name: verify
    profile: audit
```

#### Task 3.5: Parse workflow steps

**File**: `internal/step/service.go`

```go
func (s *Service) GetWorkflowSteps(workflowType string) ([]WorkflowStep, error)
```

Reads the profile frontmatter and extracts `steps` array.

---

### Phase 4: Initiative Creation Changes

**Goal**: Generate INITIATIVE.md with cycle and step table.

#### Task 4.1: Update createInitiativeMD()

**File**: `internal/initiative/service.go`

New template:
```markdown
# Initiative: {name}

**Type**: {type}
**Status**: in_progress
**Created**: {timestamp}

## Cycles

### 1. {cycle-type}/{name} (active)

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | {timestamp} |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |

## Description

<!-- Add description -->

## Goals

<!-- Define goals -->
```

#### Task 4.2: Pass workflow steps to creation

**File**: `internal/mcp/tools/initiative/tool.go`

In `handleCreate()`:
1. Detect workflow type
2. Load workflow steps from profile frontmatter
3. Pass steps to `createInitiativeMD()`

---

### Phase 5: Status and Step Tools

**Goal**: Read/write step state from INITIATIVE.md.

#### Task 5.1: Update initiative status

**File**: `internal/initiative/service.go`

`Status()` method now:
1. Reads `active.json` for initiative path
2. Parses INITIATIVE.md for cycle/step state
3. Returns combined result

Update `StatusResult`:
```go
type StatusResult struct {
    Active         bool     `json:"active"`
    InitiativeID   string   `json:"initiative_id,omitempty"`
    InitiativeType string   `json:"initiative_type,omitempty"`
    CurrentCycle   string   `json:"current_cycle,omitempty"`
    CurrentStep    string   `json:"current_step,omitempty"`
    StepStatus     string   `json:"step_status,omitempty"`
    StepsCompleted int      `json:"steps_completed"`
    StepsTotal     int      `json:"steps_total"`
    // ... existing fields
}
```

#### Task 5.2: Update step tool to write INITIATIVE.md

**File**: `internal/mcp/tools/step/tool.go`

After step execution:
1. Parse INITIATIVE.md
2. Update current step status to "completed"
3. Update next step status to "in_progress" (if any)
4. Write INITIATIVE.md

---

### Phase 6: next.md Workflow Update

**Goal**: Implement complete-or-advance logic.

#### Task 6.1: Update next.md workflow

**File**: `embed/workflows/next.md`

New logic:
1. Parse INITIATIVE.md for current cycle
2. Find step with status "in_progress"
3. Mark it "completed" with timestamp
4. Find next "pending" step (skip "skipped")
5. If found: mark "in_progress", load its profile
6. If not found: suggest `/brains.complete`

---

### Phase 7: Cleanup

**Goal**: Remove deprecated code and update tests.

#### Task 7.1: Remove deprecated fields from service.go

Remove references to:
- `state.Cycle`
- `state.CurrentStep`
- `state.Started`
- `state.LastActivity`

#### Task 7.2: Update tests

Files:
- `internal/initiative/state_test.go`
- `internal/initiative/service_test.go`
- `internal/initiative/cycle_test.go`

Update test fixtures to use new `active.json` format.

#### Task 7.3: Add INITIATIVE.md parser tests

**File**: `internal/initiative/markdown_test.go` (new)

Test cases:
- Parse single cycle
- Parse multiple cycles
- Handle malformed tables
- Update step status
- Add new step
- Preserve notes

---

## Dependency Order

```
Phase 1 (types)
    ↓
Phase 2 (parser) ← No dependencies on existing code
    ↓
Phase 3 (frontmatter) ← Parallel with Phase 2
    ↓
Phase 4 (creation) ← Needs Phase 2 + 3
    ↓
Phase 5 (status/step) ← Needs Phase 2 + 4
    ↓
Phase 6 (next.md) ← Needs Phase 5
    ↓
Phase 7 (cleanup) ← After all phases
```

## Risk Areas

1. **INITIATIVE.md parsing** - Markdown is flexible; edge cases in user-edited files
   - Mitigation: Robust regex, graceful fallbacks, preserve unparsed sections

2. **Backwards compatibility** - Existing initiatives have old format
   - Mitigation: Out of scope per spec; can add migration later if needed

3. **Concurrent edits** - Agent and user both editing INITIATIVE.md
   - Mitigation: Parse fresh on each operation; atomic write

## Success Validation

After implementation:
1. `active.json` only has `initiative` and `status`
2. `/brains.new` creates INITIATIVE.md with step table
3. `/brains.next` updates step status in INITIATIVE.md
4. `initiative status` returns cycle/step info from INITIATIVE.md
5. Agent can add/skip steps by editing the table
