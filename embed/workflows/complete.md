---
name: complete
description: Mark the current initiative as complete and clear active state
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

### AutoMode Detection

If the user input contains the keyword **automode** (case-insensitive), or if this workflow was invoked automatically by a step running in automode:

- Load the automode profile via `mcp__zombiekit__profile-compose` with `profiles: ["automode"]` and follow its "At the Complete Step" instructions.
- These instructions override the interactive confirmation prompts below (commit, Linear update, etc.).

## Complete Initiative Workflow

Goal: Properly close out an initiative, archive artifacts, and clear active state.

### Execution Steps

1. **Load Active Initiative**
   - Read `.brains/active.json`
   - If no active initiative: Report nothing to complete

2. **Completion Check**
   - Review all work items
   - Identify incomplete items
   - If incomplete items exist:
     - List them with status
     - Confirm user wants to complete anyway
     - OR suggest completing work items first

3. **Generate Summary**
   - Create completion summary in INITIATIVE.md
   - List all work items and outcomes
   - Note any incomplete items (if proceeding anyway)
   - Record completion timestamp

4. **Update INITIATIVE.md**
   ```markdown
   ## Completion

   **Completed**: {timestamp}
   **Duration**: {start to end}

   ### Outcomes
   - Feature: auth-api - Complete
   - Feature: session-mgmt - Complete
   - Refactor: middleware - Skipped (deprioritized)

   ### Notes
   {Any closing notes}
   ```

5. **Offer Commit / Push / PR** (if in git repository)
   - Run `git status --porcelain` via the `mcp__zombiekit__git` tool (or Bash fallback)
   - If command fails (not a git repo): Skip to step 6
   - If output is empty (no changes): Skip to step 6
   - If output is non-empty (changes detected):
     a. Parse output to summarize changes:
        - Count lines with `M` (modified)
        - Count lines with `A` or `??` (added/untracked)
        - Count lines with `D` (deleted)
     b. Display: "Uncommitted changes detected: X modified, Y added, Z deleted"
     c. Use `AskUserQuestion` tool:
        ```json
        {
          "questions": [{
            "question": "How would you like to handle these changes?",
            "header": "Commit & Publish",
            "multiSelect": false,
            "options": [
              {"label": "Commit, push, and open PR", "description": "Stage, commit, push branch, then open a pull request"},
              {"label": "Commit only", "description": "Stage all and generate commit message — no push"},
              {"label": "Do nothing", "description": "Complete the initiative without touching git"}
            ]
          }]
        }
        ```
     d. If "Commit, push, and open PR":
        - **IMPORTANT**: Stage BOTH implementation files AND the initiative's `history/{initiative}/` directory. The spec, research, plan, tasks, and INITIATIVE.md are part of the feature work and must be committed together. Never commit implementation files without their corresponding history artifacts.
        - Stage the relevant files via `mcp__zombiekit__git` (prefer explicit paths over `git add -A`)
        - Use `Skill` tool with `skill: "commit-message"` to generate and execute commit
        - Push the current branch via `mcp__zombiekit__git` (equivalent to `git push -u origin <branch>`)
        - Use `Skill` tool with `skill: "create-pr"` to open the pull request
        - If any step fails: Display error message, proceed to step 6
     e. If "Commit only":
        - **IMPORTANT**: Stage BOTH implementation files AND the initiative's `history/{initiative}/` directory (same rule as above).
        - Stage the relevant files via `mcp__zombiekit__git`
        - Use `Skill` tool with `skill: "commit-message"` to generate and execute commit
        - If commit fails: Display error message, proceed to step 6
     f. If "Do nothing": Proceed to step 6

6. **Offer Linear Update** (if source ticket exists)
   - Read INITIATIVE.md and look for Source section with `**Linear Ticket**: [TICKET-ID]` pattern
   - If no Source section: Try parsing initiative name for `[A-Z]+-[0-9]+` pattern as fallback
   - If no ticket found: Skip to step 7
   - If ticket found:
     a. Display: "Source ticket found: {TICKET-ID}"
     b. Use `AskUserQuestion` tool:
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
        - Generate work summary from the Outcomes section in INITIATIVE.md
        - Post comment via `mcp__linear-server__create_comment`:
          ```json
          {
            "issueId": "{TICKET-ID}",
            "body": "## Work Completed\n\n{summary of outcomes}\n\n---\n*Completed via ZombieKit initiative: {initiative-name}*"
          }
          ```
        - Update status via `mcp__linear-server__update_issue`:
          ```json
          {
            "id": "{TICKET-ID}",
            "state": "Done"
          }
          ```
        - If API fails: Display error message, proceed to step 7
     d. Proceed to step 7

7. **Clear Active State**
   - Remove or clear `.brains/active.json`
   - Initiative remains in history (never deleted)

8. **Update Flimsy** (conversation index)
   - The current session ID is available from the startup hook context (check `$SESSION_ID` or the session start system reminder)
   - Tag this conversation in flimsy with the initiative name and outcome:
     ```bash
     curl -s -X PUT "http://localhost:8090/api/v1/conversations/{session-id}/annotations" \
       -H "Content-Type: application/json" \
       -d '{"annotations": [{"key": "initiative", "value": "{initiative-name}"}, {"key": "status", "value": "complete"}]}'
     ```
   - If flimsy is not running (connection refused): Skip silently, do not block completion
   - If session ID is not available: Skip silently

9. **Report Completion**
   - Initiative name
   - Work items completed vs skipped
   - Total duration
   - History location
   - Suggested next command

### Output Format

```markdown
# Initiative Completed

**{initiative-name}**

Duration: {X days}
Location: history/{date}-{name}/

## Summary
- Features completed: 2
- Refactors completed: 0
- Bugs fixed: 1
- Items skipped: 1

## Next Steps
- Start new initiative with `/brains.new "..."`
- View history with `/brains.help`
```

### Behavior Rules

- Never delete initiative, only mark complete
- Warn about incomplete items
- Require confirmation for partial completion
- Always update INITIATIVE.md with summary
