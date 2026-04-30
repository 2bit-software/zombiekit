# Dependency Analysis

## Affected Components

### 1. Workflow Service (`internal/workflow/`)

| File | Change | Reason |
|------|--------|--------|
| `service.go` | Add `steps` field parsing to `parseWorkflow()` | Workflow frontmatter needs to include step sequence |
| `service.go` | Add `Steps` field to `Workflow` struct | Expose parsed step data |
| `service_test.go` | Add tests for step parsing from workflow frontmatter | Verify new behavior |

**Current state**: `parseWorkflow()` (line 164) only parses `name` and `description` from frontmatter. Ignores any `steps:` field.

### 2. Workflow Files (`embed/workflows/`)

| File | Change | Reason |
|------|--------|--------|
| `feature.md` | Add `steps:` to YAML frontmatter | Define step sequence for feature workflow |
| `bug.md` | Add `steps:` to YAML frontmatter | Define step sequence for bug workflow |
| `refactor.md` | Add `steps:` to YAML frontmatter | Define step sequence for refactor workflow |
| `feature-light.md` | Add `steps:` to YAML frontmatter | Define step sequence for feature-light |
| `unmanaged.md` | Add `steps:` to YAML frontmatter (minimal) | Define step sequence for unmanaged |

### 3. Step Service (`internal/step/`)

| File | Change | Reason |
|------|--------|--------|
| `service.go` | Rewrite `GetWorkflowSteps()` to read from workflow files | Currently reads from profile frontmatter |
| `types.go` | Keep `WorkflowStep` and `WorkflowMeta` types | Still needed, just sourced differently |
| `service_test.go` | Update tests to use workflow files as source | Verify new resolution path |

**Current state**: `GetWorkflowSteps()` (line 238) reads the profile file and parses its `steps:` frontmatter. This is the key function that needs to change its source from profiles to workflows.

### 4. Initiative Markdown Parser (`internal/initiative/`)

| File | Change | Reason |
|------|--------|--------|
| `markdown.go` | Add `Profile` field to `ParsedStep` | Step table now has a Profile column |
| `markdown.go` | Update `stepRowRe` regex for 4-column table | Parse `| Step | Profile | Status | Updated |` |
| `markdown.go` | Update `formatSteps()` for 4-column output | Write the new format |
| `markdown.go` | Add backwards compat for 3-column table | Old INITIATIVE.md files must still parse |
| `markdown_test.go` | Add tests for 4-column parsing and backwards compat | Cover both formats |

**Current state**: `ParsedStep` has `Name`, `Status`, `Updated`. Table regex matches 3 columns.

### 5. Initiative Service (`internal/initiative/`)

| File | Change | Reason |
|------|--------|--------|
| `service.go` | Update `createInitiativeMD()` to write Profile column | New format includes profile mapping |
| `service.go` | `WorkflowStep` already has `Name` and `Profile` — no change needed | Already correct |
| `service_test.go` | Update tests to verify Profile column in output | |

**Current state**: `createInitiativeMD()` (line 461) writes `step.Name` to the table. `WorkflowStep` struct already has `Profile` field but it's not written.

### 6. Profile Files (`embed/profiles/`)

| File | Change | Reason |
|------|--------|--------|
| `feature.md` | Remove `steps:` and `handoffs:` from frontmatter | Profiles no longer own routing |
| `bug.md` | Remove `steps:` and `handoffs:` from frontmatter | Profiles no longer own routing |
| `refactor.md` | Remove `steps:` and `handoffs:` from frontmatter | Profiles no longer own routing |

### 7. MCP Initiative Tool (`internal/mcp/tools/initiative/`)

| File | Change | Reason |
|------|--------|--------|
| `tool.go` | Update `loadWorkflowSteps()` to use workflow service | Read steps from workflow files, not profiles |
| `tool_test.go` | Update tests | Verify new resolution |

**Current state**: `loadWorkflowSteps()` (line 251) creates a `step.Service` and calls `GetWorkflowSteps()`. After refactor, this should use the workflow service directly.

### 8. Command Files (`embed/commands/`)

| File | Change | Reason |
|------|--------|--------|
| `next.md` | Update "Load Next Profile" instruction | Tell LLM to read Profile column from step table |

## Call Graph (Post-Refactor)

```
initiative create (MCP tool)
  → workflow.Service.Load(type)         # Load workflow by type name
  → workflow.Steps                      # Get step sequence from frontmatter
  → initiative.Service.Create(steps)    # Create with step sequence
  → createInitiativeMD(steps)           # Write step table WITH profile column

/brains.next (LLM reads INITIATIVE.md)
  → Parse step table: | Step | Profile | Status | Updated |
  → Find next pending step
  → Read Profile column value
  → Call profile-compose with that profile name
```

## Risk Areas

1. **Backwards compatibility**: Existing INITIATIVE.md files have 3-column tables. Parser must handle both.
2. **Profile frontmatter removal**: If any code still reads `steps:` from profiles, it will get empty results.
3. **Workflow frontmatter parsing**: `parseWorkflow()` currently uses a simple struct. Adding `steps:` requires array parsing.
