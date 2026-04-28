---
name: help
description: Show available commands, current state, and valid actions
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Help Workflow

Goal: Show state-aware contextual help based on the current initiative status.

### Step 1: Load State

Call `mcp__zombiekit__initiative` with `action: "status"` and `dir` set to the working directory.

The response contains:

| Field | Type | Description |
|-------|------|-------------|
| `active` | bool | Whether an initiative is active |
| `initiative_id` | string | Full initiative ID |
| `initiative_type` | string | `feature`, `bug`, or `refactor` |
| `current_step` | string | Name of the current step |
| `step_status` | string | Status of current step (`pending`, `in_progress`, `completed`) |
| `steps_completed` | int | Number of completed steps |
| `steps_total` | int | Total steps in workflow |
| `available_docs` | []string | Artifact filenames in initiative directory |
| `suggested_next` | string | Recommended next action |
| `history_path` | string | Relative path to initiative directory |
| `initiative_file` | string | Relative path to INITIATIVE.md |
| `files` | []string | Relative paths to all artifact files |

### Step 2: Branch on State

- If `active` is `false`: Render **No-Initiative Mode** (Step 3)
- If `active` is `true`: Render **Active-Initiative Mode** (Step 4)

### Step 3: No-Initiative Mode

Call `mcp__zombiekit__initiative` with `action: "list"` and `dir` set to the working directory to get recent initiatives.

Then render the following output (not in a code fence — render it as actual markdown):

---

**Output:**

```
## ZombieKit Help

No active initiative.

### Start New Work

    /brains.new add user authentication     (auto-detects: feature)
    /brains.new fix login timeout           (auto-detects: bug)
    /brains.new refactor auth module        (auto-detects: refactor)

### Recent Initiatives

{render a table from the initiative list response, up to 5 entries:}

    {id}  {type}  {name}  {status}

{If the list is empty, show: "No previous initiatives found."}

### Commands

    /brains.new [desc]   Start new work (auto-detects type)
    /brains.help         Show this help
```

---

### Step 4: Active-Initiative Mode

#### 4a. Parse Initiative Name

Extract a display name from `initiative_id` by stripping the UUID prefix and type.
Example: `69f0e882-feature-brains-help-contextual` → `brains-help-contextual`

#### 4b. Read INITIATIVE.md for Step Table and Source Section

Read the file at `initiative_file` to get:
1. The full step table (each step's name, status, and updated timestamp)
2. Whether a `## Source` section exists (contains Linear ticket reference)

#### 4c. Build Step Display

Use the step table parsed from INITIATIVE.md. For each step, show its actual status. Mark the current step (matching `current_step` from the status response) with `<-- current`.

#### 4d. Look Up Step Description

Use this table to find a one-line description for the current step:

| Type | Step | Description |
|------|------|-------------|
| feature | spec | Research and write business specification |
| feature | plan | Create implementation plan from spec |
| feature | tasks | Break plan into discrete implementable tasks |
| feature | implement | Execute tasks and write code |
| bug | investigate | Investigate the bug and determine root cause |
| bug | plan | Plan the fix approach |
| bug | tasks | Break fix into discrete tasks |
| bug | fix | Implement the fix |
| bug | verify | Verify the fix resolves the issue |
| refactor | analyze | Analyze code and define refactoring scope |
| refactor | plan | Plan the refactoring approach |
| refactor | tasks | Break refactor into discrete tasks |
| refactor | implement | Execute refactoring tasks |

If the `initiative_type` or `current_step` is not in this table, skip the description line.

#### 4e. Build Artifact List

List each entry from `available_docs` with `(exists)` marker. Use the `files` field for full relative paths. If `available_docs` is empty, show "No artifacts yet."

#### 4f. Build Source Section (Conditional)

If the INITIATIVE.md content (from step 4b) contains a `## Source` section, extract and display the Linear ticket reference. If no Source section exists, skip this section entirely.

#### 4g. Build Available Actions

Filter commands based on current state:

**Mid-workflow commands:**

    /brains.next        Advance to {next_step_name} step
    /brains.complete    Finish initiative ({remaining} steps remaining)
    /brains.help        Show this help
    /brains.new [desc]  Start new work (closes current initiative)

Where:
- `{next_step_name}` is the next pending step from the step table
- `{remaining}` is `steps_total - steps_completed`
- If all steps are completed, show `/brains.complete` as the primary action and omit `/brains.next`

#### 4h. Render Output

Render the following (not in a code fence — render as actual markdown):

---

**Output:**

```
## {display_name}

**Type**: {initiative_type} | **Progress**: {steps_completed}/{steps_total} | **Path**: {history_path}/

{if Source section exists:}
**Source**: [{ticket_id}]({ticket_url})
{end if}

### Progress

    {step_name}    {status}           {<-- current if matching}
    {step_name}    {status}
    ...

**Current step**: {current_step} — {step_description}

### Artifacts

    {history_path}/{filename}    (exists)
    {history_path}/{filename}    (exists)
    ...

### Available Actions

    /brains.next        Advance to {next_step} step
    /brains.complete    Finish initiative ({remaining} steps remaining)
    /brains.help        Show this help
    /brains.new [desc]  Start new work (closes current initiative)
```

---

### Step 5: Edge Cases

| Scenario | Behavior |
|----------|----------|
| No `.brains/` directory | `active` will be `false` — render no-initiative mode |
| Active but INITIATIVE.md missing | Show header with ID and type, skip progress and artifacts sections |
| All steps completed | Show progress as fully complete, `/brains.complete` as primary action, omit `/brains.next` |
| `current_step` is empty | Show step table without `<-- current` marker |
| `initiative list` returns empty | Show "No previous initiatives found." |
| Unknown `initiative_type` | Show step table from INITIATIVE.md, skip step description |
