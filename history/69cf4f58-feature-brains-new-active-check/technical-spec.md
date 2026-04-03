# Technical Spec: Active Initiative Detection

## Service Layer Change

### `internal/initiative/service.go` — Add `Abandon()` method

```go
func (s *Service) Abandon() (*AbandonResult, error) {
    state, err := s.stateManager.Load()
    if err != nil {
        return nil, err
    }

    if state.IsEmpty() {
        return nil, &InitiativeError{
            Code:    "NO_ACTIVE_INITIATIVE",
            Message: "no active initiative to abandon",
            Hint:    "There is no active initiative to abandon",
        }
    }

    initPath := filepath.Join(s.workDir, state.Initiative)
    initiativeID := filepath.Base(state.Initiative)

    // Remove history folder — tolerate if already deleted
    if err := os.RemoveAll(initPath); err != nil {
        return nil, fmt.Errorf("removing initiative folder: %w", err)
    }

    // Clear active state
    if err := s.stateManager.Clear(); err != nil {
        return nil, fmt.Errorf("clearing state: %w", err)
    }

    return &AbandonResult{
        InitiativeID: initiativeID,
        DeletedPath:  initPath,
    }, nil
}
```

### New type in `types.go`:

```go
type AbandonResult struct {
    InitiativeID string
    DeletedPath  string
}
```

## MCP Tool Layer Change

### `internal/mcp/tools/initiative/tool.go`

**Schema update** — add `"abandon"` to enum and description.

**Handler**:

```go
func (t *Tool) handleAbandon(ctx context.Context, dir string) (string, error) {
    initSvc, err := internalInit.NewService(dir)
    if err != nil {
        return "", fmt.Errorf("creating service: %w", err)
    }

    result, err := initSvc.Abandon()
    if err != nil {
        var initErr *internalInit.InitiativeError
        if errors.As(err, &initErr) {
            return "", &ToolError{
                Code:    initErr.Code,
                Message: initErr.Message,
                Hint:    initErr.Hint,
            }
        }
        return "", err
    }

    response := map[string]any{
        "action":        "abandon",
        "initiative_id": result.InitiativeID,
        "deleted_path":  result.DeletedPath,
        "abandoned_at":  time.Now().Format(time.RFC3339),
    }

    data, err := json.MarshalIndent(response, "", "  ")
    if err != nil {
        return "", fmt.Errorf("marshaling response: %w", err)
    }
    return string(data), nil
}
```

## Command Markdown Change

### `embed/commands/new.md` — Add initiative check section

Insert before "### Classification Rules":

```markdown
### Pre-Classification: Active Initiative Check

Before classifying the user's input, check for an active initiative:

1. Call `mcp__zombiekit__initiative` with `action: "status"` and `dir` set to the working directory
2. If `active: false` — skip to classification
3. If `active: true` — display the active initiative details and ask the user:

> **Active initiative detected:**
> - **ID**: {initiative_id}
> - **Name**: {name}
> - **Type**: {type}
> - **Started**: {started}
> - **Current step**: {current_step} ({completed}/{total} steps)
>
> How would you like to proceed?
> 1. **Close out early** — Mark the current initiative as complete (keeps history) and start new work
> 2. **Delete history** — Remove the current initiative entirely and start fresh
> 3. **Cancel** — Keep working on the current initiative

Use `AskUserQuestion` to present these options. Then:
- **Option 1**: Call `mcp__zombiekit__initiative` with `action: "complete"`, then proceed to classification
- **Option 2**: Call `mcp__zombiekit__initiative` with `action: "abandon"`, then proceed to classification
- **Option 3**: Stop execution. Tell the user: "Continuing with the current initiative. Use `/brains.next` to advance."
```
