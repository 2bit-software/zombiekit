# Research: Core Repository Setup

**Date**: 2025-12-21
**Feature**: 001-core-repo-setup

## Technology Decisions

### Go Module Structure

**Decision**: Use standard Go project layout with `cmd/` and `internal/`

**Rationale**:
- `cmd/brains/` contains the CLI entry point only
- `internal/` prevents external imports, enforcing API boundaries
- Matches structure defined in MASTER-DESIGN.md
- Standard pattern recognized by Go community

**Alternatives Considered**:
- Flat structure: Rejected - doesn't scale for 5+ packages
- `pkg/` for public APIs: Not needed - this is a CLI tool, not a library

### CLI Framework: urfave/cli/v2

**Decision**: Use urfave/cli/v2 as specified in MASTER-DESIGN.md

**Rationale**:
- Less boilerplate than cobra
- Well-suited for command-heavy CLIs
- Proven in mcp-genie reference project
- Good subcommand support for `brains profile compose`, etc.

**Alternatives Considered**:
- cobra: More popular but more verbose
- No framework: Too much boilerplate for subcommand structure

### Build Automation: Taskfile

**Decision**: Use go-task/Taskfile as specified

**Rationale**:
- YAML syntax is more readable than Make
- Better cross-platform support
- Proven in telegraph/ai reference projects
- Task dependencies and variables are cleaner

**Alternatives Considered**:
- Makefile: More ubiquitous but less readable
- Just: Less mature ecosystem

### Taskfile Structure

**Decision**: Split into Taskfile.yml (user tasks) and Taskfile.dev.yml (dev tasks)

**Rationale**:
- Follows mcp-genie pattern
- `task` shows user-facing commands
- `task dev -- <command>` shows development commands
- Reduces clutter for end users

**Key User Tasks**:
- `init`: Download deps, install tools
- `build`: Compile binary
- `test`: Run tests with coverage
- `lint`: Run golangci-lint
- `fmt`: Format code
- `vet`: Run go vet
- `ci`: All quality checks
- `db:up/down/migrate`: Database management

### Database: PostgreSQL with pgvector

**Decision**: Use pgvector/pgvector:pg16 Docker image on port 9432

**Rationale**:
- pgvector extension needed for conversation embeddings (Tier 2)
- Non-standard port (9432) avoids conflicts with local PostgreSQL
- Single image provides both PostgreSQL and vector support

**Configuration**:
- Database: `brains`
- User: `brains`
- Password: `brains_dev` (development only)
- Port: 9432 (mapped from container 5432)

### Test Framework

**Decision**: Go stdlib testing + gotestsum + testify

**Rationale**:
- stdlib testing is standard and fast
- gotestsum provides better output formatting
- testify/assert for cleaner assertions (optional)
- Race detector enabled by default in CI

**Test Harness Pattern**:
```go
// internal/profile/service_test.go
package profile_test

import (
    "testing"
)

func TestProfileService(t *testing.T) {
    t.Run("placeholder", func(t *testing.T) {
        // TODO: Implement when ProfileService is built
        t.Skip("not implemented")
    })
}
```

### Linting Configuration

**Decision**: Use golangci-lint with moderate strictness

**Rationale**:
- Catches common issues without being overly strict
- Consistent with mcp-genie reference
- Excludes test files from some checks (gosec, dupl)

**Enabled Linters** (subset):
- gofmt, goimports: Formatting
- govet, errcheck, staticcheck: Errors
- gosec: Security (except tests)
- misspell, unparam: Code quality

### Version Embedding

**Decision**: Use ldflags to embed version and commit at build time

**Rationale**:
- Standard Go pattern
- `brains version` shows: version (git tag), commit (short hash), build date
- Taskfile handles the ldflags automatically

**Implementation**:
```bash
go build -ldflags="-s -w -X main.version={{.VERSION}} -X main.commit={{.COMMIT}}" ...
```

## Resolved Clarifications

No NEEDS CLARIFICATION items - all technology choices defined in MASTER-DESIGN.md and spec.

## Dependencies Summary

| Dependency | Version | Purpose |
|------------|---------|---------|
| github.com/urfave/cli/v2 | v2.27+ | CLI framework |
| github.com/jackc/pgx/v5 | v5.7+ | PostgreSQL driver |
| github.com/mark3labs/mcp-go | v0.32+ | MCP protocol |
| gopkg.in/yaml.v3 | v3.0+ | YAML parsing |
| github.com/stretchr/testify | v1.9+ | Test assertions (optional) |

## Docker Compose Services

| Service | Image | Port | Purpose |
|---------|-------|------|---------|
| postgres | pgvector/pgvector:pg16 | 9432:5432 | Database with vector support |

Future services (not in this feature):
- redis (caching)
- ollama (embeddings)
