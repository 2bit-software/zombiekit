# Research: Initiative Feature Workflow (022)

**Date**: 2025-12-23
**Status**: Complete

## Executive Summary

Research for implementing the "feature" step in ZombieKit's initiative framework. The feature step creates initiative folders with cycles, copies templates, manages git branches, and returns multi-phase directives that guide LLMs through a research→create→audit→highlight workflow. Key decisions involve git integration approach, template copying patterns, and directive structuring.

## Findings

### 1. Git Integration Approach

**Decision**: Use os/exec with graceful degradation

**Rationale**:
- Zombiekit's use case (creating/switching branches) is straightforward and doesn't need go-git's complexity
- Native git is faster for basic operations
- Users working on code projects typically have git installed
- Smaller binary footprint with no additional dependencies
- Easy to expand to other git operations if needed

**Alternatives Considered**:
| Approach | Pros | Cons | Decision |
|----------|------|------|----------|
| os/exec | Simple, full git compatibility, fast | Requires git installed | **SELECTED** |
| go-git | Pure Go, portable, type-safe | 2-100x slower, incomplete features, larger binary | Rejected |

**Implementation Pattern**:
```go
type GitService struct {
    workDir string
}

func (g *GitService) EnsureBranch(stepType, name string) error {
    if !g.isGitAvailable() || !g.isGitRepository() {
        return nil // Graceful degradation
    }
    branchName := fmt.Sprintf("%s/%s", prefix, slug) // feat/name, fix/name, ref/name
    if g.branchExists(branchName) {
        return g.switchToBranch(branchName)
    }
    return g.createBranch(branchName)
}
```

**Branch Naming Conventions**:
- `feat/<name>` for features
- `fix/<name>` for bug fixes
- `ref/<name>` for refactoring

### 2. Template Copying Pattern

**Decision**: Reuse existing `copyEmbeddedFiles` pattern from `internal/cli/init.go`

**Rationale**:
- Battle-tested pattern already in codebase
- Handles recursive copying, existing file detection, error aggregation
- Uses `fs.WalkDir()` for embedded filesystem traversal
- Consistent with existing ZombieKit patterns

**Key Implementation Elements**:
```go
// Copy pattern from internal/cli/init.go
err := fs.WalkDir(embeddedFS, "templates/templates", func(path string, d fs.DirEntry, err error) error {
    // Skip directories (only copy files)
    if d.IsDir() {
        return nil
    }
    // Get relative path, read from embedded FS, write to destination
    // Use os.MkdirAll for parent directories
    // Use 0o755 for directories, 0o644 for files
})
```

**Templates to Copy**:
| Template Source | Destination | Notes |
|-----------------|-------------|-------|
| `templates/templates/spec-template.md` | `{cycle}/spec.md` | Feature specification |
| NEW: `templates/templates/research-template.md` | `{cycle}/research.md` | Research findings |
| N/A (create directory) | `{cycle}/audit/` | Empty directory for audit reports |

### 3. Step Directive Structure

**Decision**: Use multi-phase directive with explicit transition rules

**Rationale**:
- Existing steps (audit.md, implement.md, research.md) use clear phase markers
- LLM interprets phases through headers, action items, and conditional logic
- The feature step is unique in having loop-back conditions (audit → research)
- Explicit behavior rules enforce constraints (max 3 loops)

**Frontmatter Pattern**:
```yaml
---
name: feature
description: Execute the research-create-audit-highlight workflow for a new feature specification
profiles:
  - research
  - create
  - audit
files:
  - "research.md"
  - "spec.md"
  - "audit/**/*.md"
  - "../**/research.md"    # Previous cycle artifacts
  - "../**/spec.md"
type: step
---
```

**Directive Structure**:
```
## Context
[Files available, LLM responsibilities]

## Phase I: Research (Parallel Agents)
[Input, Actions, Collation, Output, Success criteria]

## Phase II: Create (Single Agent)
[Input, Actions, Output, Success criteria]

## Phase III: Audit (Parallel Agents)
[Input, Actions, Classification, Conditional transitions]

## Phase IV: Highlight (Single Agent)
[Input, Actions, User approval gate]

## Behavior Rules
[Constraints: max 3 loops, never skip phases, cite sources]
```

### 4. Cycle Management Architecture

**Decision**: Extend existing InitiativeState with cycle tracking

**Rationale**:
- Current `InitiativeState` tracks active initiative but not cycles
- Spec requires: `./history/{hex}-{name}/{hex}-{type}-{name}/`
- Cycles are typed (feature, refactor, bug) within type-agnostic initiatives
- Multiple cycles per initiative enable iterative development

**State Extension**:
```go
type InitiativeState struct {
    Initiative   string          `json:"initiative,omitempty"`
    Type         InitiativeType  `json:"type,omitempty"`
    Name         string          `json:"name,omitempty"`
    // NEW: Cycle tracking
    Cycle        string          `json:"cycle,omitempty"`        // Active cycle path
    CycleType    CycleType       `json:"cycle_type,omitempty"`   // feat, ref, fix
    CycleNumber  int             `json:"cycle_number,omitempty"` // 1, 2, 3...
    // Existing fields...
}
```

**Cycle Type**:
```go
type CycleType string

const (
    CycleFeat CycleType = "feat"  // Feature cycle
    CycleRef  CycleType = "ref"   // Refactor cycle
    CycleFix  CycleType = "fix"   // Bug fix cycle
)
```

### 5. StepResponse Extension

**Decision**: Add cycle_folder and workflow_phases to StepResponse

**Rationale**:
- Spec requires returning: directive, initiative_folder, cycle_folder, files_to_read, composed_prompt, workflow_phases
- Current StepResponse has: Directive, HistoryFolder, FilesToRead, ComposedPrompt
- Need to distinguish initiative folder from cycle folder

**Extended Response**:
```go
type StepResponse struct {
    Directive        string   `json:"directive"`
    HistoryFolder    string   `json:"history_folder"`       // Initiative root (renamed for clarity)
    CycleFolder      string   `json:"cycle_folder"`         // NEW: Active cycle folder
    FilesToRead      []string `json:"files_to_read"`
    ComposedPrompt   string   `json:"composed_prompt"`
    WorkflowPhases   []Phase  `json:"workflow_phases"`      // NEW: Structured phase descriptions
}

type Phase struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Agents      []string `json:"agents"`      // Agent types to spawn
    Outputs     []string `json:"outputs"`     // Expected artifacts
}
```

## Decision Points

### D1: Git Operations
- **Selected**: os/exec with graceful degradation
- **Impact**: Simple implementation, requires git installed for full functionality

### D2: Template Storage
- **Selected**: Embedded FS with local override (`.brains/templates/`)
- **Impact**: Consistent with existing pattern, allows customization

### D3: Cycle Auto-Detection
- **Selected**: Auto-detect based on step type + active initiative state
- **Impact**: Less friction for users, explicit override with `--new-initiative` flag

### D4: Directive Format
- **Selected**: Multi-phase markdown with clear transitions
- **Impact**: Self-documenting, LLM-parseable, supports loop-back conditions

## Recommendations

1. **Create new files**:
   - `internal/initiative/cycle.go` - Cycle management
   - `internal/step/feature.go` - Feature step implementation
   - `internal/step/git.go` - Git service
   - `templates/steps/feature.md` - Step definition
   - `templates/templates/research-template.md` - Research output template

2. **Extend existing files**:
   - `internal/initiative/types.go` - Add Cycle, CycleType types
   - `internal/initiative/state.go` - Extend InitiativeState with cycle tracking
   - `internal/step/types.go` - Extend StepResponse
   - `internal/mcp/tools/step/tool.go` - Handle feature step parameters

3. **Implementation order**:
   1. Types and cycle management (foundation)
   2. Template copying (needed for cycle creation)
   3. Git integration (optional, graceful degradation)
   4. Step response extension (API contract)
   5. Feature step logic (orchestration)
   6. Step definition and directive (LLM interface)

## Sources

- Codebase: `internal/cli/init.go` (copyEmbeddedFiles pattern)
- Codebase: `internal/initiative/service.go` (initiative patterns)
- Codebase: `internal/step/service.go` (step execution patterns)
- Codebase: `templates/steps/*.md` (directive patterns)
- Codebase: `profiles/*.md` (profile composition patterns)
- Web: go-git performance comparisons (SlideShare, GitHub issues)
- Web: Git branch naming conventions (Medium, Graphite, Zignuts)
- Web: Go error handling best practices (JetBrains, Leapcell)
