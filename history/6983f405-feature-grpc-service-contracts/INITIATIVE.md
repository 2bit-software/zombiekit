# Initiative: grpc-service-contracts

**Type**: feature
**Status**: completed
**Created**: 2026-02-04
**ID**: 6983f405-feature-grpc-service-contracts

## Cycles

### 1. feat/grpc-service-contracts (completed)

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-02-04 18:15 |
| plan | completed | 2026-02-04 18:32 |
| tasks | completed | 2026-02-04 19:14 |
| implement | completed | 2026-02-04 19:20 |

## Source

**Linear Ticket**: [DEV-110](https://linear.app/heinsight/issue/DEV-110/define-grpc-service-contracts-protobuf)
**Title**: Define gRPC Service Contracts (Protobuf)

## Description

Define the gRPC service contracts that govern all communication between the local proxy and central server. This is the foundational ticket — everything else builds on these contracts.

## Completion

**Completed**: 2026-02-04 19:20
**Duration**: ~2 hours

### Outcomes

**Files Created:**
- `buf.yaml` - Buf v2 configuration (STANDARD lint, FILE breaking)
- `buf.gen.yaml` - Code generation config (protoc-gen-go + connect-go)
- 7 proto files defining service contracts:
  - `proto/zombiekit/brains/common/v1/common.proto` - Shared types
  - `proto/zombiekit/brains/workflow/v1/workflow.proto` - WorkflowService (5 RPCs)
  - `proto/zombiekit/brains/profile/v1/profile.proto` - ProfileService (5 RPCs)
  - `proto/zombiekit/brains/search/v1/search.proto` - SearchService (3 RPCs)
  - `proto/zombiekit/brains/artifact/v1/artifact.proto` - ArtifactService (3 RPCs)
  - `proto/zombiekit/brains/config/v1/config.proto` - ConfigService (3 RPCs)
  - `proto/zombiekit/brains/llm/v1/llm.proto` - LLMService (2 RPCs, deferred)

**Generated Files:**
- 7 `.pb.go` files (protobuf message types)
- 6 `.connect.go` files (Connect-RPC service interfaces)

**Success Criteria Met:**
- SC-001: `buf lint` passes with STANDARD rules
- SC-002: `buf generate` produces valid Go code that compiles
- SC-003: All 5 core services have request/response RPCs defined
- SC-004: `buf breaking` baseline established
- SC-005: Streaming and LLM contracts defined (implementation deferred)

### Notes

All 13 implementation tasks completed successfully. Proto conventions verified:
- 21 request messages with `request_id` field
- All timestamps use `google.protobuf.Timestamp`
- 7 key entities with `reserved 10 to 19` for future multi-tenancy
