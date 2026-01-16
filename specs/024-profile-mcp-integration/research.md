# Research: Profile-MCP Integration

**Feature**: 024-profile-mcp-integration
**Date**: 2025-12-24

## Executive Summary

This research consolidates findings on how step profiles need to integrate with the MCP tool system. The existing implementation has most infrastructure in place—step loading, profile composition, and response structures. The primary work is updating profile directives to provide clear, actionable guidance for agents executing via MCP.

## 1. Current State Analysis

### 1.1 Step Profile Structure

**Source**: `internal/step/types.go`, `templates/steps/*.md`

Each step profile is a markdown file with YAML frontmatter:

```yaml
---
name: feature
description: Execute the research-create-audit-highlight workflow
profiles:          # Profiles to compose into composed_prompt
  - research
  - create
  - audit
files:             # Glob patterns relative to cycle folder
  - "research.md"
  - "spec.md"
type: step
---
# Directive content in markdown body
```

**Decision**: Maintain existing frontmatter structure. No schema changes needed.

### 1.2 StepResponse Structure

**Source**: `internal/step/types.go:66-87`

The Go backend returns:

```go
type StepResponse struct {
    Directive        string           `json:"directive"`
    HistoryFolder    string           `json:"history_folder"`  // Deprecated
    FilesToRead      []string         `json:"files_to_read"`
    ComposedPrompt   string           `json:"composed_prompt"`
    InitiativeFolder string           `json:"initiative_folder,omitempty"`
    CycleFolder      string           `json:"cycle_folder,omitempty"`
    WorkflowPhases   []Phase          `json:"workflow_phases,omitempty"`
    NextTask         *TaskInfo        `json:"next_task,omitempty"`
    Prerequisites    PrerequisiteInfo `json:"prerequisites,omitempty"`
}
```

**Decision**: Response structure is adequate. `history_folder` is deprecated in favor of `initiative_folder`.

### 1.3 Multi-Phase Workflow Implementation

**Source**: `internal/step/feature.go`

Workflow phases are built programmatically for feature/bug/refactor steps:

```go
func buildWorkflowPhases() []Phase {
    return []Phase{
        {Name: "research", Description: "...", Agents: [...], Outputs: [...], Parallel: true},
        {Name: "create", Description: "...", ...},
        {Name: "audit", Description: "...", ...},
        {Name: "highlight", Description: "...", ...},
    }
}
```

**Decision**: This hardcoded approach works but should eventually move to profile frontmatter. For now, keep in Go code.

### 1.4 Prerequisite Enforcement

**Source**: `internal/step/service.go:16-35`

Prerequisites are defined per-step:

| Step | Required Artifact | Required Status | Hint |
|------|-------------------|-----------------|------|
| plan | spec.md | approved | Run feature/bug/refactor first |
| tasks | plan.md | approved | Run plan first |
| eat | tasks.md | (existence only) | Run tasks first |

**Decision**: Prerequisite structure is complete. No changes needed.

### 1.5 Profile Composition

**Source**: `internal/step/service.go:159-163`

Profiles listed in frontmatter are composed via `profileSvc.Compose()` and returned in `composed_prompt`. This provides reusable context blocks.

**Decision**: Working correctly. Profiles should reference meaningful composable units.

## 2. Gap Analysis

### 2.1 Profile Content Gaps

| Profile | Current State | Gap | Required Change |
|---------|---------------|-----|-----------------|
| feature.md | Comprehensive 4-phase workflow | Minor | Ensure directive explicitly references MCP response fields |
| bug.md | 4-phase with classification | Minor | Mirror feature.md structure improvements |
| refactor.md | 4-phase with before/after | Minor | Mirror feature.md structure improvements |
| plan.md | Basic guidance | **Major** | Needs structured directive with constitution check, phases |
| tasks.md | Basic task format | **Major** | Needs structured directive with dependency format, parallel markers |
| eat.md | Basic implementation guidance | **Moderate** | Needs to reference `next_task` field from response |
| audit.md | Basic checklist | **Moderate** | Needs severity classification, cross-artifact alignment |
| clarify.md | Basic question guidance | **Moderate** | Needs taxonomy, question format, integration rules |

### 2.2 Missing Response Field References

Current profiles don't explicitly tell agents how to use:
- `files_to_read` - Should instruct agents to read these first
- `composed_prompt` - Should explain this contains reusable context
- `workflow_phases` - Should explain phase structure (for feature/bug/refactor)
- `next_task` - Should explain how to use this (for eat step)
- `cycle_folder` - Should reference for output file paths

**Decision**: Add a "## Response Handling" section to each profile explaining field usage.

### 2.3 Git Operations Clarity

**Source**: `internal/mcp/tools/initiative/tool.go:186-189`

Git branch creation is internal:
```go
gitSvc := step.NewGitService(dir)
_ = gitSvc.EnsureBranch(initType, name)
```

Agents receive `branch` field but never execute git commands.

**Decision**: Profiles should NOT include git instructions. Add note that git is handled automatically.

## 3. Design Decisions

### 3.1 Profile Directive Structure

**Decision**: Standardize all profiles with this structure:

```markdown
# Step Name Workflow

## Context
[What this step does, what's automatic vs agent responsibility]

## Response Handling
[How to interpret MCP response fields]

## Prerequisites
[What must exist before this step runs]

## Workflow
[Phase-by-phase or single-phase instructions]

## Output
[What artifacts to create/update]

## Success Criteria
[Checkboxes for completion verification]

## Behavior Rules
[Constraints and guidelines]
```

**Rationale**: Consistent structure makes profiles predictable and machine-parseable.

### 3.2 Multi-Phase Step Profiles

**Decision**: Feature, bug, and refactor profiles should:
1. Define each phase with Input → Actions → Output → Success Criteria
2. Include conditional transitions (audit loop, user approval gate)
3. Reference `workflow_phases` from response for phase definitions
4. Limit iteration count (3 loops before user escalation)

**Rationale**: Matches existing feature.md pattern, which is comprehensive.

### 3.3 Single-Phase Step Profiles

**Decision**: Plan, tasks, eat, audit, clarify profiles should:
1. Have simpler structure (no phase loops)
2. Focus on single artifact transformation
3. Include clear completion conditions

**Rationale**: These steps don't need multi-phase complexity.

### 3.4 Eat Step Enhancement

**Decision**: Eat step profile should:
1. Check `next_task` field in response
2. If null, indicate all tasks complete
3. Reference task ID, description, phase from `next_task`
4. Guide task-by-task implementation with TDD focus

**Rationale**: Agents need explicit guidance on task-focused execution.

### 3.5 Constitution Alignment for Plan Step

**Decision**: Plan step profile should reference constitution check as part of the planning workflow. The profile should:
1. Load `.specify/memory/constitution.md` if it exists
2. Verify plan adheres to project principles
3. Flag violations in plan output

**Rationale**: Matches spec-kit `/speckit.plan` behavior, maintains consistency.

## 4. Alternatives Considered

### 4.1 Move Phase Definitions to Frontmatter

**Considered**: Define workflow_phases in YAML frontmatter instead of Go code.

**Rejected**:
- Would require YAML schema changes
- Frontmatter parsing would need to handle nested structures
- Go code provides type safety and validation
- Keep for future enhancement when schema stabilizes

### 4.2 Merge Profiles Into Single Tool

**Considered**: Single MCP tool with all workflow logic.

**Rejected**:
- Profiles provide flexibility and customization
- Users can override profiles locally
- Separation of concerns (tool = execution, profile = guidance)

### 4.3 Dynamic Profile Loading via MCP

**Considered**: Let agents request profile content via separate MCP call.

**Rejected**:
- `composed_prompt` already provides this
- Extra round-trip adds latency
- Profiles are already included in step response

## 5. Implementation Recommendations

### Priority Order

1. **P0**: Update plan.md, tasks.md profiles (Major gaps)
2. **P1**: Update eat.md, audit.md, clarify.md profiles (Moderate gaps)
3. **P2**: Polish feature.md, bug.md, refactor.md profiles (Minor gaps)
4. **P3**: Add tests verifying profile loading and response structure

### File Changes

| File | Change Type | Priority |
|------|-------------|----------|
| `templates/steps/plan.md` | Rewrite directive | P0 |
| `templates/steps/tasks.md` | Rewrite directive | P0 |
| `templates/steps/eat.md` | Enhance with next_task handling | P1 |
| `templates/steps/audit.md` | Add severity classification | P1 |
| `templates/steps/clarify.md` | Add taxonomy and format | P1 |
| `templates/steps/feature.md` | Add response handling section | P2 |
| `templates/steps/bug.md` | Mirror feature.md improvements | P2 |
| `templates/steps/refactor.md` | Mirror feature.md improvements | P2 |
| `internal/step/service_test.go` | Add profile content tests | P3 |
| `internal/mcp/tools/step/tool_test.go` | Add response validation | P3 |

### Test Strategy

1. **Unit Tests**: Verify profile frontmatter parsing
2. **Integration Tests**: Verify step execution returns expected response structure
3. **Contract Tests**: Verify response JSON matches documented contract

## 6. Sources

- `internal/step/types.go` - Type definitions
- `internal/step/service.go` - Step execution logic
- `internal/step/feature.go` - Workflow phase builder
- `internal/mcp/tools/step/tool.go` - MCP tool implementation
- `internal/mcp/tools/initiative/tool.go` - Initiative tool implementation
- `templates/steps/*.md` - Existing profile content
- `specs/024-profile-mcp-integration/spec.md` - Feature specification
