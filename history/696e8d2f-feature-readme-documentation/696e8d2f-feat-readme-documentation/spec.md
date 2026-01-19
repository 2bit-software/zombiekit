---
status: approved
type: feature
name: readme-documentation
created: 2026-01-19
---

# README Documentation Specification

## Goal

Create a user-focused README.md that enables quick bootstrapping and provides a clear overview of ZombieKit's capabilities.

## Target Audience

Developers who use Claude Code and want structured AI-assisted development workflows.

## Non-Goals

- Contributor documentation (covered by DESIGN.md)
- CI badges (not running in CI)
- Exhaustive API reference (covered by DESIGN.md)
- Documenting all 17 skills (README shows core workflow, DESIGN.md has full list)

---

## User Scenarios & Testing

### User Story 1 - Quick Start Installation (Priority: P1)

A developer wants to install ZombieKit and start using it with Claude Code.

**Why this priority**: Without installation, nothing else works. This is the entry point.

**Document Properties to Verify**:
- Quick Start section contains exact shell commands for installation
- MCP configuration JSON block is present and valid
- Verification command is documented

**Acceptance Scenarios**:

1. **Given** the Quick Start section, **When** I read it, **Then** I see: prerequisites list, clone command, install command, MCP config JSON, and verification command.
2. **Given** the MCP config JSON, **When** I validate it, **Then** it contains valid JSON with `mcpServers.zombiekit.command` and `mcpServers.zombiekit.args` keys.

---

### User Story 2 - Understanding Capabilities (Priority: P2)

A developer wants to know what ZombieKit can do for them before investing time.

**Why this priority**: Users need to understand value proposition before committing.

**Document Properties to Verify**:
- "What Is ZombieKit?" section exists
- Section contains 2-3 paragraphs
- Section covers: (1) core insight, (2) what it provides, (3) what it does NOT do

**Acceptance Scenarios**:

1. **Given** the overview section, **When** I read it, **Then** it explicitly states that Claude Code orchestrates while ZombieKit provides memory/storage.
2. **Given** the overview section, **When** I count paragraphs, **Then** there are 2-3 paragraphs (not a wall of text).

---

### User Story 3 - Workflow Understanding (Priority: P2)

A developer wants to know how to use ZombieKit for a typical feature development.

**Why this priority**: Core value is the workflow; users need to understand the cycle.

**Document Properties to Verify**:
- Workflow section contains ASCII diagram
- Diagram shows four phases: Research, Create, Audit, Highlight
- Feedback loop from Audit back to Research is shown

**Acceptance Scenarios**:

1. **Given** the workflow section, **When** I look for a diagram, **Then** there is ASCII art showing the four-phase cycle.
2. **Given** the example workflow section, **When** I read it, **Then** it lists numbered steps from init to complete.

---

### User Story 4 - Skill Reference (Priority: P3)

A developer working on a task wants to know which skill to use.

**Why this priority**: Once users are working, they need quick reference.

**Document Properties to Verify**:
- Skills table uses markdown table format
- Table has at least 8 rows (core skills)
- Each skill has a one-line description

**Acceptance Scenarios**:

1. **Given** the skills table, **When** I scan it, **Then** skills are listed in logical workflow order (start -> plan -> implement -> finish).
2. **Given** a skill in the table, **When** I read its description, **Then** the description is one line or less.

---

## Document Structure

### Section 1: Header

**File location:** Repository root as `README.md`

**Content:**
- Project name: ZombieKit (aka "brains")
- Tagline: "Feed your codebase some brains"
- One-sentence description: Prompt composition and artifact management for Claude Code

**Format:** No badges, clean minimal header

---

### Section 2: What Is ZombieKit?

**Content (exactly 3 short paragraphs):**

1. **Core insight paragraph:** "Claude Code is the brain, brains CLI is the memory." Explain that ZombieKit does NOT orchestrate AI - Claude Code does that.

2. **Capabilities paragraph:** What it provides - structured workflows for features/bugs/refactors, persistent memory across sessions, composable prompts via profiles.

3. **Integration paragraph:** How it works - MCP server that Claude Code connects to, slash commands invoke skills, artifacts stored in `history/` folder.

---

### Section 3: Quick Start

**Prerequisites (must include version verification commands):**
- Go 1.24+ (`go version`)
- Task from taskfile.dev (`task --version`)
- Claude Code (the AI coding assistant)

**Installation Steps:**
```bash
git clone https://github.com/morganhein/zombiekit.git
cd zombiekit
task install
```

**MCP Configuration:**
Location: Claude Code MCP settings (varies by platform)
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

**Project Initialization (one-time per project):**
```bash
brains init
```

**Verification (in Claude Code):**
```
/brains.status
```

---

### Section 4: The Workflow Cycle

**Visual (ASCII-7 only for compatibility):**
```
RESEARCH (parallel) --> CREATE (single) --> AUDIT (parallel) --> HIGHLIGHT
    ^                                             |
    |                                             |
    +---------------- (if issues) ----------------+
```

**Brief explanation (4 lines max):**
- Every stage follows this pattern
- Research: parallel agents explore codebase and domain
- Create: single agent synthesizes findings into artifact
- Audit checks completeness; Highlight presents to user for approval

---

### Section 5: Core Skills

**Note at top:** "These are the primary workflow skills. See DESIGN.md for the complete list of 17 skills."

**Format:** Markdown table, grouped by purpose

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

---

### Section 6: Example Workflow

**Scenario:** Adding a new feature to your project

**Explanation paragraph:** Brief note that `brains init` is a one-time terminal command per project. All `/brains.*` commands run in Claude Code.

**Commands in order:**
1. `brains init` (terminal, one-time per project)
2. `/brains.feature "add user authentication"` (Claude Code)
3. Review and approve spec
4. `/brains.plan`
5. Review and approve plan
6. `/brains.tasks`
7. `/brains.implement`
8. `/brains.complete`

---

### Section 7: Learn More

**Format:** Markdown links

**Links:**
- [Architecture and Design](docs/DESIGN.md) - Full architecture, all skills, configuration options
- [Skill Definitions](.claude/commands/) - Individual skill documentation

---

## Requirements

### Functional Requirements

- **FR-001**: README MUST be created at repository root as `README.md`
- **FR-002**: README MUST include installation instructions that work on macOS and Linux (Windows explicitly not supported)
- **FR-003**: README MUST include prerequisite verification commands (`go version`, `task --version`)
- **FR-004**: README MUST include valid MCP configuration JSON for Claude Code
- **FR-005**: README MUST explain the workflow cycle with ASCII-7 diagram (no Unicode arrows)
- **FR-006**: README MUST list core workflow skills (9 skills) in a table, with note pointing to DESIGN.md for full list
- **FR-007**: README MUST distinguish terminal commands (`brains init`) from Claude Code skills (`/brains.*`)
- **FR-008**: README MUST NOT include CI badges
- **FR-009**: README MUST NOT duplicate detailed content from DESIGN.md (summary and links only)
- **FR-010**: README MUST be under 200 lines
- **FR-011**: README MUST use the actual git remote URL: `https://github.com/morganhein/zombiekit.git`

---

## Success Criteria

- **SC-001**: README contains all 7 sections in order
- **SC-002**: README is under 200 lines when rendered
- **SC-003**: All code blocks have correct syntax highlighting labels (bash, json)
- **SC-004**: Skills table contains exactly 9 core skills

---

## Testing Requirements

**Testing Requirements: None - documentation-only change**

This is a README file with no executable code. Validation:
- Line count check: `wc -l README.md` should be < 200
- JSON validation: MCP config block should parse as valid JSON
- Link check: Referenced files (DESIGN.md, .claude/commands/) should exist

---

## Key Decisions to Highlight

1. **File location**: Repository root as `README.md`
2. **Quick Start before concepts**: Get users running first, explain later
3. **No badges**: Per user request, cleaner look
4. **ASCII-7 diagram**: No Unicode arrows, works everywhere
5. **9 core skills, not all 17**: README is quick reference; DESIGN.md has exhaustive list
6. **Terminal vs Claude Code distinction**: Explicitly called out in example workflow
7. **Git URL verified**: Using actual remote `https://github.com/morganhein/zombiekit.git`
