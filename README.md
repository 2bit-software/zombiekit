# ZombieKit

> "Feed your codebase some brains."

Prompt composition and artifact management for Claude Code.

## What Is ZombieKit?

Claude Code is the brain; ZombieKit is the memory. ZombieKit does NOT orchestrate AI—Claude Code does that natively. Instead, ZombieKit provides persistent storage and structured workflows that Claude Code can invoke through skills.

ZombieKit gives you structured workflows for features, bugs, and refactors. It maintains persistent memory across sessions via artifacts stored in your repository. It provides composable prompts through profiles that can be mixed and matched for different contexts.

Integration is simple: ZombieKit runs as an MCP server that Claude Code connects to. Slash commands like `/brains.feature` invoke skills. All artifacts are stored in a `history/` folder in your project, versioned alongside your code.

## Quick Start

### Prerequisites

- Go 1.24+ (`go version`)
- Task from taskfile.dev (`task --version`)
- Claude Code (the AI coding assistant)

### Installation

```bash
git clone https://github.com/morganhein/zombiekit.git
cd zombiekit
task install
```

### MCP Configuration

Add to your Claude Code MCP settings:

```json
{
  "mcpServers": {
    "zombiekit": {
      "command": "brains",
      "args": ["serve"]
    }
  }
}
```

### Project Initialization

Run once per project:

```bash
brains init
```

### Verification

In Claude Code, run:

```
/brains.status
```

## The Workflow Cycle

```
RESEARCH --> CREATE --> AUDIT --> HIGHLIGHT
    ^                      |
    |                      |
    +---- (if issues) -----+
```

Every feature, bug, and refactor follows this pattern:
- **Research**: Parallel agents explore the codebase and domain
- **Create**: Single agent synthesizes findings into an artifact
- **Audit**: Checks completeness against requirements
- **Highlight**: Presents artifact to user for approval

## Core Skills

These are the primary workflow skills. See [docs/DESIGN.md](docs/DESIGN.md) for the complete list.

| Skill | Purpose |
|-------|---------|
| **Starting Work** | |
| `/brains.init` | Initialize ZombieKit in project |
| `/brains.feature` | Create feature specification |
| `/brains.bug` | Bug investigation and fix |
| `/brains.refactor` | Refactoring specification |
| **Planning** | |
| `/brains.plan` | Create implementation plan |
| `/brains.tasks` | Generate task breakdown |
| **Implementing** | |
| `/brains.implement` | Execute tasks from task list |
| **Tracking** | |
| `/brains.status` | Show current initiative status |
| `/brains.complete` | Mark initiative as done |

## Example Workflow

The `brains init` command runs once in your terminal to initialize a project. All `/brains.*` commands run inside Claude Code.

1. `brains init` — Initialize project (terminal, one-time)
2. `/brains.feature "add user authentication"` — Create spec (Claude Code)
3. Review and approve the specification
4. `/brains.plan` — Create implementation plan
5. Review and approve the plan
6. `/brains.tasks` — Generate task breakdown
7. `/brains.implement` — Execute tasks
8. `/brains.complete` — Mark initiative done

## Development Setup

For contributors or those wanting to use the recall (semantic search) features:

```bash
# Check dependencies and create .env
task dev -- setup

# Pull embedding model (required for recall)
task dev -- ollama:pull

# Start the full stack (PostgreSQL, importer, web GUI)
task up
```

All development tasks are in the dev Taskfile (`task dev -- --list` to see all):

```bash
task dev -- setup        # Create .env, check deps, install tools
task dev -- build        # Build the binary
task dev -- test         # Run tests
task dev -- ci           # Run all CI checks
task dev -- db:up        # Start PostgreSQL only
task dev -- db:migrate   # Run migrations
task dev -- recall:watch # Start Claude importer only
task dev -- gui          # Start web GUI only
```

Configuration is in `.env` (copied from `.env.example` by `task dev -- setup`). Key settings:

| Variable | Default | Purpose |
|----------|---------|---------|
| `POSTGRES_PORT` | 9432 | PostgreSQL external port |
| `BRAINS_BACKEND` | postgres | Storage backend (sqlite/postgres) |
| `BRAINS_OLLAMA_URL` | http://localhost:11434 | Ollama API endpoint |

## Hooks

ZombieKit registers coding-agent hooks that inject rules into the conversation at the right moments. Rules live in `.brains/rules/` (project-local) and `~/.brains/rules/` (global), and accumulate — all matching rules fire, none shadow.

Both Claude Code and Gemini CLI are supported. Pass `--editor claude` or `--editor gemini` to tell `brains hook` which output format to emit; when the flag is omitted, the command falls back to env detection (`CLAUDE_CODE_ENTRYPOINT`) and ultimately to Claude as the default.

| Event | What fires | Purpose |
|-------|-----------|---------|
| `SessionStart` | `brains hook --editor <e> --event session-start` | Injects **unconditional rules** (no path/command triggers) into the system prompt at session start, resume, and compaction. |
| `PreToolUse` (`Read`/`Write`/`Edit`/`MultiEdit`) | `brains hook --editor <e> --event pre-tool-use` | Injects **path-matched rules** before file operations (primary rule injection point for **Claude Code**). |
| `PreToolUse` (`Bash`) | same | Injects **command-matched rules** when a bash invocation matches a rule's `commands:` prefix (e.g. `go build` → Taskfile reminder). |
| `PostToolUse` (`Read`/`Write`/`Edit`/`MultiEdit`) | `brains hook --editor <e> --event post-tool-use` | Injects **path-matched rules** after file operations (primary rule injection point for **Gemini CLI**). |
| `SessionEnd` | `brains hook --editor <e> --event session-end` | Cleanup / session-state teardown. |

The `--event` flag is zombiekit's canonical event name; when wiring into Gemini CLI, map Gemini's `BeforeTool` to `--event pre-tool-use` in `settings.json`.

### Gemini CLI setup

Add to `.gemini/settings.json` (or `~/.gemini/settings.json`):

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [{ "type": "command", "command": "brains hook --editor gemini --event session-start" }] }
    ],
    "BeforeTool": [
      {
        "matcher": ".*",
        "hooks": [{ "type": "command", "command": "brains hook --editor gemini --event pre-tool-use" }]
      }
    ],
    "AfterTool": [
      {
        "matcher": ".*",
        "hooks": [{ "type": "command", "command": "brains hook --editor gemini --event post-tool-use" }]
      }
    ],
    "SessionEnd": [
      { "hooks": [{ "type": "command", "command": "brains hook --editor gemini --event session-end" }] }
    ]
  }
}
```

**Hooks are warnings, not hard stops.** A matched rule surfaces guidance alongside the tool call — the agent still executes the command. Each `(rule, trigger)` fires at most once per session; state resets on compaction.

### Bash command rules

Rules can declare command prefixes and file-existence gates:

```yaml
---
commands: ["go test", "go run", "go build"]
requires_files: [Taskfile.yml]
---
# Use the Taskfile
Prefer `task dev -- test` over bare `go` invocations.
```

- Commands match as whole-token prefixes; chained commands (`&&`, `||`, `;`, `|`) are split and matched independently.
- `requires_files` / `requires_files_absent` walk up from `cwd` to the enclosing git root, so subdirectory invocations still resolve a top-level `Taskfile.yml`.

See [INFRASTRUCTURE.md](INFRASTRUCTURE.md#hooks) for the full hook table, rule resolution order, and matching semantics.

## Learn More

- [Architecture and Design](docs/DESIGN.md) — Full architecture, all skills, configuration options
- [Skill Definitions](.claude/commands/) — Individual skill documentation
