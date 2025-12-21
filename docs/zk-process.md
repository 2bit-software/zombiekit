# ZombieKit Design Document

> *"Feed your codebase some brains."*

A data-driven specification and implementation framework with iterative cycles at each stage.

## Overview

ZombieKit is a CLI tool that orchestrates AI agents through a structured workflow: from initial idea through specification, planning, task breakdown, and implementation. Unlike linear tools, ZombieKit emphasizes multiple audit/feedback cycles at each stage before progressing.

## Core Concepts

| Concept | Description |
|---------|-------------|
| **Brains** | The CLI command namespace (`/brains.*`) |
| **Artifacts** | Persistent outputs: specs, plans, task lists |
| **Cycles** | Iterative research → create → audit loops within each stage |
| **Agents** | Specialized AI sub-agents for research, creation, and auditing |

## Command Reference

| Command | Purpose | Primary Skill |
|---------|---------|---------------|
| `/brains.feature` | New feature specification | spec |
| `/brains.bug` | Bug fix specification | spec |
| `/brains.refactor` | Refactoring specification | spec |
| `/brains.plan` | Implementation planning | plan |
| `/brains.tasks` | Task breakdown | tasks |
| `/brains.implement` | Execute implementation | implement |
| `/brains.research` | Standalone research | research-orchestrator |
| `/brains.update` | Modify existing artifacts | spec (edit mode) |
| `/brains.clarify` | Surface ambiguities | highlights + auditor |
| `/brains.audit` | Cross-artifact alignment check | auditor |
| `/brains.eat` | 🧟 *TBD - something fun* | ??? |

---

## Primary Workflow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ZOMBIEKIT WORKFLOW                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   USER INPUT                                                                │
│       │                                                                     │
│       ▼                                                                     │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │  /brains.feature  OR  /brains.bug  OR  /brains.refactor           │     │
│   │                                                                   │     │
│   │   ┌─────────────────────────────────────────────────────────┐     │     │
│   │   │ 1. SORT: Business spec ←→ Technical spec                │     │     │
│   │   │    • Extract business requirements                      │     │     │
│   │   │    • Separate technical constraints                     │     │     │
│   │   │    • Save technical spec for /brains.plan               │     │     │
│   │   └─────────────────────────────────────────────────────────┘     │     │
│   │                          │                                        │     │
│   │                          ▼                                        │     │
│   │   ┌─────────────────────────────────────────────────────────┐     │     │
│   │   │ 2-4. SPEC CYCLE (repeats until clean)                   │     │     │
│   │   │                                                         │     │     │
│   │   │    ┌──────────┐    ┌──────────┐    ┌──────────┐         │     │     │
│   │   │    │ RESEARCH │───▶│  CREATE  │───▶│  AUDIT   │         │     │     │
│   │   │    │ (many)   │    │ (single) │    │ (many)   │         │     │     │
│   │   │    └──────────┘    └──────────┘    └────┬─────┘         │     │     │
│   │   │         ▲                               │               │     │     │
│   │   │         └───── CRITICAL/MAJOR? ─────────┘               │     │     │
│   │   │                     YES                                 │     │     │
│   │   └─────────────────────────────────────────────────────────┘     │     │
│   │                          │ NO                                     │     │
│   │                          ▼                                        │     │
│   │   ┌─────────────────────────────────────────────────────────┐     │     │
│   │   │ 5. HIGHLIGHT to user                                    │     │     │
│   │   │    • Surface key decisions                              │     │     │
│   │   │    • Await approval or feedback                         │     │     │
│   │   └─────────────────────────────────────────────────────────┘     │     │
│   │                                                                   │     │
│   │   OUTPUT: spec.md + technical-spec.md                             │     │
│   └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
│       │                                                                     │
│       ▼                                                                     │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │  /brains.plan                                                     │     │
│   │                                                                   │     │
│   │   ┌─────────────────────────────────────────────────────────┐     │     │
│   │   │ 1. LOAD technical spec from previous stage              │     │     │
│   │   └─────────────────────────────────────────────────────────┘     │     │
│   │                          │                                        │     │
│   │                          ▼                                        │     │
│   │   ┌─────────────────────────────────────────────────────────┐     │     │
│   │   │ 2-4. PLAN CYCLE (repeats until clean)                   │     │     │
│   │   │                                                         │     │     │
│   │   │    ┌──────────┐    ┌──────────┐    ┌──────────┐         │     │     │
│   │   │    │ RESEARCH │───▶│  CREATE  │───▶│  AUDIT   │         │     │     │
│   │   │    │ (many)   │    │ (single) │    │ (many)   │         │     │     │
│   │   │    └──────────┘    └──────────┘    └────┬─────┘         │     │     │
│   │   │         ▲                               │               │     │     │
│   │   │         └───── CRITICAL/MAJOR? ─────────┘               │     │     │
│   │   └─────────────────────────────────────────────────────────┘     │     │
│   │                          │ NO                                     │     │
│   │                          ▼                                        │     │
│   │   ┌─────────────────────────────────────────────────────────┐     │     │
│   │   │ 5. HIGHLIGHT to user                                    │     │     │
│   │   └─────────────────────────────────────────────────────────┘     │     │
│   │                                                                   │     │
│   │   OUTPUT: plan.md                                                 │     │
│   └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
│       │                                                                     │
│       ▼                                                                     │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │  /brains.tasks                                                    │     │
│   │                                                                   │     │
│   │   1. Load plan.md                                                 │     │
│   │   2. Break user stories into independent sub-components           │     │
│   │   3. Ensure each task can start from fresh context                │     │
│   │   4. Output task graph with no circular dependencies              │     │
│   │                                                                   │     │
│   │   OUTPUT: tasks.md (independent, parallelizable tasks)            │     │
│   └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
│       │                                                                     │
│       ▼                                                                     │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │  /brains.implement                                                │     │
│   │                                                                   │     │
│   │   FOR EACH independent task/user story:                           │     │
│   │       • Start fresh context                                       │     │
│   │       • Load task + relevant spec sections                        │     │
│   │       • Implement using specified agents                          │     │
│   │       • Verify against acceptance criteria                        │     │
│   │                                                                   │     │
│   │   OUTPUT: Working code                                            │     │
│   └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Supporting Commands

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SUPPORTING COMMANDS                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   /brains.research                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐       │
│   │  Standalone iterative research                                  │       │
│   │                                                                 │       │
│   │  • Can be called independently or as part of other commands     │       │
│   │  • Delegates to multiple research agents                        │       │
│   │  • Collates, deduplicates, organizes results                    │       │
│   │  • Preserves sources for all findings                           │       │
│   │                                                                 │       │
│   │  OUTPUT: research.md                                            │       │
│   └─────────────────────────────────────────────────────────────────┘       │
│                                                                             │
│   /brains.update  (aliases: .alter, .change, .modify)                       │
│   ┌─────────────────────────────────────────────────────────────────┐       │
│   │  Modify existing artifacts                                      │       │
│   │                                                                 │       │
│   │  • Load existing spec/plan/tasks                                │       │
│   │  • Apply user's requested changes                               │       │
│   │  • Re-run audit cycle                                           │       │
│   │  • Highlight changes for approval                               │       │
│   │                                                                 │       │
│   │  OUTPUT: Updated artifact                                       │       │
│   └─────────────────────────────────────────────────────────────────┘       │
│                                                                             │
│   /brains.clarify                                                           │
│   ┌─────────────────────────────────────────────────────────────────┐       │
│   │  Surface ambiguities and questions                              │       │
│   │                                                                 │       │
│   │  • Audit spec and/or plan                                       │       │
│   │  • Identify unclear areas                                       │       │
│   │  • Generate clarifying questions                                │       │
│   │  • Highlight items needing user input                           │       │
│   │                                                                 │       │
│   │  OUTPUT: Clarification requests                                 │       │
│   └─────────────────────────────────────────────────────────────────┘       │
│                                                                             │
│   /brains.audit                                                             │
│   ┌─────────────────────────────────────────────────────────────────┐       │
│   │  Cross-artifact alignment check                                 │       │
│   │                                                                 │       │
│   │  • Verify spec ↔ plan alignment                                 │       │
│   │  • Verify plan ↔ tasks alignment                                │       │
│   │  • Check for drift or inconsistencies                           │       │
│   │  • Report misalignments by severity                             │       │
│   │                                                                 │       │
│   │  OUTPUT: Alignment report                                       │       │
│   └─────────────────────────────────────────────────────────────────┘       │
│                                                                             │
│   /brains.eat                                                               │
│   ┌─────────────────────────────────────────────────────────────────┐       │
│   │  🧟 BRAAAAINS                                                   │       │
│   │                                                                 │       │
│   │  Ideas:                                                         │       │
│   │  • Consume/archive completed specs                              │       │
│   │  • Learn from past projects (feed the zombie)                   │       │
│   │  • Digest external documentation into internal knowledge        │       │
│   │  • Easter egg / fun mode                                        │       │
│   │                                                                 │       │
│   │  OUTPUT: 🧠🧠🧠                                                  │       │
│   └─────────────────────────────────────────────────────────────────┘       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Agent Mapping

| Stage | Agent Type | Skills/Agents Used |
|-------|------------|-------------------|
| Research | Many (parallel) | `@agent-research` → `research-orchestrator` |
| Spec Creation | Single | `spec-creator` |
| Plan Creation | Single | `plan-creator` (TBD) |
| Task Creation | Single | `task-creator` (TBD) |
| Audit | Many (parallel) | `@agent-audit` → `auditor` → `spec-auditor` + `ai-synergy-auditor` |
| Highlight | Single | `highlights` |
| Implementation | Per-task | Specified agents per task type |

---

## Data Flow

```
USER INPUT
    │
    ▼
┌─────────────────┐
│  Business Spec  │──────────────────────────────────┐
│  (user-facing)  │                                  │
└────────┬────────┘                                  │
         │                                           │
         ▼                                           ▼
┌─────────────────┐                         ┌───────────────┐
│ Technical Spec  │────────────────────────▶│    plan.md    │
│ (implementation)│                         │               │
└─────────────────┘                         └───────┬───────┘
                                                    │
                                                    ▼
                                            ┌───────────────┐
                                            │   tasks.md    │
                                            │ (independent) │
                                            └───────┬───────┘
                                                    │
                                          ┌─────────┼─────────┐
                                          ▼         ▼         ▼
                                      ┌──────┐  ┌──────┐  ┌──────┐
                                      │Task 1│  │Task 2│  │Task N│
                                      └──┬───┘  └──┬───┘  └──┬───┘
                                         │         │         │
                                         ▼         ▼         ▼
                                      ┌─────────────────────────┐
                                      │     WORKING CODE        │
                                      └─────────────────────────┘
```

---

## Artifact Structure

```
project/
├── .zombiekit/
│   ├── config.yml           # Project configuration
│   └── filters.md           # Custom highlight filters
│
├── specs/
│   └── {feature-name}/
│       ├── spec.md          # Business specification
│       ├── technical.md     # Technical specification
│       ├── research.md      # Research findings
│       ├── plan.md          # Implementation plan
│       ├── tasks.md         # Task breakdown
│       └── audit/
│           └── {date}.md    # Audit reports
│
└── src/                     # Implementation output
```

---

## Cycle Philosophy

Each major stage follows the same pattern:

```
     ┌──────────────────────────────────────────┐
     │           ZOMBIEKIT CYCLE                │
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
     │   CRITICAL/MAJOR? ──YES──▶ Loop back     │
     │       │                                  │
     │       NO                                 │
     │       │                                  │
     │       ▼                                  │
     │   ┌────────┐                             │
     │   │HIGHLIGH│  Surface decisions          │
     │   │   T    │  Await user approval        │
     │   └───┬────┘                             │
     │       │                                  │
     │       ▼                                  │
     │   USER APPROVED? ──NO──▶ Loop back       │
     │       │                                  │
     │       YES                                │
     │       │                                  │
     │       ▼                                  │
     │   NEXT STAGE                             │
     └──────────────────────────────────────────┘
```

**Key Principle:** No stage completes until:
1. No CRITICAL or MAJOR audit issues remain
2. User has reviewed highlights and approved

---

## Next Steps

1. **Create missing skills:** `plan-creator`, `task-creator`
2. **Define CLI interface:** Argument parsing, config loading
3. **Build artifact persistence:** File I/O, state management
4. **Implement `/brains.eat`:** Whatever that ends up being 🧟