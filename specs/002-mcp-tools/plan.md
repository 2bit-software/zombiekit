# Implementation Plan: MCP Tools - Code Reasoning & Sticky Memory

**Branch**: `002-mcp-tools` | **Date**: 2025-12-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-mcp-tools/spec.md`

## Summary

Implement two MCP tools (`stickymemory` and `code-reasoning`) backed by PostgreSQL or SQLite with full CLI and MCP interfaces. The sticky memory tool provides persistent key-value storage with versioning for AI assistants. The code reasoning tool provides structured sequential reasoning with branching and revision support. Both tools are exposed via an MCP server supporting Streamable HTTP, SSE, and stdio transports.

**Compatibility Note**: SQLite implementation follows the mcp-genie patterns from `telegraph/ai/tools/mcp-genie` for consistency and potential code sharing.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: urfave/cli/v2 (CLI), mark3labs/mcp-go (MCP), pgx/v5 (PostgreSQL), modernc.org/sqlite (SQLite), slog (logging)
**Storage**: PostgreSQL 16 with pgvector (production) OR SQLite with WAL mode (development/single-user, default)
**Testing**: go test with testify, testcontainers-go for PostgreSQL integration tests, temp files for SQLite tests
**Target Platform**: Linux/macOS CLI binary
**Project Type**: Single project (existing CLI structure)
**Performance Goals**: <100ms for memory operations, <200ms for search, 10 concurrent connections
**Constraints**: No authentication (local-only), fail-fast on DB errors, structured JSON logging
**Scale/Scope**: Up to 10,000 memories, 1MB max per memory, 20 thoughts per reasoning chain
**Default Backend**: SQLite (zero-configuration), PostgreSQL via `--db-type postgres`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Reference**: MASTER-DESIGN.md principles (no formal constitution yet)

| Principle | Status | Notes |
|-----------|--------|-------|
| Tiered Dependencies | ✅ PASS | MCP tools are Tier 2 (PostgreSQL required for production) - correctly scoped |
| Service Interfaces | ✅ PASS | Tools will implement clean interfaces for testability |
| CLI Text I/O | ✅ PASS | All commands support `--format json` for machine-readable output |
| Stateless CLI | ✅ PASS | Brains CLI remains stateless; PostgreSQL/SQLite handles persistence |
| MCP Tool Integration | ✅ PASS | Follows existing `internal/mcp/` structure |

**Gate Result**: PASS - No violations. Proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/002-mcp-tools/
├── spec.md              # Feature specification (complete)
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (MCP tool schemas)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/brains/
└── main.go                      # CLI entry point (existing)

internal/
├── cli/                         # CLI commands (existing)
│   ├── root.go                  # Root command (existing)
│   ├── serve.go                 # NEW: brains serve command
│   ├── memory.go                # NEW: brains memory subcommand
│   └── db.go                    # NEW: brains db subcommand
│
├── mcp/                         # MCP server (existing skeleton)
│   ├── server.go                # MCP protocol handler (enhance)
│   └── tools/                   # NEW: tool implementations
│       ├── stickymemory/
│       │   ├── tool.go          # MCP tool interface
│       │   ├── service.go       # Business logic
│       │   └── tool_test.go
│       └── codereasoning/
│           ├── tool.go          # MCP tool interface
│           ├── session.go       # Session state management
│           └── tool_test.go
│
├── database/                    # NEW: database layer
│   ├── pool.go                  # Connection pool management
│   ├── migrations.go            # Migration runner
│   └── migrations/              # SQL migration files
│       ├── postgres/
│       │   └── 001_stickymemory.sql
│       └── sqlite/
│           └── 001_stickymemory.sql
│
├── memory/                      # NEW: sticky memory domain
│   ├── repository.go            # Repository interface
│   ├── types.go                 # Memory entity types
│   ├── postgres/                # PostgreSQL implementation
│   │   ├── repository.go
│   │   └── repository_test.go
│   └── sqlite/                  # SQLite implementation
│       ├── repository.go
│       └── repository_test.go
│
├── mo/                          # NEW: Maybe monad (from mcp-genie)
│   └── maybe.go                 # mo.Maybe[T], mo.Just[T], mo.Nothing[T]
│
└── config/                      # Configuration (existing)
    └── config.go                # Add database config

tests/
└── integration/                 # NEW: integration tests
    ├── memory_test.go           # Memory operations against real DB
    └── mcp_test.go              # MCP server integration tests
```

**Structure Decision**: Follows existing single-project Go layout from MASTER-DESIGN.md. MCP tools go under `internal/mcp/tools/`, with domain logic in separate packages (`internal/memory/`, `internal/database/`). Both PostgreSQL and SQLite implement the same Repository interface. Integration tests use testcontainers-go for PostgreSQL and temp files for SQLite.

## Complexity Tracking

> No constitution violations. Design follows MASTER-DESIGN.md patterns.

| Pattern | Justification |
|---------|---------------|
| Repository pattern for memory | Enables mocked unit tests and potential future storage backends |
| Separate database package | Isolates PostgreSQL/SQLite connection management and migrations |
| Tool interface abstraction | Required by MCP protocol; each tool implements standard interface |
| Maybe monad (mo package) | Consistency with mcp-genie; cleaner optional value handling |

---

## Constitution Check (Post-Design)

*Re-evaluation after Phase 1 design artifacts completed.*

| Principle | Status | Notes |
|-----------|--------|-------|
| Tiered Dependencies | ✅ PASS | Database package isolates PostgreSQL/SQLite; CLI gracefully errors without DB |
| Service Interfaces | ✅ PASS | Repository interface defined in data-model.md; tool interfaces in contracts/ |
| CLI Text I/O | ✅ PASS | All memory commands support `--format json`; structured logging implemented |
| Stateless CLI | ✅ PASS | Memory state in PostgreSQL/SQLite; reasoning state in-memory per-session |
| MCP Tool Integration | ✅ PASS | Follows mark3labs/mcp-go patterns from research.md |
| Testability | ✅ PASS | Repository interface enables mocking; testcontainers for integration |
| mcp-genie Compatibility | ✅ PASS | SQLite schema matches mcp-genie's `memories` table with (name, version) PK |

**Post-Design Gate Result**: PASS - Design aligns with MASTER-DESIGN.md principles and mcp-genie patterns.

---

## mcp-genie Compatibility Reference

The SQLite implementation in zombiekit MUST be compatible with mcp-genie's stickymemory implementation to enable potential code sharing and consistent behavior.

### Key Patterns from mcp-genie

1. **Storage Interface** (`pkg/tools/stickymemory/storage.go`):
   ```go
   type Storage interface {
       Set(ctx context.Context, name, content string) error
       Get(ctx context.Context, name string) (mo.Maybe[MemoryItem], error)
       Delete(ctx context.Context, name string) error
       List(ctx context.Context, search string) ([]MemoryMetadata, error)
       Clear(ctx context.Context) (int, error)
       Close() error
   }
   ```

2. **SQLite Schema** (mcp-genie's sqlite_storage.go):
   ```sql
   CREATE TABLE IF NOT EXISTS memories (
       name TEXT NOT NULL,
       version INTEGER NOT NULL,
       deleted BOOLEAN NOT NULL DEFAULT FALSE,
       content TEXT NOT NULL,
       created_at TIMESTAMP NOT NULL,
       updated_at TIMESTAMP NOT NULL,
       PRIMARY KEY (name, version)
   );
   ```

3. **Version Management**: Each `Set` creates a new version (not upsert); latest non-deleted version is returned by `Get`.

4. **Soft Deletes**: `Delete` sets `deleted=TRUE` on ALL versions of a name.

5. **Maybe Monad**: Returns `mo.Maybe[MemoryItem]` from `Get` - `Nothing` for not found, `Just(item)` for found.

6. **Name Sanitization**: `sanitizeName()` replaces invalid characters with underscores (valid: a-z, A-Z, 0-9, -, _, .)

7. **Backend Selection**: Environment variable `STICKYMEMORY_BACKEND` (sqlite/postgres), defaults to sqlite.

8. **SQLite Path**: Defaults to `~/.mcp-genie/stickymemory/memories.db` - zombiekit should use `~/.brains/memories.db`.

### Compatibility Requirements for zombiekit

- [ ] Use identical table schema (name, version as composite PK)
- [ ] Implement same Storage interface signature
- [ ] Include mo.Maybe package (can copy from mcp-genie)
- [ ] Match soft-delete behavior (delete all versions)
- [ ] Use same sanitizeName() logic
- [ ] Support same environment variable pattern (`BRAINS_BACKEND`, `BRAINS_SQLITE_PATH`)

---

## Generated Artifacts

| Artifact | Path | Description |
|----------|------|-------------|
| Research | `research.md` | Technology decisions and code patterns |
| Data Model | `data-model.md` | Entity definitions and SQL schema |
| Contracts | `contracts/stickymemory.json` | MCP tool schema for sticky memory |
| Contracts | `contracts/code-reasoning.json` | MCP tool schema for code reasoning |
| Quickstart | `quickstart.md` | Local development setup guide |

---

## Next Step

Run `/speckit.tasks` to generate implementation tasks from this plan.
