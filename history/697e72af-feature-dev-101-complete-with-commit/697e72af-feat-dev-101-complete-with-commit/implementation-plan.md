# Implementation Plan: Enhanced Initiative Lifecycle with Linear Integration

## Overview

Three interconnected features to improve initiative tracking:

1. **F1: Ticket Capture** - Record Linear ticket in INITIATIVE.md when starting from a ticket
2. **F2: Commit Offer** - Offer to commit changes when completing an initiative
3. **F3: Linear Update** - Offer to update the source ticket when completing

All features are workflow-only changes (no Go code).

## Implementation Strategy

**Workflow/profile modifications only** - leverage existing tools (Edit, AskUserQuestion, Skill, Linear MCP).

---

## Feature 1: Ticket Capture on Creation

### Files to Modify

1. `embed/workflows/new.md` - Detect ticket and fetch details
2. `embed/profiles/feature.md` - Add Source section to INITIATIVE.md
3. `embed/profiles/bug.md` - Same as feature.md
4. `embed/profiles/refactor.md` - Same as feature.md

### Step 1: Modify `embed/workflows/new.md`

Add ticket detection **after** classification, **before** loading profile:

```markdown
### Ticket Detection (after classification)

Before loading the profile:
1. Parse user input for Linear ticket pattern: `[A-Z]+-[0-9]+` (case-insensitive)
2. If found:
   - Extract ticket identifier (e.g., "DEV-101")
   - Attempt to fetch ticket via `mcp__linear-server__get_issue`
   - If successful: Store ticket ID, URL, and title
   - If fails: Display warning, proceed without ticket
3. Pass ticket info to profile via enriched arguments:
   - Original: "work on DEV-101 feature"
   - Enriched: "work on DEV-101 feature\n\n---\nLINEAR_TICKET: DEV-101\nLINEAR_URL: https://...\nLINEAR_TITLE: ..."
```

### Step 2: Modify feature/bug/refactor profiles

Add step after "Initiative Check" to update INITIATIVE.md:

```markdown
1.5 **Add Source Section** (if ticket info present)
   - Parse arguments for LINER_TICKET/LINEAR_URL/LINEAR_TITLE
   - If present, use Edit tool to add Source section to INITIATIVE.md:
     - Find line with "## Description"
     - Insert before it:
       ```
       ## Source

       **Linear Ticket**: [TICKET-ID](URL)
       **Title**: Title text

       ```
```

---

## Feature 2: Commit Offer on Completion

### Files to Modify

1. `embed/workflows/complete.md` - Add commit offer step
2. `embed/profiles/complete.md` - Keep in sync

### Step: Modify complete workflow

Insert **step 5** between "Update INITIATIVE.md" and "Clear Active State":

```markdown
5. **Offer Commit** (if in git repository)
   - Run `git status --porcelain` via Bash tool
   - If command fails (not a git repo): Skip to step 6
   - If output is empty (no changes): Skip to step 6
   - If output is non-empty (changes detected):
     a. Parse output to summarize changes (modified/added/deleted counts)
     b. Display summary: "You have uncommitted changes: X modified, Y added"
     c. Use `AskUserQuestion` tool:
        ```
        question: "Would you like to commit your changes before completing?"
        header: "Commit"
        options:
          - label: "Yes, commit changes"
          - label: "No, skip commit"
        ```
     d. If user selects "Yes":
        - Run `git add -A` to stage all changes
        - Use `Skill` tool with `skill: "commit-message"`
        - If commit fails: Display error, proceed to step 6
     e. If user selects "No" or "Other": Proceed to step 6
```

---

## Feature 3: Linear Ticket Update on Completion

### Files to Modify

Same as F2 (extend the complete workflow/profile)

### Step: Modify complete workflow

Insert **step 6** (after commit offer, before clear active state):

```markdown
6. **Offer Linear Update** (if source ticket exists)
   - Read INITIATIVE.md and check for Source section with Linear ticket
   - If no Source section: Try parsing initiative name for `[A-Z]+-[0-9]+` pattern
   - If no ticket found: Skip to step 7
   - If ticket found:
     a. Display: "Source ticket found: {TICKET-ID}"
     b. Use `AskUserQuestion` tool:
        ```
        question: "Would you like to update Linear ticket {TICKET-ID}?"
        header: "Linear"
        options:
          - label: "Yes, update ticket"
          - label: "No, skip update"
        ```
     c. If user selects "Yes":
        - Generate work summary from completion summary (outcomes, duration)
        - Post comment via `mcp__linear-server__create_comment`:
          ```
          issueId: {ticket-id}
          body: "## Work Completed\n\n{summary}\n\nCompleted via ZombieKit initiative: {initiative-name}"
          ```
        - Update status via `mcp__linear-server__update_issue`:
          ```
          id: {ticket-id}
          state: "Done"
          ```
        - If API fails: Display error, proceed to step 7
     d. If user selects "No" or "Other": Proceed to step 7
```

---

## Updated Complete Workflow Structure

**Before**:
```
1. Load Active Initiative
2. Completion Check
3. Generate Summary
4. Update INITIATIVE.md
5. Clear Active State
6. Report Completion
```

**After**:
```
1. Load Active Initiative
2. Completion Check
3. Generate Summary
4. Update INITIATIVE.md
5. Offer Commit          <-- NEW (F2)
6. Offer Linear Update   <-- NEW (F3)
7. Clear Active State
8. Report Completion
```

---

## Dependencies

All tools already exist:

| Tool | Used For |
|------|----------|
| `Bash` | `git status --porcelain`, `git add -A` |
| `AskUserQuestion` | Commit and Linear update offers |
| `Skill` | Invoke `commit-message` |
| `Edit` | Add Source section to INITIATIVE.md |
| `Read` | Read INITIATIVE.md for Source section |
| `mcp__linear-server__get_issue` | Fetch ticket on creation |
| `mcp__linear-server__create_comment` | Post summary on completion |
| `mcp__linear-server__update_issue` | Set status to Done |

---

## Testing Plan

### F1: Ticket Capture
| Scenario | Expected |
|----------|----------|
| `/brains.new "work on DEV-101"` | Source section added to INITIATIVE.md |
| `/brains.new "add feature"` | No Source section |
| `/brains.new "DEV-999"` (invalid) | Warning shown, no Source section |
| Linear MCP unavailable | No error, proceeds normally |

### F2: Commit Offer
| Scenario | Expected |
|----------|----------|
| Complete with uncommitted changes | Offer shown, commit on accept |
| Complete with no changes | No offer |
| Complete outside git repo | No offer |
| Commit fails | Error shown, completion proceeds |

### F3: Linear Update
| Scenario | Expected |
|----------|----------|
| Complete with Source section | Offer shown, updates on accept |
| Complete without Source but name has ticket | Fallback detection, offer shown |
| Complete without any ticket | No offer |
| Linear API fails | Error shown, completion proceeds |

---

## Estimated Changes

| File | Lines Added | Complexity |
|------|-------------|------------|
| `embed/workflows/new.md` | ~20 | Low |
| `embed/profiles/feature.md` | ~15 | Low |
| `embed/profiles/bug.md` | ~15 | Low |
| `embed/profiles/refactor.md` | ~15 | Low |
| `embed/workflows/complete.md` | ~40 | Medium |
| `embed/profiles/complete.md` | ~40 | Medium |

Total: ~145 lines of markdown instructions across 6 files.
