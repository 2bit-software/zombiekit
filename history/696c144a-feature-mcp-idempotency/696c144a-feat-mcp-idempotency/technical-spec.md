# Technical Specification: MCP Command Idempotency

## Architecture Overview

The implementation adds idempotency to the `initiative create` MCP command through two mechanisms:

1. **Initiative-level idempotency**: Detect when active initiative matches requested name+type and return it instead of creating a new one
2. **File-level idempotency**: Skip copying template files when destination already has content

```
┌─────────────────────────────────────────────────────────────┐
│                    handleCreate()                            │
├─────────────────────────────────────────────────────────────┤
│ 1. Validate parameters (existing)                           │
│ 2. Check for active initiative (existing)                   │
│ 3. NEW: Check if active matches name+type                   │
│    ├─ Match? → Return existing + run template copy          │
│    └─ No match? → Error (initiative already active)         │
│ 4. Create new initiative (existing, only if no active)      │
│ 5. Run template copy with skip logic                        │
│ 6. Return response with AlreadyExisted flag                 │
└─────────────────────────────────────────────────────────────┘
```

## API Changes

### Initiative Tool - Create Action

**Request:** No changes

**Response (CreateResponse):**

```go
type CreateResponse struct {
    Action         string   `json:"action"`          // "create"
    InitiativeID   string   `json:"initiative_id"`   // e.g., "696c144a-feature-auth"
    InitiativePath string   `json:"initiative_path"` // e.g., "history/696c144a-feature-auth"
    CycleID        string   `json:"cycle_id"`        // e.g., "696c144a-feat-auth"
    CyclePath      string   `json:"cycle_path"`      // Full path to cycle folder
    Branch         string   `json:"branch"`          // Git branch name
    Type           string   `json:"type"`            // "feature" | "bug" | "refactor"
    Name           string   `json:"name"`            // Normalized name
    NextStep       string   `json:"next_step"`       // Suggested next step
    // NEW FIELDS
    AlreadyExisted bool     `json:"already_existed"`        // true if returning existing
    SkippedFiles   []string `json:"skipped_files,omitempty"` // Files not overwritten
    CopiedFiles    []string `json:"copied_files,omitempty"`  // Files that were copied
}
```

**Behavior Matrix:**

| Active Initiative | Requested Name+Type | Result |
|-------------------|---------------------|--------|
| None | Any | Create new, AlreadyExisted=false |
| Matches request | Same | Return existing, AlreadyExisted=true |
| Different | Any | Error: INITIATIVE_ALREADY_ACTIVE |

## Implementation Details

### 1. FindActiveByNameAndType Method

**Location:** `internal/initiative/service.go`

```go
// FindActiveByNameAndType returns the active initiative if it matches the given name and type.
// Returns nil if no active initiative exists or if the active initiative doesn't match.
func (s *Service) FindActiveByNameAndType(name string, initType InitiativeType) (*Initiative, error) {
    active, err := s.GetActive()
    if err != nil {
        return nil, err
    }
    if active == nil {
        return nil, nil
    }

    // Use same normalization as Create() uses
    normalizedName := normalizeName(name)

    if active.Name == normalizedName && active.Type == initType {
        return active, nil
    }
    return nil, nil
}
```

**Key Design Decisions:**
- Reuses existing `normalizeName()` function for consistency
- Compares Type directly (InitiativeType is comparable)
- Returns nil on no match (not an error) - caller decides behavior

### 2. copyTemplateIfNotExists Helper

**Location:** `internal/mcp/tools/initiative/tool.go`

```go
// copyTemplateIfNotExists copies a template to destination if destination doesn't exist
// or is empty/whitespace-only. Returns whether the file was copied.
func copyTemplateIfNotExists(templateContent []byte, destPath string) (copied bool, err error) {
    // Check if destination exists
    if _, err := os.Stat(destPath); err == nil {
        // File exists - check if it has non-whitespace content
        content, err := os.ReadFile(destPath)
        if err != nil {
            return false, fmt.Errorf("reading existing file %s: %w", destPath, err)
        }
        if len(bytes.TrimSpace(content)) > 0 {
            return false, nil // Skip - file has content
        }
        // File is empty or whitespace-only, fall through to overwrite
    } else if !os.IsNotExist(err) {
        // Unexpected error (not just "file doesn't exist")
        return false, fmt.Errorf("checking file %s: %w", destPath, err)
    }

    // Copy template (file doesn't exist OR is empty/whitespace)
    if err := os.WriteFile(destPath, templateContent, 0644); err != nil {
        return false, fmt.Errorf("writing template to %s: %w", destPath, err)
    }
    return true, nil
}
```

**Key Design Decisions:**
- Takes template content as parameter (not path) - simplifies testing, works with both local overrides and embedded FS
- Uses `bytes.TrimSpace` for whitespace detection
- Returns bool for skip/copy tracking
- Wraps errors with context

### 3. Modified copyTemplatesToCycle

**Location:** `internal/mcp/tools/initiative/tool.go`

```go
// copyTemplatesToCycle copies spec and research templates to the cycle folder.
// Returns lists of skipped and copied file names.
func (t *Tool) copyTemplatesToCycle(workDir, cyclePath string) (skipped, copied []string, err error) {
    embFS := t.embeddedFS
    if embFS == nil {
        embFS = step.GetTemplateFS()
    }
    if embFS == nil {
        return nil, nil, fmt.Errorf("no embedded template filesystem available")
    }

    templates := []struct {
        src  string
        dest string
    }{
        {"templates/spec-template.md", "spec.md"},
        {"templates/research-template.md", "research.md"},
    }

    for _, tmpl := range templates {
        // First check if local override exists
        localPath := filepath.Join(workDir, ".brains", "templates", filepath.Base(tmpl.src))
        var content []byte

        if _, statErr := os.Stat(localPath); statErr == nil {
            content, err = os.ReadFile(localPath)
        } else {
            content, err = fs.ReadFile(embFS, tmpl.src)
        }

        if err != nil {
            return nil, nil, fmt.Errorf("reading template %s: %w", tmpl.src, err)
        }

        destPath := filepath.Join(cyclePath, tmpl.dest)
        wasCopied, err := copyTemplateIfNotExists(content, destPath)
        if err != nil {
            return nil, nil, fmt.Errorf("copying template %s: %w", tmpl.dest, err)
        }

        if wasCopied {
            copied = append(copied, tmpl.dest)
        } else {
            skipped = append(skipped, tmpl.dest)
        }
    }

    return skipped, copied, nil
}
```

### 4. Modified handleCreate Flow

**Location:** `internal/mcp/tools/initiative/tool.go`

```go
func (t *Tool) handleCreate(ctx context.Context, dir string, args map[string]interface{}) (string, error) {
    // ... parameter validation (unchanged) ...

    initSvc, err := internalInit.NewService(dir)
    if err != nil {
        return "", fmt.Errorf("creating initiative service: %w", err)
    }

    // Check if active initiative matches request (idempotency)
    existing, err := initSvc.FindActiveByNameAndType(name, internalInit.InitiativeType(initType))
    if err != nil {
        return "", fmt.Errorf("checking for existing initiative: %w", err)
    }

    if existing != nil {
        // Idempotent case: return existing initiative
        cyclePath, cycleID := t.findFirstCycle(existing.Path)

        // Still run template copy (safe - skips existing files)
        skipped, copied, err := t.copyTemplatesToCycle(dir, cyclePath)
        if err != nil {
            return "", fmt.Errorf("copying templates: %w", err)
        }

        resp := CreateResponse{
            Action:         "create",
            InitiativeID:   existing.ID,
            InitiativePath: existing.Path,
            CycleID:        cycleID,
            CyclePath:      cyclePath,
            Branch:         existing.ID,
            Type:           initType,
            Name:           name,
            NextStep:       initType,
            AlreadyExisted: true,
            SkippedFiles:   skipped,
            CopiedFiles:    copied,
        }
        return marshalResponse(resp)
    }

    // Check if a DIFFERENT initiative is active (error case)
    active, err := initSvc.GetActive()
    if err != nil {
        return "", fmt.Errorf("checking active initiative: %w", err)
    }
    if active != nil {
        return "", &ToolError{
            Code:    "INITIATIVE_ALREADY_ACTIVE",
            Message: fmt.Sprintf("a different initiative is already active: %s", active.ID),
            Hint:    "Complete or abandon the current initiative first with 'initiative complete'",
        }
    }

    // Create new initiative (unchanged from here)
    initiative, err := initSvc.Create(internalInit.InitiativeType(initType), name)
    // ... rest of creation flow ...

    // Update template copy call to use new signature
    skipped, copied, err := t.copyTemplatesToCycle(dir, cycle.Path)
    if err != nil {
        return "", fmt.Errorf("copying templates: %w", err)
    }

    resp := CreateResponse{
        // ... existing fields ...
        AlreadyExisted: false,
        SkippedFiles:   skipped,
        CopiedFiles:    copied,
    }
    return marshalResponse(resp)
}
```

### 5. Helper: findFirstCycle

**Location:** `internal/mcp/tools/initiative/tool.go`

```go
// findFirstCycle finds the first cycle folder in an initiative directory.
// Returns the full path and cycle ID.
func (t *Tool) findFirstCycle(initiativePath string) (cyclePath, cycleID string) {
    entries, err := os.ReadDir(initiativePath)
    if err != nil {
        return initiativePath, "" // Fall back to initiative path
    }

    for _, entry := range entries {
        if entry.IsDir() && entry.Name() != "." && entry.Name() != ".." {
            // Check if it looks like a cycle folder (contains spec.md or research.md)
            cyclePath := filepath.Join(initiativePath, entry.Name())
            if _, err := os.Stat(filepath.Join(cyclePath, "spec.md")); err == nil {
                return cyclePath, entry.Name()
            }
            if _, err := os.Stat(filepath.Join(cyclePath, "research.md")); err == nil {
                return cyclePath, entry.Name()
            }
        }
    }

    // No cycle found, use initiative path
    return initiativePath, ""
}
```

## Test Specifications

### Unit Tests: FindActiveByNameAndType

| Test Case | Setup | Input | Expected Output |
|-----------|-------|-------|-----------------|
| No active | Empty state | "foo", feature | nil, nil |
| Different name | Active: "bar-feature" | "foo", feature | nil, nil |
| Different type | Active: "foo-bug" | "foo", feature | nil, nil |
| Exact match | Active: "foo-feature" | "foo", feature | &Initiative{...}, nil |
| Name normalization | Active: "user-auth" | "User Auth", feature | &Initiative{...}, nil |

### Unit Tests: copyTemplateIfNotExists

| Test Case | Dest File State | Expected Return | Expected File State |
|-----------|-----------------|-----------------|---------------------|
| File doesn't exist | Not present | (true, nil) | Contains template |
| File has content | "Custom content" | (false, nil) | Unchanged |
| File is empty | "" (0 bytes) | (true, nil) | Contains template |
| File is whitespace | "  \n\t  " | (true, nil) | Contains template |
| Template read error | N/A | (false, error) | Unchanged |

### Integration Tests: handleCreate

| Test Case | Scenario | Expected |
|-----------|----------|----------|
| Idempotent create | Create, modify spec.md, create same | AlreadyExisted=true, spec unchanged |
| Different initiative active | Create foo, attempt create bar | Error: INITIATIVE_ALREADY_ACTIVE |
| After complete | Create, complete, create same | AlreadyExisted=false, new ID |
| Fresh create | No active | AlreadyExisted=false, new files |

## Error Handling

| Error Code | When | Response Type |
|------------|------|---------------|
| INITIATIVE_ALREADY_ACTIVE | Different initiative is active | ToolError |
| (none) | Same initiative already active | Success with AlreadyExisted=true |
| Template read error | Can't read template file | Go error (wrapped) |
| Write error | Can't write to cycle folder | Go error (wrapped) |

## Backward Compatibility

1. **Request format**: Unchanged - all existing requests work
2. **Response format**: Additive only
   - New fields: `already_existed`, `skipped_files`, `copied_files`
   - Clients ignoring unknown fields will continue to work
3. **Error behavior**: Preserved for "different initiative active" case
4. **Success behavior**: Enhanced - same name+type now returns success instead of creating duplicate

## Migration Notes

None required. The change is backward compatible and requires no migration steps.
