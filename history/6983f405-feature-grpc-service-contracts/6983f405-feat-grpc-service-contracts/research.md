---
status: complete
updated: 2026-02-04
---

# Research: gRPC Service Contracts (Protobuf)

## Executive Summary

This feature defines gRPC service contracts for communication between a local proxy and central server. The codebase already has clean service interfaces in Go that map naturally to gRPC services. Using buf with connect-go provides modern tooling (linting, breaking change detection) with full gRPC protocol compatibility.

## Findings

### Codebase Context

**Existing MCP Tool Structure**

The codebase has 6 MCP tool packages that map to gRPC services:

| MCP Tool Package | Maps To | Key Methods |
|-----------------|---------|-------------|
| `internal/mcp/tools/initiative/` | WorkflowService | Create, GetActive, Complete, List |
| `internal/mcp/tools/stickymemory/` | ArtifactService/ConfigService | Get, Set, Delete, List, Search |
| `internal/mcp/tools/recall/` | SearchService | ListConversations, GetConversation |
| `internal/mcp/tools/profile/` | ProfileService | Compose, List, Save |
| `internal/mcp/tools/workflow/` | WorkflowService | Load, List |
| `internal/mcp/tools/codereasoning/` | (future) CodeReasoningService | RecordThought, GetSession |

**Service Interface Pattern**

Services use clean interfaces with backend-agnostic implementations:

```go
// internal/recall/storage.go
type Storage interface {
    Save(ctx context.Context, content string, embedding []float32) (id string, created bool, err error)
    GetByConversation(ctx context.Context, conversationID string) ([]Chunk, error)
    Search(ctx context.Context, embedding []float32, limit int) ([]SearchResult, error)
    ListConversations(ctx context.Context, limit, offset int, project string) ([]ConversationSummary, error)
}
```

**Error Handling Pattern**

```go
type ToolError struct {
    Code    string
    Message string
    Hint    string
}
```

This maps well to gRPC error details.

**Dependencies Already Present**

- `google.golang.org/grpc v1.75.1` (indirect)
- `google.golang.org/protobuf v1.36.10` (indirect)

### Domain Knowledge

**Buf Ecosystem**

- **buf.yaml v2**: Defines modules, lint rules (STANDARD), breaking detection (FILE)
- **buf.gen.yaml v2**: Configures code generation with connect-go plugins
- **STANDARD lint**: Enforces version suffixes, enum prefixes, file naming
- **FILE breaking**: Strictest breaking change detection (recommended)

**Connect-Go vs gRPC-Go**

Connect-go is the recommended choice:
- Supports gRPC, gRPC-Web, AND Connect protocols simultaneously
- Uses standard `net/http` handlers (existing middleware works)
- Simpler API than grpc-go
- Human-readable JSON for debugging

**Proto Package Versioning**

Package format: `organization.product.service.version`
Example: `zombiekit.brains.memory.v1`

Version suffixes:
- `v1` - stable API
- `v1alpha1` - experimental
- `v1beta1` - feature-complete but may change

**Directory Structure**

```
proto/
└── zombiekit/
    └── brains/
        └── <service>/
            └── v1/
                └── <service>.proto
gen/
└── zombiekit/
    └── brains/
        └── <service>/
            └── v1/
                ├── <service>.pb.go
                └── <service>v1connect/
                    └── <service>.connect.go
```

## Decision Points

- [x] **D1**: Proto tooling - Using buf (per user preference)
- [x] **D2**: gRPC framework - Using connect-go (modern, supports all protocols)
- [x] **D3**: Package prefix - `zombiekit.brains.<service>.v1`
- [x] **D4**: Generated code location - `gen/` directory
- [ ] **D5**: Streaming for config/profile updates - Use server-streaming RPCs

## Recommendations

1. **Start with v1 API** - All services get `v1` suffix from day one, enabling future breaking changes via `v2`

2. **Use STANDARD lint** - Enforces best practices (version suffix, enum prefixes, file naming)

3. **Use FILE breaking** - Strictest detection; can relax to WIRE_JSON if needed

4. **Reserve field numbers** - Keep 10-19 reserved for future multi-tenant fields (`tenant_id`, etc.)

5. **Common types package** - Create `zombiekit.brains.common.v1` for shared types (Timestamp, Pagination, etc.)

6. **Service naming** - Follow the ticket: `WorkflowService`, `ProfileService`, `SearchService`, `ArtifactService`, `ConfigService`, `LLMService`

## Sources

- [buf.yaml v2 config - Buf Docs](https://buf.build/docs/configuration/v2/buf-yaml/)
- [buf.gen.yaml v2 config - Buf Docs](https://buf.build/docs/configuration/v2/buf-gen-yaml/)
- [Getting started with Connect-Go](https://connectrpc.com/docs/go/getting-started/)
- [Buf lint rules and categories](https://buf.build/docs/lint/rules/)
- [Buf breaking change detection](https://buf.build/docs/breaking/rules/)
- [Files and packages - Buf Docs](https://buf.build/docs/reference/protobuf-files-and-packages/)
- Linear ticket DEV-110: Define gRPC Service Contracts (Protobuf)
