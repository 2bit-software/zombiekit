# Feature Specification: gRPC Service Contracts (Protobuf)

**Feature Branch**: `morganhein/dev-110-define-grpc-service-contracts-protobuf`
**Created**: 2026-02-04
**Status**: Draft
**Source**: [DEV-110](https://linear.app/heinsight/issue/DEV-110/define-grpc-service-contracts-protobuf)
**Parent**: [DEV-109](https://linear.app/heinsight/issue/DEV-109/architecture-client-server-split-zk-server-local-proxy) - Client-Server Split Architecture

## Scope

This ticket defines **all** gRPC service contracts (proto files) for the client-server split. However, implementation is phased:

### MVP Scope (This Ticket)
Define and generate code for core request/response RPCs that enable the local proxy to function:
- WorkflowService (initiative lifecycle)
- ProfileService (profile composition - **excluding** streaming)
- SearchService (conversation search)
- ArtifactService (initiative file storage)
- ConfigService (config management - **excluding** streaming)

### Followup Scope (Separate Tickets)
Contracts defined here but implementation deferred:
- **Streaming RPCs** → DEV-??? "Profile caching + gRPC streaming push"
  - `ProfileService.SubscribeProfileUpdates`
  - `ConfigService.SubscribeConfigUpdates`
- **LLMService** → DEV-??? "LLM proxy for tertiary calls"
  - `Complete`, `CompleteStream`

## User Scenarios & Testing

### User Story 1 - Proto files compile and generate valid Go code (Priority: P1)

As a developer, I need the proto files to compile cleanly and generate valid Go code so that I can build services against these contracts.

**Why this priority**: This is the foundational requirement. Without compiling protos and valid generated code, nothing else works.

**Independent Test**: Run `buf generate` and `go build ./gen/...` - both must succeed.

**Acceptance Scenarios**:

1. **Given** proto files in `proto/` directory, **When** I run `buf generate`, **Then** Go code is generated in `gen/` without errors
2. **Given** generated Go code in `gen/`, **When** I run `go build ./gen/...`, **Then** the code compiles successfully
3. **Given** proto files, **When** I run `buf lint`, **Then** all STANDARD lint rules pass

---

### User Story 2 - Service contracts cover existing MCP functionality (Priority: P1)

As a developer, I need gRPC service definitions that match the existing MCP tool capabilities so that the gRPC layer can replace MCP for server communication.

**Why this priority**: The contracts must support all existing functionality; otherwise the migration would lose features.

**Independent Test**: Compare MCP tool methods against gRPC service RPCs - every MCP operation must have a corresponding RPC.

**Acceptance Scenarios**:

1. **Given** the existing `initiative` MCP tool, **When** I examine WorkflowService, **Then** I find RPCs for CreateInitiative, GetStatus, UpdateStep, CompleteInitiative, ListInitiatives
2. **Given** the existing `profile` MCP tool, **When** I examine ProfileService, **Then** I find RPCs for ComposeProfile, ListProfiles, GetProfile, SaveProfile
3. **Given** the existing `recall` MCP tool, **When** I examine SearchService, **Then** I find RPCs for Search, GetConversation, ListConversations
4. **Given** ArtifactService, **When** I examine the proto, **Then** I find RPCs for GetArtifact, SaveArtifact, ListArtifacts
5. **Given** ConfigService, **When** I examine the proto, **Then** I find RPCs for GetConfig, UpdateConfig

---

### User Story 3 - Streaming RPCs for real-time updates (Priority: FOLLOWUP)

> **Note**: Contract defined in this ticket; implementation deferred to "Profile caching + gRPC streaming push" sub-ticket.

As a client, I need streaming RPCs for profile and config updates so that I can receive push notifications when these change on the server.

**Why followup**: Streaming enables real-time sync without polling, but basic request/response functionality works without it. The proxy can poll initially.

**Acceptance Scenarios**:

1. **Given** ProfileService, **When** I examine the proto, **Then** I find `SubscribeProfileUpdates` returning `stream ProfileUpdateEvent`
2. **Given** ConfigService, **When** I examine the proto, **Then** I find `SubscribeConfigUpdates` returning `stream ConfigUpdateEvent`

---

### User Story 4 - LLM proxy service for tertiary calls (Priority: FOLLOWUP)

> **Note**: Contract defined in this ticket; implementation deferred to "LLM proxy" sub-ticket.

As a local agent, I need an LLMService so that I can route LLM completion requests through the central server (for billing, rate limiting, model selection).

**Why followup**: Centralizing LLM calls is a key architectural goal, but the proxy functions without it. LLM calls are described as "tertiary needs" in DEV-109.

**Acceptance Scenarios**:

1. **Given** LLMService, **When** I examine the proto, **Then** I find `Complete` RPC with CompletionRequest (model, messages[], options) returning CompletionResponse (content, usage, finish_reason)
2. **Given** LLMService, **When** I examine the proto, **Then** I find `CompleteStream` RPC returning `stream CompletionChunk` with delta content and usage info

---

### User Story 5 - Forward-compatible message design (Priority: P3)

As a maintainer, I need message types that can evolve without breaking clients so that we can add multi-tenancy later.

**Why this priority**: Future-proofing is good practice but not blocking for initial implementation.

**Independent Test**: Verify reserved field numbers in key messages.

**Acceptance Scenarios**:

1. **Given** the Initiative message, **When** I examine the proto, **Then** I find reserved field numbers 10-19 for future use
2. **Given** any request message, **When** I examine the proto, **Then** I find a `request_id` field for traceability

---

### Edge Cases

- What happens when a client sends an unknown field? (Proto3 preserves unknown fields)
- How does the system handle deprecated RPCs? (Mark with `deprecated = true` option)
- What happens when streaming connection drops? (Client must reconnect; server should support resume tokens if needed)

## Requirements

### Functional Requirements

**MVP - Core Services (request/response)**

- **FR-001**: System MUST define WorkflowService with CreateInitiative, GetStatus, UpdateStep, CompleteInitiative, ListInitiatives RPCs
- **FR-002**: System MUST define ProfileService with ComposeProfile, ListProfiles, GetProfile, SaveProfile RPCs
- **FR-003**: System MUST define SearchService with Search, GetConversation, ListConversations RPCs
- **FR-004**: System MUST define ArtifactService with GetArtifact, SaveArtifact, ListArtifacts RPCs
- **FR-005**: System MUST define ConfigService with GetConfig, UpdateConfig RPCs

**MVP - Standards**

- **FR-006**: All proto packages MUST use `zombiekit.brains.<service>.v1` naming
- **FR-007**: All messages MUST use `google.protobuf.Timestamp` for date fields
- **FR-008**: All request messages MUST include a `request_id` string field for traceability
- **FR-009**: Key entity messages MUST reserve field numbers 10-19 for future multi-tenant fields

**Followup - Streaming (define contract, defer implementation)**

- **FR-010**: ProfileService MUST define SubscribeProfileUpdates streaming RPC
- **FR-011**: ConfigService MUST define SubscribeConfigUpdates streaming RPC

**Followup - LLM Proxy (define contract, defer implementation)**

- **FR-012**: System MUST define LLMService with Complete, CompleteStream RPCs

### Key Entities

- **Initiative**: Represents a unit of work (feature, bug, refactor)
  - Fields: `id`, `name`, `type` (feature/bug/refactor), `status`, `steps[]`, `created_at`, `updated_at`
  - Reserved: 10-19 for tenant fields

- **WorkflowStep**: A step within an initiative's workflow
  - Fields: `name`, `status` (pending/in_progress/completed), `updated_at`

- **Profile**: A composable prompt template
  - Fields: `name`, `content`, `domains[]`, `dependencies[]`, `location` (local/global)
  - Reserved: 10-19 for tenant fields

- **Conversation**: A collection of conversation chunks
  - Fields: `id`, `project`, `created_at`, `updated_at`, `summary`, `total_chunks`

- **ConversationChunk**: A segment of conversation text
  - Fields: `id`, `conversation_id`, `content`, `created_at`, `sequence`

- **Artifact**: A file associated with an initiative
  - Fields: `initiative_id`, `path`, `content`, `created_at`, `updated_at`
  - Reserved: 10-19 for tenant fields

- **Config**: Configuration key-value pairs
  - Fields: `key`, `value`, `updated_at`

### Shared Types

- **Pagination**: `page_token` (string), `page_size` (int32) for list operations; responses include `next_page_token`

- **AuditIssue**: Issue from spec/code audits
  - Fields: `severity` (CRITICAL/MAJOR/MINOR), `message`, `file_path`, `line_number`, `suggestion`

- **ComplexityResult**: Result from complexity analysis
  - Fields: `score`, `factors[]` (each with name, value, weight), `recommendation`

### Streaming Event Types

- **ProfileUpdateEvent**: Server-push when profile changes
  - Fields: `event_type` (CREATED/UPDATED/DELETED), `profile_name`, `timestamp`

- **ConfigUpdateEvent**: Server-push when config changes
  - Fields: `event_type` (UPDATED), `key`, `value`, `timestamp`

### LLM Types

- **Message**: A chat message for LLM completion
  - Fields: `role` (system/user/assistant), `content`

- **CompletionRequest**: Request for LLM completion
  - Fields: `model`, `messages[]`, `max_tokens`, `temperature`, `request_id`

- **CompletionResponse**: Response from LLM completion
  - Fields: `content`, `finish_reason`, `usage` (prompt_tokens, completion_tokens, total_tokens)

- **CompletionChunk**: Streaming chunk for LLM completion
  - Fields: `delta_content`, `finish_reason`, `usage` (only in final chunk)

## Success Criteria

### Measurable Outcomes

- **SC-001**: `buf lint` passes with STANDARD rules on all proto files
- **SC-002**: `buf generate` produces valid Go code that compiles
- **SC-003**: All 5 core services (WorkflowService, ProfileService, SearchService, ArtifactService, ConfigService) have request/response RPCs defined
- **SC-004**: `buf breaking` establishes baseline (no prior version to compare)
- **SC-005**: Streaming and LLM contracts are defined (signatures exist in proto) even though implementation is deferred

## Testing Requirements

### Test Strategy

Integration tests will verify that:
1. Proto files compile with `buf build`
2. Generated Go code compiles with `go build`
3. Service interfaces are correctly generated

No runtime tests at this stage - this is a contract definition ticket. Runtime tests come when implementing the services.

### FR to Test Mapping

**MVP Tests**

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Verify WorkflowService has required RPCs via generated Go interface |
| FR-002 | Integration | Verify ProfileService has required RPCs via generated Go interface |
| FR-003 | Integration | Verify SearchService has required RPCs via generated Go interface |
| FR-004 | Integration | Verify ArtifactService has required RPCs via generated Go interface |
| FR-005 | Integration | Verify ConfigService has required RPCs via generated Go interface |
| FR-006 | Lint | `buf lint` enforces package version suffix |
| FR-007 | Manual | Review proto files for Timestamp usage |
| FR-008 | Manual | Review request messages for request_id field |
| FR-009 | Manual | Review entity messages for reserved fields |

**Followup Tests (contracts only - implementation tested in later tickets)**

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-010 | Integration | Verify ProfileService.SubscribeProfileUpdates signature exists |
| FR-011 | Integration | Verify ConfigService.SubscribeConfigUpdates signature exists |
| FR-012 | Integration | Verify LLMService has Complete, CompleteStream RPCs |

### Edge Case Coverage

- Unknown fields → Proto3 default behavior (preserved)
- Deprecated RPCs → Verify deprecation option compiles
- Empty responses → Use standard empty message patterns

## Followup Tickets

These tickets should be created to track deferred work:

### DEV-??? Profile Caching + gRPC Streaming Push

Implement the streaming RPCs defined in this ticket:
- `ProfileService.SubscribeProfileUpdates`
- `ConfigService.SubscribeConfigUpdates`
- Local profile cache with server-push invalidation
- Reconnection/retry logic for dropped streams

### DEV-??? LLM Proxy Service

Implement the LLM proxy service defined in this ticket:
- `LLMService.Complete` (synchronous completion)
- `LLMService.CompleteStream` (streaming completion)
- Billing/rate limiting integration
- Model selection and routing
