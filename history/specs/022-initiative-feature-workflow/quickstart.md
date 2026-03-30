# Quickstart: Initiative Feature Workflow (022)

**Date**: 2025-12-23
**Estimated Effort**: 3-5 implementation sessions

## Prerequisites

- Go 1.24.0+ installed
- Git installed (optional, for branch management)
- Existing zombiekit codebase with:
  - `internal/initiative/` package
  - `internal/step/` package
  - `internal/mcp/tools/step/` package

## Implementation Order

### Step 1: Add Cycle Types (30 min)

**File**: `internal/initiative/types.go`

Add after existing `InitiativeStatus`:

```go
// CycleType represents the type of cycle within an initiative.
type CycleType string

const (
    CycleFeat CycleType = "feat"
    CycleRef  CycleType = "ref"
    CycleFix  CycleType = "fix"
)

func (c CycleType) IsValid() bool {
    switch c {
    case CycleFeat, CycleRef, CycleFix:
        return true
    default:
        return false
    }
}

// CycleStatus represents the status of a cycle.
type CycleStatus string

const (
    CycleStatusTemplate   CycleStatus = "template"
    CycleStatusInProgress CycleStatus = "in_progress"
    CycleStatusAudited    CycleStatus = "audited"
    CycleStatusApproved   CycleStatus = "approved"
)

// Cycle represents a single workflow pass within an initiative.
type Cycle struct {
    ID           string      `json:"id"`
    Type         CycleType   `json:"type"`
    Name         string      `json:"name"`
    Path         string      `json:"path"`
    Status       CycleStatus `json:"status"`
    InitiativeID string      `json:"initiative_id"`
    Number       int         `json:"number"`
    CreatedAt    time.Time   `json:"created_at"`
    UpdatedAt    time.Time   `json:"updated_at"`
}
```

Update `InitiativeState` (simplified - pointer only, no status fields):

```go
// InitiativeState tracks ONLY which initiative and cycle are currently active.
// Status is NOT stored here - read from INITIATIVE.md frontmatter.
type InitiativeState struct {
    // Pointer to active initiative (path only)
    Initiative   string    `json:"initiative,omitempty"`

    // Pointer to active cycle within the initiative
    Cycle        string    `json:"cycle,omitempty"`

    // Timestamps for activity tracking
    Started      time.Time `json:"started,omitempty"`
    LastActivity time.Time `json:"last_activity,omitempty"`

    // Current step name (for resume capability)
    CurrentStep  string    `json:"current_step,omitempty"`
}
```

### Step 2: Add Cycle Management (1 hour)

**File**: `internal/initiative/cycle.go` (new)

```go
package initiative

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
)

// CreateCycle creates a new cycle within an initiative.
func (s *Service) CreateCycle(initPath string, cycleType CycleType, name string) (*Cycle, error) {
    // Validate cycle type
    if !cycleType.IsValid() {
        return nil, &InitiativeError{
            Code:    "INVALID_CYCLE_TYPE",
            Message: fmt.Sprintf("invalid cycle type '%s'", cycleType),
            Hint:    "Type must be one of: feat, ref, fix",
        }
    }

    // Get next cycle number
    cycleNum, err := s.getNextCycleNumber(initPath)
    if err != nil {
        return nil, err
    }

    // Generate cycle ID
    cycleID := s.generateCycleID(cycleType, name)
    cyclePath := filepath.Join(initPath, cycleID)

    // Create cycle directory
    if err := os.MkdirAll(cyclePath, 0755); err != nil {
        return nil, fmt.Errorf("creating cycle directory: %w", err)
    }

    // Create audit subdirectory
    auditPath := filepath.Join(cyclePath, "audit")
    if err := os.MkdirAll(auditPath, 0755); err != nil {
        return nil, fmt.Errorf("creating audit directory: %w", err)
    }

    now := time.Now()
    cycle := &Cycle{
        ID:           cycleID,
        Type:         cycleType,
        Name:         name,
        Path:         cyclePath,
        Status:       CycleStatusTemplate,
        InitiativeID: filepath.Base(initPath),
        Number:       cycleNum,
        CreatedAt:    now,
        UpdatedAt:    now,
    }

    return cycle, nil
}

func (s *Service) generateCycleID(cycleType CycleType, name string) string {
    timestamp := fmt.Sprintf("%08x", time.Now().Unix())
    return fmt.Sprintf("%s-%s-%s", timestamp, cycleType, name)
}

func (s *Service) getNextCycleNumber(initPath string) (int, error) {
    entries, err := os.ReadDir(initPath)
    if err != nil {
        return 1, nil // First cycle if can't read
    }

    count := 0
    for _, entry := range entries {
        if entry.IsDir() && entry.Name() != "audit" {
            count++
        }
    }
    return count + 1, nil
}
```

### Step 3: Add Git Service (45 min)

**File**: `internal/step/git.go` (new)

```go
package step

import (
    "fmt"
    "os/exec"
    "regexp"
    "strings"
)

type GitService struct {
    workDir string
}

func NewGitService(workDir string) *GitService {
    return &GitService{workDir: workDir}
}

// EnsureBranch creates or switches to a branch for the initiative.
func (g *GitService) EnsureBranch(initType, name string) error {
    if !g.isGitAvailable() || !g.isGitRepository() {
        return nil // Graceful degradation
    }

    branchName, err := g.formatBranchName(initType, name)
    if err != nil {
        return err
    }

    if g.branchExists(branchName) {
        return g.switchToBranch(branchName)
    }
    return g.createBranch(branchName)
}

func (g *GitService) isGitAvailable() bool {
    _, err := exec.LookPath("git")
    return err == nil
}

func (g *GitService) isGitRepository() bool {
    cmd := exec.Command("git", "rev-parse", "--git-dir")
    cmd.Dir = g.workDir
    return cmd.Run() == nil
}

func (g *GitService) branchExists(branchName string) bool {
    cmd := exec.Command("git", "rev-parse", "--verify", branchName)
    cmd.Dir = g.workDir
    return cmd.Run() == nil
}

func (g *GitService) switchToBranch(branchName string) error {
    cmd := exec.Command("git", "checkout", branchName)
    cmd.Dir = g.workDir
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("switching to branch %s: %w\nOutput: %s", branchName, err, output)
    }
    return nil
}

func (g *GitService) createBranch(branchName string) error {
    cmd := exec.Command("git", "checkout", "-b", branchName)
    cmd.Dir = g.workDir
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("creating branch %s: %w\nOutput: %s", branchName, err, output)
    }
    return nil
}

func (g *GitService) formatBranchName(initType, name string) (string, error) {
    slug := strings.ToLower(name)
    slug = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(slug, "-")
    slug = strings.Trim(slug, "-")

    if slug == "" {
        return "", fmt.Errorf("name normalizes to empty string")
    }

    prefixMap := map[string]string{
        "feature":  "feat",
        "bug":      "fix",
        "refactor": "ref",
    }

    prefix, ok := prefixMap[initType]
    if !ok {
        return "", fmt.Errorf("unknown initiative type '%s'", initType)
    }

    return fmt.Sprintf("%s/%s", prefix, slug), nil
}
```

### Step 4: Extend StepResponse (30 min)

**File**: `internal/step/types.go`

Add:

```go
type StepResponse struct {
    // Existing fields
    Directive      string   `json:"directive"`
    HistoryFolder  string   `json:"history_folder"` // Deprecated, use InitiativeFolder
    FilesToRead    []string `json:"files_to_read"`
    ComposedPrompt string   `json:"composed_prompt"`

    // NEW: Cycle support
    InitiativeFolder string  `json:"initiative_folder"`
    CycleFolder      string  `json:"cycle_folder,omitempty"`
    WorkflowPhases   []Phase `json:"workflow_phases,omitempty"`
}

type Phase struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Agents      []string `json:"agents"`
    Outputs     []string `json:"outputs"`
    Parallel    bool     `json:"parallel"`
}
```

### Step 5: Add Feature Step Handler (1 hour)

**File**: `internal/step/feature.go` (new)

```go
package step

import (
    "io/fs"
    "os"
    "path/filepath"

    "github.com/2bit-software/zombiekit/internal/initiative"
)

// executeFeatureStep handles the feature step workflow.
func (s *Service) executeFeatureStep(step *Step, opts *ExecuteOptions) (*StepResponse, error) {
    // Validate required parameters
    if opts == nil || opts.Name == "" {
        return nil, &StepError{
            Code:    "MISSING_NAME",
            Message: "name parameter is required for feature step",
            Hint:    "Provide a name for the feature (e.g., 'user-auth')",
        }
    }

    // Determine initiative type (default to feature)
    initType := initiative.InitiativeType(opts.Type)
    if opts.Type == "" {
        initType = initiative.TypeFeature
    }
    if !initType.IsValid() {
        return nil, &StepError{
            Code:    "INVALID_TYPE",
            Message: "invalid initiative type",
            Hint:    "Type must be one of: feature, bug, refactor",
        }
    }

    // Get or create initiative
    initSvc, err := initiative.NewService(s.workDir)
    if err != nil {
        return nil, err
    }

    var init *initiative.Initiative
    var cycle *initiative.Cycle

    // Check for active initiative
    state, err := s.stateManager.Load()
    if err != nil {
        return nil, err
    }

    if state.IsEmpty() || opts.NewInitiative {
        // Create new initiative
        init, err = initSvc.Create(initType, opts.Name)
        if err != nil {
            return nil, err
        }

        // Create first cycle
        cycleType := mapInitTypeToCycleType(initType)
        cycle, err = initSvc.CreateCycle(init.Path, cycleType, opts.Name)
        if err != nil {
            return nil, err
        }

        // Create git branch
        gitSvc := NewGitService(s.workDir)
        _ = gitSvc.EnsureBranch(string(initType), opts.Name)
    } else {
        // Add cycle to existing initiative
        initPath := filepath.Join(s.workDir, state.Initiative)
        cycleType := mapInitTypeToCycleType(initType)
        cycle, err = initSvc.CreateCycle(initPath, cycleType, opts.Name)
        if err != nil {
            return nil, err
        }
    }

    // Copy templates to cycle folder
    if err := s.copyTemplatesToCycle(cycle.Path); err != nil {
        return nil, err
    }

    // Update state
    // ... state update code ...

    // Build response
    return &StepResponse{
        Directive:        step.Directive,
        HistoryFolder:    cycle.Path, // Backward compat
        InitiativeFolder: filepath.Dir(cycle.Path),
        CycleFolder:      cycle.Path,
        FilesToRead:      s.resolveFiles(step.Files, cycle.Path),
        ComposedPrompt:   "", // Will be composed if profiles exist
        WorkflowPhases:   buildWorkflowPhases(),
    }, nil
}

func mapInitTypeToCycleType(t initiative.InitiativeType) initiative.CycleType {
    switch t {
    case initiative.TypeFeature:
        return initiative.CycleFeat
    case initiative.TypeRefactor:
        return initiative.CycleRef
    case initiative.TypeBug:
        return initiative.CycleFix
    default:
        return initiative.CycleFeat
    }
}

func (s *Service) copyTemplatesToCycle(cyclePath string) error {
    // Implementation using copyEmbeddedFiles pattern
    // See research.md for details
    return nil
}

func buildWorkflowPhases() []Phase {
    return []Phase{
        {Name: "research", Description: "Gather context and domain knowledge",
         Agents: []string{"research-codebase", "research-domain"},
         Outputs: []string{"research.md"}, Parallel: true},
        {Name: "create", Description: "Synthesize specification from research",
         Agents: []string{"spec-writer"},
         Outputs: []string{"spec.md"}, Parallel: false},
        {Name: "audit", Description: "Check specification quality",
         Agents: []string{"audit-completeness", "audit-ai-readiness"},
         Outputs: []string{"audit/{date}.md"}, Parallel: true},
        {Name: "highlight", Description: "Present for user approval",
         Agents: []string{"highlighter"},
         Outputs: []string{}, Parallel: false},
    }
}
```

### Step 6: Add Feature Step Definition (30 min)

**File**: `templates/steps/feature.md` (new)

Create the step definition markdown file with the directive content from the contracts/step-directive.md document.

### Step 7: Add Templates (30 min)

**File**: `templates/templates/research-template.md` (new)

```markdown
---
status: template
updated: {timestamp}
---

# Research: {Feature Name}

## Executive Summary
{2-3 sentence overview - to be filled during research phase}

## Findings

### Codebase Context
{Findings from codebase exploration}

### Domain Knowledge
{Findings from domain research}

## Decision Points
{Decisions that need to be made}

## Recommendations
{Recommended approaches with rationale}

## Sources
{List of sources}
```

### Step 8: Update MCP Tool (30 min)

**File**: `internal/mcp/tools/step/tool.go`

Add new parameters to input schema and handle feature step in Execute method.

### Step 9: Add Tests (1 hour)

- `internal/initiative/cycle_test.go`
- `internal/step/feature_test.go`
- `internal/step/git_test.go`

## Verification

### Unit Tests

```bash
go test ./internal/initiative/... -v
go test ./internal/step/... -v
```

### Integration Test

```bash
# Initialize ZombieKit
brains init

# Create feature via MCP
# Use Claude Code to call mcp_zombiekit__step with step="feature", name="test-feature"

# Verify structure
ls -la history/
ls -la history/*/     # Initiative folder
ls -la history/*/*/   # Cycle folder
```

### Expected Output

```
history/
└── 675d8a3f-feature-test-feature/
    ├── INITIATIVE.md
    └── 675d8a40-feat-test-feature/
        ├── research.md
        ├── spec.md
        └── audit/
```

## Key Implementation Notes

1. **Graceful Git Degradation**: Git operations should never cause failure
2. **Template Override**: Local `.brains/templates/` takes precedence over embedded
3. **Atomic State Updates**: Use temp file + rename for state file writes
4. **Backward Compatibility**: Keep `history_folder` in response
5. **Profile Composition**: Compose all step profiles before returning
