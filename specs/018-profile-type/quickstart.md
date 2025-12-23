# Quickstart: Profile Type Classification

**Feature**: 018-profile-type
**Date**: 2025-12-23

## Overview

This feature adds a `type` field to profile YAML frontmatter, allowing profiles to be classified as Action, Domain, or Step profiles.

## Usage

### Defining a Profile Type

Add the `type` field to your profile's YAML frontmatter:

```yaml
---
name: my-profile
description: A sample profile
type: action
---

Your profile content here...
```

### Valid Type Values

| Type | Purpose |
|------|---------|
| `action` | Prompts for doing work within workflow steps |
| `domain` | Domain knowledge and context prompts |
| `step` | Workflow step profiles (e.g., specify, research, clarify) |

### Example Profiles

**Action Profile** (`~/.brains/profiles/code-review.md`):
```yaml
---
name: code-review
description: Reviews code for quality and best practices
type: action
---

Review the provided code for:
- Code quality and readability
- Potential bugs or issues
- Performance considerations
```

**Domain Profile** (`~/.brains/profiles/golang-expert.md`):
```yaml
---
name: golang-expert
description: Go/Golang domain expertise
type: domain
---

You are an expert Go developer with deep knowledge of:
- Go idioms and best practices
- Concurrency patterns (goroutines, channels)
- Standard library usage
```

**Step Profile** (`~/.brains/profiles/specify.md`):
```yaml
---
name: specify
description: Feature specification workflow step
type: step
---

This step guides the creation of detailed feature specifications...
```

## Viewing Types

### Web Interface

1. Navigate to the Profiles page
2. Each profile shows a colored badge indicating its type:
   - Purple badge: Action profiles
   - Green badge: Domain profiles
   - Blue badge: Step profiles
   - Gray badge: Unknown or no type

3. Click a profile to see type in the metadata section

### CLI

Profile type appears in JSON output:

```bash
brains profile show my-profile --json
```

```json
{
  "name": "my-profile",
  "type": "action",
  "description": "A sample profile",
  ...
}
```

## Notes

- The `type` field is optional - existing profiles work without it
- Type values are case-insensitive (Action, action, ACTION all work)
- Original casing is preserved for display
- Unknown type values are accepted for forward compatibility
