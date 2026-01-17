# Initiative: profile-creator

**Type**: feature
**Status**: complete
**Created**: 2026-01-15T17:20:42-08:00
**ID**: 6969926a-feature-profile-creator

## Description

Add a profile creation workflow to ZombieKit that dogfoods the existing research → create → audit → approve cycle. This enables users to create new profiles through a guided process, with output stored in either local (.brains/profiles/) or global (~/.brains/profiles/) directories, making them immediately discoverable by the system.

## Goals

- Create a `/brains.profile.new` command/workflow for guided profile creation
- Reuse the existing research-create-audit cycle for profile development
- Support both local and global profile storage (user chooses at creation time)
- Ensure new profiles are immediately discoverable without restart
- Validate profile structure (frontmatter, required fields) during creation

## Progress

### Implementation Tasks
- ✅ T001: Add WriteRequest/WriteResponse types
- ✅ T002: Add Write method to profile service
- ✅ T003: Add HandleWrite handler
- ✅ T004: Register profile-write tool
- ✅ T005: Add Write method tests (8 tests)
- ✅ T006: Add HandleWrite tests (6 tests)
- ✅ T007: Create profile-new.md workflow profile
- ⏳ T008: Manual verification (deferred)
- ⏳ T009: End-to-end test (deferred)

## Completion

**Completed**: 2026-01-15
**Duration**: Same day

### Outcomes
- `profile-write` MCP tool: Complete
- `profile-new` workflow profile: Complete
- Test coverage: 14 new tests
- Manual verification: Pending rebuild

### Files Changed
- `internal/mcp/tools/profile/types.go` (new)
- `internal/profile/service.go` (Write method)
- `internal/mcp/tools/profile/tool.go` (HandleWrite)
- `internal/mcp/server.go` (tool registration)
- `internal/profile/service_test.go` (tests)
- `internal/mcp/tools/profile/tool_test.go` (tests)
- `profiles/profile-new.md` (new)

### Notes
T008/T009 require rebuild to verify embedded profile discoverability. Core implementation complete and tested.
