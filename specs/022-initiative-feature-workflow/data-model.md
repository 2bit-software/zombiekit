# Data Model: Initiative Feature Workflow (022)

**Date**: 2025-12-23
**Status**: Draft

## Entity Overview

This feature extends the existing initiative framework with **cycle** support and enhances the step response for the feature workflow.

```
┌──────────────────────────────────────────────────────────────────────┐
│                         INITIATIVE                                    │
│  (Top-level container at ./history/{hex}-{name}/)                    │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │ INITIATIVE.md (frontmatter = source of truth for status)       │  │
│  │   status: active | blocked | completed                         │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                       │
│  ┌──────────────────────┐  ┌──────────────────────┐                  │
│  │      CYCLE 1         │  │      CYCLE 2         │                  │
│  │  (feat-{name}/)      │  │  (ref-{name}/)       │                  │
│  │                      │  │                      │                  │
│  │  - spec.md           │  │  - spec.md           │                  │
│  │  - research.md       │  │  - research.md       │                  │
│  │  - audit/            │  │  - audit/            │                  │
│  └──────────────────────┘  └──────────────────────┘                  │
└──────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
                         ┌─────────────────────────────┐
                         │   ACTIVE POINTER            │
                         │ (.brains/active.json)       │
                         │                             │
                         │ initiative: path only       │
                         │ cycle: path only            │
                         │ (NO status - read from MD)  │
                         └─────────────────────────────┘
```

## Entities

### Initiative (Existing - No Changes)

The top-level container for a unit of work. Type-agnostic; can contain multiple cycles.

```go
// Location: internal/initiative/types.go (existing)
type Initiative struct {
    ID        string           `json:"id"`          // e.g., "675d8a3f-feature-user-auth"
    Type      InitiativeType   `json:"type"`        // feature, bug, refactor
    Name      string           `json:"name"`        // "user-auth"
    Path      string           `json:"path"`        // Absolute path
    Status    InitiativeStatus `json:"status"`      // active, completed
    CreatedAt time.Time        `json:"created_at"`
    UpdatedAt time.Time        `json:"updated_at"`
}
```

### Cycle (NEW)

A single workflow pass within an initiative. Contains artifacts for one specification workflow.

```go
// Location: internal/initiative/types.go (new)
type Cycle struct {
    ID           string      `json:"id"`            // e.g., "675d8a40-feat-user-auth"
    Type         CycleType   `json:"type"`          // feat, ref, fix
    Name         string      `json:"name"`          // "user-auth"
    Path         string      `json:"path"`          // Absolute path to cycle folder
    Status       CycleStatus `json:"status"`        // template, in_progress, audited, approved
    InitiativeID string      `json:"initiative_id"` // Parent initiative ID
    Number       int         `json:"number"`        // Cycle number within initiative (1, 2, 3...)
    CreatedAt    time.Time   `json:"created_at"`
    UpdatedAt    time.Time   `json:"updated_at"`
}

type CycleType string

const (
    CycleFeat CycleType = "feat"  // Feature cycle
    CycleRef  CycleType = "ref"   // Refactor cycle
    CycleFix  CycleType = "fix"   // Bug fix cycle
)

type CycleStatus string

const (
    CycleStatusTemplate   CycleStatus = "template"    // Blank templates created
    CycleStatusInProgress CycleStatus = "in_progress" // Workflow executing
    CycleStatusAudited    CycleStatus = "audited"     // Passed audit
    CycleStatusApproved   CycleStatus = "approved"    // User approved
)
```

### InitiativeState (Simplified - Pointer Only)

Tracks ONLY which initiative and cycle are currently active. Status is NOT stored here.

**Design Decision**: Status lives in `INITIATIVE.md` frontmatter as the single source of truth.
This avoids duplication and sync issues between the state file and markdown files.

```go
// Location: internal/initiative/types.go (simplified)
type InitiativeState struct {
    // Pointer to active initiative (path only)
    Initiative   string    `json:"initiative,omitempty"`     // Relative path to initiative folder

    // Pointer to active cycle within the initiative
    Cycle        string    `json:"cycle,omitempty"`          // Relative path to active cycle folder

    // Timestamps for activity tracking
    Started      time.Time `json:"started,omitempty"`        // When this initiative was set as active
    LastActivity time.Time `json:"last_activity,omitempty"`  // Last step execution timestamp

    // Current step name (for resume capability)
    CurrentStep  string    `json:"current_step,omitempty"`   // Last executed step name
}

// NOTE: Status fields have been REMOVED from InitiativeState:
// - Initiative status → read from INITIATIVE.md frontmatter
// - Cycle status → read from cycle artifacts (spec.md, research.md frontmatter)
// - Type, Name → derived from folder names or INITIATIVE.md
```

### StepResponse (Extended)

Response structure from executing a step via MCP.

```go
// Location: internal/step/types.go (extended)
type StepResponse struct {
    // Existing fields
    Directive      string   `json:"directive"`        // Step instruction text
    HistoryFolder  string   `json:"history_folder"`   // Renamed: Initiative root path
    FilesToRead    []string `json:"files_to_read"`    // Files to read
    ComposedPrompt string   `json:"composed_prompt"`  // Merged profile content

    // NEW: Cycle support
    InitiativeFolder string   `json:"initiative_folder"` // Explicit initiative path
    CycleFolder      string   `json:"cycle_folder"`      // Active cycle path
    WorkflowPhases   []Phase  `json:"workflow_phases"`   // Phase descriptions
}

type Phase struct {
    Name        string   `json:"name"`        // "research", "create", "audit", "highlight"
    Description string   `json:"description"` // Human-readable phase description
    Agents      []string `json:"agents"`      // Agent types to spawn
    Outputs     []string `json:"outputs"`     // Expected artifacts
    Parallel    bool     `json:"parallel"`    // Whether agents run in parallel
}
```

## File Artifacts

### INITIATIVE.md (With Frontmatter - Source of Truth)

Located at initiative root. Contains metadata and cycle list.

**Status is stored in YAML frontmatter** - this is the single source of truth for initiative status.

```markdown
---
status: active
type: feature
created: 2025-12-23T10:30:00Z
updated: 2025-12-23T14:45:00Z
---

# Initiative: {name}

**ID**: {hex}-{name}

## Description

{Description text}

## Goals

{Goal list}

## Cycles

| # | Type | Status | Created |
|---|------|--------|---------|
| 1 | feat | approved | 2025-12-23 |
| 2 | ref | in_progress | 2025-12-24 |

## Progress

{Progress notes}
```

**Frontmatter Fields:**
- `status`: `active` | `blocked` | `completed` (REQUIRED - source of truth)
- `type`: `feature` | `bug` | `refactor` (initiative type)
- `created`: ISO timestamp when initiative was created
- `updated`: ISO timestamp of last status change

### research.md (NEW - Cycle Artifact)

Located in cycle folder. Collated research findings.

```markdown
---
status: template|in_progress|complete
updated: {ISO timestamp}
---

# Research: {Feature Name}

## Executive Summary
{2-3 sentence overview}

## Findings

### Category 1
- Finding with [source]

### Category 2
- Finding with [source]

## Decision Points
- {Decision needed} - Options: A, B, C

## Recommendations
- {Recommended approach with rationale}

## Sources
- {Source 1}
- {Source 2}
```

### spec.md (Existing Template - Used in Cycle)

Located in cycle folder. Feature specification.

```markdown
---
status: template|draft|audited|approved
updated: {ISO timestamp}
---

# Feature Specification: {Feature Name}

{Existing spec template structure}
```

### audit/{date}.md (NEW - Cycle Artifact)

Located in cycle folder's audit/ subdirectory. Audit report.

```markdown
---
date: {ISO date}
iteration: {1|2|3}
result: pass|fail
---

# Audit Report: {Feature Name}

## Summary
- Critical: {count}
- Major: {count}
- Minor: {count}

## Findings

### CRITICAL

#### [C1] {Title}
- **Location**: {artifact}:{section}
- **Issue**: {description}
- **Fix**: {suggestion}

### MAJOR
...

## Alignment Matrix
| Requirement | Coverage |
|-------------|----------|
| R1 | Yes |

## Recommendations
{Next steps based on findings}
```

## State Transitions

### Cycle Status Flow

```
template → in_progress → audited → approved
     │           │
     │           └──→ (loop back if CRITICAL/MAJOR)
     │
     └──→ (never directly to audited/approved)
```

### Initiative Status Flow

```
active ──→ blocked ──→ active
  │                      │
  └───────→ completed ←──┘
```

## Relationships

```
Initiative 1:N Cycle
    - An initiative contains one or more cycles
    - Cycles are ordered by creation (cycle_number)
    - Only one cycle is active at a time

Cycle 1:1 CycleType
    - Each cycle has exactly one type (feat, ref, fix)
    - Type determines git branch prefix

Cycle 1:N Artifact
    - Each cycle contains: spec.md, research.md, audit/
    - Artifacts are created during template copying
    - Artifacts are populated during workflow execution

InitiativeState 1:1 Cycle
    - State tracks the currently active cycle
    - Changing cycles updates state
```

## Folder Structure

### After Feature Step Creation

```
./history/
└── {hex}-{name}/                          # Initiative folder
    ├── INITIATIVE.md                      # Initiative metadata
    └── {hex}-feat-{name}/                 # First cycle (feature)
        ├── research.md                    # [blank template]
        ├── spec.md                        # [blank template]
        └── audit/                         # [empty directory]
```

### After Multiple Cycles

```
./history/
└── {hex}-{name}/                          # Initiative folder
    ├── INITIATIVE.md                      # Updated with cycle history
    ├── {hex1}-feat-{name}/                # Cycle 1 (feature) - complete
    │   ├── research.md
    │   ├── spec.md
    │   └── audit/
    │       └── 2025-12-23.md
    └── {hex2}-ref-{name}/                 # Cycle 2 (refactor) - active
        ├── research.md
        ├── spec.md
        └── audit/
```

## Validation Rules

### Cycle
- `ID` must be unique within initiative
- `Type` must be valid CycleType (feat, ref, fix)
- `Name` must be normalized slug (lowercase, alphanumeric, hyphens)
- `Number` must be sequential (1, 2, 3...)
- `InitiativeID` must reference existing initiative

### InitiativeState (active.json)
- `Initiative` path must exist in filesystem
- `Cycle` path must exist within the initiative folder
- File contains NO status fields (status read from INITIATIVE.md)

### INITIATIVE.md
- Must exist in initiative folder
- Must have YAML frontmatter with `status` field
- `status` must be valid: `active`, `blocked`, or `completed`
- Cycles table must match actual cycle folders

## ID Generation

### Initiative ID
```
{hex-timestamp}-{type}-{normalized-name}
Example: 675d8a3f-feature-user-auth
```

### Cycle ID
```
{hex-timestamp}-{cycle-type}-{normalized-name}
Example: 675d8a40-feat-user-auth
```

### Hex Timestamp Format
```go
timestamp := fmt.Sprintf("%08x", time.Now().Unix())
```
