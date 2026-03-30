# Implementation Plan: gRPC Service Contracts

**Status**: Ready for implementation
**Estimated Complexity**: Low-Medium (proto authoring, no runtime logic)

## Overview

This plan covers creating proto files and buf configuration for the gRPC service contracts. No spikes needed - buf and connect-go are well-documented, and we're defining contracts only (no server/client implementation).

## Prerequisites

Before starting implementation:

1. Install required tools:
   ```bash
   go install github.com/bufbuild/buf/cmd/buf@latest
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
   ```

2. Add Go dependency:
   ```bash
   go get connectrpc.com/connect
   ```

## Implementation Steps

### Phase 1: Buf Configuration (Steps 1-2)

#### Step 1: Create buf.yaml

**File**: `buf.yaml` (repository root)

```yaml
version: v2
modules:
  - path: proto
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

**Verification**: `buf config ls-modules` shows the proto module

---

#### Step 2: Create buf.gen.yaml

**File**: `buf.gen.yaml` (repository root)

```yaml
version: v2
plugins:
  - local: protoc-gen-go
    out: gen
    opt: paths=source_relative
  - local: protoc-gen-connect-go
    out: gen
    opt: paths=source_relative
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/2bit-software/zombiekit/gen
```

**Verification**: File exists and is valid YAML

---

### Phase 2: Common Types (Step 3)

#### Step 3: Create common.proto

**File**: `proto/zombiekit/brains/common/v1/common.proto`

Shared types used across services:
- Pagination (PageRequest, PageResponse)
- Timestamps (use google.protobuf.Timestamp)
- Error details (if needed)

**Verification**: `buf lint` passes

---

### Phase 3: Core Service Protos (Steps 4-8)

#### Step 4: Create workflow.proto (WorkflowService)

**File**: `proto/zombiekit/brains/workflow/v1/workflow.proto`

RPCs:
- `CreateInitiative(CreateInitiativeRequest) returns (CreateInitiativeResponse)`
- `GetStatus(GetStatusRequest) returns (GetStatusResponse)`
- `UpdateStep(UpdateStepRequest) returns (UpdateStepResponse)`
- `CompleteInitiative(CompleteInitiativeRequest) returns (CompleteInitiativeResponse)`
- `ListInitiatives(ListInitiativesRequest) returns (ListInitiativesResponse)`

Messages: Initiative, WorkflowStep, InitiativeType enum, StepStatus enum

**Verification**: `buf lint` passes, `buf generate` produces Go code

---

#### Step 5: Create profile.proto (ProfileService)

**File**: `proto/zombiekit/brains/profile/v1/profile.proto`

RPCs (MVP):
- `ComposeProfile(ComposeProfileRequest) returns (ComposeProfileResponse)`
- `ListProfiles(ListProfilesRequest) returns (ListProfilesResponse)`
- `GetProfile(GetProfileRequest) returns (GetProfileResponse)`
- `SaveProfile(SaveProfileRequest) returns (SaveProfileResponse)`

RPCs (Followup - define now, implement later):
- `SubscribeProfileUpdates(SubscribeProfileUpdatesRequest) returns (stream ProfileUpdateEvent)`

Messages: Profile, ProfileLocation enum, ProfileUpdateEvent, EventType enum

**Verification**: `buf lint` passes, `buf generate` produces Go code

---

#### Step 6: Create search.proto (SearchService)

**File**: `proto/zombiekit/brains/search/v1/search.proto`

RPCs:
- `Search(SearchRequest) returns (SearchResponse)`
- `GetConversation(GetConversationRequest) returns (GetConversationResponse)`
- `ListConversations(ListConversationsRequest) returns (ListConversationsResponse)`

Messages: Conversation, ConversationChunk, SearchResult

**Verification**: `buf lint` passes, `buf generate` produces Go code

---

#### Step 7: Create artifact.proto (ArtifactService)

**File**: `proto/zombiekit/brains/artifact/v1/artifact.proto`

RPCs:
- `GetArtifact(GetArtifactRequest) returns (GetArtifactResponse)`
- `SaveArtifact(SaveArtifactRequest) returns (SaveArtifactResponse)`
- `ListArtifacts(ListArtifactsRequest) returns (ListArtifactsResponse)`

Messages: Artifact

**Verification**: `buf lint` passes, `buf generate` produces Go code

---

#### Step 8: Create config.proto (ConfigService)

**File**: `proto/zombiekit/brains/config/v1/config.proto`

RPCs (MVP):
- `GetConfig(GetConfigRequest) returns (GetConfigResponse)`
- `UpdateConfig(UpdateConfigRequest) returns (UpdateConfigResponse)`

RPCs (Followup - define now, implement later):
- `SubscribeConfigUpdates(SubscribeConfigUpdatesRequest) returns (stream ConfigUpdateEvent)`

Messages: Config, ConfigUpdateEvent

**Verification**: `buf lint` passes, `buf generate` produces Go code

---

### Phase 4: Followup Service Proto (Step 9)

#### Step 9: Create llm.proto (LLMService)

**File**: `proto/zombiekit/brains/llm/v1/llm.proto`

RPCs (define now, implement in followup ticket):
- `Complete(CompletionRequest) returns (CompletionResponse)`
- `CompleteStream(CompletionRequest) returns (stream CompletionChunk)`

Messages: Message, CompletionRequest, CompletionResponse, CompletionChunk, Usage, Role enum

**Verification**: `buf lint` passes, `buf generate` produces Go code

---

### Phase 5: Verification (Steps 10-11)

#### Step 10: Run full lint and generate

```bash
buf lint
buf generate
go build ./gen/...
```

All commands must pass.

---

#### Step 11: Update .gitignore (if needed)

Ensure `gen/` is tracked in git (generated code should be committed for this project to enable go get without buf).

---

## Dependency Order

```
Step 1 (buf.yaml)
    ↓
Step 2 (buf.gen.yaml)
    ↓
Step 3 (common.proto) ─────────────────────┐
    ↓                                      │
Steps 4-9 (service protos, can parallelize)│
    ↓                                      │
Step 10 (verification) ←───────────────────┘
    ↓
Step 11 (gitignore)
```

## Files Created

| File | Purpose |
|------|---------|
| `buf.yaml` | Buf module configuration |
| `buf.gen.yaml` | Code generation configuration |
| `proto/zombiekit/brains/common/v1/common.proto` | Shared types |
| `proto/zombiekit/brains/workflow/v1/workflow.proto` | WorkflowService |
| `proto/zombiekit/brains/profile/v1/profile.proto` | ProfileService |
| `proto/zombiekit/brains/search/v1/search.proto` | SearchService |
| `proto/zombiekit/brains/artifact/v1/artifact.proto` | ArtifactService |
| `proto/zombiekit/brains/config/v1/config.proto` | ConfigService |
| `proto/zombiekit/brains/llm/v1/llm.proto` | LLMService |

## Success Criteria

- [ ] `buf lint` passes with no errors
- [ ] `buf generate` produces Go code in `gen/`
- [ ] `go build ./gen/...` compiles successfully
- [ ] All 6 services have proto definitions
- [ ] All request messages have `request_id` field
- [ ] Key entities have reserved fields 10-19
- [ ] Timestamps use `google.protobuf.Timestamp`
