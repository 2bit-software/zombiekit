# Refactor Plan: Data-Driven Workflows

## Overview

7 atomic steps, each independently committable. Steps 1-3 are foundation (can't be parallelized). Steps 4-5 are the core change. Steps 6-7 are cleanup.

---

## Step 1: Add regression baseline tests

**Goal**: Lock in current behavior before changing anything.

**Files**:
- `internal/initiative/markdown_test.go` — add `TestParseInitiativeMD_ThreeColumnBaseline`
- `internal/step/service_test.go` — add `TestGetWorkflowSteps_CurrentBehavior`
- `internal/initiative/service_test.go` — add `TestCreateInitiativeMD_CurrentFormat`

**Tests to add**:
```go
// Verify 3-column table parsing is preserved
func TestParseInitiativeMD_ThreeColumnBaseline(t *testing.T) {
    // Write a 3-column INITIATIVE.md
    // Parse it
    // Assert steps parsed correctly with Name, Status, Updated
    // Assert Profile field is empty (doesn't exist yet, but will after step 3)
}

// Verify GetWorkflowSteps currently reads from profile frontmatter
func TestGetWorkflowSteps_CurrentBehavior(t *testing.T) {
    // Set up a profile with steps: frontmatter
    // Call GetWorkflowSteps
    // Assert returns the steps from profile
}

// Verify INITIATIVE.md format output
func TestCreateInitiativeMD_CurrentFormat(t *testing.T) {
    // Create initiative with steps
    // Read INITIATIVE.md
    // Assert 3-column format
}
```

**Commit**: `test(workflow): add regression baselines before data-driven refactor`

---

## Step 2: Add `steps:` parsing to workflow service

**Goal**: `parseWorkflow()` can read step sequences from workflow frontmatter.

**Files**:
- `internal/workflow/service.go` — extend `parseWorkflow()` and `Workflow` struct
- `internal/workflow/service_test.go` — add tests for step parsing

**Changes**:
```go
// Extend Workflow struct
type Workflow struct {
    Name        string
    Description string
    Steps       []WorkflowStep  // NEW
    Content     string
    Path        string
    Source      string
}

// WorkflowStep defines a step in a workflow sequence.
type WorkflowStep struct {
    Name     string   `yaml:"name"`
    Profiles []string `yaml:"profiles"`
}

// Update parseWorkflow to parse steps from frontmatter
func parseWorkflow(content []byte, name, path, source string) (*Workflow, error) {
    // Parse extended frontmatter:
    // ---
    // name: feature
    // description: ...
    // steps:
    //   - name: spec
    //     profiles: [feature]
    //   - name: plan
    //     profiles: [plan]
    // ---
}
```

**Tests to add**:
```go
func TestParseWorkflow_WithSteps(t *testing.T) {
    // Workflow with steps in frontmatter → Steps field populated
}

func TestParseWorkflow_WithoutSteps(t *testing.T) {
    // Workflow without steps → Steps field is nil/empty (not an error)
}

func TestParseWorkflow_MultipleProfilesPerStep(t *testing.T) {
    // Step with profiles: [implement, automode] → both captured
}

func TestParseWorkflow_EmptyStepsArray(t *testing.T) {
    // steps: [] → empty slice, not nil
}

func TestService_Load_WithSteps(t *testing.T) {
    // Load a workflow file with steps → verify steps accessible on result
}
```

**Commit**: `feat(workflow): parse steps array from workflow frontmatter`

---

## Step 3: Add Profile column to INITIATIVE.md parser

**Goal**: `ParsedStep` gains a `Profile` field. Parser handles both 3-column and 4-column tables.

**Files**:
- `internal/initiative/markdown.go` — extend `ParsedStep`, update regex, update formatter
- `internal/initiative/markdown_test.go` — exhaustive tests for both formats

**Changes**:
```go
type ParsedStep struct {
    Name    string
    Profile string     // NEW — profile name(s) for this step
    Status  StepStatus
    Updated string
}

// New regex for 4-column: | Step | Profile | Status | Updated |
var stepRowRe4Col = regexp.MustCompile(`^\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|`)

// Detection: if header contains "Profile", use 4-column parser
// Otherwise use 3-column parser (backwards compat)
```

**Format**:
```markdown
| Step | Profile | Status | Updated |
|------|---------|--------|---------|
| spec | feature | in_progress | 2026-04-30 10:00 |
| plan | plan | pending | - |
| tasks | tasks | pending | - |
| implement | implement | pending | - |
```

**Tests to add**:
```go
func TestParseInitiativeMD_FourColumnTable(t *testing.T) {
    // Parse 4-column → Profile field populated
}

func TestParseInitiativeMD_ThreeColumnTable_BackwardsCompat(t *testing.T) {
    // Parse 3-column → Profile field is empty, rest works
}

func TestParsedInitiative_FormatSteps_WithProfiles(t *testing.T) {
    // Steps with Profile set → 4-column output
}

func TestParsedInitiative_FormatSteps_WithoutProfiles(t *testing.T) {
    // Steps with empty Profile → 3-column output (legacy compat)
}

func TestParsedInitiative_RoundTrip_FourColumn(t *testing.T) {
    // Write 4-col → parse → assert equality
}

func TestParsedInitiative_RoundTrip_ThreeColumn(t *testing.T) {
    // Write 3-col → parse → assert equality
}

func TestParseStepRow_FourColumn(t *testing.T) {
    // Individual row parsing for 4-column
}

func TestParseStepRow_ThreeColumn(t *testing.T) {
    // Individual row parsing for 3-column (regression)
}

func TestParsedInitiative_UpdateStepStatus_WithProfile(t *testing.T) {
    // Update status preserves Profile field
}

func TestParsedInitiative_WriteTo_PreservesProfileColumn(t *testing.T) {
    // Parse → modify step → write → re-parse → Profile still there
}
```

**Commit**: `feat(initiative): support 4-column step table with Profile column`

---

## Step 4: Wire workflow steps into initiative creation

**Goal**: `loadWorkflowSteps()` reads from workflow files. `createInitiativeMD()` writes Profile column.

**Files**:
- `internal/mcp/tools/initiative/tool.go` — rewrite `loadWorkflowSteps()` to use workflow service
- `internal/initiative/service.go` — update `createInitiativeMD()` to write Profile column
- `internal/mcp/tools/initiative/tool_test.go` — update/add tests
- `internal/initiative/service_test.go` — update tests

**Changes**:
```go
// tool.go: loadWorkflowSteps now reads from workflow service
func loadWorkflowSteps(dir, initType string) ([]internalInit.WorkflowStep, error) {
    wfSvc := workflow.NewServiceForSubdir(dir, "workflows")
    wf, err := wfSvc.Load(initType)
    if err != nil {
        return nil, nil // No workflow defined, not an error
    }
    if len(wf.Steps) == 0 {
        return nil, nil
    }
    // Convert workflow.WorkflowStep → initiative.WorkflowStep
    initSteps := make([]internalInit.WorkflowStep, len(wf.Steps))
    for i, ws := range wf.Steps {
        initSteps[i] = internalInit.WorkflowStep{
            Name:    ws.Name,
            Profile: strings.Join(ws.Profiles, ","), // Join multiple profiles
        }
    }
    return initSteps, nil
}

// service.go: createInitiativeMD writes Profile column
func (s *Service) createInitiativeMD(init *Initiative, steps []WorkflowStep) error {
    // If any step has a Profile, use 4-column format
    // Otherwise use 3-column format (backwards compat for tests/legacy)
}
```

**Tests to add**:
```go
func TestLoadWorkflowSteps_FromWorkflowFile(t *testing.T) {
    // Set up workflow file with steps → verify correct steps returned
}

func TestLoadWorkflowSteps_NoWorkflowFile(t *testing.T) {
    // No workflow file → returns nil, nil (not error)
}

func TestLoadWorkflowSteps_WorkflowWithoutSteps(t *testing.T) {
    // Workflow exists but no steps field → returns nil, nil
}

func TestCreateInitiativeMD_WritesProfileColumn(t *testing.T) {
    // Create with steps that have profiles → verify 4-column table
}

func TestCreateInitiativeMD_Integration(t *testing.T) {
    // Create initiative → parse INITIATIVE.md → verify steps have profiles
}
```

**Commit**: `refactor(initiative): read step sequence from workflow files, write Profile column`

---

## Step 5: Add `steps:` to workflow files

**Goal**: Embedded workflow files contain their step sequences in frontmatter.

**Files**:
- `embed/workflows/feature.md` — add steps frontmatter
- `embed/workflows/bug.md` — add steps frontmatter
- `embed/workflows/refactor.md` — add steps frontmatter
- `embed/workflows/feature-light.md` — add steps frontmatter
- `embed/workflows/unmanaged.md` — add steps frontmatter (empty or minimal)

**Format for feature.md**:
```yaml
---
name: feature
description: Feature specification workflow...
steps:
  - name: spec
    profiles: [feature]
  - name: plan
    profiles: [plan]
  - name: tasks
    profiles: [tasks]
  - name: implement
    profiles: [implement]
---
```

**Format for bug.md**:
```yaml
---
name: bug
description: Bug investigation workflow...
steps:
  - name: investigate
    profiles: [bug]
  - name: plan
    profiles: [plan]
  - name: tasks
    profiles: [tasks]
  - name: fix
    profiles: [implement]
  - name: verify
    profiles: [audit]
---
```

**Format for refactor.md**:
```yaml
---
name: refactor
description: Refactoring workflow...
steps:
  - name: analyze
    profiles: [refactor]
  - name: plan
    profiles: [plan]
  - name: tasks
    profiles: [tasks]
  - name: implement
    profiles: [implement]
---
```

**Tests to add**:
```go
func TestEmbeddedWorkflows_HaveSteps(t *testing.T) {
    // Load each embedded workflow → verify Steps is non-empty
    // Verify each step has at least one profile
}

func TestEmbeddedWorkflows_StepProfilesExist(t *testing.T) {
    // For each step's profiles, verify the profile actually exists in embedded profiles
}
```

**Commit**: `feat(workflows): add step sequences to embedded workflow frontmatter`

---

## Step 6: Update `next` command to read Profile column

**Goal**: The LLM instruction in `embed/commands/next.md` tells the agent to read the Profile column.

**Files**:
- `embed/commands/next.md` — update "Load Next Profile" section

**Change in next.md**:
```markdown
5. **Load Next Profile**
   - Read the Profile column from the step table row
   - If Profile column exists: use that value as the profile name(s)
   - If Profile column is missing (legacy 3-column table): use the step name as profile name
   - If multiple profiles (comma-separated): compose all of them
   - Call `mcp__zombiekit__profile-compose` with the resolved profile(s)
```

**Commit**: `docs(commands): update next command to read Profile column from step table`

---

## Step 7: Remove `steps:` and `handoffs:` from profile frontmatter

**Goal**: Profiles no longer contain routing metadata. They're pure instruction content.

**Files**:
- `embed/profiles/feature.md` — remove `steps:` and `handoffs:` from frontmatter
- `embed/profiles/bug.md` — remove `steps:` and `handoffs:` from frontmatter
- `embed/profiles/refactor.md` — remove `steps:` and `handoffs:` from frontmatter
- `internal/step/service.go` — remove or deprecate `GetWorkflowSteps()` (now unused)

**Tests to add**:
```go
func TestGetWorkflowSteps_FallbackToWorkflow(t *testing.T) {
    // Profile has no steps, workflow has steps → returns workflow steps
}

func TestGetWorkflowSteps_Deprecated(t *testing.T) {
    // If we remove GetWorkflowSteps entirely, verify nothing calls it
}
```

**Commit**: `refactor(profiles): remove routing metadata from profile frontmatter`

---

## Execution Order

```
Step 1 (tests)       → baseline tests, no code changes
Step 2 (workflow)    → additive, no breaking changes
Step 3 (parser)      → additive, backwards compatible
Step 4 (wiring)      → the actual behavior change (critical)
Step 5 (data)        → populates the new frontmatter
Step 6 (command)     → LLM instruction update
Step 7 (cleanup)     → removes old metadata
```

Steps 1-3 are safe to land on main individually. Step 4+5 should be landed together (4 without 5 would mean no workflow has steps defined yet). Step 6 and 7 are independent cleanup.

## Rollback Strategy

- **Steps 1-3**: Pure additions. Revert is trivial but unnecessary — nothing breaks.
- **Step 4**: Revert `loadWorkflowSteps()` to read profiles instead of workflows. One function change.
- **Step 5**: Revert frontmatter additions. No code depends on them until step 4 is active.
- **Step 6**: Revert next.md. LLM falls back to using step name as profile (current behavior).
- **Step 7**: Re-add frontmatter to profiles. Only needed if step 4 is also reverted.

## Multiple Profiles Per Step

The `profiles: [implement, automode]` case is handled by:
1. Storing as comma-separated in the Profile column: `implement,automode`
2. `next` command splits on comma and passes array to `profile-compose`
3. `profile-compose` already supports multiple profiles

This enables automode propagation without special-casing in the `next` command.
