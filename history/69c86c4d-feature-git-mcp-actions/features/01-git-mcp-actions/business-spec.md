# Business Spec: Git MCP Actions

## Problem Statement

Claude Code skills that perform git operations (committing, staging, creating PRs) currently rely on shell script wrappers and coarse-grained Bash tool permissions (`Bash(git:*)`, `Bash(gh pr:*)`). This creates several problems:

1. **Permission sprawl** -- Skills must declare broad Bash wildcard permissions to execute git commands
2. **Shell script maintenance** -- Wrapper scripts duplicate validation logic that belongs in the application layer
3. **No unified interface** -- Each skill reimplements git interaction patterns independently
4. **Fragile permissioning** -- Users must approve Bash execution for each git operation, or grant blanket wildcards

## Desired Outcome

MCP tool endpoints that provide git and GitHub operations as first-class tools, so:
- Skills can call `mcp__zombiekit__git-info` instead of `Bash(git status:*)`
- Input validation happens server-side (reject flags, validate paths, check branch protection)
- Tools are individually configurable (enable/disable per tool or per category)
- The existing commit-message and create-pr skills can be rewritten to use MCP tools instead of Bash

## Proposed MCP Tools

### 1. `git-info` -- Repository Context

Returns current repository state in a single call.

**Operations:**
| Operation | Returns | Notes |
|-----------|---------|-------|
| `status` | Branch name, short status, tracking info, staged changes flag | Combines multiple git commands into one response |
| `log` | Recent commits (configurable count, default 10) | Optionally scoped to `base..HEAD` |
| `diff-stat` | Changed files summary vs a base ref | For PR context gathering |

**Parameters:**
- `operation` (required): `status`, `log`, `diff-stat`
- `base` (optional): Base ref for log/diff-stat (default: `main`)
- `count` (optional): Number of log entries (default: 10)

### 2. `git-diff` -- Diff Content

Returns actual diff content (not just stat).

**Parameters:**
- `scope` (required): `all` (staged + unstaged), `staged`, `unstaged`
- `base` (optional): Compare against base ref instead of HEAD
- `paths` (optional): Limit diff to specific file paths

**Returns:** Diff text content.

### 3. `git-stage` -- Stage Files

Stages specific files for commit.

**Parameters:**
- `files` (required): List of file paths to stage

**Validation:**
- Rejects paths that look like flags (`-*`)
- Validates each file exists in working tree
- Never stages all files (no `.` or `-A` equivalent)

**Returns:** Updated status after staging.

### 4. `git-commit` -- Create Commit

Creates a commit with a message.

**Parameters:**
- `message` (required): Commit message text (multi-line supported)
- `author` (optional): Override author string

**Validation:**
- Message must be non-empty
- Staged changes must exist (fails if nothing staged)
- Does not support `--amend` or `--no-verify` (by design)

**Returns:** Commit hash, branch, short log entry.

### 5. `git-push` -- Push to Remote

Pushes current branch to remote.

**Parameters:**
- `set_upstream` (optional, boolean): Whether to set upstream tracking (default: false)
- `remote` (optional): Remote name (default: `origin`)

**Validation:**
- Refuses to push `main` or `master` branches
- Checks that current branch has commits ahead of remote

**Returns:** Push result (success/failure, remote URL, branch).

### 6. `gh-pr` -- GitHub PR Operations

Manages pull requests via the `gh` CLI.

**Operations:**
| Operation | Purpose | Parameters |
|-----------|---------|-----------|
| `view` | Check if PR exists for current branch | None (uses current branch) |
| `create` | Create a new PR | `title`, `body`, `base` (default: main), `draft` (boolean) |
| `comment` | Add comment to a PR | `pr_number`, `body` |

**Validation:**
- `create` requires non-empty title and body
- `create` fails if PR already exists for current branch
- `comment` requires valid PR number

**Returns:** PR URL, number, title (for view/create); comment URL (for comment).

## Out of Scope

- **ccexport integration** -- Conversation export is a separate concern (external binary)
- **Branch creation/switching** -- Already handled by worktree manager
- **Merge/rebase operations** -- Dangerous, better left to manual control
- **Force push** -- Explicitly excluded for safety

## Configuration

All tools belong to a `git` category in the config system:

```toml
[tools.git]
enabled = true  # Enables all git-* and gh-* tools

[tools.git-push]
enabled = false  # Override: disable push specifically
```

## Success Criteria

1. The commit-message skill can be rewritten using only `git-info`, `git-diff`, `git-stage`, and `git-commit` MCP tools (no Bash calls)
2. The create-pr skill can be rewritten using `git-info`, `git-diff`, `git-push`, and `gh-pr` MCP tools (Bash only for ccexport)
3. Each tool validates inputs server-side, matching or exceeding the safety of the current shell scripts
4. Tools can be individually enabled/disabled via config
