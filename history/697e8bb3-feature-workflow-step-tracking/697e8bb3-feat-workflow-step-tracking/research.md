---
status: complete
updated: 2026-01-31
---

# Research: Workflow Step Tracking

## Executive Summary

The current ZombieKit workflow system lacks explicit step tracking in INITIATIVE.md. Workflows define phases in code (`internal/step/feature.go`) and step definitions (`embed/steps/*.md`), but the initiative file doesn't surface these to users. This feature adds visible step tracking to INITIATIVE.md with status indicators that update as work progresses.

## Findings

### Codebase Context

**Current Initiative Structure:**
- `active.json` points to initiative path, has no status field (just `initiative`, `started`, `last_activity`)
- `INITIATIVE.md` has: Description, Goals, Progress sections (freeform)
- Progress section is unstructured text - no machine-readable status
- No step listing at creation time

**Workflow Phase Definitions:**
- `internal/step/feature.go` defines phases: research → create → audit → highlight
- `internal/step/service.go` has `stepPrerequisites` map for plan, tasks, eat steps
- Step definitions in `embed/steps/*.md` have frontmatter with profiles, files
- Workflow profiles in `embed/profiles/*.md` describe execution steps as prose

**State Management:**
- `internal/initiative/service.go` handles create/complete/status
- `createInitiativeMD()` generates template with static sections
- State updates via `stateManager.Save()` - only tracks `CurrentStep` string

**Existing Progress Patterns:**
- DEV-101 initiative shows Goals as checkboxes: `- [x] Add ticket detection...`
- Completion section added manually with outcomes list
- No standard format for in-progress tracking

### Domain Knowledge

**Workflow Visualization Patterns:**
- GitHub Actions: Shows stages with ✓/⊘/○ icons
- Linear: Workflow columns with card movement
- Make/Taskfile: Target dependencies shown in DAG

**Progress Indicator Standards:**
- Checkbox markdown: `- [ ]` pending, `- [x]` complete
- Status badges: `![status](badge-url)` for visual indicators
- Emoji status: ⬜ pending, 🔄 in-progress, ✅ complete, ⏭️ skipped

**Structured Status in Markdown:**
- YAML frontmatter for machine-readable metadata
- Table format for multi-step status display
- Section-based status (common in RFC documents)

## Decision Points

- [x] **D1**: Where to define workflow steps - In workflow definition files (embed/workflows/*.md and embed/profiles/*.md)
- [x] **D2**: Status indicator format - Use emoji + checkbox hybrid: ⬜ pending, 🔄 in-progress, ✅ complete
- [x] **D3**: When to update status - Agent updates after completing each step (profile responsibility)
- [x] **D4**: active.json changes - Add `status: "in-progress" | "complete"` field

## Recommendations

1. **Add Workflow Steps table to INITIATIVE.md** - Show all steps upfront at creation time with pending status
2. **Use profile instructions to update status** - Agent reads profile, updates INITIATIVE.md at phase transitions
3. **Minimal code changes** - Most changes in templates and profiles, not Go code
4. **Status field in active.json** - Simple string to indicate overall initiative state

## Sources

- `internal/initiative/service.go:386-410` - createInitiativeMD implementation
- `internal/step/feature.go:1-35` - buildWorkflowPhases definition
- `internal/step/types.go:108-134` - Phase and StepPrerequisite structs
- `embed/steps/feature.md` - Feature step workflow definition
- `embed/profiles/feature.md` - Feature profile with execution steps
- `embed/workflows/complete.md` - Complete workflow with status updates
- `history/697e72af-feature-dev-101-complete-with-commit/INITIATIVE.md` - Example of completed initiative
