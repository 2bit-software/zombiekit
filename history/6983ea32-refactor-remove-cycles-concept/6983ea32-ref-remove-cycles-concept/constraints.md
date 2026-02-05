# Constraints: Behavior That Must Not Change

## Public Interface Stability

### MCP Tool: `initiative`

The following behaviors must remain stable:

| Action | Required Behavior |
|--------|-------------------|
| `create` | Creates initiative folder, returns initiative ID and path, sets active state |
| `status` | Returns current step, available docs, suggested next step |
| `complete` | Clears active state, marks initiative complete |
| `list` | Returns all initiatives with ID, type, name, status |

### MCP Tool: `step`

| Behavior | Must Remain |
|----------|-------------|
| Step execution | Requires active initiative |
| File resolution | Resolves `files:` patterns to actual paths |
| Prerequisites | Checks for required artifacts before step execution |
| Profile composition | Composes profiles for step directives |

### INITIATIVE.md Format

The file must continue to:
- Store initiative metadata (type, status, created date, ID)
- Track step progress with status and timestamps
- Be parseable by `ParseInitiativeMD`

### Folder Structure

```
history/{initiative-id}/
  INITIATIVE.md
  spec.md
  research.md
  plan.md
  tasks.md
  audit/
```

**Note**: The `audit/` subdirectory remains for audit artifacts.

## External Dependencies

1. **Profile system**: No changes - profiles still compose the same way
2. **Step definitions**: No changes - steps still resolve files and compose profiles
3. **Git service**: Branch creation uses initiative ID, not cycle ID
4. **Template system**: Templates still copy to the working folder (now initiative folder)

## Test Contracts

These test behaviors define the public contract:

1. `TestService_Create` - Creates initiative with INITIATIVE.md
2. `TestService_Status` - Returns step progress from INITIATIVE.md
3. `TestService_Complete` - Clears active state
4. `TestService_List` - Enumerates initiative folders
5. `TestParseInitiativeMD` - Parses initiative metadata and steps
6. `TestStepService_Execute` - Executes steps within initiative context
