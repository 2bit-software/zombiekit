# Implementation Plan: Core Repository Setup

**Branch**: `001-core-repo-setup` | **Date**: 2025-12-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-core-repo-setup/spec.md`

## Summary

Create the foundational repository structure for ZombieKit (brains CLI) including:
- Go module with urfave/cli/v2 CLI skeleton
- Taskfile.yml with build, test, lint, and database automation
- Docker Compose for PostgreSQL with pgvector
- Test harnesses for planned packages (profile, spec, mcp, web, conversation)
- golangci-lint configuration for code quality

## Technical Context

**Language/Version**: Go 1.22+ (per MASTER-DESIGN.md)
**Primary Dependencies**:
- urfave/cli/v2 (CLI framework)
- pgx/v5 (PostgreSQL driver)
- mark3labs/mcp-go (MCP protocol)
- gopkg.in/yaml.v3 (YAML parsing for profiles)

**Storage**: PostgreSQL 16 with pgvector extension (Tier 2 features only)
**Testing**: Go stdlib testing + gotestsum for better output
**Target Platform**: macOS/Linux (CLI tool, single binary)
**Project Type**: Single Go module producing CLI binary
**Performance Goals**: Build in <30s, tests in <60s
**Constraints**: Tier 1 features work without Docker/database
**Scale/Scope**: 5 internal packages with test harnesses

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Note**: Project constitution is not yet defined. Using standard Go best practices:

| Principle | Status | Notes |
|-----------|--------|-------|
| Single binary output | PASS | Building `brains` CLI binary |
| Test infrastructure | PASS | Test harnesses for each package |
| Clear package boundaries | PASS | internal/ structure from MASTER-DESIGN.md |
| Minimal dependencies | PASS | Only spec-required dependencies |
| Tiered functionality | PASS | Tier 1 works without Docker |

**Result**: No violations. Proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/001-core-repo-setup/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (N/A - infrastructure only)
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - no API contracts)
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
zombiekit/
├── cmd/
│   └── brains/
│       └── main.go              # CLI entry point
│
├── internal/
│   ├── cli/                     # Command implementations
│   │   ├── root.go              # Root command, global flags
│   │   └── version.go           # Version command
│   │
│   ├── config/                  # Configuration system
│   │   └── config.go            # Config struct (placeholder)
│   │
│   ├── profile/                 # Profile domain
│   │   └── service.go           # ProfileService (placeholder)
│   │
│   ├── spec/                    # Spec domain
│   │   └── service.go           # SpecService (placeholder)
│   │
│   ├── conversation/            # Conversation domain
│   │   └── service.go           # ConversationService (placeholder)
│   │
│   ├── mcp/                     # MCP server & tools
│   │   └── server.go            # MCP server (placeholder)
│   │
│   └── web/                     # Web frontend
│       └── server.go            # HTTP server (placeholder)
│
├── migrations/                  # PostgreSQL schemas (placeholder)
│   └── .gitkeep
│
├── profiles/                    # Default global profiles (placeholder)
│   └── .gitkeep
│
├── configs/
│   └── .golangci.yml            # Linter configuration
│
├── Taskfile.yml                 # Build automation
├── docker-compose.yml           # Development services
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
└── .gitignore                   # Git ignore patterns
```

**Structure Decision**: Single Go module following MASTER-DESIGN.md layout. All packages under `internal/` for encapsulation. Test harnesses co-located with packages (e.g., `internal/profile/service_test.go`).

## Complexity Tracking

No violations to justify. Structure follows standard Go CLI patterns.
