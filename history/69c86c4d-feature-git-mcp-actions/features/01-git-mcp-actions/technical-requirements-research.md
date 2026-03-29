# Technical Requirements Research: Git MCP Actions

## Implementation Hints from User

- "We don't have to have shell scripts and other 'hacks' to get around the permissioning"
- Goal is to replace Bash tool calls with native MCP tool calls
- Skills currently rely on `allowed-tools: Bash(git:*)` patterns which are coarse-grained

## Architectural Considerations

### Where to Put the Code

Follow existing pattern: `/internal/mcp/tools/git/tool.go`

Reuse existing git utilities:
- `/internal/worktree/manager.go` already has `exec.CommandContext` wrappers for git
- `/internal/step/git.go` has branch naming utilities

### Tool Granularity Options

**Option A: Single `git` tool with operation parameter (like stickymemory)**
- Pros: Single registration, consistent with existing patterns
- Cons: Large parameter surface, hard to describe all operations in one schema

**Option B: Multiple focused tools (git-info, git-stage, git-commit, git-push, gh-pr)**
- Pros: Clear separation, each tool has tight parameter schema, easier to permission individually
- Cons: More registration boilerplate

**Option C: Two tools - `git` for local ops, `gh` for GitHub ops**
- Pros: Clean domain split (local git vs remote GitHub), moderate number of tools
- Cons: `git` tool still has many operations

**Recommendation:** Option B aligns best with the goal of replacing specific Bash permissions with specific MCP tool permissions. Each tool can be enabled/disabled independently in config.

### Security Model

The commit-message skill's wrapper scripts provide input validation (reject flags, validate paths). The MCP tools should replicate this:
- Validate file paths before staging (no flags, file must exist)
- Validate commit message is non-empty
- Validate branch is not main/master before push
- Validate PR title/body are present before creation

### GitHub CLI Dependency

The `gh` CLI is an external dependency. Two approaches:
1. Shell out to `gh` (simpler, same as current approach)
2. Use GitHub API directly via `go-github` library (more control, no external dependency)

For now, shelling out to `gh` is the pragmatic choice -- it handles auth, pagination, and all the edge cases. Can be swapped later.

### MCP Logging Constraint

Per CLAUDE.md: MCP tools MUST NOT write to stdout. All git command execution should capture output and return it via the MCP response. The existing `exec.CommandContext` pattern in worktree/manager.go already does this correctly.

## Config Integration

New tools need entries in `KnownTools` in `/internal/config/tools.go`:
```
"git-info"
"git-stage"
"git-commit"
"git-push"
"gh-pr"
```

Or as a category: `tools.git.enabled = true` to control all git tools at once.
