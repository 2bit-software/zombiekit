# Quickstart: Update Step Types

## Overview

This feature updates the available workflow steps to a streamlined set of nine:
- **feature, bug, refactor** - Initiative creation steps (replace init + specify)
- **plan** - Implementation planning
- **tasks** - Task list generation
- **eat** - Implementation execution (replaces implement)
- **audit, clarify** - Quality assurance
- **complete** - Initiative completion

## Key Changes

### Removed Steps
- `init` - Merged into feature/bug/refactor
- `specify` - Merged into feature/bug/refactor
- `implement` - Renamed to `eat`

### New Steps
- `bug` - Starts a bug investigation initiative
- `refactor` - Starts a refactor initiative

### Renamed Steps
- `implement` → `eat` - More memorable, follows "BRAAAAINS" theme

## Implementation Tasks

### 1. Update Step Templates

**Delete**:
- `templates/steps/init.md`
- `templates/steps/specify.md`

**Rename**:
- `templates/steps/implement.md` → `templates/steps/eat.md`

**Create**:
- `templates/steps/bug.md` - Bug investigation directive
- `templates/steps/refactor.md` - Refactor specification directive

### 2. Add Prerequisite Checking

In `internal/step/service.go`:

```go
// Add prerequisite map
var stepPrerequisites = map[string]StepPrerequisite{
    "plan":  {RequiredArtifact: "spec.md", RequiredStatus: "approved"},
    "tasks": {RequiredArtifact: "plan.md", RequiredStatus: "approved"},
    "eat":   {RequiredArtifact: "tasks.md"},
}

// Add check in Execute()
func (s *Service) Execute(stepName string, opts *ExecuteOptions) (*StepResponse, error) {
    // Check prerequisites first
    if err := s.checkPrerequisite(stepName); err != nil {
        return nil, err
    }
    // ... rest of execution
}
```

### 3. Add Bug/Refactor Step Handlers

Simple wrappers that delegate to feature step with preset type:

```go
func (s *Service) executeBugStep(step *Step, opts *ExecuteOptions) (*StepResponse, error) {
    if opts == nil {
        opts = &ExecuteOptions{}
    }
    opts.Type = "bug"
    return s.executeFeatureStep(step, opts)
}

func (s *Service) executeRefactorStep(step *Step, opts *ExecuteOptions) (*StepResponse, error) {
    if opts == nil {
        opts = &ExecuteOptions{}
    }
    opts.Type = "refactor"
    return s.executeFeatureStep(step, opts)
}
```

### 4. Update Service.Execute() Switch

```go
switch stepName {
case "feature":
    return s.executeFeatureStep(step, opts)
case "bug":
    return s.executeBugStep(step, opts)
case "refactor":
    return s.executeRefactorStep(step, opts)
case "complete":
    return s.executeCompleteStep(step, opts)
// Remove "init" case entirely
default:
    // Standard step execution with prerequisite checking
}
```

### 5. Update Loader Error Hint

In `internal/step/loader.go`:

```go
// Change from:
Hint: "Available steps: init, specify, plan, tasks, implement, audit, clarify, complete"
// To:
Hint: "Available steps: feature, bug, refactor, plan, tasks, eat, audit, clarify, complete"
```

## Testing Checklist

- [ ] `feature` step creates initiative with spec.md template
- [ ] `bug` step creates bug-type initiative
- [ ] `refactor` step creates refactor-type initiative
- [ ] `plan` blocks if spec.md not approved
- [ ] `tasks` blocks if plan.md not approved
- [ ] `eat` blocks if tasks.md doesn't exist
- [ ] `init` step returns UNKNOWN_STEP error
- [ ] `specify` step returns UNKNOWN_STEP error
- [ ] `implement` step returns UNKNOWN_STEP error
- [ ] All nine new steps are listed by `ListSteps()`

## File Changes Summary

| Action | File |
|--------|------|
| Modify | internal/step/service.go |
| Modify | internal/step/loader.go |
| Create | internal/step/bug.go |
| Create | internal/step/refactor.go |
| Create | internal/step/prerequisite.go |
| Delete | templates/steps/init.md |
| Delete | templates/steps/specify.md |
| Rename | templates/steps/implement.md → eat.md |
| Create | templates/steps/bug.md |
| Create | templates/steps/refactor.md |
