# ZombieKit Master Design Document

> *"Feed your codebase some brains."*

## Executive Summary

ZombieKit (aka "brains") is a **prompt composition and artifact management system** that integrates with Claude Code. It does NOT orchestrate AI agents directly—Claude Code's native Task tool handles all orchestration. The `brains` CLI provides:

1. **Prompt Composition** - Hierarchical profile system that composes prompts from multiple sources
2. **Artifact Storage** - Filesystem-based storage for specs, plans, and tasks
3. **MCP Tools** - Sticky memory and code-thinking tools
4. **Conversation Import** - Background service indexing Claude conversations
5. **Web Management** - UI for profiles, artifacts, and search

**Key Insight:** All workflow orchestration happens in Claude Code via skills and agents. The `brains` CLI is a stateless utility that agents call to compose their prompts.

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CLAUDE CODE                                        │
│                      (Orchestration Layer)                                   │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │  .claude/skills/brains-feature/SKILL.md                             │   │
│   │                                                                     │   │
│   │  Orchestrates: research → create → audit → highlight                │   │
│   │  Uses Task tool to spawn agents in parallel                         │   │
│   └──────────────────────────────┬──────────────────────────────────────┘   │
│                                  │                                          │
│                    ┌─────────────┼─────────────┐                            │
│                    ▼             ▼             ▼                            │
│   ┌────────────────────┐ ┌────────────────┐ ┌────────────────────┐         │
│   │ .claude/agents/    │ │ .claude/agents/│ │ .claude/agents/    │         │
│   │ research-codebase  │ │ research-domain│ │ research-security  │         │
│   └─────────┬──────────┘ └───────┬────────┘ └─────────┬──────────┘         │
│             │                    │                    │                     │
└─────────────┼────────────────────┼────────────────────┼─────────────────────┘
              │                    │                    │
              ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           BRAINS CLI                                         │
│                    (Prompt Compositor & Storage)                             │
│                                                                             │
│   $ ./brains profiles compose research,codebase                             │
│   $ ./brains profiles compose research,domain,papi                          │
│   $ ./brains profiles compose research,security                             │
│                                                                             │
│                              │                                              │
│                              ▼                                              │
│                    Returns composed prompt text                             │
│                    (agent uses this as its context)                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Part 1: Separation of Concerns

### What Claude Code Does (Orchestration)

| Responsibility | How |
|----------------|-----|
| Workflow sequencing | Skills define research → create → audit → highlight flow |
| Parallel execution | Task tool spawns multiple agents concurrently |
| Agent invocation | Skills/agents invoke each other via Task tool |
| User interaction | Claude Code handles all prompts, approvals, feedback |
| LLM reasoning | All AI inference happens in Claude Code |

### What Brains CLI Does (Composition & Storage)

| Responsibility | How |
|----------------|-----|
| Prompt composition | `./brains profiles compose X,Y,Z` returns merged text |
| Profile resolution | Walks directory tree, merges by precedence |
| Profile registry | Tracks known `.brains` directories |
| Artifact CRUD | Save/load specs, plans, tasks to filesystem |
| Conversation import | Background service, PostgreSQL storage |
| MCP tools | sticky-memory, code-thinking |
| Web UI | Profile/artifact/search management |

### What Brains CLI Does NOT Do

- ❌ Orchestrate agents (Claude Code does this)
- ❌ Make LLM API calls (Claude Code does this)
- ❌ Define workflow logic (Skills do this)
- ❌ Spawn parallel tasks (Task tool does this)

### Tiered Feature Architecture

The CLI is designed with **tiered dependencies** so core features work without external services:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  TIER 1: Zero Dependencies (works immediately after install)                │
├─────────────────────────────────────────────────────────────────────────────┤
│  brains init                    Create .brains/ directory                   │
│  brains profile compose         Compose prompts from profiles               │
│  brains profile list/show       View available profiles                     │
│  brains profile validate        Check for errors                            │
│  brains spec new                Create spec directory                       │
│  brains spec list               List spec directories                       │
│  brains spec search             Full-text search (grep-based)               │
│  brains registry *              Manage .brains directories                  │
│  brains version/doctor          System info and health check                │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  TIER 2: Requires PostgreSQL                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  brains import                  Import Claude conversations                 │
│  brains convo search            Search imported conversations               │
│  brains convo stats             Import statistics                           │
│  brains serve                   MCP server (sticky-memory needs DB)         │
│  brains web                     Web UI (needs DB for conversations)         │
│  brains db *                    Database migrations                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Error handling for Tier 2 without database:**
```bash
$ brains import
Error: This feature requires PostgreSQL.

To set up the database:
  1. Start PostgreSQL (see Taskfile: task db:start)
  2. Run migrations: brains db migrate

Or use Tier 1 features that don't require a database.
```

### Service Interfaces

Each domain exposes a clean interface for testability and decoupling:

```go
// internal/profile/service.go
type ProfileService interface {
    Compose(ctx context.Context, names []string, cwd string) (*ComposeResult, error)
    List(ctx context.Context) ([]ProfileInfo, error)
    Show(ctx context.Context, name string, raw bool) (*Profile, error)
    Validate(ctx context.Context) ([]ValidationError, error)
}

// internal/spec/service.go
type SpecService interface {
    New(ctx context.Context, title string) (string, error)  // returns path
    List(ctx context.Context) ([]SpecInfo, error)
    Search(ctx context.Context, query string) ([]SearchResult, error)
}

// internal/conversation/service.go
type ConversationService interface {
    Import(ctx context.Context, path string) error
    Search(ctx context.Context, query string, opts SearchOpts) ([]SearchResult, error)
    SearchVector(ctx context.Context, embedding []float32, limit int) ([]SearchResult, error)
    Stats(ctx context.Context) (*ImportStats, error)
}
```

Web handlers and CLI commands inject these interfaces, enabling:
- Unit testing with mocks
- Swapping implementations (e.g., different databases)
- Clear dependency boundaries

---

## Part 2: Claude Code Integration

### File Layout (Claude Code Side)

```
.claude/
├── skills/
│   ├── brains-feature/
│   │   └── SKILL.md              # /brains.feature entry point
│   ├── brains-plan/
│   │   └── SKILL.md              # /brains.plan entry point
│   ├── brains-tasks/
│   │   └── SKILL.md              # /brains.tasks entry point
│   ├── brains-research/
│   │   └── SKILL.md              # /brains.research standalone
│   └── brains-audit/
│       └── SKILL.md              # /brains.audit standalone
│
└── agents/
    ├── brains-orchestrator.md    # Coordinates workflow phases
    │
    ├── research-codebase.md      # Codebase exploration expert
    ├── research-domain.md        # Domain knowledge expert
    ├── research-security.md      # Security considerations
    │
    ├── spec-creator.md           # Synthesizes research into spec
    ├── plan-creator.md           # Creates implementation plan
    ├── task-creator.md           # Breaks plan into tasks
    │
    ├── audit-completeness.md     # Checks coverage
    ├── audit-ai-consumer.md      # Checks AI-friendliness
    ├── audit-alignment.md        # Checks artifact consistency
    │
    └── highlighter.md            # Surfaces key decisions
```

### Skill Example: `/brains.feature`

```markdown
---
name: brains-feature
description: Create a new feature specification using the ZombieKit workflow. Use when starting a new feature from scratch.
allowed-tools: Task, Bash, Read, Write
---

# Feature Specification Workflow

This skill orchestrates the full spec cycle: research → create → audit → highlight.

## Workflow

1. **Research Phase** - Spawn research agents in parallel:
   - research-codebase: Explore existing code patterns
   - research-domain: Gather domain knowledge
   - (user-specified domain agents)

2. **Create Phase** - Spawn spec-creator agent:
   - Synthesizes research into spec.md and technical.md

3. **Audit Phase** - Spawn auditors in parallel:
   - audit-completeness
   - audit-ai-consumer
   - (user-specified auditors)

4. **Loop or Highlight**:
   - If CRITICAL/MAJOR issues: loop back to research
   - Otherwise: highlight key decisions for user approval

## Agent Selection

User can specify additional profiles:
```
/brains.feature papi,database "Add user metrics"
```

This passes `papi,database` to research and audit agents.

## Artifact Storage

Save artifacts using brains CLI:
```bash
./brains artifact save --type spec --title "user-metrics" < spec.md
```
```

### Agent Example: Research Agent Wrapper

```markdown
---
name: research-database
description: Database research expert. Use when investigating database schemas, queries, or data patterns.
tools: Bash, Read, Grep, Glob
---

# Database Research Agent

## Setup

First, compose your specialized prompt:

```bash
CONTEXT=$(./brains profiles compose research,database)
```

The above command returns your full research context including:
- Research methodology
- Database-specific heuristics
- Project-specific database conventions

## Your Task

Using the context above, investigate the requested topic with focus on:
- Schema design and relationships
- Query patterns and performance
- Data flow and transformations
- Migration history

## Output Format

Provide findings as structured markdown suitable for the spec-creator agent.
```

### How Parallel Execution Works

Claude Code's Task tool automatically parallelizes when a skill spawns multiple agents:

```markdown
## Research Phase

Spawn these agents in parallel to gather information:

1. Use the **research-codebase** agent to explore existing patterns
2. Use the **research-domain** agent to gather domain knowledge
3. Use the **research-security** agent to identify security considerations

Wait for all agents to complete, then collate their findings.
```

Claude Code sees "spawn these agents in parallel" and invokes them concurrently via the Task tool.

---

## Part 3: Brains CLI Design

### Global Flags

All commands support these global flags:

| Flag | Description |
|------|-------------|
| `--format <raw\|json>` | Output format (default: `raw` for human-readable) |
| `--quiet` | Suppress non-essential output |
| `--verbose` | Include additional metadata |

### Commands

```bash
# ─────────────────────────────────────────────────────────────
# SETUP
# ─────────────────────────────────────────────────────────────

brains init                              # Create .brains/ in current directory
brains init --global                     # Create ~/.brains/ if not exists

# ─────────────────────────────────────────────────────────────
# PROFILES (composition & management)
# ─────────────────────────────────────────────────────────────

brains profile compose <a,b,c>           # Output composed prompt text
brains profile compose <a,b,c> --format json   # Structured output for agents
brains profile compose <a,b,c> --dry-run       # Validate without output
brains profile compose <a,b,c> --stats         # Include token estimates
brains profile list                      # List available profiles
brains profile show <name>               # Show resolved profile content
brains profile show <name> --raw         # Show just local file (no inheritance)
brains profile validate                  # Check for circular deps, missing refs
brains profile create <name>             # Create new profile (stub)

# ─────────────────────────────────────────────────────────────
# REGISTRY (track .brains directories)
# ─────────────────────────────────────────────────────────────

brains registry list                     # Show all registered .brains directories
brains registry add <path>               # Register a directory
brains registry remove <path>            # Unregister a directory

# ─────────────────────────────────────────────────────────────
# SPECS (directory management for workflows)
# ─────────────────────────────────────────────────────────────

brains spec new <title>                  # Create spec dir, print path
brains spec list                         # List spec directories
brains spec list --format json           # Structured output for agents
brains spec search <query>               # Full-text search across specs

# ─────────────────────────────────────────────────────────────
# SERVICES
# ─────────────────────────────────────────────────────────────

brains serve                             # Start MCP server (default: stdio)
brains serve --mode sse --port 8080      # SSE mode for web clients

brains web                               # Start web UI (default: :3000)
brains web --port 8080

brains import                            # Run conversation import (once)
brains import --watch                    # Run as daemon (continuous)

# ─────────────────────────────────────────────────────────────
# DATABASE
# ─────────────────────────────────────────────────────────────

brains db migrate                        # Run pending migrations
brains db status                         # Show migration status

# ─────────────────────────────────────────────────────────────
# CONVERSATIONS
# ─────────────────────────────────────────────────────────────

brains convo search <query>              # Search imported conversations
brains convo stats                       # Show import statistics

# ─────────────────────────────────────────────────────────────
# UTILITY
# ─────────────────────────────────────────────────────────────

brains version                           # Show version
brains doctor                            # Check system health
```

### Exit Codes

All commands use consistent exit codes for machine-readable error handling:

| Code | Name | Description |
|------|------|-------------|
| 0 | `SUCCESS` | Command completed successfully |
| 1 | `GENERAL_ERROR` | Unspecified error |
| 2 | `INVALID_ARGS` | Invalid arguments or flags |
| 3 | `NOT_FOUND` | Profile, spec, or resource not found |
| 4 | `CIRCULAR_DEP` | Circular dependency detected |
| 5 | `PERMISSION` | File permission error |
| 10 | `DB_CONNECTION` | Database connection failed |
| 11 | `DB_MIGRATION` | Migration required or failed |

### JSON Output Format

When `--format json` is used, all commands return structured responses:

**Success response:**
```json
{
  "success": true,
  "data": { /* command-specific payload */ },
  "metadata": {
    "duration_ms": 45,
    "warnings": []
  }
}
```

**Error response:**
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Profile 'nonexistent' not found",
    "details": {
      "requested": "nonexistent",
      "searched_paths": ["/project/.brains/profiles/", "~/.brains/profiles/"],
      "suggestion": "Did you mean 'research'?"
    }
  }
}
```

**Example: `brains profile compose` with JSON:**
```json
{
  "success": true,
  "data": {
    "content": "# Research Methodology\n\nWhen investigating...",
    "profiles_used": ["research", "database"],
    "resolution": [
      {"name": "research", "source": "~/.brains/profiles/research.md", "inherited": false},
      {"name": "database", "source": "/project/.brains/profiles/database.md", "inherited": true}
    ]
  },
  "metadata": {
    "duration_ms": 23,
    "total_chars": 4582,
    "estimated_tokens": 1200,
    "warnings": []
  }
}
```

**Example: `brains spec list` with JSON:**
```json
{
  "success": true,
  "data": {
    "specs": [
      {
        "id": "675d8a3f-user-authentication",
        "title": "user-authentication",
        "path": "specs/675d8a3f-user-authentication",
        "created": "2024-01-15T10:30:00Z",
        "artifacts": ["spec", "plan", "tasks"],
        "last_modified": "2024-01-15T14:22:00Z"
      }
    ]
  },
  "metadata": {
    "total_count": 1,
    "duration_ms": 12
  }
}
```

### Spec Directory Management

The `brains spec` commands manage spec directories without abstracting away the filesystem. Agents handle all file I/O directly.

**Creating a new spec:**
```bash
$ brains spec new "user authentication"
specs/675d8a3f-user-authentication

# Agent then writes files directly:
$ cat > specs/675d8a3f-user-authentication/spec.md << 'EOF'
# User Authentication
...
EOF
```

**Listing specs:**
```bash
$ brains spec list
675d8a3f-user-authentication    2024-01-15  (spec, plan, tasks)
675d8b12-payment-refactor       2024-01-14  (spec, plan)
675d8c01-fix-login-bug          2024-01-13  (spec)
```

**Searching:**
```bash
$ brains spec search "OAuth"
specs/675d8a3f-user-authentication/spec.md:42: Support OAuth 2.0 providers
specs/675d8a3f-user-authentication/technical.md:15: OAuth flow diagram
```

### Directory Layout (Brains CLI)

```
zombiekit/
├── cmd/
│   └── brains/                   # Single CLI binary
│       └── main.go
│
├── internal/
│   ├── cli/                      # Command implementations
│   │   ├── root.go               # Root command, global flags
│   │   ├── init.go               # brains init
│   │   ├── profile.go            # brains profile *
│   │   ├── registry.go           # brains registry *
│   │   ├── spec.go               # brains spec *
│   │   ├── serve.go              # brains serve
│   │   ├── web.go                # brains web
│   │   ├── import.go             # brains import
│   │   ├── db.go                 # brains db *
│   │   └── convo.go              # brains convo *
│   │
│   ├── config/                   # Configuration system
│   │   ├── config.go             # Config struct and loading
│   │   ├── loader.go             # File discovery and parsing
│   │   └── defaults.go           # Default values
│   │
│   ├── profile/                  # Profile domain
│   │   ├── service.go            # ProfileService interface + impl
│   │   ├── resolver.go           # Directory tree walking
│   │   ├── composer.go           # Merge multiple profiles
│   │   ├── registry.go           # Track known .brains dirs
│   │   ├── loader.go             # Parse profile files
│   │   ├── validator.go          # Check for errors
│   │   └── types.go
│   │
│   ├── spec/                     # Spec domain
│   │   ├── service.go            # SpecService interface + impl
│   │   ├── manager.go            # Create, list, search
│   │   ├── naming.go             # Hex-timestamp conventions
│   │   └── types.go
│   │
│   ├── conversation/             # Conversation domain
│   │   ├── service.go            # ConversationService interface
│   │   ├── repository.go         # Repository interface
│   │   ├── postgres/             # PostgreSQL implementation
│   │   │   └── repository.go
│   │   ├── importer/             # File watcher + import logic
│   │   ├── parser/               # Claude conversation parser
│   │   └── embedder/             # Embedding provider interface
│   │       ├── embedder.go       # Interface
│   │       └── ollama/           # Ollama implementation
│   │
│   ├── mcp/                      # MCP server & tools
│   │   ├── server.go             # MCP protocol handler
│   │   ├── framework/            # Tool interface (from mcp-genie)
│   │   │   └── tool.go
│   │   └── tools/                # MCP tool implementations
│   │       ├── stickymemory/
│   │       └── codereasoning/
│   │
│   └── web/                      # Web frontend
│       ├── server.go             # HTTP server setup
│       ├── handlers/             # Route handlers
│       │   ├── profile.go
│       │   ├── spec.go
│       │   └── conversation.go
│       └── ui/                   # Embedded frontend assets
│
├── profiles/                     # Default global profiles
│   ├── research.md               # Base research methodology
│   ├── spec-creator.md           # Spec creation guidelines
│   ├── audit-completeness.md     # Completeness audit rules
│   └── ...
│
└── migrations/                   # PostgreSQL schemas
```

---

## Part 4: Profile System

### What is a Profile?

A **profile** is a composable unit of prompt content. It contains:
- Instructions and methodology
- Domain-specific knowledge
- Rules and constraints
- References to other profiles

### Profile Format Specification

```markdown
---
name: database                    # Required, must match filename (without .md)
description: Database expertise   # Optional, for UI/discovery
includes:                         # Optional, other profiles to pull in
  - research
  - sql-conventions
inherits: true                    # Optional, default true
---

# Database Expert Profile

All markdown below frontmatter is the actual prompt content.

## Methodology

When investigating database-related topics:
1. Start with schema exploration
2. Trace data flow through queries
3. Check for indexing and performance patterns

## Conventions

- Use PostgreSQL-style naming (snake_case)
- Prefer explicit JOINs over implicit
- Always consider NULL handling
```

### Frontmatter Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | Yes | - | Identifier, must match filename (e.g., `database` for `database.md`) |
| `description` | No | - | Human-readable purpose, shown in UI and `profiles list` |
| `includes` | No | `[]` | List of other profiles to pull in before this one |
| `inherits` | No | `true` | Whether to prepend same-named profile from parent directory |

### Merge Rules

When `./brains profiles compose research,database,papi` is called:

**Step 1: Resolve each profile independently**

For each profile name in the list:
1. Find the profile file (walk directory tree, closest to CWD wins)
2. Resolve `includes` recursively (depth-first, deduplicate)
3. If `inherits: true`, prepend content from same-named profile in parent directories

**Step 2: Concatenate in order**

```
┌─────────────────────────────────────────────────────────┐
│  Final Output                                           │
├─────────────────────────────────────────────────────────┤
│  [research profile content]                             │
│  ─────────────────────────                              │
│  [database profile content]                             │
│  ─────────────────────────                              │
│  [papi profile content]                                 │
└─────────────────────────────────────────────────────────┘
```

- Left-to-right order (first profile listed appears first in output)
- Later profiles appear later in the output
- No conflict resolution - concatenation only
- LLM interprets combined context naturally

**Step 3: Deduplication**

If the same profile would be included multiple times (via `includes` or explicit listing), it appears only once at its first occurrence.

### Inheritance Example

Given:
```
~/.brains/profiles/database.md           # Global: general SQL patterns
/project/.brains/profiles/database.md    # Local: project-specific conventions
```

With `inherits: true` (default):
```
┌─────────────────────────────────────────────────────────┐
│  Resolved "database" profile                            │
├─────────────────────────────────────────────────────────┤
│  [~/.brains/profiles/database.md content]               │  ← parent first
│  ─────────────────────────────────────────              │
│  [/project/.brains/profiles/database.md content]        │  ← local second
└─────────────────────────────────────────────────────────┘
```

With `inherits: false`:
```
┌─────────────────────────────────────────────────────────┐
│  Resolved "database" profile                            │
├─────────────────────────────────────────────────────────┤
│  [/project/.brains/profiles/database.md content]        │  ← local only
└─────────────────────────────────────────────────────────┘
```

### Includes Example

Given `database.md`:
```yaml
---
name: database
includes:
  - research
  - sql-conventions
---
```

Resolution order:
1. Load `research` profile (and its includes, recursively)
2. Load `sql-conventions` profile (and its includes, recursively)
3. Load `database` profile content
4. Concatenate: `research` + `sql-conventions` + `database`

### Hierarchical Resolution

Profiles resolve by walking up the directory tree:

```
/home/user/projects/myapp/src/feature/
       │
       ▼ (check first - highest priority)
/home/user/projects/myapp/src/feature/.brains/profiles/
       │
       ▼
/home/user/projects/myapp/.brains/profiles/  ← git root
       │
       ▼ (check last - lowest priority)
~/.brains/profiles/  ← global defaults
```

**Resolution rule:** Closest to CWD wins for conflicts.

### Composition Example

```bash
$ ./brains profiles compose research,database,papi

# Output: merged markdown combining:
# - ~/.brains/profiles/research.md (base research methodology)
# - ~/.brains/profiles/database.md (database expertise)
# - /project/.brains/profiles/papi.md (project-specific API patterns)
# With conflicts resolved by precedence (papi > database > research)
```

### Profile Registry

Brains maintains a registry of known `.brains` directories:

| Source | How Added |
|--------|-----------|
| Discovered | First `./brains profiles compose` call from a directory |
| Explicit | `./brains profiles add <path>` |
| Web UI | Manual addition |

This enables the web UI to show profiles across all projects.

---

## Part 5: Spec Storage

### Design Principle

The `brains` CLI manages spec *directories* but not file contents. Agents use Claude Code's native file I/O (Read, Write tools) for all content operations. This keeps the CLI simple and leverages what Claude Code already does well.

### Directory Structure

```
project/
├── .brains/
│   ├── config.yml              # Project settings
│   └── profiles/               # Local profile overrides
│
└── specs/
    └── {hex-timestamp}-{title}/
        ├── spec.md             # Business requirements
        ├── technical.md        # Technical constraints
        ├── research.md         # Research findings
        ├── plan.md             # Implementation plan
        ├── tasks.md            # Task breakdown
        └── audit/
            └── {date}.md       # Audit reports
```

### Naming Convention

- `hex-timestamp` = hex-encoded Unix timestamp (e.g., `675d8a3f`)
- `title` = AI-summarized slug (e.g., `user-metrics`)
- Full: `675d8a3f-user-metrics/`

### CLI vs Agent Responsibilities

| Task | Who Does It | How |
|------|-------------|-----|
| Create spec directory | CLI | `brains spec new "title"` |
| Write spec.md content | Agent | Claude Code Write tool |
| Read technical.md | Agent | Claude Code Read tool |
| List all specs | CLI | `brains spec list` |
| Search spec content | CLI | `brains spec search "query"` |
| Pass spec path between phases | Skill | Orchestration via Task tool |

### Agent Workflow Example

```markdown
# In brains-feature skill:

## Step 1: Create spec directory
```bash
SPEC_DIR=$(brains spec new "user authentication")
# Returns: specs/675d8a3f-user-authentication
```

## Step 2: Run research agents (parallel)
Pass $SPEC_DIR to each agent...

## Step 3: Spec creator writes files
The spec-creator agent writes directly:
- $SPEC_DIR/spec.md
- $SPEC_DIR/technical.md
- $SPEC_DIR/research.md

## Step 4: Continue to /brains.plan
Pass $SPEC_DIR to the plan skill...
```

---

## Part 6: Workflow Commands

### Command Namespace

All `/brains.*` commands are Claude Code skills that orchestrate agents:

| Command | Skill | Purpose |
|---------|-------|---------|
| `/brains.feature` | brains-feature | New feature specification |
| `/brains.bug` | brains-bug | Bug fix specification |
| `/brains.refactor` | brains-refactor | Refactoring specification |
| `/brains.plan` | brains-plan | Implementation planning |
| `/brains.tasks` | brains-tasks | Task breakdown |
| `/brains.eat` | brains-eat | Execute tasks (implement) |
| `/brains.research` | brains-research | Standalone research |
| `/brains.audit` | brains-audit | Alignment check |
| `/brains.clarify` | brains-clarify | Surface ambiguities |
| `/brains.update` | brains-update | Modify existing artifact |
| `/brains.revise` | brains-revise | Re-enter cycle to update spec |

### The Core Cycle

Each major command follows the same pattern:

```
┌─────────────────────────────────────────────────────────────┐
│                    THE ZOMBIEKIT CYCLE                       │
│                                                             │
│    ┌──────────────────────────────────────────────────┐     │
│    │  RESEARCH          (many agents, parallel)       │     │
│    │  • Agents call: ./brains profiles compose ...    │     │
│    │  • Each gets domain-specific prompt              │     │
│    │  • Results collated by orchestrator              │     │
│    └─────────────────────┬────────────────────────────┘     │
│                          │                                   │
│                          ▼                                   │
│    ┌──────────────────────────────────────────────────┐     │
│    │  CREATE            (single agent)                │     │
│    │  • Agent calls: ./brains profiles compose ...    │     │
│    │  • Synthesizes research into artifact            │     │
│    │  • Saves via: ./brains artifact save ...         │     │
│    └─────────────────────┬────────────────────────────┘     │
│                          │                                   │
│                          ▼                                   │
│    ┌──────────────────────────────────────────────────┐     │
│    │  AUDIT             (many agents, parallel)       │     │
│    │  • Each auditor has specialized profile          │     │
│    │  • Check completeness, AI-friendliness, etc.     │     │
│    │  • Return findings with severity levels          │     │
│    └─────────────────────┬────────────────────────────┘     │
│                          │                                   │
│              ┌───────────┴───────────┐                       │
│              ▼                       ▼                       │
│    CRITICAL/MAJOR found?      MINOR/NONE only               │
│              │                       │                       │
│              ▼                       ▼                       │
│         LOOP BACK              HIGHLIGHT                     │
│         to RESEARCH            to user                       │
│                                      │                       │
│                           ┌──────────┴──────────┐            │
│                           ▼                     ▼            │
│                      APPROVED              FEEDBACK          │
│                           │                     │            │
│                           ▼                     ▼            │
│                      NEXT STAGE            LOOP BACK         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Conflict Resolution

When auditors produce conflicting findings (e.g., "too complex" vs "insufficiently detailed"), the conflict is surfaced to the user. The user decides which feedback to incorporate. The system never silently resolves disagreements.

---

## Part 7: MCP Server & Tools

### Available Tools

| Tool | Purpose |
|------|---------|
| `sticky-memory` | Persistent key-value storage across sessions |
| `code-thinking` | Structured sequential reasoning |

### Server Modes

```bash
# STDIO mode (Claude Desktop)
./brains serve --mode stdio

# SSE mode (web browser)
./brains serve --mode sse --port 8080
```

---

## Part 8: Conversation Import

### Background Service

```bash
# Start import daemon
./brains import --watch

# One-time import
./brains import --once
```

### Storage

- PostgreSQL with pgvector extension
- Full-text search via GIN indexes
- Vector embeddings via local Ollama

### Message Categorization

| Category | Description |
|----------|-------------|
| `system_prompt` | Original system instructions |
| `ai_response` | Claude's responses |
| `user_message` | User questions/followups |
| `file_reference` | Links to local files |
| `url_reference` | Links to external URLs |

---

## Part 9: Web UI

### Capabilities

1. **Profile Management**
   - View profiles across all registered projects
   - Edit profile content
   - Test composition results

2. **Artifact Browser**
   - View specs, plans, tasks
   - See audit history
   - Track workflow progress

3. **Conversation Search**
   - Full-text search
   - Vector similarity search
   - Filter by project/date/role

### Access

```bash
./brains web --port 3000
# Open http://localhost:3000
```

---

## Part 10: Iterative Development & Backtracking

The spec → plan → tasks → implement chain is not strictly linear. Implementation often reveals spec issues. The system handles this through the `/brains.revise` command.

### The Revise Command

```bash
/brains.revise <spec-path> "reason for revision"
```

**What it does:**
1. Archives current artifacts with version suffix (e.g., `spec.v1.md`)
2. Re-enters the research phase with new context
3. Creates updated artifacts while preserving history
4. Runs audit to ensure changes are consistent

### Artifact Versioning

When revising, previous versions are preserved:

```
specs/675d8a3f-user-authentication/
├── spec.md              # Current version
├── spec.v1.md           # First version (archived)
├── spec.v2.md           # Second version (archived)
├── technical.md         # Current version
├── technical.v1.md      # First version (archived)
├── plan.md
├── tasks.md
├── revision-log.md      # History of revisions
└── audit/
```

### Revision Log Format

```markdown
# Revision Log

## v3 (current) - 2024-01-16
**Reason:** Implementation revealed OAuth flow needs PKCE
**Changed:** spec.md (OAuth section), technical.md (auth flow diagram)
**By:** /brains.revise during implementation

## v2 - 2024-01-15
**Reason:** Security audit found missing rate limiting
**Changed:** technical.md (rate limiting requirements)
**By:** /brains.audit feedback

## v1 - 2024-01-14
**Reason:** Initial specification
**Created:** spec.md, technical.md
**By:** /brains.feature
```

### When to Revise vs Update

| Scenario | Command | Why |
|----------|---------|-----|
| Typo or minor clarification | `/brains.update` | No re-research needed |
| New requirement discovered | `/brains.revise` | Needs full cycle |
| Implementation blocked | `/brains.revise` | Spec was incomplete |
| Audit found major issue | `/brains.revise` | Need to rethink approach |
| Scope change from user | `/brains.revise` | Fundamental change |

### Agent Decision Tree

```
Implementation blocked or issue discovered
    │
    ├─► Minor issue (typo, unclear wording)
    │   └─► Use /brains.update (quick fix)
    │
    └─► Major issue (missing requirement, wrong approach)
        └─► Use /brains.revise
            │
            ├─► Archives current artifacts
            ├─► Re-enters research phase
            ├─► Creates new versions
            └─► Runs audit on changes
```

---

## Part 11: Agent Integration Patterns

This section provides concrete patterns for agents calling the brains CLI.

### Pattern 1: Research Agent Setup

```bash
#!/bin/bash
# In a research agent

# Step 1: Compose context with error handling
if ! CONTEXT=$(brains profile compose research,database --format raw 2>/dev/null); then
    EXIT_CODE=$?
    case $EXIT_CODE in
        3) echo "ERROR: Profile not found" >&2 ;;
        4) echo "ERROR: Circular dependency" >&2 ;;
        *) echo "ERROR: Composition failed (code: $EXIT_CODE)" >&2 ;;
    esac
    exit 1
fi

# Step 2: CONTEXT now contains your full research prompt
# Use it as system instructions for your investigation
```

### Pattern 2: Spec Creation Workflow

```bash
# In brains-feature skill

# Step 1: Create spec directory
SPEC_DIR=$(brains spec new "user authentication" --format raw)
if [ $? -ne 0 ]; then
    echo "Failed to create spec directory"
    exit 1
fi

# Step 2: Pass to research agents (Claude Code handles parallelism)
# Each agent writes to $SPEC_DIR/research.md

# Step 3: Spec creator writes files using Claude Code Write tool
# Writes to: $SPEC_DIR/spec.md, $SPEC_DIR/technical.md

# Step 4: Continue to next phase
echo "Spec created at: $SPEC_DIR"
```

### Pattern 3: JSON Output Parsing

```bash
# When structured data is needed

# Get spec list as JSON
SPECS_JSON=$(brains spec list --format json)

# Check for success
SUCCESS=$(echo "$SPECS_JSON" | jq -r '.success')
if [ "$SUCCESS" != "true" ]; then
    ERROR=$(echo "$SPECS_JSON" | jq -r '.error.message')
    echo "Error: $ERROR" >&2
    exit 1
fi

# Extract data
LATEST_SPEC=$(echo "$SPECS_JSON" | jq -r '.data.specs[0].path')
echo "Latest spec: $LATEST_SPEC"
```

### Pattern 4: Error Recovery

```bash
# Graceful fallback when optional profile missing

# Try with custom domain profile
if brains profile compose research,custom-domain --dry-run 2>/dev/null; then
    CONTEXT=$(brains profile compose research,custom-domain --format raw)
else
    # Fallback to base profile
    echo "WARN: custom-domain profile not found, using research only" >&2
    CONTEXT=$(brains profile compose research --format raw)
fi
```

### Pattern 5: Auditor Conflict Handling

```
Audit phase complete
    │
    ├─► All auditors agree (PASS)
    │   └─► Continue to highlight phase
    │
    ├─► All auditors agree (FAIL with same issues)
    │   └─► Loop back to research with combined feedback
    │
    └─► Auditors disagree (conflicting findings)
        └─► Surface conflict to user
            │
            ├─► User selects which feedback to incorporate
            └─► Agent continues with chosen direction
```

### Pattern 6: Context Size Management

```bash
# Check composition size before using

RESULT=$(brains profile compose research,database,security --format json --stats)

TOKENS=$(echo "$RESULT" | jq -r '.metadata.estimated_tokens')

if [ "$TOKENS" -gt 50000 ]; then
    echo "WARN: Large context ($TOKENS tokens), consider reducing profiles" >&2
    # Optionally fall back to fewer profiles
fi
```

### Example: Complete Agent Wrapper

```markdown
---
name: research-database
description: Database research expert
tools: Bash, Read, Grep, Glob
---

# Database Research Agent

## Setup

First, compose your specialized prompt:

```bash
# Get composed context with error handling
RESULT=$(brains profile compose research,database --format json)

if [ "$(echo "$RESULT" | jq -r '.success')" != "true" ]; then
    ERROR=$(echo "$RESULT" | jq -r '.error.message')
    echo "Failed to compose profiles: $ERROR" >&2
    exit 1
fi

# Extract the content
CONTEXT=$(echo "$RESULT" | jq -r '.data.content')

# Log resolution for debugging
echo "Profiles used: $(echo "$RESULT" | jq -r '.data.profiles_used | join(", ")')" >&2
```

## Your Task

Using the context above, investigate the requested topic.

## Error Handling

If you encounter issues:
- Exit code 3: Profile not found → check available profiles with `brains profile list`
- Exit code 4: Circular dependency → report to user, cannot proceed
- Exit code 5: Permission error → check file permissions

## Output Format

Return findings as structured markdown for the spec-creator agent.
```

---

## Part 12: Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go 1.22+ | Single binary, concurrency, proven in mcp-genie |
| CLI Framework | urfave/cli/v2 | Lightweight, less boilerplate than cobra |
| Database | PostgreSQL 16 + pgvector | Full-text + vector search |
| DB Driver | pgx/v5 | Native PostgreSQL driver, better than database/sql |
| Build | Taskfile | Readable YAML, proven in telegraph/ai |
| MCP | mark3labs/mcp-go | Only mature Go MCP library |
| Web | stdlib net/http | Lightweight, sufficient for management UI |
| YAML | gopkg.in/yaml.v3 | Profile frontmatter parsing |
| File Watch | fsnotify | For `brains import --watch` |
| Logging | slog | Go 1.22+ stdlib structured logging |

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
│   • LLM inference                • Conversation import      │
│                                  • Web UI                   │
│                                                             │
│   Skills call agents ──────────► Agents call CLI            │
│   Agents return results ◄─────── CLI returns prompts        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

The core insight: **Claude Code is the brain, brains CLI is the memory.**
