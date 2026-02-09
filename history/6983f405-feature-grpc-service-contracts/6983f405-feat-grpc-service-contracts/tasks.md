# Tasks: gRPC Service Contracts

**Total Tasks**: 13
**Parallel Opportunities**: Tasks T004-T009 can run in parallel after T003
**Complexity**: Medium (9 files)

## Dependency Graph

```
T001 → T002 → T003 ──┬── T004 [P]
                     ├── T005 [P]
                     ├── T006 [P]
                     ├── T007 [P]
                     ├── T008 [P]
                     └── T009 [P]
                           ↓
                         T010 → T011 → T012 → T013
```

## Prerequisites (Manual - Before Starting)

```bash
# Install required tools
go install github.com/bufbuild/buf/cmd/buf@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

# Add Go dependency
go get connectrpc.com/connect
```

---

## Phase 1: Buf Configuration

- [ ] **T001** [FR-006] Create buf.yaml at repository root
  - File: `buf.yaml`
  - Content: v2 config with proto module, STANDARD lint, FILE breaking
  - Acceptance: File exists, valid YAML

- [ ] **T002** [FR-006] Create buf.gen.yaml at repository root
  - File: `buf.gen.yaml`
  - Content: v2 config with protoc-gen-go and protoc-gen-connect-go plugins
  - Acceptance: File exists, valid YAML
  - Depends: T001

---

## Phase 2: Common Types

- [ ] **T003** [FR-007] Create common.proto with shared types
  - File: `proto/zombiekit/brains/common/v1/common.proto`
  - Content: PageRequest, PageResponse, AuditIssue, ComplexityResult
  - Acceptance: `buf lint` passes
  - Depends: T002

---

## Phase 3: Core Service Protos (Parallelizable)

- [ ] **T004** [P] [FR-001] Create workflow.proto (WorkflowService)
  - File: `proto/zombiekit/brains/workflow/v1/workflow.proto`
  - RPCs: CreateInitiative, GetStatus, UpdateStep, CompleteInitiative, ListInitiatives
  - Messages: Initiative, WorkflowStep, enums
  - Acceptance: `buf lint` passes
  - Depends: T003

- [ ] **T005** [P] [FR-002, FR-010] Create profile.proto (ProfileService)
  - File: `proto/zombiekit/brains/profile/v1/profile.proto`
  - RPCs: ComposeProfile, ListProfiles, GetProfile, SaveProfile, SubscribeProfileUpdates (streaming)
  - Messages: Profile, ProfileUpdateEvent, enums
  - Acceptance: `buf lint` passes
  - Depends: T003

- [ ] **T006** [P] [FR-003] Create search.proto (SearchService)
  - File: `proto/zombiekit/brains/search/v1/search.proto`
  - RPCs: Search, GetConversation, ListConversations
  - Messages: Conversation, ConversationChunk, SearchResult
  - Acceptance: `buf lint` passes
  - Depends: T003

- [ ] **T007** [P] [FR-004] Create artifact.proto (ArtifactService)
  - File: `proto/zombiekit/brains/artifact/v1/artifact.proto`
  - RPCs: GetArtifact, SaveArtifact, ListArtifacts
  - Messages: Artifact
  - Acceptance: `buf lint` passes
  - Depends: T003

- [ ] **T008** [P] [FR-005, FR-011] Create config.proto (ConfigService)
  - File: `proto/zombiekit/brains/config/v1/config.proto`
  - RPCs: GetConfig, UpdateConfig, SubscribeConfigUpdates (streaming)
  - Messages: Config, ConfigUpdateEvent
  - Acceptance: `buf lint` passes
  - Depends: T003

- [ ] **T009** [P] [FR-012] Create llm.proto (LLMService)
  - File: `proto/zombiekit/brains/llm/v1/llm.proto`
  - RPCs: Complete, CompleteStream (both streaming followup)
  - Messages: Message, CompletionRequest, CompletionResponse, CompletionChunk, Usage
  - Acceptance: `buf lint` passes
  - Depends: T003

---

## Phase 4: Verification

- [ ] **T010** Run buf lint on all proto files
  - Command: `buf lint`
  - Acceptance: Exit code 0, no errors
  - Depends: T004-T009

- [ ] **T011** Run buf generate to produce Go code
  - Command: `buf generate`
  - Acceptance: Go files generated in `gen/` directory
  - Depends: T010

- [ ] **T012** Verify generated Go code compiles
  - Command: `go build ./gen/...`
  - Acceptance: Exit code 0, no compile errors
  - Depends: T011

- [ ] **T013** [FR-008, FR-009] Manual review of proto conventions
  - Check: All request messages have `request_id` field
  - Check: All timestamps use `google.protobuf.Timestamp`
  - Check: Key entities have `reserved 10 to 19`
  - Acceptance: All conventions followed
  - Depends: T011

---

## FR Traceability

| FR | Task(s) | Description |
|----|---------|-------------|
| FR-001 | T004 | WorkflowService RPCs |
| FR-002 | T005 | ProfileService RPCs |
| FR-003 | T006 | SearchService RPCs |
| FR-004 | T007 | ArtifactService RPCs |
| FR-005 | T008 | ConfigService RPCs |
| FR-006 | T001, T002 | Package naming via buf config |
| FR-007 | T003, T013 | Timestamp usage |
| FR-008 | T013 | request_id traceability |
| FR-009 | T013 | Reserved fields |
| FR-010 | T005 | ProfileService streaming |
| FR-011 | T008 | ConfigService streaming |
| FR-012 | T009 | LLMService |

---

## Execution Order

**Sequential path (critical)**:
1. T001 → T002 → T003 → T010 → T011 → T012 → T013

**Parallel batch** (after T003):
- T004, T005, T006, T007, T008, T009 can all run simultaneously

**Optimal execution**:
1. T001, T002, T003 (sequential setup)
2. T004-T009 (parallel proto creation)
3. T010, T011, T012, T013 (sequential verification)
