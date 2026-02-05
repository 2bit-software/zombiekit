---
name: next
description: Advance to the next workflow step based on INITIATIVE.md state
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Next Step Workflow

Goal: Read step state from INITIATIVE.md and advance to the next step.

### Execution Steps

1. **Load Active State**
   - Read `.brains/active.json` to get initiative path
   - If no active initiative: Report error and suggest `/brains.new`

2. **Parse INITIATIVE.md**
   - Read `{initiative_path}/INITIATIVE.md`
   - Find the active cycle (status = "active")
   - Parse the step table to find current and next step

3. **Determine Action**
   - If current step is `in_progress`: Mark as `completed`, advance next `pending` to `in_progress`
   - If all steps are `completed`/`skipped`: Suggest `/brains.complete`
   - If no `in_progress` step: Start the first `pending` step

4. **Update INITIATIVE.md**
   - Update the step table with new status and timestamp
   - Use atomic write (temp file + rename)

5. **Load Next Profile**
   - Use `mcp__zombiekit__profile-compose` with the step's profile
   - Pass through any user arguments

### Step Status Values

| Status | Meaning |
|--------|---------|
| `pending` | Not yet started |
| `in_progress` | Currently active (only one at a time) |
| `completed` | Successfully finished |
| `skipped` | Intentionally bypassed |

### INITIATIVE.md Cycle Format

```markdown
### 1. feat/feature-name (active)

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-31 10:30 |
| plan | in_progress | 2026-01-31 11:00 |
| tasks | pending | - |
| implement | pending | - |
```

### Explicit Step Navigation

If the argument is a workflow step name, jump directly to that step (forwards or backwards):
- `next spec` - Jump to spec/analyze step
- `next plan` - Jump to plan step
- `next tasks` - Jump to tasks step
- `next implement` - Jump to implement step
- `next verify` - Jump to verify step

This allows:
- **Backwards navigation** - Return to an earlier step to revise work
- **Skip steps** - Jump ahead when intermediate steps aren't needed
- **Restart steps** - Re-run a step that was previously completed

When jumping to a step:
1. Mark any `in_progress` step as `completed` first
2. Set the target step to `in_progress`
3. Update timestamps for both changes
4. Load the appropriate profile

### Alternate Path Handling

For detours that don't affect the main step sequence:
- `next audit` - Run audit profile
- `next clarify` - Run clarification profile
- `next research` - Do more research

These overrides run the profile without changing step status.

### Output Format

```markdown
## Current Status
Initiative: {initiative-id}
Cycle: {cycle-number}. {type}/{name}
Current step: {step-name} ({status})
Progress: {completed}/{total} steps

## Next Step
Advancing to: **{next-step-name}**

Loading {profile} profile...
```

### Complete-or-Advance Logic

```
if current_step.status == "in_progress":
    current_step.status = "completed"
    current_step.updated = now()

next_step = find_first_pending_step()
if next_step:
    next_step.status = "in_progress"
    next_step.updated = now()
    load_profile(next_step.profile)
else:
    suggest("/brains.complete")
```

### Error Conditions

**No active initiative:**
```
No active initiative found.

Use `/brains.new <description>` to start a new initiative.
```

**No active cycle:**
```
Initiative exists but has no active cycle.

This may indicate the initiative needs to be restarted or is in an inconsistent state.
```

**All steps complete:**
```
All steps complete for {initiative-name}.

Options:
- `/brains.complete` - Mark initiative done
- `/brains.new <description>` - Start new work
```
