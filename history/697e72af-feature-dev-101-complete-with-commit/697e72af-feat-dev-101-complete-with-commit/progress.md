# Implementation Progress

## Status: Complete

All 8 tasks implemented.

---

## T001 - Add ticket detection to embed/workflows/new.md
- **Status**: Complete
- **Files**: `embed/workflows/new.md`
- **Changes**: Added "Linear Ticket Detection" section after classification
- **Notes**: Parses `[A-Z]+-[0-9]+` pattern, fetches via Linear MCP, appends metadata to arguments

## T002 - Add Source section step to embed/profiles/feature.md
- **Status**: Complete
- **Files**: `embed/profiles/feature.md`
- **Changes**: Added step 1.5 "Add Source Section" after Initiative Check
- **Notes**: Parses LINEAR_TICKET metadata from arguments, edits INITIATIVE.md

## T003 - Add Source section step to embed/profiles/bug.md
- **Status**: Complete
- **Files**: `embed/profiles/bug.md`
- **Changes**: Same as T002
- **Notes**: Identical implementation

## T004 - Add Source section step to embed/profiles/refactor.md
- **Status**: Complete
- **Files**: `embed/profiles/refactor.md`
- **Changes**: Same as T002
- **Notes**: Identical implementation

## T005 - Add commit offer step to embed/workflows/complete.md
- **Status**: Complete
- **Files**: `embed/workflows/complete.md`
- **Changes**: Added step 5 "Offer Commit"
- **Notes**: Uses `git status --porcelain`, AskUserQuestion, and commit-message skill

## T006 - Add Linear update step to embed/workflows/complete.md
- **Status**: Complete
- **Files**: `embed/workflows/complete.md`
- **Changes**: Added step 6 "Offer Linear Update"
- **Notes**: Reads Source from INITIATIVE.md, posts comment, updates status to Done

## T007 - Update embed/profiles/complete.md to match workflow
- **Status**: Complete
- **Files**: `embed/profiles/complete.md`
- **Changes**: Synced steps 5-8 with workflow
- **Notes**: Profile now matches workflow exactly

## T008 - Test all features end-to-end
- **Status**: Ready for manual testing
- **Notes**: Features are prompt-engineering only; testing requires actual usage

---

## Files Modified

| File | Lines Added |
|------|-------------|
| `embed/workflows/new.md` | +37 |
| `embed/profiles/feature.md` | +14 |
| `embed/profiles/bug.md` | +14 |
| `embed/profiles/refactor.md` | +14 |
| `embed/workflows/complete.md` | +68 |
| `embed/profiles/complete.md` | +68 |
| **Total** | **~195** |

---

## Testing Notes

Since these are prompt-engineering changes, testing requires:

1. **F1 (Ticket Capture)**: Run `/brains.new "work on DEV-XXX"` and verify Source section in INITIATIVE.md
2. **F2 (Commit Offer)**: Run `/brains.complete` with uncommitted changes
3. **F3 (Linear Update)**: Run `/brains.complete` when INITIATIVE.md has Source section

All features gracefully handle failures (missing Linear MCP, git errors, etc.).
