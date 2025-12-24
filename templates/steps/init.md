---
name: init
description: Initialize a new initiative
type: step
---
# Initialize New Initiative

Create a new initiative (feature, bug, or refactor) and set it as the active initiative for the project.

## Required Parameters
- **type**: The type of initiative (feature, bug, refactor)
- **name**: A slug-friendly name for the initiative (e.g., "user-auth", "login-crash")

## Process
1. Create a new folder in `history/` with naming format: `{hex-timestamp}-{type}-{name}`
2. Create `INITIATIVE.md` in the folder with metadata
3. Update `.brains/active.json` to track this as the active initiative
4. Return the path to the new initiative folder
