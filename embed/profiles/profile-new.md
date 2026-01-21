---
name: profile-new
description: Create a new ZombieKit profile through guided workflow. Dogfoods the research-create-audit-highlight cycle.
type: skill
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty). The input may contain:
- Profile name (required if not provided, prompt for it)
- Profile description/purpose
- Storage location preference (local/global)

## Outline

Goal: Create a new profile through the research → create → audit → highlight → write cycle.

## Workflow Phases

### Phase 1: Gather Inputs

Collect required information from user:

1. **Profile Name** (required)
   - If provided in arguments, use it
   - Otherwise, ask: "What should this profile be named?"
   - Names are normalized: lowercase, alphanumeric + hyphens

2. **Profile Purpose** (required)
   - If provided in arguments, use it
   - Otherwise, ask: "What should this profile help with? Describe its purpose."

3. **Storage Location** (required)
   - Ask user to choose:
     - **Local** (`.brains/profiles/`): Project-specific, not shared
     - **Global** (`~/.brains/profiles/`): Available in all projects

### Phase 2: Research

Understand existing profile patterns:

1. **List Existing Profiles**
   ```
   Use profile-list tool to enumerate available profiles
   ```

2. **Read Representative Profiles**
   - Read 2-3 existing profiles that are similar in purpose
   - Note their structure: frontmatter fields, body organization, content patterns

3. **Gather Context**
   - What type of profile is this? (domain knowledge, action, skill, step)
   - What existing profiles might it complement or extend?

### Phase 3: Create

Generate the profile content:

1. **Frontmatter** (YAML)
   ```yaml
   ---
   name: {normalized-name}
   description: {user-provided description or derived from purpose}
   type: domain  # Default to domain unless clearly another type
   includes: []  # Optional: profiles to compose before this one
   inherits: true  # Default
   ---
   ```

2. **Body Content**
   - Organize content logically based on purpose
   - Follow patterns observed in research phase
   - Keep content focused and actionable

### Phase 4: Audit

Validate the generated profile:

1. **Frontmatter Validity**
   - Parse YAML and check for syntax errors
   - Verify required fields: `name`, `description`

2. **Include Validation**
   - If `includes` field has entries, verify each profile exists via `profile-list`
   - Check for circular dependencies

3. **Classify Issues**
   - **CRITICAL**: Invalid YAML, missing required fields → blocks write
   - **MAJOR**: Non-existent includes → should fix before write
   - **MINOR**: Missing optional fields, style issues → acceptable

### Phase 5: Highlight

Present summary for user approval:

```
## Profile Summary

**Name**: {name}
**Storage**: {local|global} ({path})
**Type**: {type}
**Description**: {description}

### Content Preview
{first 10 lines of body}

### Audit Results
{any issues found, or "No issues found"}

---

Approve this profile? (yes/no)
```

### Phase 6: Write

On user approval:

1. **Call profile-save tool**
   ```json
   {
     "name": "{name}",
     "content": "{full profile content}",
     "location": "{local|global}",
     "overwrite": false
   }
   ```

2. **Handle Response**
   - If success: Display path and confirmation
   - If PROFILE_EXISTS: Ask user to overwrite or choose different name

3. **Verify**
   - Confirm profile appears in `profile-list`
   - Profile is ready for use via `profile-compose`

## Behavior Rules

- Always gather all required inputs before research phase
- Default profile type to `domain` unless purpose clearly indicates otherwise
- Never skip the audit phase
- Always present summary before writing
- Handle existing profile gracefully (offer overwrite or rename)

## Success Output

```
Profile created successfully!

Path: {absolute-path-to-profile}
Name: {profile-name}
Location: {local|global}

You can now use this profile with:
- profile-compose: Include "{profile-name}" in profiles list
- profile-list: Verify it appears in the list
```
