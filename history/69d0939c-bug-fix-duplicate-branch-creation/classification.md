# Classification

**Type**: Spec Gap

The initiative MCP tool was updated to create branches during `create`, but the workflow markdown files were not updated to account for this. The workflows still instruct the agent to create branches independently.

## Evidence

- `initiative create` returns `branch` field showing it created the branch
- All 5 workflow files still have a "Create Branch" step that duplicates this
