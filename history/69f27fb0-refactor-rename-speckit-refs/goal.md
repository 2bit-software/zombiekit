# Goal: Rename speckit references to brains/zombiekit

## Improvement Goal

Replace all remaining "speckit" and ".specify" references in active template and profile files with the correct "brains" or "zombiekit" equivalents. The old speckit command surface (`/speckit.plan`, `/speckit.tasks`, `/speckit.checklist`) was replaced by the brains workflow system (`/brains.new` → `/brains.next` step progression), but the embedded templates still reference the old commands.

## What "Better" Means

- **Consistency**: All user-facing template text references the current command surface
- **Clarity**: New users reading templates won't encounter dead references to a system that no longer exists
- **Maintainability**: No confusion about which commands to use

## Success Criteria

1. Zero occurrences of `speckit`, `spec-kit`, or `.specify` in `embed/templates/` and `embed/profiles/`
2. All replacements correctly map old commands to their brains/zombiekit equivalents
3. Templates remain accurate descriptions of the current workflow
