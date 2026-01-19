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

## Learn More

- [Architecture and Design](docs/DESIGN.md) — Full architecture, all skills, configuration options
- [Skill Definitions](.claude/commands/) — Individual skill documentation
