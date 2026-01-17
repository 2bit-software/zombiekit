# Implementation Plan: Profile Import Subcommand

**Branch**: `005-profile-import` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/005-profile-import/spec.md`

## Summary

Add an `import` subcommand to `brains profiles` that converts Claude agents (from `.claude/agents/`) to brains profiles (in `.brains/profiles/`). The import preserves scope (local→local, global→global), sets `inherits: false` for all imported profiles, discards Claude-specific fields (model, color), and overwrites existing profiles on collision.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: urfave/cli/v2 (CLI), adrg/frontmatter (YAML parsing), gopkg.in/yaml.v3 (YAML)
**Storage**: File-based (.md files with YAML frontmatter)
**Testing**: go test with testify/assert
**Target Platform**: Cross-platform CLI (Linux, macOS, Windows)
**Project Type**: Single CLI application
**Performance Goals**: < 5 seconds for typical import (under 50 agents)
**Constraints**: Builds on 004-source-interface; uses existing ProfileSourceInterface
**Scale/Scope**: Typically 1-50 agents per import

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution is not yet configured (template placeholders only). Proceeding with standard Go best practices:

- [x] **Interface Segregation**: Leverages existing ProfileSourceInterface for reading Claude agents
- [x] **Single Responsibility**: Import logic isolated in dedicated importer package/service
- [x] **Testability**: Pure functions for conversion, interfaces for file I/O
- [x] **Simplicity**: One source type supported (claude), extensible pattern for future

## Project Structure

### Documentation (this feature)

```text
specs/005-profile-import/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # N/A - CLI, not API
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── profile/
│   ├── importer.go         # NEW: Import service for source-to-brains conversion
│   ├── importer_test.go    # NEW: Unit tests for import logic
│   ├── source.go           # EXISTING: ProfileSourceInterface (used for reading)
│   ├── brains_source.go    # EXISTING: Used as write target
│   ├── claude_source.go    # EXISTING: Used as read source
│   ├── frontmatter.go      # EXISTING: Brains frontmatter serialization
│   ├── claude_frontmatter.go # EXISTING: Claude frontmatter parsing
│   └── types.go            # EXISTING: ImportResult type added
├── cli/
│   └── profile.go          # EXISTING: Add import subcommand
```

**Structure Decision**: Single project structure. This feature adds a new `importer.go` file to `internal/profile/` and modifies `internal/cli/profile.go` to add the import subcommand.

## Complexity Tracking

No constitution violations. The design is minimal:
1. Reuses existing ClaudeSource for reading agents
2. Reuses existing BrainsSource for writing profiles
3. Simple conversion function between frontmatter formats
