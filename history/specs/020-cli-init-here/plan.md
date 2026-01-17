# Implementation Plan: CLI Init Enhancement

**Branch**: `020-cli-init-here` | **Date**: 2025-12-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/020-cli-init-here/spec.md`

## Summary

Enhance the `brains init` command to perform full ZombieKit setup by default: creating `.claude/commands/` with embedded Claude Code skills and `.brains/templates/` with embedded specification templates. The implementation extends the existing embedded filesystem pattern (`go:embed`) to include two additional asset collections, and modifies `internal/cli/init.go` to copy these assets to the target directory.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: urfave/cli/v2 (CLI framework), embed (stdlib for filesystem embedding)
**Storage**: File-based (copying embedded files to local filesystem)
**Testing**: go test with testify assertions (existing test framework)
**Target Platform**: Cross-platform (macOS, Linux, Windows)
**Project Type**: Single CLI binary
**Performance Goals**: Complete initialization in under 30 seconds (typically <1 second)
**Constraints**: Single binary deployment, no external dependencies at runtime
**Scale/Scope**: 15 command files + 5 template files to embed and copy

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution file contains placeholder content. Based on project patterns observed:

| Principle | Status | Notes |
|-----------|--------|-------|
| Library-First | PASS | Feature extends existing `internal/cli/init.go` and reuses `internal/profile` patterns |
| CLI Interface | PASS | All functionality exposed via `brains init` CLI command |
| Test-First | PASS | Test file `init_test.go` defined in project structure |
| Observability | PASS | Verbose file-by-file output during initialization |
| Simplicity | PASS | Uses existing `go:embed` pattern, minimal new code |

**Post-Design Re-evaluation (Phase 1 Complete)**:
- All gates PASS
- Design follows existing patterns (embed.go, internal/cli structure)
- No new dependencies introduced
- No complexity justifications required

## Project Structure

### Documentation (this feature)

```text
specs/020-cli-init-here/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
# Existing structure (Go CLI project)
cmd/brains/
└── main.go              # Entry point, registers embedded assets

internal/
├── cli/
│   └── init.go          # MODIFY: Add full setup logic with --force flag
└── profile/
    └── embedded.go      # REFERENCE: Pattern for embedded filesystem handling

# New embedded assets
embed.go                 # MODIFY: Add commands and templates embed directives
integrations/
└── claude/
    └── commands/        # SOURCE: 15 .md files to embed (brains.*.md)
templates/
└── templates/           # SOURCE: 5 .md files to embed (*-template.md)

# Tests
internal/cli/
└── init_test.go         # NEW: Unit tests for init command
```

**Structure Decision**: Single Go CLI project following existing patterns. The feature modifies `embed.go` at repository root to add two new embedded filesystems, and updates `internal/cli/init.go` to copy files from embedded assets to local directories.

## Complexity Tracking

No violations requiring justification. The implementation follows existing patterns for embedded filesystems.
