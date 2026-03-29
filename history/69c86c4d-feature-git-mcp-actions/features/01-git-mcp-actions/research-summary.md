# Research Summary: Git MCP Actions

## Skill Audit Findings

### commit-message Skill

**Location:** `~/.claude/skills.pre-link/commit-message/`

**Architecture:** Three validated shell scripts wrapped by a SKILL.md manifest. All git operations are routed through wrapper scripts -- raw git commands are explicitly forbidden.

**Shell Scripts:**
| Script | Git Commands | Purpose |
|--------|-------------|---------|
| `git-info.sh` | `git branch --show-current`, `git status --short`, `git log --oneline -10`, `git diff HEAD` | Gather repo context |
| `git-stage.sh` | `git add -- <files>` | Stage specific files (validates paths, rejects flags) |
| `git-commit.sh` | `git diff --cached --quiet`, `git commit -F <message-file>` | Create commit (validates message file, checks staged changes) |

**Security Model:** Input validation in each script (reject flags, check file existence), `set -euo pipefail`, no eval/command substitution of untrusted input.

**Distinct Git Operations Needed:**
1. Get current branch name
2. Get short status
3. Get recent commit log (last 10)
4. Get full diff (staged + unstaged)
5. Stage specific files
6. Check if staged changes exist
7. Create commit from message file

### create-pr Skill

**Location:** `~/.claude/skills/create-pr/SKILL.md`

**Architecture:** Monolithic skill file with direct Bash tool calls (no wrapper scripts). Uses wildcard permissions: `Bash(git:*)`, `Bash(gh pr:*)`, `Bash(gh api:*)`, `Bash(ccexport:*)`.

**Distinct Operations:**
| Category | Commands | Purpose |
|----------|---------|---------|
| Git Read | `git branch --show-current`, `git log main..HEAD --oneline`, `git status --porcelain`, `git status -sb`, `git diff main...HEAD --stat` | Context gathering |
| Git Write | `git push -u origin <branch>`, `git push` | Push to remote |
| GitHub Read | `gh pr view --json url,title` | Check existing PR |
| GitHub Write | `gh pr create --base main --title --body`, `gh pr comment <number> --body-file` | Create PR & comment |
| External | `ccexport -f markdown --no-thinking --no-agents -o <file> <session-id>` | Export conversation |

**Sub-skill dependency:** Invokes `commit-message` when working directory is dirty.

## Existing Codebase Patterns

### MCP Tool Registration Pattern

Tools follow a consistent pattern in `/internal/mcp/tools/`:
1. Tool struct with dependencies injected via constructor
2. `Definition()` method returning metadata
3. `Execute(ctx, args)` dispatching to operation handlers
4. Handlers return `(string, error)` with JSON responses
5. Registration via `s.mcpServer.AddTool()` in `server.go`

### Existing Git Code

Two relevant packages already exist:
- **`/internal/step/git.go`** - Branch naming, creation, switching
- **`/internal/worktree/manager.go`** - Worktree CRUD, branch resolution, `exec.CommandContext` for git

Both handle git availability gracefully and provide error classification.

### Config System

Tools are registered in `KnownTools` list in `/internal/config/tools.go`. Enable/disable per-tool or per-category.

## Combined Operation Matrix

Deduplicating across both skills, these are the distinct git/GitHub operations needed:

### Git Read Operations
| Operation | Used By | MCP Endpoint Candidate |
|-----------|---------|----------------------|
| Current branch name | Both | `git-info` |
| Short status | commit-message | `git-info` |
| Porcelain status | create-pr | `git-info` |
| Recent commit log | Both | `git-info` |
| Full diff (HEAD) | commit-message | `git-diff` |
| Diff stat vs base | create-pr | `git-diff` |
| Remote tracking status | create-pr | `git-info` |
| Staged changes check | commit-message | `git-info` |

### Git Write Operations
| Operation | Used By | MCP Endpoint Candidate |
|-----------|---------|----------------------|
| Stage specific files | commit-message | `git-stage` |
| Create commit from message | commit-message | `git-commit` |
| Push to remote | create-pr | `git-push` |

### GitHub Operations
| Operation | Used By | MCP Endpoint Candidate |
|-----------|---------|----------------------|
| Check existing PR | create-pr | `gh-pr-view` |
| Create PR | create-pr | `gh-pr-create` |
| Comment on PR | create-pr | `gh-pr-comment` |

### External Tool Operations
| Operation | Used By | MCP Endpoint Candidate |
|-----------|---------|----------------------|
| Export conversation | create-pr | Out of scope (ccexport is separate) |
