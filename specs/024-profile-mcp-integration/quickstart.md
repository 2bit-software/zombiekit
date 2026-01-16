# Quickstart: Profile-MCP Integration

**Feature**: 024-profile-mcp-integration
**Date**: 2025-12-24

## Prerequisites

- Go 1.24.0+
- Working zombiekit installation (`go build -o bin/brains ./cmd/brains`)
- An initialized project (`brains init`)

## Development Setup

```bash
# Clone and build
cd /path/to/zombiekit
go mod tidy
go build -o bin/brains ./cmd/brains

# Run tests
go test ./internal/step/... -v
go test ./internal/mcp/tools/step/... -v
```

## Key Files

### Step Profiles (Primary Change Area)

Location: `templates/steps/`

| File | Description |
|------|-------------|
| `feature.md` | Multi-phase specification workflow |
| `bug.md` | Bug investigation workflow |
| `refactor.md` | Refactoring specification workflow |
| `plan.md` | Implementation planning |
| `tasks.md` | Task breakdown generation |
| `eat.md` | Task execution |
| `audit.md` | Cross-artifact alignment |
| `clarify.md` | Ambiguity detection |

### Go Code

| File | Description |
|------|-------------|
| `internal/step/types.go` | Step, StepResponse, Phase types |
| `internal/step/service.go` | Step execution logic |
| `internal/step/feature.go` | Workflow phase builder |
| `internal/step/loader.go` | Profile loading |
| `internal/mcp/tools/step/tool.go` | MCP step tool |
| `internal/mcp/tools/initiative/tool.go` | MCP initiative tool |

## Testing Changes

### Unit Tests

```bash
# Run step service tests
go test ./internal/step/... -v -count=1

# Run MCP tool tests
go test ./internal/mcp/tools/step/... -v -count=1
go test ./internal/mcp/tools/initiative/... -v -count=1
```

### Manual Testing

1. **Initialize project**:
   ```bash
   ./bin/brains init
   ```

2. **Create initiative via MCP** (or use Claude Code):
   ```json
   {
     "action": "create",
     "dir": ".",
     "type": "feature",
     "name": "test-feature"
   }
   ```

3. **Execute step**:
   ```json
   {
     "step": "feature",
     "dir": "."
   }
   ```

4. **Verify response** contains:
   - `directive` (non-empty)
   - `files_to_read` (resolved paths)
   - `workflow_phases` (4 phases for feature)
   - `cycle_folder` (valid path)

## Profile Structure

Each step profile follows this structure:

```markdown
---
name: step-name
description: What this step does
profiles:
  - profile1
  - profile2
files:
  - "pattern/*.md"
type: step
---
# Step Name Workflow

## Context
Agent responsibilities and automatic behaviors.

## Response Handling
How to interpret MCP response fields.

## Workflow / Phases
Phase-by-phase or single-phase instructions.

## Output
What artifacts to create.

## Success Criteria
- [ ] Criterion 1
- [ ] Criterion 2

## Behavior Rules
1. Rule 1
2. Rule 2
```

## Common Tasks

### Updating a Profile

1. Edit `templates/steps/{step}.md`
2. Rebuild: `go build -o bin/brains ./cmd/brains`
3. Test: `go test ./internal/step/... -v -run TestLoader`

### Adding Response Handling Section

Add after "## Context":

```markdown
## Response Handling

When you receive the MCP response, process fields in this order:

1. **Check `prerequisites.met`**: If false, follow `prerequisites.hint`
2. **Read `files_to_read`**: Load each file for context
3. **Parse `workflow_phases`**: (Multi-phase only) Understand phase structure
4. **Check `next_task`**: (Eat step only) Know which task to implement
5. **Follow directive**: Execute according to this document
6. **Output to `cycle_folder`**: Save artifacts here
```

### Testing Profile Loading

```go
func TestProfileLoading(t *testing.T) {
    svc, err := step.NewService(".")
    require.NoError(t, err)

    s, err := svc.GetStep("feature")
    require.NoError(t, err)

    assert.Equal(t, "feature", s.Name)
    assert.Contains(t, s.Directive, "Research")
    assert.Equal(t, step.SourceEmbedded, s.Source)
}
```

## Debugging

### Profile Not Loading

Check source priority:
1. Local: `.brains/steps/{step}.md`
2. Global: `~/.brains/steps/{step}.md`
3. Embedded: `templates/steps/{step}.md`

### Empty composed_prompt

Verify:
1. Profile names in frontmatter are valid
2. Profiles exist in profile system
3. Profile service initialized correctly

### Files Not Resolving

Verify:
1. Patterns are relative to cycle folder
2. Files actually exist at those paths
3. Glob patterns are valid

## Next Steps After Implementation

1. Run full test suite: `go test ./... -v`
2. Manual workflow test: Create initiative → Run feature step → Verify response
3. Update embedded FS: Ensure `embed.go` includes `templates/steps/`
4. Build release: `go build -o bin/brains ./cmd/brains`
