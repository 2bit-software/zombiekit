# Tasks: Workflow Entrypoints

## Complexity Analysis

- **Files affected**: 7 (2 modify, 2 new, 3 delete)
- **Estimated lines**: ~100 additions, ~50 deletions
- **Cross-module dependencies**: Low (MCP tool → profile service)
- **Classification**: Simple (<5 files)

## Task List

### Phase 1: MCP Tool + Service Changes

- [ ] T001 [P1] Add `ComposeOptions` struct with `WorkflowOnly` field to `internal/profile/service.go`
- [ ] T002 [P1] Add `ComposeWithOptions` method to Service that filters by type before composition `internal/profile/service.go`
- [ ] T003 [P1] Add `workflow` boolean parameter to HandleCompose in `internal/mcp/tools/profile/tool.go`
- [ ] T004 [P1] Write unit test for workflow filter in `internal/profile/service_test.go`
- [ ] T005 [P1] Write unit test for MCP tool workflow parameter in `internal/mcp/tools/profile/tool_test.go`

### Phase 2: New Workflow Profile

- [ ] T006 [P2] Create `profiles/new.md` with `type: workflow` and classification instructions

### Phase 3: New Claude Command

- [ ] T007 [P3] Create `integrations/claude/commands/brains.new.md` that loads workflow profile with `workflow: true`

### Phase 4: Remove Legacy Commands

- [ ] T008 [P4] Delete `integrations/claude/commands/brains.feature.md`
- [ ] T009 [P4] Delete `integrations/claude/commands/brains.bug.md`
- [ ] T010 [P4] Delete `integrations/claude/commands/brains.refactor.md`
- [ ] T011 [P4] Update `internal/cli/init_test.go` to reference `brains.new.md` instead of deleted commands

### Phase 5: Local Command Cleanup

- [ ] T012 [P5] Delete `.claude/commands/brains.feature.md`
- [ ] T013 [P5] Delete `.claude/commands/brains.bug.md`
- [ ] T014 [P5] Delete `.claude/commands/brains.refactor.md`
- [ ] T015 [P5] Create `.claude/commands/brains.new.md` (copy from integrations)

## Parallel Execution Opportunities

```
T001, T002 → T003 → T004, T005 (sequential within phase)
T006 (independent)
T007 (depends on T006)
T008, T009, T010, T011 (parallel)
T012, T013, T014, T015 (parallel)
```

## Acceptance Criteria

### T001-T003: MCP Tool Changes
- [ ] `profile-compose` accepts optional `workflow` boolean parameter
- [ ] `workflow: true` returns only profiles with `type: workflow`
- [ ] `workflow: false` or omitted returns profiles without `type: workflow`
- [ ] Existing behavior unchanged when parameter omitted

### T004-T005: Unit Tests
- [ ] Test confirms workflow filter returns correct profile type
- [ ] Test confirms name collision resolved by filter
- [ ] Tests pass with `go test ./...`

### T006: Workflow Profile
- [ ] `profiles/new.md` has `type: workflow` in frontmatter
- [ ] Contains `$ARGUMENTS` placeholder
- [ ] Contains classification instructions for feature/bug/refactor
- [ ] Embedded in binary via existing `EmbeddedProfiles`

### T007: Claude Command
- [ ] `brains.new.md` calls profile-compose with `workflow: true`
- [ ] Passes `$ARGUMENTS` to the profile

### T008-T015: Command Cleanup
- [ ] Legacy commands removed from embedded source
- [ ] Legacy commands removed from local `.claude/commands/`
- [ ] Tests updated to use `brains.new.md`
- [ ] `brains init` copies new command, not old ones

## Traceability

| Task | Plan Phase | Spec Section |
|------|------------|--------------|
| T001-T005 | Phase 1 | MCP Tool Change |
| T006 | Phase 2 | File: profiles/new.md |
| T007 | Phase 3 | File: brains.new.md |
| T008-T011 | Phase 4 | Commands to Delete |
| T012-T015 | Phase 4 | Local cleanup |

## Suggested Execution Order

1. T001 → T002 → T003 (service changes flow to MCP tool)
2. T004, T005 (tests in parallel)
3. T006 (new profile)
4. T007 (new command)
5. T008, T009, T010 (delete legacy embedded commands in parallel)
6. T011 (update tests)
7. T012, T013, T014, T015 (local cleanup in parallel)

## Next Step

Run `/brains.implement` to begin implementation.
