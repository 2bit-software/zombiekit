---
status: complete
updated: 2026-01-19
---

# Research: Simplified Command Structure

## Executive Summary

ZombieKit's current architecture separates MCP tools (`initiative`, `step`) from Claude Code skills, requiring users to know 10+ commands. The codebase supports consolidation into a unified `workflow` tool with 5 user-facing commands. Intent detection should run in skill profiles (Claude side) while MCP tools remain deterministic.

## Findings

### Codebase Context

**Current Tool Architecture:**
- `internal/mcp/tools/initiative/tool.go` - Initiative lifecycle (create/status/complete/list)
- `internal/mcp/tools/step/tool.go` - Step execution (feature/bug/refactor/plan/tasks/eat)
- `internal/mcp/server.go` - Tool registration via `registerTools()`
- `internal/config/tools.go` - Tool enable/disable configuration

**Step Service Flow:**
1. Load step definition from local → global → embedded
2. Check prerequisites (spec.md approved before plan, etc.)
3. Resolve file patterns to actual paths
4. Compose profiles
5. Return StepResponse with directive and context

**Initiative Service Flow:**
1. Create initiative folder in `history/`
2. Set active state in `.brains/active.json`
3. Create cycle subfolder
4. Copy templates (spec.md, research.md)

**Key Files:**
| File | Purpose |
|------|---------|
| `internal/step/service.go` | Step execution with prerequisites |
| `internal/step/loader.go` | Three-tier step loading |
| `internal/initiative/service.go` | Initiative CRUD |
| `internal/initiative/state.go` | Active state management |
| `internal/initiative/types.go` | Data structures |

**Existing Patterns to Preserve:**
- Idempotent creation (`FindActiveByNameAndType`)
- Prerequisite validation (`stepPrerequisites` map)
- Template copying with skip-if-exists
- Profile composition per step

### Domain Knowledge

**Intent Detection Patterns:**
- Keyword-based classification effective for structured domains
- Confidence thresholds (0.7-0.8) prevent misrouting
- Always provide explicit override option
- Disambiguation prompts should offer 2-4 choices max

**Workflow State Machines:**
- Forward transitions: require prerequisites
- Backward transitions: preserve artifacts
- Track step history for audit trail
- Define alternatives explicitly in transitions

**Similar Tools:**
- GitHub CLI: unified `gh` with subcommands
- Terraform: `plan`/`apply`/`destroy` workflow
- Git: unified interface with subcommands

## Decision Points

- [x] **D1**: Single unified tool vs. keep separate tools
  - **Decision**: Unified `workflow` tool with actions
  - **Rationale**: Cleaner API, easier maintenance, better UX

- [x] **D2**: Intent detection location (MCP vs. skill profile)
  - **Decision**: Skill profile (Claude side)
  - **Rationale**: MCP tools should be deterministic; Claude has LLM

- [x] **D3**: Registry caching strategy
  - **Decision**: No cache, embedded fallback
  - **Rationale**: Always fresh data, fallback handles failures

- [x] **D4**: Sub-task structure
  - **Decision**: Subfolder within initiative
  - **Rationale**: Clear hierarchy, easy navigation

- [ ] **D5**: Migration strategy (immediate vs. deprecation period)
  - **Options**: A) Immediate removal, B) 2-release deprecation
  - **Recommendation**: B - deprecation with warnings

## Recommendations

### 1. Create Unified Workflow Tool
Consolidate `initiative` and `step` tools into single `workflow` tool with actions: `new`, `step`, `next`, `complete`, `help`.

**Rationale:** Single registration, consistent API, easier to understand.

### 2. Intent Detection in Skill Profiles
Implement keyword-based classification in the `/brains.new` skill profile, NOT in MCP tool.

**Rationale:** Leverages Claude's LLM capabilities, keeps MCP tools deterministic.

### 3. Registry-Driven Navigation
Create workflow registry returning available steps and valid transitions per workflow type.

**Rationale:** Enables validation, supports `/brains.help` context, future extensibility.

### 4. Phased Migration
Deprecate old skills with warnings, remove after 2 release cycles.

**Rationale:** Avoids breaking existing users, provides learning opportunity.

### 5. Sub-tasks as Nested Initiatives
Create subfolder structure for sub-tasks within parent initiative.

**Rationale:** Clear hierarchy, preserves parent context, easy to navigate.

## Sources

- Codebase analysis: `internal/mcp/tools/`, `internal/step/`, `internal/initiative/`
- Linear ticket DEV-83 with business specification
- GitHub CLI design patterns
- Industry workflow engine patterns
