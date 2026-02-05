# Feature Specification: Realtime Task Markdown Updates

**Feature Branch**: `69839ae9-feature-realtime-task-markdown-updates`
**Created**: 2026-02-04
**Status**: Draft
**Linear Ticket**: [DEV-84](https://linear.app/heinsight/issue/DEV-84/while-performing-implement-the-tasks-markdown-file-should-be-updated)

## Summary

Update the `/implement` (eat) step profile to explicitly instruct the agent to:
1. Use Claude's built-in task tools (TaskCreate, TaskUpdate, TaskList) for session tracking
2. Keep `tasks.md` in sync by marking tasks complete as they finish

This is a profile/documentation change, not a code change.

## User Scenarios & Testing

### User Story 1 - Dual Task Tracking (Priority: P1)

As a developer using the `/implement` workflow, I want the agent to track progress using BOTH Claude's task tools AND the tasks.md file, so that progress is visible in the UI during the session AND persisted in the markdown file for later review.

**Why this priority**: Core functionality - without explicit instructions, agents may use only one tracking method.

**Independent Test**: Manual verification by running `/implement` and observing that both the UI task list and tasks.md are updated.

**Acceptance Scenarios**:

1. **Given** an implement session starts, **When** agent reads tasks.md, **Then** agent creates corresponding TaskCreate entries for tracking
2. **Given** agent completes a task, **When** marking it done, **Then** agent updates BOTH TaskUpdate AND edits tasks.md
3. **Given** tasks.md has `- [ ] T005 ...`, **When** task completes, **Then** tasks.md shows `- [x] T005 ...`

## Requirements

### Functional Requirements

- **FR-001**: Eat step profile MUST instruct agent to use TaskCreate for incomplete tasks
- **FR-002**: Eat step profile MUST instruct agent to call TaskUpdate when task completes
- **FR-003**: Eat step profile MUST instruct agent to edit tasks.md when task completes
- **FR-004**: Eat step profile MUST explain why both are needed (session vs persistent tracking)

## Success Criteria

- **SC-001**: eat.md profile contains explicit dual-tracking instructions
- **SC-002**: Agent following the profile updates both task tools and tasks.md

## Testing Requirements

Testing Requirements: None - this is a profile/documentation change. Verification is manual observation that agents follow the updated instructions.

## Changes Made

Updated `embed/steps/eat.md`:
1. Added "Track progress using Claude's built-in task tools" to responsibilities
2. Added "Keep tasks.md in sync" to responsibilities
3. Updated Step 1 to include task initialization with TaskCreate
4. Updated Step 3 to show BOTH tracking methods are required
5. Added "Dual Tracking Required" to behavior rules
6. Updated success criteria to include both tracking methods
