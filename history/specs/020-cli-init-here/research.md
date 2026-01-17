# Research: CLI Init Enhancement

**Feature**: 020-cli-init-here
**Date**: 2025-12-23

## 1. Embedded Filesystem Pattern in Go

### Decision
Use Go's `embed` package with multiple `//go:embed` directives in a single file (`embed.go` at repository root).

### Rationale
- Already established pattern in the codebase (`embed.go` with `EmbeddedProfiles`)
- Compile-time embedding ensures single binary deployment
- `embed.FS` implements `fs.FS` interface, enabling standard library compatibility
- No runtime file path dependencies

### Alternatives Considered
| Alternative | Rejected Because |
|-------------|------------------|
| External file distribution | Adds deployment complexity, requires file path management |
| go-bindata/go-rice | Deprecated in favor of stdlib embed, adds dependency |
| Download at runtime | Requires network, fails offline, version drift |

### Implementation Pattern (from existing codebase)
```go
// embed.go at repository root
//go:embed profiles/*
var EmbeddedProfiles embed.FS

//go:embed integrations/claude/commands/*
var EmbeddedCommands embed.FS

//go:embed templates/templates/*
var EmbeddedTemplates embed.FS
```

## 2. File Copying Strategy

### Decision
Copy files individually with verbose output, skip existing files unless `--force` is provided.

### Rationale
- User wants to see each file as it's processed (clarification session decision)
- Skipping existing files prevents accidental overwrites
- `--force` flag provides explicit control for updates
- Continue on individual file failures, report summary at end

### Best Practices
1. **Directory creation**: Use `os.MkdirAll` with 0755 permissions
2. **File writing**: Use `os.WriteFile` with 0644 permissions (default for non-executable)
3. **Existence check**: Use `os.Stat` before writing, skip if exists and no `--force`
4. **Error handling**: Log individual failures, continue with remaining files, return aggregate error

### Output Format
```
Created .claude/
Created .claude/commands/
  Copied brains.feature.md
  Copied brains.plan.md
  Skipped brains.init.md (exists)
Created .brains/
Created .brains/templates/
  Copied spec-template.md
  Copied plan-template.md

Initialized ZombieKit: 18 files copied, 2 skipped
```

## 3. Flag Design

### Decision
- Default (no flags): Full local setup in current directory
- `--global`: Creates in `~/.brains/` (existing behavior, unchanged)
- `--force`: Overwrite existing files

### Rationale
- Clarification session determined default should do full setup
- `--global` maintains backward compatibility
- `--force` follows Unix convention (`cp -f`, `rm -f`)

### Flag Interactions
| Flags | Behavior |
|-------|----------|
| (none) | Full setup in `./.claude/commands/` and `./.brains/templates/` |
| `--global` | Setup in `~/.brains/` only (existing behavior, no commands) |
| `--force` | Full local setup, overwrite existing files |
| `--global --force` | Global setup, overwrite existing (edge case) |

## 4. Registry Integration

### Decision
Register the `.brains` directory in the profile registry after successful initialization.

### Rationale
- Existing pattern in current init implementation
- Enables profile discovery across projects
- Best effort (errors ignored) to not block initialization

### Implementation (from existing code)
```go
rm, err := profile.NewRegistryManager()
if err == nil {
    _ = rm.Register(brainsDir)
}
```

## 5. Error Handling Strategy

### Decision
Continue on file copy failures, aggregate and report at end.

### Rationale
- Partial initialization is better than no initialization
- User can address individual failures
- Matches spec edge case: "System reports the specific file that failed and continues"

### Error Categories
| Error Type | Handling |
|------------|----------|
| Directory not writable | Fatal, return immediately with clear message |
| Empty embedded FS | Fatal, suggest reinstalling binary |
| Individual file copy failure | Log, continue, include in summary |
| Registry failure | Ignore (best effort) |

## 6. Cross-Platform Considerations

### Decision
Use `filepath.Join` for all path construction, use `fs.FS` interface for embedded access.

### Rationale
- `filepath.Join` handles OS-specific path separators
- Embedded paths use forward slashes (Go embed convention)
- Target paths use OS conventions

### Implementation Notes
- Embedded FS always uses `/` as separator (per Go spec)
- Use `filepath.Join` when writing to local filesystem
- Use `path.Join` or string concatenation for embedded paths

## Summary

All technical decisions align with existing codebase patterns:
- Extend `embed.go` with two new embedded filesystems
- Modify `init.go` to copy files using standard library functions
- Add `--force` flag to existing command structure
- Maintain backward compatibility with `--global` flag
