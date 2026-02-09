# Progress Log: gRPC Service Contracts

## Prerequisites
- [x] buf CLI installed (v1.45.0)
- [x] protoc-gen-go installed (v1.36.5)
- [x] protoc-gen-connect-go installed (via go install)
- [x] connectrpc.com/connect added to go.mod

---

## T001 - Create buf.yaml
- Status: Complete
- Files: buf.yaml
- Notes: v2 config with proto module, STANDARD lint, FILE breaking

## T002 - Create buf.gen.yaml
- Status: Complete
- Files: buf.gen.yaml
- Notes: v2 config with protoc-gen-go and protoc-gen-connect-go plugins

## T003 - Create common.proto
- Status: Complete
- Files: proto/zombiekit/brains/common/v1/common.proto
- Notes: PageRequest, PageResponse, AuditIssue, ComplexityResult

## T004 - Create workflow.proto
- Status: Complete
- Files: proto/zombiekit/brains/workflow/v1/workflow.proto
- Notes: WorkflowService with 5 RPCs, Initiative, WorkflowStep messages

## T005 - Create profile.proto
- Status: Complete
- Files: proto/zombiekit/brains/profile/v1/profile.proto
- Notes: ProfileService with 5 RPCs (4 MVP + 1 streaming followup)

## T006 - Create search.proto
- Status: Complete
- Files: proto/zombiekit/brains/search/v1/search.proto
- Notes: SearchService with 3 RPCs, Conversation, ConversationChunk

## T007 - Create artifact.proto
- Status: Complete
- Files: proto/zombiekit/brains/artifact/v1/artifact.proto
- Notes: ArtifactService with 3 RPCs

## T008 - Create config.proto
- Status: Complete
- Files: proto/zombiekit/brains/config/v1/config.proto
- Notes: ConfigService with 3 RPCs (2 MVP + 1 streaming followup)

## T009 - Create llm.proto
- Status: Complete
- Files: proto/zombiekit/brains/llm/v1/llm.proto
- Notes: LLMService with 2 RPCs (both streaming, implementation deferred)

## T010 - Run buf lint
- Status: Complete
- Command: `buf lint`
- Notes: All STANDARD lint rules pass (after fixing response naming)

## T011 - Run buf generate
- Status: Complete
- Command: `buf generate`
- Notes: 13 Go files generated in gen/ directory

## T012 - Verify Go code compiles
- Status: Complete
- Command: `go build ./gen/...`
- Notes: Exit code 0, no compile errors

## T013 - Manual review of conventions
- Status: Complete
- Checks:
  - [x] All 21 request messages have `request_id` field
  - [x] All timestamps use `google.protobuf.Timestamp` (13 usages)
  - [x] All 7 key entities have `reserved 10 to 19`
    - Initiative, WorkflowStep, Profile, Conversation, ConversationChunk, Artifact, Config

---

## Summary

**Files Created:**
- buf.yaml
- buf.gen.yaml
- proto/zombiekit/brains/common/v1/common.proto
- proto/zombiekit/brains/workflow/v1/workflow.proto
- proto/zombiekit/brains/profile/v1/profile.proto
- proto/zombiekit/brains/search/v1/search.proto
- proto/zombiekit/brains/artifact/v1/artifact.proto
- proto/zombiekit/brains/config/v1/config.proto
- proto/zombiekit/brains/llm/v1/llm.proto

**Files Generated:**
- gen/zombiekit/brains/*/v1/*.pb.go (7 files)
- gen/zombiekit/brains/*/v1/*connect/*.connect.go (6 files)

**All success criteria met:**
- SC-001: `buf lint` passes with STANDARD rules
- SC-002: `buf generate` produces valid Go code that compiles
- SC-003: All 5 core services have request/response RPCs defined
- SC-004: `buf breaking` baseline established (no prior version)
- SC-005: Streaming and LLM contracts defined
