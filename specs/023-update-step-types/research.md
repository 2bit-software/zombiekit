# Research: Update Step Types

**Date**: 2025-12-23
**Feature**: 023-update-step-types

## Executive Summary

The existing step system already has most infrastructure in place. The feature step (`feature.go`) already handles initiative creation with cycles and uses the specify workflow (research-create-audit-highlight phases). The main work involves: (1) removing legacy steps (init, specify, implement), (2) adding bug/refactor as first-class step types, (3) renaming implement to eat, and (4) adding prerequisite enforcement.

## Findings by Category

### Current Step Implementation

**Decision**: Extend existing pattern in `internal/step/service.go`

**Rationale**: The service already has special handling for init, feature, and complete steps. Adding bug/refactor follows the same pattern.

**Current Step Handlers**:
- `executeInitStep()` - Creates initiative, to be removed
- `executeFeatureStep()` - Creates initiative with cycles, already supports type parameter
- `executeCompleteStep()` - Marks initiative complete

**Key Insight**: The feature step already accepts a `Type` parameter that maps to initiative types (feature, bug, refactor). Bug and refactor steps can delegate to the same handler with preset types.

### Step Template Structure

**Decision**: Keep current YAML frontmatter + markdown directive format

**Rationale**: Works well, no changes needed to the format itself.

**Current Templates** (in `templates/steps/`):
- audit.md, clarify.md, complete.md - Keep as-is
- feature.md - Keep as-is (already has full specify workflow)
- implement.md - Rename to eat.md
- init.md - Remove
- specify.md - Remove (functionality now in feature.md)
- plan.md, tasks.md - Keep as-is

**New Templates Needed**:
- bug.md - Bug investigation workflow directive
- refactor.md - Refactor specification workflow directive

### Loader Hint Update

**Decision**: Update error hint in `loader.go` line 107-108

**Current**:
```go
Hint: "Available steps: init, specify, plan, tasks, implement, audit, clarify, complete"
```

**New**:
```go
Hint: "Available steps: feature, bug, refactor, plan, tasks, eat, audit, clarify, complete"
```

### Prerequisite Enforcement

**Decision**: Add prerequisite validation in `Service.Execute()`

**Rationale**: FR-011 requires hard blocking with guidance. Implement as a prerequisite map checked before step execution.

**Prerequisite Map**:
```go
var stepPrerequisites = map[string]StepPrerequisite{
    "plan":  {RequiredArtifact: "spec.md", RequiredStatus: "approved", Hint: "Run feature/bug/refactor first"},
    "tasks": {RequiredArtifact: "plan.md", RequiredStatus: "approved", Hint: "Run plan first"},
    "eat":   {RequiredArtifact: "tasks.md", RequiredStatus: "", Hint: "Run tasks first"},
}
```

### Initiative Type Handling

**Decision**: Bug and refactor steps delegate to feature step handler with preset types

**Rationale**: The feature step already handles all three initiative types via the Type parameter. Dedicated steps just provide better UX and type-specific directives.

**Implementation**:
```go
func (s *Service) executeBugStep(step *Step, opts *ExecuteOptions) (*StepResponse, error) {
    if opts == nil {
        opts = &ExecuteOptions{}
    }
    opts.Type = "bug"
    return s.executeFeatureStep(step, opts)
}
```

### Files to Modify

1. **internal/step/service.go**
   - Remove `executeInitStep()` case
   - Add `executeBugStep()` and `executeRefactorStep()` cases
   - Change `implement` case to `eat`
   - Add prerequisite validation in `Execute()`

2. **internal/step/loader.go**
   - Update error hint with new step names

3. **internal/step/bug.go** (NEW)
   - Simple wrapper that sets Type="bug" and delegates to executeFeatureStep

4. **internal/step/refactor.go** (NEW)
   - Simple wrapper that sets Type="refactor" and delegates to executeFeatureStep

5. **templates/steps/**
   - Delete: init.md, specify.md
   - Rename: implement.md → eat.md
   - Add: bug.md, refactor.md

## Alternatives Considered

### Alternative 1: Keep legacy steps as aliases
**Rejected**: User explicitly stated "we don't care about legacy" - breaking change acceptable.

### Alternative 2: Merge bug/refactor into feature step only
**Rejected**: Having dedicated `/brains.bug` and `/brains.refactor` commands improves discoverability and allows type-specific directives.

### Alternative 3: Complex prerequisite state machine
**Rejected**: Simple artifact-exists check is sufficient. No need for elaborate state tracking.

## Open Questions

None - all clarifications resolved in spec phase.

## Sources

- `internal/step/service.go` - Current step execution logic
- `internal/step/feature.go` - Feature step implementation with initiative/cycle handling
- `internal/step/loader.go` - Step loading and error messages
- `templates/steps/*.md` - Current step templates
- Spec clarifications - Legacy removal, hard block enforcement
