# Feature: Enhanced Initiative Lifecycle with Linear Integration

## Problem Statement

The initiative lifecycle has gaps in tracking and completion:

1. **On creation**: When starting work from a Linear ticket, the connection isn't recorded
2. **On completion**: Users forget to commit changes and update the originating ticket

This creates gaps where work is logically "done" but not properly recorded or traced back to its source.

## User Stories

### US1: Ticket Capture on Creation
As a developer starting an initiative from a Linear ticket,
I want the ticket reference to be automatically recorded in INITIATIVE.md,
So that the connection between the initiative and ticket is preserved for later use.

### US2: Commit Offer on Completion
As a developer completing an initiative,
I want to be offered the option to commit my changes,
So that I have the opportunity to preserve my work in version control before the initiative closes.

### US3: Linear Ticket Update on Completion
As a developer completing an initiative that originated from a Linear ticket,
I want the ticket to be automatically marked as done with a work summary,
So that the ticket reflects the completed work without manual updates.

---

## Feature 1: Ticket Capture on Creation

### Acceptance Criteria

1. **During `/brains.new`**: After classifying the work type, check if user input mentions a Linear ticket
2. **If ticket identifier found** (e.g., "DEV-101", "work on PROJ-42"):
   - Fetch ticket details via `mcp__linear-server__get_issue`
   - After initiative is created, update INITIATIVE.md to add a "Source" section
3. **If no ticket identifier**: Proceed normally without Source section
4. **If Linear MCP unavailable**: Proceed normally, skip ticket lookup

### INITIATIVE.md Update

Add a "Source" section after the ID line:

```markdown
# Initiative: dev-101-complete-with-commit

**Type**: feature
**Status**: active
**Created**: 2026-01-31T13:22:55-08:00
**ID**: 697e72af-feature-dev-101-complete-with-commit

## Source

**Linear Ticket**: [DEV-101](https://linear.app/heinsight/issue/DEV-101/...)
**Title**: Have the /brains.complete command also offer to write a commit

## Description
...
```

### Implementation Location

Modify `embed/workflows/new.md` to:
1. Parse user input for ticket pattern `[A-Z]+-[0-9]+`
2. If found, fetch ticket via Linear MCP before loading profile
3. Pass ticket info to the profile (via arguments or store in context)

Modify `embed/profiles/feature.md` (and bug.md, refactor.md) to:
1. After initiative creation, check if ticket info was captured
2. If yes, use Edit tool to add Source section to INITIATIVE.md

### Edge Cases

- **Ticket not found**: Display warning, proceed without Source section
- **Multiple ticket identifiers**: Use the first match
- **Linear MCP unavailable**: Skip ticket lookup silently

---

## Feature 2: Commit Offer on Completion

### Acceptance Criteria

1. **After displaying completion summary**: Before calling `mcp__zombiekit__initiative` with action `complete`, check for uncommitted changes
2. **If uncommitted changes exist**: Use `AskUserQuestion` tool to offer the commit choice
3. **If user accepts**:
   - Stage all changes with `git add -A`
   - Invoke the `commit-message` skill via the `Skill` tool
4. **If user declines or doesn't respond**: Proceed with completion
5. **If no uncommitted changes**: Skip the offer silently

### Change Detection

```bash
git status --porcelain
# Empty = no changes, skip offer
# Non-empty = changes exist, show offer
```

### User Experience

```
/brains.complete

[Summary of work items...]

**Uncommitted changes detected:**
- 3 files modified
- 1 file added

[AskUserQuestion: "Would you like to commit your changes?"]
> Yes, commit changes

[Skill: commit-message invoked]
[Initiative marked complete]
```

### Edge Cases

- **Not a git repository**: Skip commit offer silently
- **Commit fails**: Display error, proceed with completion
- **Staged + unstaged changes**: Stage all with `git add -A`

---

## Feature 3: Linear Ticket Update on Completion

### Acceptance Criteria

1. **After commit offer**: Check INITIATIVE.md for Source section with Linear ticket
2. **If ticket found**: Use `AskUserQuestion` to offer updating the ticket
3. **If user accepts**:
   - Generate work summary from completion summary
   - Post summary as comment via `mcp__linear-server__create_comment`
   - Update ticket status to "Done" via `mcp__linear-server__update_issue`
4. **If user declines**: Proceed with completion
5. **If no ticket in Source**: Skip offer silently

### User Experience

```
/brains.complete

[Summary...]
[Commit offer handled...]

**Source ticket found: DEV-101**

[AskUserQuestion: "Update Linear ticket DEV-101?"]
> Yes, update ticket

[Comment posted to DEV-101]
[Ticket status updated to Done]
[Initiative marked complete]
```

### Edge Cases

- **No Source section**: Parse initiative name as fallback (pattern `[A-Z]+-[0-9]+`)
- **Ticket already done**: Post comment, skip status update
- **Linear API fails**: Display error, proceed with completion
- **Linear MCP unavailable**: Skip offer silently

---

## Tool Integration

### AskUserQuestion - Commit Offer

```yaml
questions:
  - question: "Would you like to commit your changes before completing?"
    header: "Commit"
    multiSelect: false
    options:
      - label: "Yes, commit changes"
        description: "Generate a commit message and commit all changes"
      - label: "No, skip commit"
        description: "Complete without committing"
```

### AskUserQuestion - Linear Update Offer

```yaml
questions:
  - question: "Would you like to update Linear ticket {TICKET-ID}?"
    header: "Linear"
    multiSelect: false
    options:
      - label: "Yes, update ticket"
        description: "Post summary and mark as Done"
      - label: "No, skip update"
        description: "Complete without updating ticket"
```

### Linear MCP Tools

```
mcp__linear-server__get_issue     - Fetch ticket details on creation
mcp__linear-server__create_comment - Post work summary on completion
mcp__linear-server__update_issue   - Set status to Done on completion
```

---

## Files to Modify

| File | Feature | Changes |
|------|---------|---------|
| `embed/workflows/new.md` | F1 | Add ticket detection and lookup before profile load |
| `embed/profiles/feature.md` | F1 | Add Source section to INITIATIVE.md after creation |
| `embed/profiles/bug.md` | F1 | Same as feature.md |
| `embed/profiles/refactor.md` | F1 | Same as feature.md |
| `embed/workflows/complete.md` | F2, F3 | Add commit offer and Linear update steps |
| `embed/profiles/complete.md` | F2, F3 | Keep in sync with workflow |

---

## Constraints

- Must not block completion on any failure (commit, Linear API, etc.)
- Must gracefully handle missing Linear MCP
- Must use existing `commit-message` skill
- No new Go code required (workflow-only implementation)
- Timeout/cancellation: Proceed without action if user doesn't respond
