# ZombieKit Design Document

> *"Feed your codebase some brains."*

An AI-assisted software development framework with iterative cycles, auditable history, and decoupled change management.

---

## Core Concepts

### Initiative

An **initiative** is a user's goal or request that may require multiple specifications, refactors, and bug fixes to complete. It serves as an umbrella for related work that shares context.

```
history/
└── 2024-01-15-add-user-authentication/
    ├── INITIATIVE.md           # Goal, context, status
    ├── features/
    │   ├── 001-auth-api/
    │   └── 002-session-management/
    ├── refactors/
    │   └── 001-extract-middleware/
    └── bugs/
        └── 001-token-expiry-edge-case/
```

**INITIATIVE.md** contains:
- Original user request
- Updated/refined goals as work progresses
- Shared context useful across all work items
- Completion criteria
- Status and decision log

### Work Item Types

| Type | Changes Spec? | Changes Code? | When to Use |
|------|---------------|---------------|-------------|
| **Feature** | ✅ | ✅ | New or modified user-visible capability |
| **Refactor** | ❌ | ✅ | Restructure code without changing behavior |
| **Bug** | Maybe | ✅ | Something doesn't work as expected |

### Decoupling Principle

When a request contains multiple logically independent changes, ZombieKit separates them into distinct work items. Each can have its own research, testing, and implementation path.

**Example:** "Add version display to CLI"
- **Feature 001:** Version display in CLI output
- **Feature 002:** Build system integration for version embedding

These may share an initiative but proceed independently, allowing:
- Different research requirements
- Independent testing strategies
- Parallel implementation if desired
- Cleaner commit history

---

## Folder Structure

```
project/
├── .brains/                    # Configuration & overrides
│   ├── config.toml             # ZombieKit settings
│   ├── agents/                 # Custom agent overrides
│   └── templates/              # Custom templates
│
├── history/                    # Auditable changelog (committed)
│   ├── 2024-01-15-user-auth/
│   │   ├── INITIATIVE.md
│   │   ├── features/
│   │   │   └── 001-auth-api/
│   │   │       ├── business-spec.md
│   │   │       ├── technical-spec.md
│   │   │       ├── technical-requirements-research.md
│   │   │       ├── research-summary.md
│   │   │       ├── implementation-plan.md
│   │   │       ├── proof-tests/
│   │   │       └── audit-reports/
│   │   ├── refactors/
│   │   └── bugs/
│   └── 2024-01-20-performance-optimization/
│       └── ...
│
└── src/
```

---

## The Specification Split

During initial specification creation, ZombieKit separates concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                     USER INPUT                                  │
│  "I want auth with JWT tokens using RS256 and Redis sessions"  │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │      SEPARATION PROCESS       │
              │                               │
              │  Extract technical impl       │
              │  details from business need   │
              └───────────────┬───────────────┘
                              │
            ┌─────────────────┴─────────────────┐
            ▼                                   ▼
┌───────────────────────┐           ┌───────────────────────────────┐
│   business-spec.md    │           │ technical-requirements-       │
│                       │           │ research.md                   │
│ • User can log in     │           │                               │
│ • Sessions persist    │           │ • JWT with RS256 (user pref)  │
│ • Tokens expire       │           │ • Redis for sessions          │
│                       │           │ • Research: alternatives?     │
└───────────────────────┘           └───────────────────────────────┘
                                                  │
                                                  ▼
                                    ┌───────────────────────────────┐
                                    │  Used during /brains.plan     │
                                    │  to inform technical-spec.md  │
                                    └───────────────────────────────┘
```

**Why this matters:**
- Business spec stays focused on "what", not "how"
- Technical preferences aren't lost—they're preserved for planning
- Research agents can challenge or validate technical assumptions
- Clear separation of concerns throughout the workflow

---

## Workflow Modes

### Standard Mode (Default)

Full research → create → audit → highlight cycles at each stage. Maximum thoroughness.

```
/brains feature "add user authentication"
```

### Fast Mode

Reduced cycles for simpler changes or when speed matters more than exhaustive research.

```
/brains feature --fast "add user authentication"
```

| Aspect | Standard | Fast |
|--------|----------|------|
| Research depth | Comprehensive, multiple agents | Single-pass, key sources only |
| Audit iterations | Until no MAJOR issues | Single audit pass |
| Proof tests | Full TDD cycle | Smoke tests only |
| User checkpoints | After each phase | Only at completion |

---

## Commands

### Work Item Commands

```bash
/brains feature "description"      # New/add feature spec
/brains refactor "description"     # New/add refactoring work
/brains bug "description"          # New/add bug investigation

# All support --fast and --new flags
/brains feature --fast "description"   # Reduced thoroughness
/brains feature --new "description"    # Force new initiative
```

**State inference:**
- No active initiative → Creates new initiative, adds work item
- Active initiative → Adds work item to current
- `--new` flag → Completes current initiative, starts new one

### Initiative Management

```bash
/brains status
# Shows current initiative, work items, and next steps

/brains complete
# Marks initiative as complete, clears active state
```

### Workflow Commands

```bash
/brains plan                       # Create implementation plan (includes proof spikes)
/brains tasks                      # Generate task list
/brains implement                  # Execute tasks
```

### Research & Audit

```bash
/brains research "topic"           # Standalone research
/brains audit                      # Audit current artifacts
/brains clarify                    # Identify gaps needing clarification
```

### Server & Infrastructure

```bash
/brains serve
# Single command that starts:
#   • MCP server (SSE transport, default)
#   • Web interface server
#   • Conversation importer
#   • Vector embedding service connection
```

---

## Active Initiative State

ZombieKit tracks the current initiative in `.brains/active.json` (gitignored):

```json
{
  "initiative": "history/2024-01-15-add-user-auth",
  "started": "2024-01-15T10:30:00Z",
  "last_activity": "2024-01-15T14:22:00Z",
  "current_work_item": "features/001-login-endpoint"
}
```

**State inference flow:**

```
┌────────────────────────────────────────────────────────┐
│  /brains <command> "description"                       │
│       │                                                │
│       ▼                                                │
│  Active initiative? ──NO──▶ Create new, set active     │
│       │                                                │
│      YES                                               │
│       │                                                │
│       ▼                                                │
│  --new flag? ──YES──▶ Complete current, create new     │
│       │                                                │
│      NO                                                │
│       │                                                │
│       ▼                                                │
│  Add to current initiative                             │
└────────────────────────────────────────────────────────┘
```

**Status output example:**

```
$ /brains status

Initiative: add-user-authentication (active)
Started: 2024-01-15

Work Items:
  ✅ features/001-login-endpoint (complete)
  🔄 features/002-session-management (in progress - planning)
  ⏳ refactors/001-extract-middleware (queued)

Next: Continue with /brains plan, or add more work items
```

---

## Task Complexity Management

Before generating tasks, ZombieKit evaluates complexity:

```
┌─────────────────────────────────────────────────────────────────┐
│                    COMPLEXITY ANALYSIS                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Factors evaluated:                                             │
│  • Number of files affected                                     │
│  • Cross-module dependencies                                    │
│  • Estimated lines of change                                    │
│  • Number of new interfaces/contracts                           │
│  • Testing surface area                                         │
│                                                                 │
│  Thresholds:                                                    │
│  • Simple: < 5 files, < 200 LOC, single module                 │
│  • Medium: 5-15 files, 200-500 LOC, 2-3 modules                │
│  • Complex: > 15 files, > 500 LOC, 4+ modules                  │
│                                                                 │
│  If Complex:                                                    │
│  → Automatically split into multiple task lists                 │
│  → Each list is independently implementable                     │
│  → Dependency order is calculated                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Planning with Proof Spikes

The planning phase includes lightweight implementation spikes to validate assumptions before committing to a full plan.

```
┌─────────────────────────────────────────────────────────────────┐
│                      /brains plan                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. RESEARCH                                                    │
│     • Load technical-requirements-research.md                   │
│     • Investigate implementation approaches                     │
│     • Identify unknowns and risks                               │
│                                                                 │
│  2. SPIKE (where needed)                                        │
│     • Write minimal proof-of-concept code for risky areas       │
│     • Validate interfaces, APIs, library behavior               │
│     • Document findings in spike-results.md                     │
│                                                                 │
│  3. PLAN                                                        │
│     • Create implementation plan informed by spike results      │
│     • Order steps based on dependencies                         │
│     • Flag remaining uncertainties                              │
│                                                                 │
│  4. AUDIT                                                       │
│     • Verify plan is complete and achievable                    │
│     • Check for gaps between spec and plan                      │
│                                                                 │
│  Outputs:                                                       │
│  • implementation-plan.md                                       │
│  • spike-results.md (if spikes were needed)                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**When spikes are triggered:**
- External API/library usage with unclear behavior
- Performance-critical paths needing validation
- Integration points with existing code
- Any area flagged as "uncertain" in research

**Spike artifacts are temporary** — they validate assumptions but aren't production code.

---

## Architecture

### Server Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    /brains serve                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌────────────────┐  │
│  │   MCP Server    │  │   Web Server    │  │  Conversation  │  │
│  │   (SSE)         │  │                 │  │  Importer      │  │
│  │                 │  │  • Dashboard    │  │                │  │
│  │  • Tool calls   │  │  • History view │  │  • Watch dirs  │  │
│  │  • Agent comms  │  │  • Status       │  │  • Parse logs  │  │
│  └────────┬────────┘  └────────┬────────┘  └───────┬────────┘  │
│           │                    │                    │           │
│           └────────────────────┼────────────────────┘           │
│                                │                                │
│                                ▼                                │
│                    ┌───────────────────────┐                    │
│                    │   Vector Service      │                    │
│                    │   Interface           │                    │
│                    │                       │                    │
│                    │   → Local Ollama      │                    │
│                    │   → (extensible)      │                    │
│                    └───────────────────────┘                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Vector Embedding Interface

ZombieKit calls an LLM for vector embeddings (not for reasoning—that's Claude Code's job).

```
┌─────────────────────────────────────────────────────────────────┐
│                 EMBEDDING SERVICE INTERFACE                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Default: Local Ollama instance                                 │
│                                                                 │
│  Interface:                                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  POST /embed                                             │   │
│  │  {                                                       │   │
│  │    "text": "content to embed",                           │   │
│  │    "model": "nomic-embed-text"  // configurable          │   │
│  │  }                                                       │   │
│  │                                                          │   │
│  │  Response:                                               │   │
│  │  {                                                       │   │
│  │    "embedding": [0.1, 0.2, ...],                         │   │
│  │    "model": "nomic-embed-text",                          │   │
│  │    "dimensions": 768                                     │   │
│  │  }                                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Configuration (.brains/config.toml):                           │
│  [embedding]                                                    │
│  provider = "ollama"        # ollama | openai | custom          │
│  endpoint = "http://localhost:11434"                            │
│  model = "nomic-embed-text"                                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Note:** This is the ONLY LLM call ZombieKit makes directly. All reasoning/generation is delegated to Claude Code.

---

## Work Item Artifacts

### Feature

```
features/001-feature-name/
├── business-spec.md                    # What (user-visible behavior)
├── technical-spec.md                   # How (implementation design)
├── technical-requirements-research.md  # User's tech preferences + research
├── research-summary.md                 # Domain research findings
├── implementation-plan.md              # Ordered steps
├── spike-results.md                    # Findings from proof spikes (if any)
└── audit-reports/
    ├── 001-spec-audit.md
    └── 002-plan-audit.md
```

### Refactor

```
refactors/001-refactor-name/
├── goal.md                   # What improvement, why
├── constraints.md            # Behavior that MUST NOT change
├── dependency-analysis.md    # What code is affected
├── refactor-plan.md          # Ordered atomic steps
├── safety-net.md             # Existing + added test coverage
└── progress.md               # Status, blockers, decisions
```

### Bug

```
bugs/001-bug-name/
├── report.md                 # Original bug report
├── reproduction.md           # Steps, environment, failing test
├── investigation.md          # Root cause analysis
├── classification.md         # Spec gap vs impl error
├── fix-plan.md               # Required changes
├── spec-update.md            # If spec gap: the spec change
└── verification.md           # Tests added, regression results
```

---

## Configuration

**.brains/config.toml**

```toml
[general]
default_mode = "standard"     # standard | fast

[server]
mcp_transport = "sse"         # sse | stdio
mcp_port = 3000
web_port = 8080

[embedding]
provider = "ollama"
endpoint = "http://localhost:11434"
model = "nomic-embed-text"

[database]
# Vector embeddings stored in postgres with pgvector
connection = "postgresql://localhost:5432/zombiekit"

[complexity]
simple_max_files = 5
simple_max_loc = 200
medium_max_files = 15
medium_max_loc = 500
# Above medium = complex, auto-split

[agents]
# Override default agents
spec_research = "custom-research-agent"
```

---

## Workflow Summary

```
┌─────────────────────────────────────────────────────────────────┐
│                    ZOMBIEKIT WORKFLOW                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  /brains feature|refactor|bug ──▶ Create/add work item          │
│       │                          (auto-creates initiative       │
│       │                           if none active)               │
│       │                                                         │
│       │    ┌──────────────────────────────────────────────┐    │
│       │    │            QUALITY LOOP                      │    │
│       │    │  research → create → audit → highlight       │    │
│       │    │  (single pass in --fast mode)                │    │
│       │    └──────────────────────────────────────────────┘    │
│       ▼                                                         │
│  /brains plan ──▶ Create implementation plan                    │
│       │           (includes proof spikes where needed)          │
│       │                                                         │
│       │    (same quality loop)                                  │
│       ▼                                                         │
│  /brains tasks ──▶ Generate task list(s)                        │
│       │                                                         │
│       │    (complexity check: split if needed)                  │
│       ▼                                                         │
│  /brains implement ──▶ Execute tasks                            │
│       │                                                         │
│       ▼                                                         │
│  /brains complete ──▶ Mark initiative done                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Open Questions

1. **Fast mode specifics:** What exactly gets skipped? Current thinking: single iteration through quality loop instead of cycling until no issues. Should this be configurable per-project?

2. **Spike artifact retention:** Should spike code be kept in the history for reference, or discarded after informing the plan?

3. **Initiative naming:** Currently derived from first work item + timestamp. Should users be able to rename? Override?
