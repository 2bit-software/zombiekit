# Technical Requirements: gRPC Service Contracts

**User Preference**: Use buf ecosystem for proto management and code generation.

## Tooling Setup

### Required Tools

```bash
# Buf CLI
go install github.com/bufbuild/buf/cmd/buf@latest

# Go protobuf plugin
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Connect-go plugin (recommended over grpc-go)
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
```

### Go Dependency

```bash
go get connectrpc.com/connect
```

## Configuration Files

### buf.yaml (repository root)

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

### buf.gen.yaml (repository root)

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

## Directory Structure

```
zombiekit/
├── buf.yaml
├── buf.gen.yaml
├── proto/
│   └── zombiekit/
│       └── brains/
│           ├── common/
│           │   └── v1/
│           │       └── common.proto      # Shared types
│           ├── workflow/
│           │   └── v1/
│           │       └── workflow.proto    # WorkflowService
│           ├── profile/
│           │   └── v1/
│           │       └── profile.proto     # ProfileService
│           ├── search/
│           │   └── v1/
│           │       └── search.proto      # SearchService
│           ├── artifact/
│           │   └── v1/
│           │       └── artifact.proto    # ArtifactService
│           ├── config/
│           │   └── v1/
│           │       └── config.proto      # ConfigService
│           └── llm/
│               └── v1/
│                   └── llm.proto         # LLMService
└── gen/
    └── zombiekit/
        └── brains/
            └── <service>/
                └── v1/
                    ├── <service>.pb.go
                    └── <service>v1connect/
                        └── <service>.connect.go
```

## Proto Package Naming

Format: `zombiekit.brains.<service>.v1`

Examples:
- `zombiekit.brains.common.v1`
- `zombiekit.brains.workflow.v1`
- `zombiekit.brains.profile.v1`

## Build Commands

```bash
# Generate Go code from protos
buf generate

# Lint proto files
buf lint

# Check for breaking changes (against main branch)
buf breaking --against '.git#branch=main'

# Build/validate protos only
buf build
```

## Reserved Field Numbers

Reserve 10-19 in entity messages for future multi-tenant fields:

```protobuf
message Initiative {
  string id = 1;
  // ... other fields

  reserved 10 to 19;
  reserved "tenant_id", "organization_id";
}
```

## Connect-go vs gRPC-go

Using connect-go because:

1. **Multi-protocol support**: Handles gRPC, gRPC-Web, and Connect protocols from single implementation
2. **Standard library compatible**: Uses `net/http` handlers; existing middleware works
3. **Simpler API**: Less boilerplate than grpc-go
4. **JSON debugging**: Connect protocol uses human-readable JSON

## Streaming Implementation Notes

For `SubscribeProfileUpdates` and `SubscribeConfigUpdates`:

- Use server-streaming RPCs (`returns (stream EventType)`)
- Client maintains long-lived connection
- Server pushes events as they occur
- Consider heartbeat/keepalive for connection health
