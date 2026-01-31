# Technical Specification: Enhanced Initiative Lifecycle with Linear Integration

## Architecture

No new Go components. All features implemented through workflow/profile markdown modifications.

```
┌──────────────────────────────────────────────────────────────────┐
│                        /brains.new                               │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│ embed/workflows/new.md                                           │
│ ┌──────────────────┐                                             │
│ │ Ticket Detection │──► mcp__linear-server__get_issue            │
│ └────────┬─────────┘                                             │
│          │ (enriched args)                                       │
│          ▼                                                       │
│ ┌──────────────────┐                                             │
│ │ Load Profile     │──► feature.md / bug.md / refactor.md        │
│ └──────────────────┘                                             │
└──────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│ embed/profiles/feature.md (or bug.md, refactor.md)               │
│ ┌──────────────────┐                                             │
│ │ Create Initiative│──► mcp__zombiekit__initiative               │
│ └────────┬─────────┘                                             │
│          │                                                       │
│          ▼                                                       │
│ ┌──────────────────┐                                             │
│ │ Add Source       │──► Edit tool (INITIATIVE.md)                │
│ └──────────────────┘                                             │
└──────────────────────────────────────────────────────────────────┘


┌──────────────────────────────────────────────────────────────────┐
│                      /brains.complete                            │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│ embed/workflows/complete.md                                      │
│                                                                  │
│ [Steps 1-4: existing completion flow]                            │
│          │                                                       │
│          ▼                                                       │
│ ┌──────────────────┐                                             │
│ │ Step 5: Commit   │──► Bash (git status) ──► AskUserQuestion    │
│ │ Offer            │──► Bash (git add -A) ──► Skill (commit-msg) │
│ └────────┬─────────┘                                             │
│          │                                                       │
│          ▼                                                       │
│ ┌──────────────────┐                                             │
│ │ Step 6: Linear   │──► Read (INITIATIVE.md) ──► AskUserQuestion │
│ │ Update           │──► mcp__linear-server__create_comment       │
│ │                  │──► mcp__linear-server__update_issue         │
│ └────────┬─────────┘                                             │
│          │                                                       │
│          ▼                                                       │
│ [Steps 7-8: clear state, report]                                 │
└──────────────────────────────────────────────────────────────────┘
```

---

## Feature 1: Ticket Capture on Creation

### File Changes

#### `embed/workflows/new.md`

**Change type**: Content addition after classification section
**New section**: "Ticket Detection"

```markdown
### Ticket Detection

After classification, before loading the profile:

1. Parse user input for Linear ticket pattern: `[A-Z]+-[0-9]+` (case-insensitive)
   - Examples: "DEV-101", "proj-42", "TEAM-1234"
2. If pattern found:
   - Extract and uppercase the ticket identifier
   - Call `mcp__linear-server__get_issue` with the identifier
   - If successful: Append metadata to arguments:
     ```
     ---
     LINEAR_TICKET: DEV-101
     LINEAR_URL: https://linear.app/...
     LINEAR_TITLE: Ticket title here
     ```
   - If fails (404, MCP unavailable): Display warning, proceed without metadata
3. Load the profile with enriched arguments
```

#### `embed/profiles/feature.md` (and bug.md, refactor.md)

**Change type**: New step after "Initiative Check"
**New step**: "Add Source Section"

```markdown
1.5 **Add Source Section** (if ticket info present)
   - Check if arguments contain `LINEAR_TICKET:` line
   - If not present: Skip to step 2
   - If present:
     a. Parse LINEAR_TICKET, LINEAR_URL, LINEAR_TITLE from arguments
     b. Read current INITIATIVE.md
     c. Use Edit tool to insert before "## Description":
        ```markdown
        ## Source

        **Linear Ticket**: [LINEAR_TICKET](LINEAR_URL)
        **Title**: LINEAR_TITLE

        ```
```

### Tool Contracts

#### Linear MCP - get_issue

**Input**:
```json
{
  "id": "DEV-101"
}
```

**Output** (used fields):
```json
{
  "identifier": "DEV-101",
  "title": "Have the /brains.complete command also offer to write a commit",
  "url": "https://linear.app/heinsight/issue/DEV-101/..."
}
```

**Error handling**:
- 404: Display "Ticket DEV-101 not found", proceed without Source
- MCP unavailable: Skip silently

---

## Feature 2: Commit Offer on Completion

### File Changes

#### `embed/workflows/complete.md`

**Change type**: New step 5 inserted
**Step renumbering**: Old 5→7, old 6→8

```markdown
5. **Offer Commit** (if in git repository)
   - Run `git status --porcelain` via Bash tool
   - If command fails (exit code != 0): Skip to step 6
   - If output is empty: Skip to step 6
   - If output is non-empty:
     a. Parse output to count changes:
        - Modified: lines with `M` in first two columns
        - Added: lines starting with `A` or `??`
        - Deleted: lines with `D` in first two columns
     b. Display: "You have uncommitted changes: X modified, Y added, Z deleted"
     c. Use AskUserQuestion:
        ```json
        {
          "questions": [{
            "question": "Would you like to commit your changes before completing?",
            "header": "Commit",
            "multiSelect": false,
            "options": [
              {"label": "Yes, commit changes", "description": "Stage all and generate commit message"},
              {"label": "No, skip commit", "description": "Complete without committing"}
            ]
          }]
        }
        ```
     d. If "Yes, commit changes":
        - Run `git add -A` via Bash
        - Call Skill tool: `{"skill": "commit-message"}`
        - If error: Display "Commit failed: {error}", proceed to step 6
     e. Proceed to step 6
```

#### `embed/profiles/complete.md`

Keep in sync with workflow (identical step 5 content).

### Tool Contracts

#### Bash - git status

**Input**: `git status --porcelain`

**Output interpretation**:
| Output | Action |
|--------|--------|
| Exit != 0 | Skip commit offer (not git repo) |
| Empty | Skip commit offer (no changes) |
| Non-empty | Show commit offer |

#### Bash - git add

**Input**: `git add -A`

**Purpose**: Stage all changes before commit-message skill

#### Skill - commit-message

**Input**: `{"skill": "commit-message"}`

**Behavior**: Generates conventional commit message, stages, commits

---

## Feature 3: Linear Ticket Update on Completion

### File Changes

#### `embed/workflows/complete.md`

**Change type**: New step 6 inserted (after commit offer)

```markdown
6. **Offer Linear Update** (if source ticket exists)
   - Read INITIATIVE.md via Read tool
   - Look for Source section with pattern: `**Linear Ticket**: \[([\w-]+)\]`
   - If not found: Check initiative name for `[A-Z]+-[0-9]+` pattern (fallback)
   - If no ticket found: Skip to step 7
   - If ticket found:
     a. Display: "Source ticket found: {TICKET-ID}"
     b. Use AskUserQuestion:
        ```json
        {
          "questions": [{
            "question": "Would you like to update Linear ticket {TICKET-ID}?",
            "header": "Linear",
            "multiSelect": false,
            "options": [
              {"label": "Yes, update ticket", "description": "Post summary and mark as Done"},
              {"label": "No, skip update", "description": "Complete without updating ticket"}
            ]
          }]
        }
        ```
     c. If "Yes, update ticket":
        - Generate summary from completion outcomes
        - Call mcp__linear-server__create_comment:
          ```json
          {
            "issueId": "{TICKET-ID}",
            "body": "## Work Completed\n\n{summary}\n\n---\n*Completed via ZombieKit*"
          }
          ```
        - Call mcp__linear-server__update_issue:
          ```json
          {
            "id": "{TICKET-ID}",
            "state": "Done"
          }
          ```
        - If error: Display "Linear update failed: {error}", proceed to step 7
     d. Proceed to step 7
```

#### `embed/profiles/complete.md`

Keep in sync with workflow.

### Tool Contracts

#### Linear MCP - create_comment

**Input**:
```json
{
  "issueId": "DEV-101",
  "body": "## Work Completed\n\n- Feature implemented\n- Tests added\n\n---\n*Completed via ZombieKit*"
}
```

**Error handling**: Display error, proceed with completion

#### Linear MCP - update_issue

**Input**:
```json
{
  "id": "DEV-101",
  "state": "Done"
}
```

**Error handling**: Display error, proceed with completion

---

## Updated Workflow Structures

### `embed/workflows/new.md`

```
Before:
1. Classification Task
2. After Classification → Load Profile

After:
1. Classification Task
2. Ticket Detection (NEW)
3. After Classification → Load Profile (with enriched args)
```

### `embed/workflows/complete.md`

```
Before:
1. Load Active Initiative
2. Completion Check
3. Generate Summary
4. Update INITIATIVE.md
5. Clear Active State
6. Report Completion

After:
1. Load Active Initiative
2. Completion Check
3. Generate Summary
4. Update INITIATIVE.md
5. Offer Commit (NEW - F2)
6. Offer Linear Update (NEW - F3)
7. Clear Active State
8. Report Completion
```

---

## Error Handling Summary

| Scenario | Detection | Recovery |
|----------|-----------|----------|
| Not a git repo | `git status` exit != 0 | Skip commit offer silently |
| No git changes | `git status` empty output | Skip commit offer silently |
| Commit fails | Skill returns error | Display error, proceed |
| No ticket in Source | Pattern not found | Try name fallback, then skip |
| Linear MCP unavailable | Tool call fails | Skip Linear offers silently |
| Ticket not found (404) | API error | Display warning, proceed |
| Linear API error | create_comment/update_issue fails | Display error, proceed |

---

## Security Considerations

- No secrets exposed in workflow content
- Linear API accessed via existing MCP (authentication handled by MCP server)
- Git operations are standard (status, add, commit)
- User explicitly consents to each action via AskUserQuestion
