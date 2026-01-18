# Business Specification: MCP Command Idempotency

## Problem Statement

MCP tools that create files or initialize state can overwrite existing work when called multiple times. This causes data loss when:
- A user accidentally runs the same command twice
- An AI agent retries a failed operation
- Network issues cause duplicate requests

**Example scenario**: User runs `initiative create` with `name: "auth-feature"`. They work on the specification for an hour. They accidentally run the same command again. Their work is gone - replaced by blank templates.

## User Stories

### US-1: Initiative Creation Protection
**As a** user creating an initiative
**I want** the system to detect if an initiative with the same name/type already exists
**So that** I don't accidentally overwrite my existing work

**Acceptance Criteria:**
- If an initiative with matching `name` AND `type` exists in history, return the existing initiative instead of creating a new one
- Response should indicate whether a new initiative was created or an existing one was returned
- User's existing files in the initiative directory must remain untouched

### US-2: Template Copy Protection
**As a** user with an existing initiative
**I want** template copying to skip files that already exist
**So that** my modified specs/plans are not overwritten by blank templates

**Acceptance Criteria:**
- When copying templates to a cycle folder, check if each target file exists
- If file exists and has content (non-empty), skip copying the template
- If file exists but is empty, overwrite with template
- Log which files were skipped vs copied

### US-4: Profile Write Safety (Already Implemented)
**As a** user writing profiles
**I want** the default behavior to reject overwrites
**So that** I don't accidentally lose my custom profile configurations

**Status:** Already implemented via `overwrite: false` default. No changes needed.

## Out of Scope

- Undo/rollback functionality for overwrites
- Version history of files
- Backup creation before operations
- Changes to read-only tools (profile-compose, profile-list, feature)
- Changes to database-backed tools (stickymemory - already idempotent)
- Changes to in-memory tools (code-reasoning - already idempotent)
- **Step tool idempotency** - deferred to future work; step tool currently regenerates files on each run (acceptable behavior for now)
- Concurrency control (file locking) - filesystem operations are not atomic; first write wins in race conditions (rare edge case for human-initiated operations)

## Success Metrics

1. **Zero data loss incidents** - Verified by: Integration test creates initiative, modifies spec.md, re-runs create command, verifies spec.md content unchanged
2. **Clear response messaging** - Verified by: Response includes `already_existed: true/false` field and `skipped_files` array when applicable
3. **Backward compatibility** - Verified by: All existing MCP tool calls continue to function; new response fields are additive only (clients ignore unknown fields)

## Edge Cases

| Scenario | Expected Behavior |
|----------|-------------------|
| Same name, different type | Create new initiative (name+type is the unique key) |
| Same name, same type, different description | Return existing (description is not part of uniqueness) |
| Initiative exists but NOT in active state | Create new initiative (inactive initiatives are treated as archived) |
| File exists but is empty (0 bytes) | Overwrite with template |
| File exists with only whitespace | Treat as empty, overwrite with template (check: `len(bytes.TrimSpace(content)) == 0`) |

## Glossary

- **Initiative**: A tracked unit of work containing cycles, specs, and plans
- **Cycle**: A phase within an initiative (e.g., feature cycle, bug cycle)
- **Idempotent**: An operation that produces the same result regardless of how many times it's executed
- **Template**: Starter files copied when initializing new work items
- **Active Initiative**: An initiative referenced in `.brains/active.json`; only one initiative can be active at a time
- **Inactive Initiative**: An initiative in `history/` folder but not referenced in `.brains/active.json`; treated as archived for idempotency purposes
