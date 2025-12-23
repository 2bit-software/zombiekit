# Implementation Plan: Embedded Profile Fallback

**Branch**: `019-embedded-profile-fallback` | **Date**: 2025-12-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/019-embedded-profile-fallback/spec.md`

## Summary

Add embedded default profiles to the brains CLI binary so users can use profile features immediately without configuration. The `./profiles/` directory (containing 15+ default profiles) will be embedded at compile time using Go's `//go:embed` directive and served as the lowest-precedence fallback source after local, parent, and global directories.

## Technical Context

**Language/Version**: Go 1.24.0
**Primary Dependencies**: urfave/cli/v2 (CLI), mark3labs/mcp-go (MCP), adrg/frontmatter (YAML parsing)
**Storage**: File-based profiles (.md files with YAML frontmatter), embedded filesystem (embed.FS)
**Testing**: go test with stretchr/testify
**Target Platform**: Linux/macOS/Windows CLI
**Project Type**: Single (CLI tool with internal packages)
**Performance Goals**: N/A (profile loading is not performance-critical)
**Constraints**: Binary size increase acceptable (~50KB for 15 profile files)
**Scale/Scope**: 15+ embedded profiles, single-user CLI tool

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution file is a placeholder template. No formal gates are defined. Proceeding with standard Go best practices:

| Gate | Status | Notes |
|------|--------|-------|
| Library-First | PASS | Changes are in internal/profile package |
| CLI Interface | PASS | Existing CLI commands extended, not new commands |
| Test-First | PASS | Will add unit tests for embedded source |
| Simplicity | PASS | Using Go's standard embed directive, minimal new code |

## Project Structure

### Documentation (this feature)

```text
specs/019-embedded-profile-fallback/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # N/A (no API changes)
└── tasks.md             # Phase 2 output (from /speckit.tasks)
```

### Source Code (repository root)

```text
profiles/                     # Embedded at build time (source files)
├── audit.md
├── bug.md
├── clarify.md
├── complete.md
├── eat.md
├── feature.md
├── implement.md
├── init.md
├── plan.md
├── refactor.md
├── research.md
├── revise.md
├── status.md
├── tasks.md
└── update.md

internal/
├── profile/
│   ├── types.go             # Add SourceEmbedded constant
│   ├── resolver.go          # Extend FindProfileDirs() for embedded
│   ├── embedded.go          # NEW: embed.FS declaration and loader
│   ├── embedded_source.go   # NEW: EmbeddedSource implementing ProfileSourceInterface
│   ├── embedded_test.go     # NEW: Tests for embedded profiles
│   ├── brains_source.go     # Update to chain embedded source
│   └── service.go           # No changes (uses source interface)
├── mcp/
│   └── tools/profile/
│       └── tool.go          # No changes (uses service)
└── cli/
    └── profile.go           # No changes (uses service)

tests/
└── integration/
    └── profile_embedded_test.go  # NEW: Integration tests
```

**Structure Decision**: Single project structure. All changes are within the existing `internal/profile` package, following the established pattern. No new commands or packages needed.

## Complexity Tracking

No violations to justify. The implementation uses Go's standard `embed` package with minimal new code:
- 1 new source type constant
- 1 new embed.FS variable
- 1 new ProfileSourceInterface implementation (~100 LOC)
- Extension of existing BrainsSource to chain embedded as fallback
