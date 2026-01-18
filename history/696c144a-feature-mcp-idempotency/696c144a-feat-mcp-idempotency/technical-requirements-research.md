# Technical Requirements Research

## Implementation Hints from Ticket

The ticket (DEV-76) states:
> "we don't want blank templates to get copied over previous work if the same command is called twice"

This indicates:
1. Focus is on **template copying** operations
2. The concern is **data loss** from overwrites
3. "Same command called twice" suggests **idempotency** requirement

## Codebase Patterns to Follow

### Existing Idempotency Pattern: profile-write

Location: `internal/profile/service.go` lines 216-233

```go
// Check if file exists
if _, err := os.Stat(fullPath); err == nil {
    if !overwrite {
        return ProfileExistsError{Path: fullPath}
    }
}

// Atomic write pattern
tmpFile := fullPath + ".tmp"
if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
    return err
}
return os.Rename(tmpFile, fullPath)
```

Key patterns:
- Explicit `overwrite` flag (default: false)
- Pre-existence check before write
- Atomic write via temp file + rename
- Custom error type for exists condition

### Files Requiring Changes

1. **`internal/mcp/tools/initiative/tool.go`**
   - `handleCreate()` method (lines 123-210)
   - `copyTemplatesToCycle()` method (lines 315-356)

2. **`internal/initiative/service.go`**
   - Add method to find existing initiative by name+type

3. **`internal/mcp/tools/initiative/types.go`**
   - Add `AlreadyExisted` field to response
   - Add `SkippedFiles` field for template copy reporting

## Go Patterns to Use

### Check-before-write pattern (with whitespace handling)
```go
func copyTemplateIfNotExists(src, dst string) (copied bool, err error) {
    // Check if destination file exists
    if _, err := os.Stat(dst); err == nil {
        // File exists - read content to check if non-empty
        content, err := os.ReadFile(dst)
        if err != nil {
            return false, fmt.Errorf("failed to read existing file: %w", err)
        }
        // Skip if file has non-whitespace content
        if len(bytes.TrimSpace(content)) > 0 {
            return false, nil // skipped - file has content
        }
    }
    // Copy the template (either file doesn't exist or is empty/whitespace-only)
    templateContent, err := os.ReadFile(src)
    if err != nil {
        return false, fmt.Errorf("failed to read template: %w", err)
    }
    if err := os.WriteFile(dst, templateContent, 0644); err != nil {
        return false, fmt.Errorf("failed to write template: %w", err)
    }
    return true, nil
}
```

### Initiative lookup by name+type (active only)
```go
// FindActiveByNameAndType returns the active initiative if it matches the given name and type.
// Returns nil if no active initiative exists or if it doesn't match.
// This is intentionally simple: only the active initiative can be "found" for idempotency.
// Inactive initiatives in history/ are treated as archived and don't block creation.
func (s *Service) FindActiveByNameAndType(name, typ string) (*Initiative, error) {
    active, err := s.GetActive()
    if err != nil {
        return nil, err
    }
    if active == nil {
        return nil, nil // No active initiative
    }
    // Normalize name for comparison (same normalization as Create uses)
    normalizedName := normalizeInitiativeName(name)
    if active.Name == normalizedName && string(active.Type) == typ {
        return active, nil
    }
    return nil, nil // Active initiative doesn't match
}
```

### Templates copied during initiative creation
The `copyTemplatesToCycle()` function in `internal/mcp/tools/initiative/tool.go` copies these files:
- `spec-template.md` → `spec.md`
- `research-template.md` → `research.md`

These are the ONLY files that need idempotency protection in this feature.

## Response Design

### InitiativeCreateResponse (updated)
```go
type InitiativeCreateResponse struct {
    Action         string   `json:"action"`
    InitiativeID   string   `json:"initiative_id"`
    InitiativePath string   `json:"initiative_path"`
    CycleID        string   `json:"cycle_id"`
    CyclePath      string   `json:"cycle_path"`
    Branch         string   `json:"branch"`
    Type           string   `json:"type"`
    Name           string   `json:"name"`
    NextStep       string   `json:"next_step"`
    // New fields
    AlreadyExisted bool     `json:"already_existed"`
    SkippedFiles   []string `json:"skipped_files,omitempty"`
    CopiedFiles    []string `json:"copied_files,omitempty"`
}
```

## Testing Strategy

Add tests to `internal/mcp/tools/initiative/tool_test.go` (create if doesn't exist) and `internal/initiative/service_test.go`.

### Unit Tests

1. **`TestFindActiveByNameAndType`** in `internal/initiative/service_test.go`:
   - No active initiative → returns nil
   - Active initiative with different name → returns nil
   - Active initiative with different type → returns nil
   - Active initiative matching name+type → returns the initiative
   - Name normalization works (spaces, case)

2. **`TestCopyTemplateIfNotExists`** in `internal/mcp/tools/initiative/tool_test.go`:
   - File doesn't exist → copies template, returns (true, nil)
   - File exists with content → skips, returns (false, nil)
   - File exists but empty (0 bytes) → copies template, returns (true, nil)
   - File exists with only whitespace → copies template, returns (true, nil)
   - Template source missing → returns error

### Integration Tests

3. **`TestHandleCreateIdempotent`** in `internal/mcp/tools/initiative/tool_test.go`:
   - Create initiative with name="foo", type="feature"
   - Modify spec.md with custom content
   - Call handleCreate again with same name+type
   - Verify: returns existing initiative (AlreadyExisted=true)
   - Verify: spec.md content unchanged
   - Verify: SkippedFiles includes "spec.md"

4. **`TestHandleCreateAfterComplete`** in `internal/mcp/tools/initiative/tool_test.go`:
   - Create initiative with name="bar", type="feature"
   - Complete the initiative
   - Call handleCreate with same name+type
   - Verify: creates new initiative (AlreadyExisted=false)

## Implementation Order

1. Add `FindActiveByNameAndType` to `internal/initiative/service.go`
2. Add unit tests for `FindActiveByNameAndType`
3. Add `copyTemplateIfNotExists` helper to `internal/mcp/tools/initiative/tool.go`
4. Add unit tests for `copyTemplateIfNotExists`
5. Modify `handleCreate` to check for existing initiative first
6. Modify `copyTemplatesToCycle` to use the new helper and track skipped/copied files
7. Update response types in `internal/mcp/tools/initiative/types.go`
8. Add integration tests

## Error Handling

- When returning an existing initiative, return a **success response** (not an error) with `AlreadyExisted: true`
- File system errors (permission denied, disk full) should propagate as errors
- Missing template source is an error (should not happen in normal operation)

## Dependencies

- No new dependencies required
- Uses standard library: `os`, `path/filepath`, `bytes`
