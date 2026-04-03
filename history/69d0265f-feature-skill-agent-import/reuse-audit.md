# Reuse Audit

## Summary
- Duplicates: 3 (scopeDir, ImportResult, frontmatter serialization)
- Overlaps: 3 (directory walking, shim detection, single-file copy)
- Related: 0
- No match: 3 (DiscoverableItem, raw map frontmatter parse, copyDir)

## Findings

### DUPLICATE

#### scopeDir function
- **Existing**: `internal/skill/install.go:TargetDir()` — resolves global/local to `~/.claude/skills/` or `.claude/skills/`
- **Decision**: Use directly for Claude skill shim paths. For brains/profiles destination, use `profile.BrainsSource.GetInitDir()` or inline the pattern.

#### ImportResult-style structs
- **Existing**: `internal/profile/types.go:ImportResult` + `ImportFailure`
- **Decision**: Create new — existing struct is tied to profile import (Created/Overwritten/Failed counts). Our result needs `Imported/Skipped/Shims` arrays with different semantics. The shape is similar but the contract differs enough that extending would be awkward.

#### Agent frontmatter serialization
- **Existing**: `internal/profile/importer.go:convertClaudeToBrains()` — marshals struct to YAML with `---` delimiters
- **Decision**: Extract the serialization pattern. Use `yaml.Marshal` + delimiter wrapping, same approach.

### OVERLAP

#### Directory walking for Claude files
- **Existing**: `internal/profile/claude_source.go:FindProfileDirs()` — walks `~/.claude/agents/`
- **Similarity**: Same home resolution and directory listing, but hardcoded to `agents/` and returns `ResolvedDirectory` not `DiscoverableItem`
- **Decision**: Create new — the discovery function needs to parse frontmatter, check for shims, and return a different type. The walk logic is trivial (`os.ReadDir`), not worth coupling to `ClaudeSource`.

#### Shim detection
- **Existing**: `internal/skill/install.go:GenerateContent()` defines the shim pattern implicitly
- **Decision**: Create new `IsShim(body string) bool` function in `internal/skill/`. Simple `strings.Contains` check.

#### Single-file copy
- **Existing**: `internal/worktree/manager.go:copyFiles()` — unexported, method-bound
- **Decision**: Create new `copyDirContents()` — the worktree function is too coupled to its context.

### NONE

- **DiscoverableItem struct**: No equivalent. New struct needed.
- **Raw map frontmatter parse**: All existing parsers use typed structs. Use `frontmatter.Parse(r, &map[string]any{})`.
- **copyDir utility**: No directory copy exists. Write new.

## Plan Changes

1. **Step 1**: Use `recall/claude.DefaultClaudePath()` for `~/.claude` resolution instead of manual home dir logic
2. **Step 2**: Add `IsShim()` to `internal/skill/` package (small function, belongs near `GenerateContent`)
3. **Step 2**: Use `skill.TargetDir()` for skill shim destination paths
4. **Step 2**: Own `ImportResult` type — don't extend `profile.ImportResult`
5. **Step 2**: Follow `convertClaudeToBrains()` pattern for YAML serialization but don't import it
