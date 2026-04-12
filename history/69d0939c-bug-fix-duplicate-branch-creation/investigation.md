# Investigation

## Root Cause

The initiative MCP tool (`internal/mcp/tools/initiative/tool.go:214-216`) calls `gitSvc.EnsureBranch()` during `create`, which creates and checks out the branch. The workflow markdown files then instruct the agent to create a branch again in step 3.

## Relevant Files

- `internal/mcp/tools/initiative/tool.go:216` — `gitSvc.EnsureBranch(initType, name)` called during create
- `internal/step/git.go:72` — actual `git checkout -b` execution
- `embed/workflows/feature.md:49-55` — step 3 "Create Branch"
- `embed/workflows/bug.md:49-55` — step 3 "Create Branch"
- `embed/workflows/refactor.md:49-55` — step 3 "Create Branch"
- `embed/workflows/feature-light.md:52-59` — step 3 "Create Branch"
- `embed/workflows/unmanaged.md:42-70` — step 3 "Create Branch"

## Key Finding

The initiative `create` response already returns `branch` and `already_existed` fields. The workflows should use this info to skip branch creation when the initiative was freshly created.
