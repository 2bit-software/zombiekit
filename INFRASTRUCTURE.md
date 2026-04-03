# ZombieKit Infrastructure Overview

Reference document for developers. Covers the full Claude Code + zombiekit + brains runtime topology: where things live, how they connect, and the resolution order for every extensible component.

---

## Executables

| Binary | Installed Path | Notes |
|--------|---------------|-------|
| `brains` | `~/.local/bin/brains` | Main CLI; MCP server, hook handler, profile/workflow engine |
| `bs` | `~/.local/bin/bs` | Symlink → `brains` |
| `orchestrator` | `./bin/orchestrator` (local only) | Distributed workflow orchestration daemon |
| `zk-server` | `./bin/zk-server` (local only) | PostgreSQL + Ollama wrapper for recall features |

`task install` builds and installs only `brains` + `bs`. The others are run locally via `task orch` / `task server`.

---

## Claude Code Configuration

**Config root:** `~/.claude/`
**`settings.json`** is symlinked to `/Users/morgan/Projects/personal/ai/claude/settings.json`

### Registered MCP Servers

Stored in `~/.claude/.mcp.json`:

```
zombiekit   stdio   user-scope   ~/.local/bin/brains serve --mode stdio
```

### Hooks

| Event | Matcher | Command | Timeout |
|-------|---------|---------|---------|
| `SessionStart` | `startup\|resume\|compact` | `brains hook --event session-start` | 10s |
| `PreToolUse` | `Read\|Write\|Edit\|MultiEdit` | `brains hook --event pre-tool-use` | 10s |
| `PostToolUse` | `Write\|Edit` | `prek-check.sh`, `sentrux-gate.sh` | 30s / 60s |
| `PostToolUse` | `Grep` | `grep-mcp-hint.sh` | 5s |
| `PostToolUse` | `*` | `codeforge-hook.sh` (async) | 10s |
| `PostToolUseFailure` | — | `codeforge-hook.sh` (async) | — |
| `SessionEnd` | — | `brains hook --event session-end` | 5s |
| `TaskCompleted`, `Notification`, `SubagentStop` | — | `codeforge-hook.sh` (async) | — |

`SessionStart` injects unconditional rules into the system prompt.
`PreToolUse` injects path-matched rules before file operations.
Both use session-state tracking to avoid duplicate injection; state resets on compaction.

### Folders

| Folder | Points To |
|--------|-----------|
| `~/.claude/skills/` | `~/Projects/personal/ai/claude/skills/` (symlink) |
| `~/.brains/rules/` | `~/Projects/personal/ai/claude/rules/` (symlink) |

---

## Global Brains Data

| Path | Symlink Target | Contents |
|------|---------------|----------|
| `~/.brains/profiles/` | `~/Projects/personal/ai/profiles/` | 84 profiles/skills |
| `~/.brains/rules/` | `~/Projects/personal/ai/claude/rules/` | 25 rule `.md` files |
| `~/.brains/workflows/` | `~/Projects/personal/zombiekit/embed/workflows/` | 6 embedded workflows (symlinked for live dev) |
| `~/.brains/memories.db` | — | SQLite, session sticky memory |
| `~/.brains/registry.json` | — | Profile/workflow registry cache |

No symlinks point from the zombiekit source tree into these folders. Profiles and workflows are distributed by embedding them in the binary at compile time via `//go:embed embed/profiles/*` and `//go:embed embed/workflows/*`.

---

## Embedded Assets

Both registries are declared in `embed.go` and registered at startup in `cmd/brains/main.go`:

```go
profile.SetEmbeddedFS(zombiekit.EmbeddedProfiles)   // 36 profiles
workflow.SetEmbeddedFS(zombiekit.EmbeddedWorkflows)  // 6 workflows
```

Source trees:

```
embed/
├── profiles/          36 .md files (feature, bug, refactor, research, audit, ...)
├── workflows/         6 .md files  (new, feature-light, next, complete, help, ...)
├── templates/         Step templates used by workflows
├── scripts/           Helper scripts (commit-message, permissions-audit, repo-auditor)
└── integrations/claude/commands/
```

---

## Resolution Order

### Profiles (first match wins — local shadows global shadows embedded)

```
1. .brains/profiles/          project-local
2. ../.brains/profiles/       parent dirs, walked up to git root
3. ~/.brains/profiles/        global  (~84 profiles, symlinked)
4. [binary embedded]          fallback (~36 profiles, compiled in)
```

Implemented in `internal/profile/brains_source.go` → `FindProfileDirs()`.

### Workflows (first match wins)

```
1. .brains/workflows/         project-local
2. ~/.brains/workflows/       global (currently unused)
3. [binary embedded]          fallback (6 workflows, compiled in)
```

Implemented in `internal/workflow/service.go`.

### Rules (all are collected — they accumulate, not shadow)

```
1. .brains/rules/             project-local  (highest priority for ordering)
2. ../.brains/rules/          parent dirs, walked up to git root
3. ~/.brains/rules/           global (~25 rule files, symlinked)
```

All matching rules are injected; there is no shadowing. Rules are matched by path pattern at `PreToolUse` time. Unconditional rules (no path pattern) fire at `SessionStart`.

Implemented in `internal/rules/resolver.go` + `internal/rules/matcher.go`.

---

## MCP Tools Exposed by zombiekit

The `brains serve --mode stdio` process exposes these tools to Claude Code:

| Tool | Purpose |
|------|---------|
| `profile-compose` | Resolve + merge profiles into a single prompt |
| `profile-list` | List all available profiles (across all sources) |
| `profile-save` | Write a profile to local or global storage |
| `workflow-compose` | Load and return a workflow prompt |
| `skill-install` | Install a profile as a Claude Code skill (writes `SKILL.md`) |
| `initiative` | Workflow lifecycle: create, status, complete, list |
| `code-reasoning` | Sequential thinking with branching |
| `stickymemory` | Persistent key/value memory across sessions |
| `git` | Git operations (status, log, diff, stage, commit, push) |
| `gh-pr` | GitHub PR operations (view, create, comment, edit) |
| `recall-*` | Conversation search/memory (requires PostgreSQL + Ollama) |

---

## Skill Install Flow

`skill-install` (`internal/mcp/tools/skillinstall/tool.go`):

1. Resolves the named profile through the normal resolution chain
2. Generates a `SKILL.md` from the profile content
3. Writes to `~/.claude/skills/{name}/SKILL.md` (global) or `.claude/skills/{name}/SKILL.md` (local)

Currently installed global skills: `commit-message`, `create-pr`.

---

## Workflow Invocation from Claude Code

```
User invokes /brains.new (or any brains.* skill)
→ Claude Code calls mcp__zombiekit__workflow-compose {name}
→ internal/workflow/service.go::Load(name)
→ Resolution chain: local → global → embedded
→ Workflow markdown returned as prompt content
→ Claude follows steps; calls mcp__zombiekit__initiative to track state
```

`new.md` is the dispatcher workflow — detects Linear ticket refs, classifies intent (feature/bug/refactor), and routes to the appropriate subworkflow.

---

## Hook Execution Flow

```
SessionStart (startup/resume/compact)
→ brains hook --event session-start
→ internal/hook/handler.go
→ internal/rules/resolver.go: walk CWD → git root → ~/.brains/rules/
→ Filter: unconditional rules only (no path patterns)
→ Deduplicate against already-injected set for this session
→ Output rule bodies as markdown → injected into Claude Code system prompt

PreToolUse (Read/Write/Edit/MultiEdit)
→ brains hook --event pre-tool-use
→ Same resolver, but filter: rules whose path patterns match the tool's target file
→ Output matching rules → injected before file operation
```

Session state (injected rule tracking) lives in `~/.brains/` (likely `memories.db`). Resets on compaction.
