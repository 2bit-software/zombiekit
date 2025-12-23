# Implementation Plan: Profile Composition System

**Branch**: `003-profiles` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-profiles/spec.md`

## Summary

Implement a hierarchical profile composition system that allows composable prompts with inheritance between local (`.brains/profiles/`) and global (`~/.brains/profiles/`) directories. Profiles are markdown files with optional YAML frontmatter that can include other profiles and inherit from parent directories. The system exposes functionality via both CLI commands and MCP tools.

## Technical Context

**Language/Version**: Go 1.24+ (per go.mod)
**Primary Dependencies**: urfave/cli/v2 (CLI), mark3labs/mcp-go (MCP), gopkg.in/yaml.v3 (YAML parsing)
**Storage**: File-based profiles (.md files), JSON registry (~/.brains/registry.json) with flock
**Testing**: go test with testify/assert
**Target Platform**: Linux/macOS/Windows (cross-platform CLI)
**Project Type**: Single CLI application with MCP server
**Performance Goals**: <1 second for composing 3+ profiles under 50KB total (per SC-001)
**Constraints**: OS-level file locking for registry, no arbitrary recursion depth limits
**Scale/Scope**: Typical usage: 10-50 profiles per project, 1-5 include levels

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution is not yet configured (template placeholders present). Proceeding with standard Go project conventions:
- ✅ Tests alongside implementation (`*_test.go` files)
- ✅ Clear package boundaries (internal/profile, internal/cli, internal/mcp)
- ✅ CLI and MCP interfaces for all functionality

## Project Structure

### Documentation (this feature)

```text
specs/003-profiles/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (CLI and MCP tool schemas)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── profile/
│   ├── service.go           # Core profile operations (exists, needs implementation)
│   ├── service_test.go      # Tests (exists)
│   ├── frontmatter.go       # YAML frontmatter parsing
│   ├── frontmatter_test.go
│   ├── resolver.go          # Directory walking and profile resolution
│   ├── resolver_test.go
│   ├── composer.go          # DAG building and composition
│   ├── composer_test.go
│   ├── registry.go          # Registry file management with flock
│   ├── registry_test.go
│   └── types.go             # Profile, ProfileSource, CompositionResult types
├── cli/
│   ├── profile.go           # profile subcommand (compose, list, show, create, validate)
│   └── profile_test.go
│   ├── init.go              # init command for .brains/ directory
│   └── init_test.go
└── mcp/
    └── tools/
        └── profile/
            ├── tool.go      # MCP tool implementations
            └── tool_test.go
```

**Structure Decision**: Follows existing Go project structure with internal packages. Profile logic in `internal/profile/`, CLI commands in `internal/cli/`, MCP tools in `internal/mcp/tools/profile/`.

## Complexity Tracking

No constitution violations identified.
