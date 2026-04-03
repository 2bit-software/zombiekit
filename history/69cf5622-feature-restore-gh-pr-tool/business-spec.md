# Business Spec: Restore gh-pr Tool

## Summary

Re-add the gh-pr MCP tool that was previously removed, restoring all original functionality and including the edit action that was added just before removal.

## Functional Requirements

- **view**: Check if a PR exists for the current branch, return URL/title/number/state
- **create**: Open a new PR with title, body, base branch, and optional draft flag
- **comment**: Add a comment to an existing PR by number
- **edit**: Update an existing PR's title and/or body (new capability)

## Acceptance Criteria

- gh-pr tool is registered in the MCP server when enabled in config
- All four actions (view, create, comment, edit) work via the MCP protocol
- Edit action requires pr_number and at least one of title or body
- Body content uses --body-file temp files to avoid shell escaping issues
- Tool returns structured JSON responses for all actions
- Tool gracefully handles missing gh CLI with a clear error message
- Existing tests pass

## Out of Scope

- Merge/close PR actions
- Label/reviewer management
- Multi-repo support
