# Goal: Remove Cycles Concept

## What "Better" Means

**Current State**: Initiatives contain nested cycle folders, adding an extra layer of directory hierarchy. Each initiative can have multiple cycles (feat/ref/fix passes), tracked in INITIATIVE.md with cycle tables.

**Target State**: Initiatives are single-level units of work. Artifacts (spec.md, plan.md, tasks.md, etc.) live directly in the initiative folder. No cycle subfolder, no cycle tracking in INITIATIVE.md.

## Why This Improves the System

1. **Reduced Complexity**: One less folder level to navigate, understand, and maintain
2. **Clearer Mental Model**: Initiative = unit of work, done. No "cycles within cycles" confusion
3. **Simpler Code Paths**: Less branching logic for "which folder am I in?"
4. **Cleaner File Paths**: `history/{init}/spec.md` instead of `history/{init}/{cycle}/spec.md`

## Success Criteria

1. `initiative create` creates a flat initiative folder with artifacts at the root
2. INITIATIVE.md tracks steps directly, no "Cycles" section with nested headers
3. Step execution operates on initiative folder, not cycle folder
4. All existing tests pass (after updating expectations)
5. MCP responses no longer include `cycle_id` or `cycle_path` fields
6. Templates copy to initiative folder, not cycle subfolder

## Out of Scope

- **Multi-pass workflows**: If someone wants to refactor after a feature, they complete the initiative and start a new one
- **Cycle history**: No migration of existing multi-cycle initiatives
- **Backwards compatibility**: This is a breaking change for the folder structure
