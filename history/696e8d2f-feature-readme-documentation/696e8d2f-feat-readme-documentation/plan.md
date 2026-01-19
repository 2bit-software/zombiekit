---
status: draft
type: plan
name: readme-documentation
created: 2026-01-19
spec: spec.md
---

# Implementation Plan: README Documentation

## Overview

Create a user-focused README.md at the repository root following the approved specification.

**Complexity:** Low - single markdown file, no code, no dependencies
**Spikes Required:** None - documentation only
**Estimated Tasks:** 1 (write the file) + 1 (validation)

---

## Technical Context

- **Output file:** `/Users/morgan/Projects/personal/zombiekit/README.md`
- **Format:** GitHub-flavored Markdown
- **Constraints:** < 200 lines, ASCII-7 only for diagrams, no badges

---

## Implementation Phases

### Phase 1: Create README.md

**Single task:** Write the complete README.md file following spec sections 1-7.

**Section-by-section content:**

#### Section 1: Header
```markdown
# ZombieKit

> "Feed your codebase some brains."

Prompt composition and artifact management for Claude Code.
```

#### Section 2: What Is ZombieKit?
Three paragraphs covering:
1. Core insight - Claude Code orchestrates, ZombieKit provides memory
2. Capabilities - workflows, persistent memory, composable prompts
3. Integration - MCP server, slash commands, history folder

#### Section 3: Quick Start
- Prerequisites with version check commands
- Clone/install commands (using verified URL)
- MCP configuration JSON block
- Project init command
- Verification command

#### Section 4: The Workflow Cycle
- ASCII-7 diagram showing Research → Create → Audit → Highlight
- 4-line explanation of each phase

#### Section 5: Core Skills
- Note about 17 total skills, showing 9 core ones
- Table grouped by purpose (Starting Work, Planning, Implementing, Tracking)

#### Section 6: Example Workflow
- Explanation of terminal vs Claude Code distinction
- Numbered steps from init to complete

#### Section 7: Learn More
- Markdown links to DESIGN.md and .claude/commands/

---

### Phase 2: Validation

**Validation checks:**
1. Line count: `wc -l README.md` < 200
2. JSON validation: MCP config block parses correctly
3. Link check: DESIGN.md and .claude/commands/ exist
4. Section count: All 7 sections present
5. Skills count: Exactly 9 skills in table

---

## Requirement Traceability

| Requirement | Implementation |
|-------------|----------------|
| FR-001 | File created at repository root |
| FR-002 | Prerequisites section with macOS/Linux note |
| FR-003 | Version check commands in prerequisites |
| FR-004 | MCP JSON block in Quick Start |
| FR-005 | ASCII-7 diagram in Workflow Cycle |
| FR-006 | 9 skills in table with DESIGN.md note |
| FR-007 | Terminal/Claude Code distinction in Example Workflow |
| FR-008 | No badges in Header |
| FR-009 | Links to DESIGN.md, not duplication |
| FR-010 | < 200 lines total |
| FR-011 | URL: https://github.com/morganhein/zombiekit.git |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Line count exceeds 200 | Low | Low | Trim explanations, keep concise |
| ASCII diagram renders poorly | Low | Low | Test in GitHub preview |

---

## Dependencies

None. This is a standalone documentation file.

---

## Not Included (Deferred)

- CI badges (per user request)
- Full skill documentation (lives in DESIGN.md)
- Contributor guide (separate concern)
- Windows support documentation

---

## Suggested Next Step

After plan approval: `/brains.tasks` to generate task breakdown.
