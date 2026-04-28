# Technical Spec: Contextual /brains.help

## Changes by File

### 1. `internal/mcp/tools/initiative/types.go`

Add 3 fields to `StatusResponse` struct (after `CurrentStep`):

```go
type StatusResponse struct {
    Action         string   `json:"action"`
    Active         bool     `json:"active"`
    InitiativeID   string   `json:"initiative_id,omitempty"`
    InitiativeType string   `json:"initiative_type,omitempty"`
    CurrentStep    string   `json:"current_step,omitempty"`
    StepStatus     string   `json:"step_status,omitempty"`      // NEW
    StepsCompleted int      `json:"steps_completed,omitempty"`  // NEW
    StepsTotal     int      `json:"steps_total,omitempty"`      // NEW
    AvailableDocs  []string `json:"available_docs,omitempty"`
    SuggestedNext  string   `json:"suggested_next,omitempty"`
    HistoryPath    string   `json:"history_path,omitempty"`
    InitiativeFile string   `json:"initiative_file,omitempty"`
    Files          []string `json:"files,omitempty"`
}
```

### 2. `internal/mcp/tools/initiative/tool.go`

In `handleStatus()`, add field mappings at line ~289 (after `CurrentStep`):

```go
resp := StatusResponse{
    Action:         "status",
    Active:         status.Active,
    InitiativeID:   status.InitiativeID,
    InitiativeType: status.InitiativeType,
    CurrentStep:    status.CurrentStep,
    StepStatus:     status.StepStatus,          // NEW
    StepsCompleted: status.StepsCompleted,       // NEW
    StepsTotal:     status.StepsTotal,           // NEW
    AvailableDocs:  status.AvailableDocs,
    SuggestedNext:  status.SuggestedNext,
    HistoryPath:    status.HistoryPath,
    InitiativeFile: status.InitiativeFile,
    Files:          status.Files,
}
```

### 3. `internal/initiative/service.go`

Replace `findAvailableDocs()` (lines 392-411) with a directory scan:

```go
func (s *Service) findAvailableDocs(initiativePath string) []string {
    var available []string

    entries, err := os.ReadDir(initiativePath)
    if err != nil {
        return available
    }

    for _, entry := range entries {
        if entry.IsDir() {
            if entry.Name() == "contracts" {
                available = append(available, "contracts/")
            }
            continue
        }
        name := entry.Name()
        if strings.HasSuffix(name, ".md") && name != InitiativeMDFile {
            available = append(available, name)
        }
    }

    return available
}
```

No sort needed — `os.ReadDir` returns entries sorted by name per Go spec. Follows pattern from `internal/step/loader.go:loadAllFromDir()`.

### 4. `embed/commands/help.md`

Full rewrite. The file is a prompt template that instructs the AI agent. Structure:

```markdown
---
name: help
description: Show available commands, current state, and valid actions
---

## User Input
{pass through $ARGUMENTS}

## Help Workflow

### Step 1: Load State
Call mcp__zombiekit__initiative with action: "status"

### Step 2: Render Output
Branch on active field...

#### No Initiative Mode
- Header, getting started examples
- Call initiative list, render recent table
- Available commands (brains.new, brains.help only)

#### Active Initiative Mode
- Read initiative_file for full step table and Source section
- Render: header, progress, step context, artifacts, source (if exists), actions
- Step lookup tables embedded in the template
- Command filtering rules embedded in the template
```

## Data Flow

```
User runs /brains.help
  → Claude loads embed/commands/help.md via workflow-load
  → help.md instructs Claude to:
    1. Call initiative status MCP tool
       → Returns: active, id, type, current_step, step_status,
                  steps_completed, steps_total, available_docs,
                  suggested_next, history_path, initiative_file, files
    2. If active: Read initiative_file (INITIATIVE.md)
       → Parse step table for per-step status
       → Parse Source section for Linear ticket
    3. If active: Call initiative list (for context)
    4. Render markdown output based on state
```

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| No `.brains/` directory | `active: false`, show no-initiative mode |
| Active but INITIATIVE.md missing/corrupt | Show header with ID/type, skip progress/step sections |
| All steps completed | Show progress as complete, primary action = `/brains.complete` |
| No `current_step` (empty) | Show step table from INITIATIVE.md without marker |
| `initiative list` returns empty | Skip "Recent Initiatives" section |
| Unknown `initiative_type` | Show generic step list from INITIATIVE.md, skip step descriptions |
