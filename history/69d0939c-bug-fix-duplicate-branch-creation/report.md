# Bug Report

## Symptoms

When running `/brains.new` with a feature (or bug/refactor) workflow, the agent errors out trying to create a git branch that already exists. The error occurs because:

1. Step 1 calls `mcp__zombiekit__initiative` with `action: "create"`, which internally calls `gitSvc.EnsureBranch()` — creating and checking out the branch.
2. Step 3 ("Create Branch") instructs the agent to create and checkout a branch via `mcp__zombiekit__git` or `git checkout -b`, which fails because the branch already exists.

## Error

```
fatal: a branch named 'XYZ' already exists
```

## Environment

- All workflow types: feature, feature-light, bug, refactor, unmanaged
- `internal/mcp/tools/initiative/tool.go:216` — `gitSvc.EnsureBranch(initType, name)`

## Steps to Reproduce

1. Run `/brains.new` with any feature/bug/refactor input
2. Initiative creation succeeds and creates the branch
3. Agent reaches step 3 and attempts to create the same branch
4. Branch creation fails with "already exists" error
