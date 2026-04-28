# Implementation Plan: Contextual /brains.help

## Overview

4 files to modify, 0 new files. The changes fall into two categories:
1. **Go prerequisite** (steps 1-2): Surface already-computed data through the MCP response
2. **Help command rewrite** (step 3): Replace the static template with state-aware instructions

## Step 1: Add missing fields to StatusResponse

**File**: `internal/mcp/tools/initiative/types.go`
**Lines**: 34-45 (StatusResponse struct)

Add 3 fields to the `StatusResponse` struct:

```go
StepStatus     string `json:"step_status,omitempty"`
StepsCompleted int    `json:"steps_completed,omitempty"`
StepsTotal     int    `json:"steps_total,omitempty"`
```

**File**: `internal/mcp/tools/initiative/tool.go`
**Lines**: 284-295 (handleStatus resp construction)

Add the 3 field mappings from StatusResult to StatusResponse:

```go
StepStatus:     status.StepStatus,
StepsCompleted: status.StepsCompleted,
StepsTotal:     status.StepsTotal,
```

**Verification**: Run `go build ./internal/mcp/tools/initiative/...` and existing tests (`go test ./internal/initiative/...`).

**Risk**: Low. Adding fields to a JSON response is backwards-compatible. No consumers will break.

## Step 2: Expand findAvailableDocs to scan all .md files

**File**: `internal/initiative/service.go`
**Lines**: 392-411 (findAvailableDocs function)

Replace the hardcoded `knownDocs` list with a directory scan that finds all `.md` files (excluding `INITIATIVE.md` itself, which is already surfaced via `initiative_file`). Keep the `contracts/` directory check.

```go
func (s *Service) findAvailableDocs(initiativePath string) []string {
    var available []string

    entries, err := os.ReadDir(initiativePath)
    if err != nil {
        return available
    }

    for _, entry := range entries {
        if entry.IsDir() {
            // Check for known directories like contracts/
            if entry.Name() == "contracts" {
                available = append(available, "contracts/")
            }
            continue
        }
        name := entry.Name()
        if strings.HasSuffix(name, ".md") && name != "INITIATIVE.md" {
            available = append(available, name)
        }
    }

    return available
}
```

Note: No `sort.Strings` — follows existing pattern from `internal/step/loader.go:loadAllFromDir()` which uses the same `os.ReadDir` + `.md` filter pattern without sorting. `os.ReadDir` returns entries sorted by name already.

**Verification**: Run existing test `TestService_Status` (which creates `INITIATIVE.md` in the test dir). Add a test case that creates additional `.md` files and verifies they appear in `available_docs`.

**Risk**: Low. Changes scan from allowlist to blocklist (exclude INITIATIVE.md). `os.ReadDir` returns deterministic order. The `Files` field in the response will now include more entries — this is additive and won't break consumers.

## Step 3: Rewrite help.md command

**File**: `embed/commands/help.md`

Replace the entire file with state-aware rendering instructions. The structure:

### 3a. Frontmatter (unchanged)
```yaml
---
name: help
description: Show available commands, current state, and valid actions
---
```

### 3b. Execution instructions

1. **Call `mcp__zombiekit__initiative` with `action: "status"`**
2. **Branch on `active` field**:
   - `false` → render no-initiative mode (3c)
   - `true` → render active-initiative mode (3d)

### 3c. No-initiative mode template

Instructions to render:
- Header: "## ZombieKit Help"
- "No active initiative." message
- "### Start New Work" section with example commands
- Call `mcp__zombiekit__initiative` with `action: "list"` and render "### Recent Initiatives" table (up to 5)
- "### Other Commands" section showing `/brains.help`

### 3d. Active-initiative mode template

Instructions to render:
- **Header**: `## {name}` (parsed from initiative_id by stripping the UUID prefix and type)
- **Metadata line**: Type, progress fraction (`steps_completed`/`steps_total`), history path
- **Progress section**: Step list with current marked. Use step names from a lookup table keyed by `initiative_type`:
  - `feature`: spec, plan, tasks, implement
  - `bug`: investigate, plan, tasks, fix, verify
  - `refactor`: analyze, plan, tasks, implement
  - Mark `current_step` with `<-- current`, completed steps with checkmark-style indicator, pending with blank
- **Step context**: One-line description from embedded lookup table:
  - feature/spec: "Research and write business specification"
  - feature/plan: "Create implementation plan from spec"
  - feature/tasks: "Break plan into discrete implementable tasks"
  - feature/implement: "Execute tasks and write code"
  - bug/investigate: "Investigate the bug and determine root cause"
  - bug/plan: "Plan the fix approach"
  - bug/tasks: "Break fix into discrete tasks"
  - bug/fix: "Implement the fix"
  - bug/verify: "Verify the fix resolves the issue"
  - refactor/analyze: "Analyze code and define refactoring scope"
  - refactor/plan: "Plan the refactoring approach"
  - refactor/tasks: "Break refactor into discrete tasks"
  - refactor/implement: "Execute refactoring tasks"
- **Artifacts section**: List `available_docs` with "(exists)" marker, using `files` for paths
- **Source section** (conditional): Read `initiative_file`, check for `## Source` section, render ticket reference if found
- **Available actions**: Filtered command list per FR-5 rules

### 3e. Step status rendering logic

The command must instruct the AI to build a step table by:
1. Getting the step list for the initiative type
2. For each step: determine status by comparing to `current_step` and `step_status`
   - Steps before `current_step` in the list → "completed" (or check INITIATIVE.md for actual status)
   - Step matching `current_step` → show `step_status` value + `<-- current` marker
   - Steps after `current_step` → "pending"

Note: This is an approximation. For precise status per step, the command should instruct the AI to read INITIATIVE.md and parse the step table directly. The MCP fields give the current step; INITIATIVE.md gives all step statuses.

**Decision**: Have the help command read INITIATIVE.md anyway (for Source section per FR-6), so use the parsed step table for exact statuses rather than inferring from position.

## Step 4: Test the changes

### Go tests
- Run `go test ./internal/initiative/...` — existing tests
- Run `go test ./internal/mcp/tools/initiative/...` — existing tests
- Manually verify: call the MCP tool and confirm new fields appear in response

### Help command tests (manual)
- Run `/brains.help` with no active initiative → verify no-initiative output
- Run `/brains.help` with current active initiative → verify active output with correct steps, artifacts, progress

## Dependency Order

```
Step 1 (types.go + tool.go) ──┐
                               ├── Step 3 (help.md) ── Step 4 (test)
Step 2 (service.go)  ─────────┘
```

Steps 1 and 2 are independent and can be done in parallel. Step 3 depends on both. Step 4 validates everything.

## Estimated Scope

- Step 1: ~6 lines changed across 2 files
- Step 2: ~20 lines changed in 1 file
- Step 3: ~150-200 lines (full help.md rewrite)
- Step 4: Manual testing + optional test additions

Total: ~230 lines across 3 files (types.go, tool.go, service.go, help.md)
