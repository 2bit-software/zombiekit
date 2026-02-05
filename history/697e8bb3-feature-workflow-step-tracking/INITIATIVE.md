# Initiative: workflow-step-tracking

**Type**: feature
**Status**: active
**Created**: 2026-01-31T15:09:39-08:00
**ID**: 697e8bb3-feature-workflow-step-tracking

## Source

**Linear Ticket**: [DEV-103](https://linear.app/heinsight/issue/DEV-103/we-should-have-the-workflow-define-the-steps)
**Title**: we should have the workflow define the steps

## Description

Make workflows define their steps upfront in INITIATIVE.md with visible status indicators. When an initiative is created, the user immediately sees all workflow phases listed. As work progresses through each phase, the status updates to show current position (pending → in-progress → complete). This provides a clear roadmap and progress tracking for any initiative.

## Goals

- [x] Add "Workflow Steps" section to INITIATIVE.md at creation time
- [x] Define step lists per workflow type (feature, bug, refactor)
- [x] Add profile instructions to update step status at phase transitions
- [x] Add status field to active.json (in-progress/complete)

## Progress

- 2026-01-31: Initiative created, research and spec written
- 2026-01-31: Implementation complete (T001-T024). Core functionality done:
  - InitiativeState simplified to minimal pointer (initiative, started, status)
  - INITIATIVE.md parser with cycle/step table support
  - Workflow profiles have steps: frontmatter
  - createInitiativeMD generates Cycles section with step table
  - Status() parses INITIATIVE.md for cycle/step info
  - UpdateState() writes step status to INITIATIVE.md
  - next.md workflow rewritten for INITIATIVE.md-based state
- 2026-01-31: All tests complete (T025-T027):
  - T025: Added TestService_Status tests for new Status() behavior
  - T026: Created markdown_test.go with comprehensive parser tests
  - T027: Fixed step/service_test.go for new path behavior (no separate cycle folder)
