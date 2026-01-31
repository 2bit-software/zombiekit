# Technical Requirements

## Implementation Approach

**Workflow-only change** - modify `embed/workflows/complete.md` to include commit offer instructions after the completion summary and before the initiative complete tool call.

## Files to Modify

1. `embed/workflows/complete.md` - Add commit offer step between summary and completion
2. `embed/profiles/complete.md` - Keep in sync with workflow (for profile-based usage)

## Insertion Point

Current workflow structure (from `embed/workflows/complete.md`):

```
1. Load Active Initiative
2. Completion Check
3. Generate Summary
4. Update INITIATIVE.md
5. Clear Active State          <-- INSERT commit offer BEFORE this
6. Report Completion
```

New step 5 becomes "Offer Commit" and current step 5 becomes step 6.

## Commit Offer Implementation

### Step 5: Offer Commit

```markdown
5. **Offer Commit** (if in git repository)
   - Run `git status --porcelain` via Bash tool
   - If command fails (not a git repo): Skip to step 6
   - If output is empty (no changes): Skip to step 6
   - If output is non-empty:
     a. Parse output to summarize changes
     b. Use AskUserQuestion tool to offer commit
     c. If user accepts: Use Skill tool with `commit-message`
     d. If commit fails: Display error, continue to step 6
     e. If user declines: Continue to step 6
```

### AskUserQuestion Usage

```yaml
AskUserQuestion tool parameters:
  questions:
    - question: "Would you like to commit your changes before completing the initiative?"
      header: "Commit"
      multiSelect: false
      options:
        - label: "Yes, commit changes"
          description: "Generate a commit message and commit all changes"
        - label: "No, skip commit"
          description: "Complete the initiative without committing"
```

### Skill Tool Usage

```yaml
Skill tool parameters:
  skill: "commit-message"
```

The `commit-message` skill is defined in the system prompt and:
- Generates conventional commit messages explaining WHY changes exist
- Should be invoked after work is complete
- Handles the actual git commit execution

## Git Status Parsing

```bash
# Run this to detect changes
git status --porcelain

# Parsing logic:
# - Empty output = no changes, skip offer
# - Any output = changes exist, offer commit
# - First column: staged status (M=modified, A=added, D=deleted, R=renamed, ?=untracked)
# - Second column: unstaged status
```

### Change Count Derivation (Optional Enhancement)

```bash
# Count by type (for user-facing summary):
# Modified: lines matching /^.M|M./
# Added: lines matching /^A/
# Deleted: lines matching /^.D|D./
# Untracked: lines matching /^\?\?/
```

Note: Exact counts are nice-to-have. A simple "You have uncommitted changes" is sufficient.

## Error Handling

| Scenario | Behavior |
|----------|----------|
| `git status` fails | Skip commit offer silently (assume not a git repo) |
| No changes | Skip commit offer silently |
| User declines | Proceed to completion |
| User doesn't respond | Proceed to completion |
| `commit-message` skill fails | Display "Commit failed: [error]. Proceeding with completion." |
| Pre-commit hook fails | Same as above |

## Testing Considerations

The implementation is entirely in the workflow markdown. Testing will be manual:

1. Run `/brains.complete` with uncommitted changes → should offer commit
2. Run `/brains.complete` with no changes → should skip offer
3. Run `/brains.complete` outside git repo → should skip offer
4. Accept commit offer → should invoke commit-message skill
5. Decline commit offer → should complete normally
