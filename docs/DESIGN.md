# ZombieKit Design Document

> *"Feed your codebase some brains."*

## Overview

ZombieKit (aka "brains") is a **prompt composition and artifact management system** that integrates with Claude Code. It does NOT orchestrate AI agents directly—Claude Code's native Task tool handles all orchestration. The `brains` CLI provides:

1. **Prompt Composition** - Hierarchical profile system that composes prompts from multiple sources
2. **Artifact Storage** - Filesystem-based storage for specs, plans, and tasks
3. **MCP Tools** - Sticky memory and code-reasoning tools
4. **Initiative Management** - Track multi-step development workflows
5. **Web Management** - UI for profiles, memories, and search

**Key Insight:** All workflow orchestration happens in Claude Code via skills/commands. The `brains` CLI is a stateless utility that agents call to compose their prompts.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CLAUDE CODE                                        │
│                      (Orchestration Layer)                                   │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │  .claude/commands/brains.feature.md                                 │   │
│   │                                                                     │   │
│   │  Orchestrates: research → create → audit → highlight                │   │
│   │  Uses Task tool to spawn agents in parallel                         │   │
│   └──────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           BRAINS CLI / MCP                                   │
│                    (Prompt Compositor & Storage)                             │
│                                                                             │
│   MCP Tools:                        CLI Commands:                           │
│   • profile-compose                 • brains profile compose X,Y,Z          │
│   • stickymemory                    • brains init                           │
│   • code-reasoning                  • brains serve                          │
│   • step (workflow execution)       • brains gui                            │
│   • feature (template delivery)     • brains memory ...                     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Separation of Concerns

| Claude Code Does | Brains CLI Does |
|------------------|-----------------|
| Workflow sequencing | Prompt composition |
| Parallel execution | Profile resolution |
| Agent invocation | Artifact storage |
| User interaction | MCP tools |
| LLM reasoning | Initiative tracking |

### What Brains CLI Does NOT Do

- Orchestrate agents (Claude Code does this)
- Make LLM API calls (Claude Code does this)
- Define workflow logic (Skills do this)
- Spawn parallel tasks (Task tool does this)

---

## CLI Commands

### Setup

```bash
brains init                    # Create .brains/ + .claude/commands/ in current directory
brains init --global           # Create ~/.brains/ only
brains init --force            # Overwrite existing files
```

### Profiles

```bash
brains profile compose <a,b,c>           # Output composed prompt text
brains profile compose <a,b,c> --json    # Structured JSON output
brains profile list                      # List available profiles
brains profile show <name>               # Show resolved profile content
brains profile validate                  # Check for circular deps, missing refs
brains profile create <name>             # Create new profile stub
brains profile import <path>             # Import profile from file
```

### Memory (Sticky Memory)

```bash
brains memory list                       # List all memories
brains memory get <name>                 # Get specific memory
brains memory set <name> <content>       # Store memory
brains memory delete <name>              # Delete memory
brains memory search <query>             # Search memories
brains memory clear                      # Clear all memories
```

### Database

```bash
brains db migrate                        # Run pending migrations
brains db status                         # Show migration status
brains db import --from sqlite --to pg   # Import between backends
```

### Services

```bash
brains serve                             # Start MCP server (default: stdio)
brains serve --transport sse --port 8080 # SSE mode for web clients
brains serve --transport http            # HTTP mode

brains gui                               # Start web UI (default: :8080)
brains gui --port 3000
```

### Utility

```bash
brains version                           # Show version and build info
```

---

## Profile System

### What is a Profile?

A **profile** is a composable unit of prompt content. It contains:
- Instructions and methodology
- Domain-specific knowledge
- Rules and constraints
- References to other profiles

### Profile Format

```markdown
---
name: database
description: Database expertise
type: domain                    # domain | action
includes:
  - research
inherits: true
---

# Database Expert Profile

All markdown below frontmatter is the actual prompt content.

## Methodology

When investigating database-related topics:
1. Start with schema exploration
2. Trace data flow through queries
```

### Frontmatter Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | Yes | - | Must match filename (e.g., `database` for `database.md`) |
| `description` | No | - | Human-readable purpose |
| `type` | No | - | Profile type: `domain` or `action` |
| `includes` | No | `[]` | Other profiles to pull in |
| `inherits` | No | `true` | Prepend same-named profile from parent |

### Resolution Order

Profiles resolve by walking up the directory tree:

```
/project/src/feature/.brains/profiles/  ← highest priority
       ↓
/project/.brains/profiles/              ← git root
       ↓
~/.brains/profiles/                     ← global defaults
       ↓
(embedded fallback profiles)            ← built into binary
```

**Resolution rule:** Closest to CWD wins for conflicts.

### Composition

When `brains profile compose research,database` is called:

1. **Resolve each profile** - Walk directory tree, closest wins
2. **Process includes** - Recursively, depth-first, deduplicate
3. **Handle inheritance** - If `inherits: true`, prepend parent content
4. **Concatenate** - Left-to-right order, no conflict resolution

---

## MCP Tools

### stickymemory

Persistent key-value storage across sessions.

| Operation | Description |
|-----------|-------------|
| `get` | Retrieve a memory by name |
| `set` | Store content with a name |
| `list` | List all memories |
| `delete` | Remove a memory |
| `search` | Search by name pattern |
| `clear` | Delete all memories |

**Storage:** SQLite (default) or PostgreSQL

### code-reasoning

Sequential thinking tool for problem-solving with branching and revision.

| Parameter | Description |
|-----------|-------------|
| `thought` | Current reasoning step |
| `thought_number` | Position in sequence |
| `total_thoughts` | Estimated total |
| `next_thought_needed` | Continue or finish |
| `branch_id` | For alternative approaches |
| `is_revision` | Correcting earlier thought |

### profile-compose

Compose multiple profiles via MCP.

| Parameter | Description |
|-----------|-------------|
| `profiles` | List of profile names |
| `working_directory` | Override CWD for resolution |

### step

Execute a workflow step within an initiative.

| Parameter | Description |
|-----------|-------------|
| `step` | Step name (init, specify, plan, tasks, implement, audit, clarify, complete) |
| `description` | For init step - initiative description |
| `initiative` | Override active initiative |

**Returns:**
- `directive` - Instructions for the step
- `history_folder` - Path to initiative folder
- `files_to_read` - Glob patterns for context
- `composed_prompt` - Merged profile content

### feature

Returns the feature step template from `~/.brains/templates/step.feature.md`.

---

## Initiative System

An **initiative** is a user's goal that may require multiple specifications, refactors, and bug fixes.

### Folder Structure

```
history/
└── 675d8a3f-feature-user-auth/
    ├── INITIATIVE.md           # Goal, context, status
    ├── spec.md                 # Business specification
    ├── technical.md            # Technical specification
    ├── plan.md                 # Implementation plan
    ├── tasks.md                # Task breakdown
    └── audit/
        └── 2024-01-15.md       # Audit reports
```

### State Tracking

Active initiative tracked in `.brains/active.json`:

```json
{
  "initiative_id": "675d8a3f-feature-user-auth",
  "initiative_type": "feature",
  "started_at": "2024-01-15T10:30:00Z"
}
```

### Workflow Steps

| Step | Purpose |
|------|---------|
| `init` | Create new initiative |
| `specify` | Create/update specification |
| `plan` | Create implementation plan |
| `tasks` | Generate task list |
| `implement` | Execute tasks |
| `audit` | Check artifact alignment |
| `clarify` | Surface ambiguities |
| `complete` | Mark initiative done |

---

## Configuration

### File Locations

| Location | Purpose |
|----------|---------|
| `.brains/config.toml` | Project-local config |
| `~/.config/brains/config.toml` | Global config (Unix) |
| `%APPDATA%\brains\config.toml` | Global config (Windows) |

### Configuration Options

```toml
[tools]
stickymemory = true
codereasoning = true
profile = true
step = true
feature = true

[storage]
backend = "sqlite"              # sqlite | postgres
sqlite_path = "~/.brains/memories.db"
postgres_url = "postgresql://localhost:5432/zombiekit"
connection_timeout = "5s"

[server]
transport = "stdio"             # stdio | sse | http
port = 8080
log_level = "info"              # debug | info | warn | error
```

---

## Storage

### SQLite (Default)

- Location: `~/.brains/memories.db`
- WAL mode for concurrent access
- Versioning with soft deletes
- No setup required

### PostgreSQL

- Required for: multi-user deployments
- Supports: pgvector for embeddings (future)
- Migrations: `brains db migrate`

---

## The Workflow Cycle

Each major stage follows the same pattern:

```
┌──────────────────────────────────────────┐
│           THE ZOMBIEKIT CYCLE            │
│                                          │
│   ┌────────┐                             │
│   │RESEARCH│  Many agents, parallel      │
│   │        │  Collate & dedupe           │
│   └───┬────┘                             │
│       │                                  │
│       ▼                                  │
│   ┌────────┐                             │
│   │ CREATE │  Single agent               │
│   │        │  Structured output          │
│   └───┬────┘                             │
│       │                                  │
│       ▼                                  │
│   ┌────────┐                             │
│   │ AUDIT  │  Many agents, parallel      │
│   │        │  Completeness + AI-ready    │
│   └───┬────┘                             │
│       │                                  │
│       ▼                                  │
│   CRITICAL/MAJOR? ──YES──► Loop back     │
│       │                                  │
│       NO                                 │
│       │                                  │
│       ▼                                  │
│   HIGHLIGHT to user                      │
│       │                                  │
│       ▼                                  │
│   USER APPROVED? ──NO──► Loop back       │
│       │                                  │
│      YES → NEXT STAGE                    │
└──────────────────────────────────────────┘
```

**Key Principle:** No stage completes until:
1. No CRITICAL or MAJOR audit issues remain
2. User has reviewed highlights and approved

---

## Slash Commands (Claude Code Skills)

| Command | Purpose |
|---------|---------|
| `/brains.feature` | New feature specification |
| `/brains.bug` | Bug investigation |
| `/brains.refactor` | Refactoring specification |
| `/brains.plan` | Implementation planning |
| `/brains.tasks` | Task breakdown |
| `/brains.eat` | Execute implementation |
| `/brains.research` | Standalone research |
| `/brains.audit` | Alignment check |
| `/brains.clarify` | Surface ambiguities |
| `/brains.status` | Show initiative status |
| `/brains.complete` | Mark initiative done |
| `/brains.next` | Continue to next phase |

---

## Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.24+ |
| CLI Framework | urfave/cli/v2 |
| MCP Server | mark3labs/mcp-go |
| Database (default) | modernc.org/sqlite |
| Database (optional) | pgx/v5 (PostgreSQL) |
| Web Router | go-chi/chi/v5 |
| Config | BurntSushi/toml |
| YAML | gopkg.in/yaml.v3 |
| Frontmatter | adrg/frontmatter |
| Logging | slog (stdlib) |

---

## Embedded Assets

Four filesystems embedded at build time:

| Asset | Source | Purpose |
|-------|--------|---------|
| Profiles | `profiles/` | Default global profiles |
| Commands | `.claude/commands/` | Claude Code skills |
| Templates | `templates/` | Artifact templates |
| Steps | `.brains/steps/` | Step definitions |

---

## Open Questions

1. **Conversation Import** - Designed but not yet implemented. Will import Claude Code conversations for full-text and vector search.

2. **Vector Embeddings** - Interface defined for Ollama, but not yet integrated.

3. **Fast Mode** - Reduced thoroughness option discussed but not implemented.

4. **Proof Spikes** - Lightweight implementation validation during planning. Designed but not implemented.

---

## Summary

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   CLAUDE CODE                    BRAINS CLI                 │
│   ════════════                   ══════════                 │
│                                                             │
│   • Orchestration                • Prompt composition       │
│   • Agent spawning               • Profile resolution       │
│   • Parallel execution           • Artifact storage         │
│   • User interaction             • MCP tools                │
│   • LLM inference                • Initiative tracking      │
│                                  • Web UI                   │
│                                                             │
│   Skills call agents ──────────► Agents call CLI            │
│   Agents return results ◄─────── CLI returns prompts        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**The core insight: Claude Code is the brain, brains CLI is the memory.**
