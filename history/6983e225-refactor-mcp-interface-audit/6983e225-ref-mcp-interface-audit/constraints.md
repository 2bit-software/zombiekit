# Constraints

## Behavior That MUST NOT Change

1. The `step` tool with `step: "feature"` must continue to work
2. All other MCP tools must remain functional:
   - stickymemory
   - code-reasoning
   - profile-compose, profile-list, profile-save
   - workflow-compose
   - initiative
   - step
   - recall-list-conversations, recall-read-conversation

## Public Interfaces That Must Remain Stable

- MCP tool names (except for the ones being removed)
- Tool input schemas
- Tool response formats

## External Dependencies

- None affected - this is purely internal cleanup
