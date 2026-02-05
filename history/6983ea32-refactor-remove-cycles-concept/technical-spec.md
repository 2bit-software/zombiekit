# Technical Specification: Remove Cycles Concept

## Data Model Changes

### Before: Nested Structure

```
ParsedInitiative
├── Name, Type, Status, Created
└── Cycles []ParsedCycle
    ├── Number, Type, Name, Status
    └── Steps []ParsedStep
        └── Name, Status, Updated
```

### After: Flat Structure

```
ParsedInitiative
├── Name, Type, Status, Created
└── Steps []ParsedStep
    └── Name, Status, Updated
```

---

## Type Deletions

### internal/initiative/types.go

```go
// DELETE - lines 102-156
type CycleType string
const (
    CycleFeat CycleType = "feat"
    CycleRef  CycleType = "ref"
    CycleFix  CycleType = "fix"
)
func (c CycleType) IsValid() bool { ... }
func (c CycleType) String() string { ... }

type CycleStatus string
const (
    CycleStatusTemplate   CycleStatus = "template"
    CycleStatusInProgress CycleStatus = "in_progress"
    CycleStatusAudited    CycleStatus = "audited"
    CycleStatusApproved   CycleStatus = "approved"
)
func (s CycleStatus) IsValid() bool { ... }
func (s CycleStatus) String() string { ... }

// DELETE - lines 158-178
type Cycle struct {
    ID           string
    Type         CycleType
    Name         string
    Path         string
    Status       CycleStatus
    InitiativeID string
    Number       int
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### internal/initiative/markdown.go

```go
// DELETE
type ParsedCycle struct {
    Number int
    Type   string
    Name   string
    Status string
    Steps  []ParsedStep
}

// DELETE method
func (p *ParsedInitiative) ActiveCycle() *ParsedCycle

// DELETE regex
var cycleHeaderRe = regexp.MustCompile(`^###\s+(\d+)\.\s+(\w+)/([^\s]+)\s+\((\w+)\)`)
```

---

## Type Modifications

### internal/initiative/markdown.go

```go
// MODIFY ParsedInitiative
type ParsedInitiative struct {
    Name    string        // unchanged
    Type    string        // unchanged
    Status  string        // unchanged
    Created time.Time     // unchanged
    Steps   []ParsedStep  // CHANGED: was Cycles []ParsedCycle
}
```

### internal/initiative/service.go

```go
// MODIFY StatusResult - remove cycle fields
type StatusResult struct {
    Active         bool
    InitiativeID   string
    InitiativeType string
    CurrentStep    string
    StepStatus     string
    // CycleID        string  // REMOVE
    // CurrentCycle   int     // REMOVE
    StepsCompleted int
    StepsTotal     int
    AvailableDocs  []string
    SuggestedNext  string
    HistoryPath    string
    InitiativeFile string
    Files          []string
}
```

### internal/mcp/tools/initiative/types.go

```go
// MODIFY CreateResponse - remove cycle fields
type CreateResponse struct {
    Action         string
    InitiativeID   string
    InitiativePath string
    // CycleID        string   // REMOVE
    // CyclePath      string   // REMOVE
    Branch         string
    Type           string
    Name           string
    NextStep       string
    AlreadyExisted bool
    SkippedFiles   []string
    CopiedFiles    []string
}

// MODIFY StatusResponse - remove cycle field
type StatusResponse struct {
    Action         string
    Active         bool
    InitiativeID   string
    InitiativeType string
    CurrentStep    string
    // CycleID        string   // REMOVE
    AvailableDocs  []string
    SuggestedNext  string
    HistoryPath    string
    InitiativeFile string
    Files          []string
}
```

### internal/step/types.go

```go
// MODIFY StepResponse - deprecate CycleFolder
type StepResponse struct {
    Directive        string
    HistoryFolder    string    // deprecated, use InitiativeFolder
    FilesToRead      []string
    ComposedPrompt   string
    InitiativeFolder string
    // CycleFolder      string   // REMOVE or keep as alias for InitiativeFolder
    WorkflowPhases   []Phase
    NextTask         *TaskInfo
    Prerequisites    PrerequisiteInfo
}
```

---

## Function Changes

### internal/initiative/markdown.go

#### ParseInitiativeMD()

**Before**: Looks for `### N. type/name (status)` headers, parses steps under each cycle.

**After**: Looks for `## Steps` header, parses step table directly.

```go
func ParseInitiativeMD(path string) (*ParsedInitiative, error) {
    // ...scanner setup...

    inStepTable := false

    for scanner.Scan() {
        line := scanner.Text()

        // Parse title and metadata (unchanged)

        // NEW: Detect step table start
        if strings.HasPrefix(line, "| Step ") {
            inStepTable = true
            continue
        }

        // Skip table separator
        if strings.HasPrefix(line, "|---") {
            continue
        }

        // Parse step rows directly into parsed.Steps
        if inStepTable {
            if matches := stepRowRe.FindStringSubmatch(line); matches != nil {
                step := ParsedStep{
                    Name:    strings.TrimSpace(matches[1]),
                    Status:  parseStepStatus(matches[2]),
                    Updated: strings.TrimSpace(matches[3]),
                }
                parsed.Steps = append(parsed.Steps, step)
            } else if !strings.HasPrefix(line, "|") {
                inStepTable = false
            }
        }
    }

    return parsed, nil
}
```

#### CurrentStep()

```go
// BEFORE
func (p *ParsedInitiative) CurrentStep() *ParsedStep {
    cycle := p.ActiveCycle()
    if cycle == nil { return nil }
    for i := range cycle.Steps {
        if cycle.Steps[i].Status == StepInProgress {
            return &cycle.Steps[i]
        }
    }
    return nil
}

// AFTER
func (p *ParsedInitiative) CurrentStep() *ParsedStep {
    for i := range p.Steps {
        if p.Steps[i].Status == StepInProgress {
            return &p.Steps[i]
        }
    }
    return nil
}
```

#### NextStep()

```go
// AFTER
func (p *ParsedInitiative) NextStep() *ParsedStep {
    foundCurrent := false
    for i := range p.Steps {
        if p.Steps[i].Status == StepInProgress {
            foundCurrent = true
            continue
        }
        if foundCurrent && p.Steps[i].Status == StepPending {
            return &p.Steps[i]
        }
    }
    if !foundCurrent {
        for i := range p.Steps {
            if p.Steps[i].Status == StepPending {
                return &p.Steps[i]
            }
        }
    }
    return nil
}
```

#### UpdateStepStatus()

```go
// BEFORE
func (p *ParsedInitiative) UpdateStepStatus(cycleNum int, stepName string, status StepStatus, timestamp string) error

// AFTER
func (p *ParsedInitiative) UpdateStepStatus(stepName string, status StepStatus, timestamp string) error {
    for i := range p.Steps {
        if p.Steps[i].Name == stepName {
            p.Steps[i].Status = status
            p.Steps[i].Updated = timestamp
            return nil
        }
    }
    return fmt.Errorf("step not found: %s", stepName)
}
```

#### AddStep()

```go
// BEFORE
func (p *ParsedInitiative) AddStep(cycleNum int, afterStep string, newStep ParsedStep) error

// AFTER
func (p *ParsedInitiative) AddStep(afterStep string, newStep ParsedStep) error {
    if afterStep == "" {
        p.Steps = append([]ParsedStep{newStep}, p.Steps...)
        return nil
    }
    for i := range p.Steps {
        if p.Steps[i].Name == afterStep {
            newSteps := make([]ParsedStep, 0, len(p.Steps)+1)
            newSteps = append(newSteps, p.Steps[:i+1]...)
            newSteps = append(newSteps, newStep)
            newSteps = append(newSteps, p.Steps[i+1:]...)
            p.Steps = newSteps
            return nil
        }
    }
    return fmt.Errorf("step not found: %s", afterStep)
}
```

#### WriteTo()

```go
func (p *ParsedInitiative) WriteTo(path string) error {
    // Read original, find ## Steps section
    // Replace with formatted step table
    // Write atomically
}

func (p *ParsedInitiative) formatSteps() []string {
    var lines []string
    lines = append(lines, "| Step | Status | Updated |")
    lines = append(lines, "|------|--------|---------|")
    for _, step := range p.Steps {
        row := fmt.Sprintf("| %s | %s | %s |", step.Name, step.Status, step.Updated)
        lines = append(lines, row)
    }
    return lines
}
```

### internal/initiative/service.go

#### createInitiativeMD()

```go
func (s *Service) createInitiativeMD(init *Initiative, steps []WorkflowStep) error {
    var builder strings.Builder

    // Header section (unchanged)
    builder.WriteString(fmt.Sprintf("# Initiative: %s\n\n", init.Name))
    // ...metadata...

    // Steps section (CHANGED from Cycles)
    if len(steps) > 0 {
        builder.WriteString("## Steps\n\n")
        builder.WriteString("| Step | Status | Updated |\n")
        builder.WriteString("|------|--------|--------|\n")
        for i, step := range steps {
            status := "pending"
            updated := "-"
            if i == 0 {
                status = "in_progress"
                updated = time.Now().Format("2006-01-02 15:04")
            }
            builder.WriteString(fmt.Sprintf("| %s | %s | %s |\n", step.Name, status, updated))
        }
        builder.WriteString("\n")
    }

    // Rest unchanged
}
```

#### Status()

```go
func (s *Service) Status() (*StatusResult, error) {
    // ...load state, get active initiative...

    if err == nil && parsed != nil {
        stepsTotal = len(parsed.Steps)

        for _, step := range parsed.Steps {
            if step.Status == StepCompleted || step.Status == StepSkipped {
                stepsCompleted++
            }
            if step.Status == StepInProgress {
                currentStep = step.Name
                stepStatus = string(step.Status)
            }
        }

        if currentStep == "" {
            if next := parsed.NextStep(); next != nil {
                currentStep = next.Name
                stepStatus = string(next.Status)
            }
        }
    }

    return &StatusResult{
        Active:         true,
        InitiativeID:   init.ID,
        InitiativeType: string(init.Type),
        CurrentStep:    currentStep,
        StepStatus:     stepStatus,
        // CycleID and CurrentCycle removed
        StepsCompleted: stepsCompleted,
        StepsTotal:     stepsTotal,
        // ...rest unchanged
    }, nil
}
```

### internal/mcp/tools/initiative/tool.go

#### handleCreate()

```go
func (t *Tool) handleCreate(ctx context.Context, dir string, args map[string]interface{}) (string, error) {
    // ...validation and idempotency check...

    // REMOVE: cycle creation
    // cycleType := mapInitTypeToCycleType(...)
    // cycle, err := initSvc.CreateCycle(...)

    // Copy templates directly to initiative folder
    skipped, copied, err := t.copyTemplatesToInitiative(dir, initiative.Path)

    // Git branch (unchanged)

    resp := CreateResponse{
        Action:         "create",
        InitiativeID:   initiative.ID,
        InitiativePath: initiative.Path,
        // CycleID and CyclePath removed
        Branch:         branchName,
        Type:           initType,
        Name:           name,
        NextStep:       nextStep,
        AlreadyExisted: false,
        SkippedFiles:   skipped,
        CopiedFiles:    copied,
    }

    return marshalResponse(resp)
}
```

Delete these functions entirely:
- `mapInitTypeToCycleType()`
- `findFirstCycle()`

Rename:
- `copyTemplatesToCycle()` → `copyTemplatesToInitiative()`

### internal/step/service.go

#### Execute()

```go
func (s *Service) Execute(stepName string, opts *ExecuteOptions) (*StepResponse, error) {
    // ...load step, get initiative context...

    // historyFolder is the initiative folder
    // cyclePath variable removed - just use historyFolder

    resp := &StepResponse{
        Directive:        step.Directive,
        HistoryFolder:    historyFolder,
        InitiativeFolder: historyFolder,
        // CycleFolder removed
        FilesToRead:      filesToRead,
        ComposedPrompt:   composedPrompt,
        Prerequisites:    PrerequisiteInfo{Met: true},
    }

    return resp, nil
}
```

#### UpdateState()

```go
func (s *Service) UpdateState(stepName string, initiativeID string) error {
    // ...load state, parse INITIATIVE.md...

    // BEFORE: cycle := parsed.ActiveCycle()
    // AFTER: work directly with parsed.Steps

    now := time.Now().Format("2006-01-02 15:04")

    for i := range parsed.Steps {
        if parsed.Steps[i].Status == initiative.StepInProgress {
            parsed.Steps[i].Status = initiative.StepCompleted
            parsed.Steps[i].Updated = now
        }
    }

    for i := range parsed.Steps {
        if parsed.Steps[i].Name == stepName {
            parsed.Steps[i].Status = initiative.StepInProgress
            parsed.Steps[i].Updated = now
            break
        }
    }

    return parsed.WriteTo(mdPath)
}
```

---

## INITIATIVE.md Format Change

### Before (Cycles)

```markdown
# Initiative: user-auth

**Type**: feature
**Status**: in_progress
**Created**: 2026-02-04
**ID**: abc123-feature-user-auth

## Cycles

### 1. feat/user-auth (active)

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-02-04 10:00 |
| plan | in_progress | 2026-02-04 11:00 |
| tasks | pending | - |
| implement | pending | - |

## Description
...
```

### After (Flat Steps)

```markdown
# Initiative: user-auth

**Type**: feature
**Status**: in_progress
**Created**: 2026-02-04
**ID**: abc123-feature-user-auth

## Steps

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-02-04 10:00 |
| plan | in_progress | 2026-02-04 11:00 |
| tasks | pending | - |
| implement | pending | - |

## Description
...
```

---

## Folder Structure Change

### Before

```
history/
  abc123-feature-user-auth/
    INITIATIVE.md
    def456-feat-user-auth/    <-- Cycle folder
      spec.md
      research.md
      plan.md
      tasks.md
      audit/
```

### After

```
history/
  abc123-feature-user-auth/
    INITIATIVE.md
    spec.md                    <-- Direct in initiative folder
    research.md
    plan.md
    tasks.md
    audit/
```

---

## API Response Changes

### CreateResponse

```json
// BEFORE
{
  "action": "create",
  "initiative_id": "abc123-feature-user-auth",
  "initiative_path": "/path/to/history/abc123-feature-user-auth",
  "cycle_id": "def456-feat-user-auth",
  "cycle_path": "/path/to/history/abc123-feature-user-auth/def456-feat-user-auth",
  "branch": "abc123-feature-user-auth",
  ...
}

// AFTER
{
  "action": "create",
  "initiative_id": "abc123-feature-user-auth",
  "initiative_path": "/path/to/history/abc123-feature-user-auth",
  "branch": "abc123-feature-user-auth",
  ...
}
```

### StatusResponse

```json
// BEFORE
{
  "action": "status",
  "active": true,
  "initiative_id": "abc123-feature-user-auth",
  "cycle_id": "1-feat-user-auth",
  "current_step": "plan",
  ...
}

// AFTER
{
  "action": "status",
  "active": true,
  "initiative_id": "abc123-feature-user-auth",
  "current_step": "plan",
  ...
}
```

---

## Test Updates Required

### markdown_test.go

All test markdown strings need:
1. `## Cycles` → `## Steps`
2. Remove `### N. type/name (status)` headers
3. Update assertions: `parsed.Cycles[0].Steps[0]` → `parsed.Steps[0]`
4. Remove `ActiveCycle()` tests
5. Update `UpdateStepStatus` and `AddStep` signatures

### service_test.go

1. Update INITIATIVE.md content assertions
2. Remove CycleID from StatusResult checks

### tool_test.go (MCP)

1. Remove CycleID/CyclePath from response assertions
2. Update path expectations for template locations

### step/service_test.go

1. Update any cycle path references
2. CycleFolder assertions removed or changed to InitiativeFolder
