---
name: step
description: Jump to a specific workflow step within the current initiative
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding.

## Step Navigation Workflow

Goal: Jump to a specific step in the current work item's workflow, loading the appropriate profile.

### Execution Steps

1. **Load Active State**
   - Read `.brains/active.json`
   - If no active initiative: Report error and suggest `/brains.new`

2. **Parse Step Argument**
   - Extract the requested step name from arguments
   - Valid steps depend on work item type (see below)

3. **Validate Step**
   - Check step name is valid for current work item type
   - If invalid: Show available steps for current context

4. **Load Step Profile**
   - Use `mcp__zombiekit__profile-compose` to load the requested profile
   - Pass through any additional arguments

### Valid Steps by Work Item Type

**Feature** (spec -> plan -> tasks -> implement)
| Step | Profile | Description |
|------|---------|-------------|
| `spec` | `feature` | Create/revise the feature specification |
| `plan` | `plan` | Create implementation plan from spec |
| `tasks` | `tasks` | Generate task list from plan |
| `implement` | `implement` | Execute tasks |
| `audit` | `audit` | Cross-check artifact alignment |
| `clarify` | `clarify` | Identify ambiguities |

**Bug** (report -> investigate -> fix-plan -> implement)
| Step | Profile | Description |
|------|---------|-------------|
| `report` | `bug` | Document the bug report |
| `investigate` | `bug` | Investigate root cause |
| `fix-plan` | `plan` | Create fix implementation plan |
| `implement` | `implement` | Execute the fix |

**Refactor** (goal -> analysis -> plan -> tasks -> implement)
| Step | Profile | Description |
|------|---------|-------------|
| `goal` | `refactor` | Define refactoring goals |
| `analysis` | `refactor` | Analyze dependencies |
| `plan` | `plan` | Create refactor plan |
| `tasks` | `tasks` | Generate task list |
| `implement` | `implement` | Execute refactoring |

### Common Steps (all types)
| Step | Profile | Description |
|------|---------|-------------|
| `research` | `research` | Standalone research |
| `update` | `update` | Modify existing artifacts |
| `revise` | `revise` | Re-enter workflow cycle |

### Error Handling

If step is not recognized:
```
Unknown step: "{step}"

Available steps for {work-item-type}:
- spec: Create/revise specification
- plan: Create implementation plan
- tasks: Generate task list
- implement: Execute tasks

Use: /brains.step <step-name>
```

### After Loading Profile

Once the profile is loaded, execute it with the current work item context. The profile will:
1. Load relevant artifacts from the work item directory
2. Guide the user through that phase
3. Suggest the next step when complete
