# Implementation Plan: Profile Source Abstraction

**Branch**: `004-source-interface` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-source-interface/spec.md`

## Summary

Add a `--source` argument to profile commands supporting "brains" (default) and "claude" sources. The implementation introduces a `ProfileSource` interface to abstract profile operations, enabling read/write operations across different profile backends. The brains source wraps existing behavior while the claude source reads/writes from `.claude/agents/` directories.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: urfave/cli/v2 (CLI), adrg/frontmatter (YAML parsing), gopkg.in/yaml.v3 (YAML)
**Storage**: File-based (.md files with YAML frontmatter)
**Testing**: go test with testify/assert
**Target Platform**: Cross-platform CLI (Linux, macOS, Windows)
**Project Type**: Single CLI application
**Performance Goals**: < 1 second for typical operations (already met by existing implementation)
**Constraints**: Backward compatible with existing brains profile commands
**Scale/Scope**: Hundreds of profiles across local/global directories

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution is not yet configured (template placeholders only). Proceeding with standard Go best practices:

- [x] **Interface Segregation**: ProfileSource interface will be minimal and focused
- [x] **Backward Compatibility**: Default to brains source, no breaking changes
- [x] **Testability**: Interface enables mocking for unit tests
- [x] **Simplicity**: Two concrete implementations, no over-abstraction

## Project Structure

### Documentation (this feature)

```text
specs/004-source-interface/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - CLI, not API)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── profile/
│   ├── source.go           # NEW: ProfileSource interface definition
│   ├── brains_source.go    # NEW: BrainsSource implementation (wraps existing)
│   ├── claude_source.go    # NEW: ClaudeSource implementation
│   ├── claude_frontmatter.go # NEW: Claude agent frontmatter parsing
│   ├── resolver.go         # EXISTING: Directory resolution (refactor to use interface)
│   ├── service.go          # EXISTING: Profile service (refactor to accept source)
│   ├── composer.go         # EXISTING: Profile composition
│   ├── frontmatter.go      # EXISTING: Brains frontmatter parsing
│   └── types.go            # EXISTING: Type definitions (extend for claude fields)
├── cli/
│   ├── profile.go          # EXISTING: Add --source flag to all subcommands
│   └── init.go             # EXISTING: Add --source flag for init
```

**Structure Decision**: Single project structure. This feature adds new files to the existing `internal/profile/` package and modifies the CLI layer to accept the new `--source` flag.

## Complexity Tracking

No constitution violations. The ProfileSource interface is justified by:
1. Clear need for two implementations (brains, claude)
2. Different directory structures and frontmatter formats
3. Enables future sources without modifying existing code
