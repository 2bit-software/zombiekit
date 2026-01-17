# Tasks: Profile Source Abstraction

**Input**: Design documents from `/specs/004-source-interface/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, quickstart.md

**Tests**: Not explicitly requested in spec. Test tasks omitted.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Go CLI application at repository root
- Source code in `internal/` following existing structure
- CLI commands in `internal/cli/`
- Profile logic in `internal/profile/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and interface definition

- [X] T001 Define ProfileSource interface in internal/profile/source.go
- [X] T002 Define SourceType enum and factory function in internal/profile/source.go
- [X] T003 [P] Add --source flag to profile command group in internal/cli/profile.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core implementations that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Create BrainsSource implementation wrapping existing Resolver in internal/profile/brains_source.go
- [X] T005 [P] Create ClaudeFrontmatter struct and parsing in internal/profile/claude_frontmatter.go
- [X] T006 Create ClaudeSource implementation in internal/profile/claude_source.go
- [X] T007 Extend Profile struct with Model and Color fields in internal/profile/types.go
- [X] T008 Refactor Service to accept ProfileSource interface in internal/profile/service.go
- [X] T009 Create NewSource factory function in internal/profile/source.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Use Brains Profiles (Default Behavior) (Priority: P1) 🎯 MVP

**Goal**: Preserve backward compatibility - all existing profile commands work unchanged without --source flag

**Independent Test**: Run `brains profile list` and `brains profile compose research` without any `--source` flag and verify behavior matches the current implementation.

### Implementation for User Story 1

- [X] T010 [US1] Wire BrainsSource as default in profile list command in internal/cli/profile.go
- [X] T011 [US1] Wire BrainsSource as default in profile show command in internal/cli/profile.go
- [X] T012 [US1] Wire BrainsSource as default in profile compose command in internal/cli/profile.go
- [X] T013 [US1] Wire BrainsSource as default in profile create command in internal/cli/profile.go
- [X] T014 [US1] Wire BrainsSource as default in profile validate command in internal/cli/profile.go
- [X] T015 [US1] Verify all existing behavior works without --source flag

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - List Claude Agents as Profiles (Priority: P1)

**Goal**: Users can list Claude agents from `.claude/agents/` directories with `--source claude`

**Independent Test**: Create sample agents in `.claude/agents/` and `~/.claude/agents/`, run `brains profile list --source claude`, and verify all agents are shown with correct metadata.

### Implementation for User Story 2

- [X] T016 [US2] Implement FindProfileDirs in ClaudeSource for local/global resolution in internal/profile/claude_source.go
- [X] T017 [US2] Implement LoadProfiles in ClaudeSource with claude frontmatter parsing in internal/profile/claude_source.go
- [X] T018 [US2] Wire ClaudeSource to profile list command when --source claude in internal/cli/profile.go
- [X] T019 [US2] Include Model and Color fields in JSON output for list command in internal/cli/profile.go
- [X] T020 [US2] Handle local > global precedence when names conflict in internal/profile/claude_source.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Show Claude Agent Details (Priority: P1)

**Goal**: Users can inspect full content of a specific Claude agent

**Independent Test**: Create an agent with known content, run `brains profile show agent-name --source claude`, and verify the full content is displayed correctly.

### Implementation for User Story 3

- [X] T021 [US3] Implement GetInheritanceChain in ClaudeSource (simpler than brains - no parent traversal) in internal/profile/claude_source.go
- [X] T022 [US3] Wire ClaudeSource to profile show command when --source claude in internal/cli/profile.go
- [X] T023 [US3] Support --raw flag for Claude agents in internal/cli/profile.go
- [X] T024 [US3] Add error with suggestions when agent not found in internal/profile/claude_source.go

**Checkpoint**: At this point, User Stories 1, 2, AND 3 should all work independently

---

## Phase 6: User Story 5 - Create New Claude Agent (Priority: P1)

**Goal**: Users can create new Claude agents with proper structure and frontmatter template

**Independent Test**: Run `brains profile create my-agent --source claude` and verify the file is created at `.claude/agents/my-agent.md` with valid Claude agent frontmatter.

### Implementation for User Story 5

- [X] T025 [US5] Implement CreateProfile in ClaudeSource with Claude frontmatter template in internal/profile/claude_source.go
- [X] T026 [US5] Wire ClaudeSource to profile create command when --source claude in internal/cli/profile.go
- [X] T027 [US5] Support --global flag for creating in ~/.claude/agents/ in internal/profile/claude_source.go
- [X] T028 [US5] Return error if directory doesn't exist (suggest init command) in internal/profile/claude_source.go

**Checkpoint**: At this point, User Stories 1, 2, 3, AND 5 should all work independently

---

## Phase 7: User Story 4 - Compose Claude Agents (Priority: P2)

**Goal**: Users can compose multiple Claude agents together

**Independent Test**: Create multiple agents, run `brains profile compose agent1,agent2 --source claude`, and verify the combined output.

### Implementation for User Story 4

- [X] T029 [US4] Implement LoadAllProfiles in ClaudeSource in internal/profile/claude_source.go
- [X] T030 [US4] Wire ClaudeSource to profile compose command when --source claude in internal/cli/profile.go
- [X] T031 [US4] Handle includes field for agent composition in internal/profile/claude_source.go
- [X] T032 [US4] Ensure duplicate includes appear only once in internal/profile/service.go

**Checkpoint**: At this point, User Stories 1-5 should all work independently

---

## Phase 8: User Story 6 - Validate Claude Agents (Priority: P2)

**Goal**: Users can check Claude agent configurations for errors

**Independent Test**: Create agents with intentional errors (invalid YAML, missing fields) and verify validate detects them.

### Implementation for User Story 6

- [X] T033 [US6] Wire ClaudeSource to profile validate command when --source claude in internal/cli/profile.go
- [X] T034 [US6] Detect circular includes in Claude agents in internal/profile/claude_source.go
- [X] T035 [US6] Report YAML parse errors with file paths in internal/profile/claude_source.go
- [X] T036 [US6] Report success when all agents valid in internal/cli/profile.go

**Checkpoint**: At this point, User Stories 1-6 should all work independently

---

## Phase 9: User Story 7 - Initialize Claude Agents Directory (Priority: P2)

**Goal**: Users can set up the Claude agents directory structure

**Independent Test**: Run init in a new directory and verify `.claude/agents/` is created.

### Implementation for User Story 7

- [X] T037 [US7] Implement GetInitDir in ClaudeSource in internal/profile/claude_source.go
- [X] T038 [US7] Add --source flag to init command in internal/cli/init.go
- [X] T039 [US7] Create .claude/agents/ for local init with --source claude in internal/cli/init.go
- [X] T040 [US7] Create ~/.claude/agents/ for global init with --source claude in internal/cli/init.go
- [X] T041 [US7] Handle case where directory already exists gracefully in internal/cli/init.go

**Checkpoint**: All user stories should now be independently functional

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T042 [P] Ensure consistent error messages include source name across all operations
- [X] T043 [P] Update help text for all profile commands to document --source flag
- [X] T044 Run quickstart.md validation scenarios manually

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2)
- **Polish (Phase 10)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Independently testable
- **User Story 3 (P1)**: Can start after Foundational (Phase 2) - Independently testable
- **User Story 5 (P1)**: Can start after Foundational (Phase 2) - Independently testable
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - May use US3 for inheritance chain
- **User Story 6 (P2)**: Can start after Foundational (Phase 2) - Independently testable
- **User Story 7 (P2)**: Can start after Foundational (Phase 2) - Independently testable

### Within Each User Story

- Core interface implementation before command wiring
- Command wiring before edge case handling
- Story complete before moving to next priority

### Parallel Opportunities

- T003 (--source flag) can run in parallel with T001, T002
- T004, T005 can run in parallel (different files)
- T010-T014 can run in parallel (different commands, same pattern)
- T016, T017, T020 can run in parallel with T018, T019 (implementation vs wiring)
- All [P] marked tasks can run in parallel within their phase

---

## Parallel Example: Foundational Phase

```bash
# Launch these two tasks in parallel (different files):
Task: "Create BrainsSource implementation wrapping existing Resolver in internal/profile/brains_source.go"
Task: "Create ClaudeFrontmatter struct and parsing in internal/profile/claude_frontmatter.go"
```

---

## Parallel Example: User Story 1

```bash
# Launch all command wiring tasks in parallel (same pattern, different commands):
Task: "Wire BrainsSource as default in profile list command in internal/cli/profile.go"
Task: "Wire BrainsSource as default in profile show command in internal/cli/profile.go"
Task: "Wire BrainsSource as default in profile compose command in internal/cli/profile.go"
Task: "Wire BrainsSource as default in profile create command in internal/cli/profile.go"
Task: "Wire BrainsSource as default in profile validate command in internal/cli/profile.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (backward compatibility)
4. **STOP and VALIDATE**: Run existing brains profile commands without --source
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP - backward compat!)
3. Add User Story 2 → List Claude agents → Deploy/Demo
4. Add User Story 3 → Show Claude agents → Deploy/Demo
5. Add User Story 5 → Create Claude agents → Deploy/Demo
6. Add User Story 4 → Compose Claude agents → Deploy/Demo
7. Add User Story 6 → Validate Claude agents → Deploy/Demo
8. Add User Story 7 → Init Claude directory → Deploy/Demo
9. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (critical - backward compat)
   - Developer B: User Story 2 + 3 (Claude read operations)
   - Developer C: User Story 5 + 7 (Claude write operations)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- User Story 5 is listed before User Story 4 due to P1 priority
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
