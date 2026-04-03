# Technical Spec: Skill/Agent Import

## Architecture

```
Claude Code Files                 Zombiekit Profiles
─────────────────                 ──────────────────
~/.claude/skills/foo/SKILL.md ──→ ~/.brains/profiles/foo/SKILL.md
                                        + scripts/
~/.claude/agents/bar.md       ──→ ~/.brains/profiles/bar/SKILL.md

                              ←── (optional shim back)
~/.claude/skills/foo/SKILL.md ←── shim → profile-compose
~/.claude/agents/bar.md       ←── shim → profile-compose (full frontmatter)
```

## Frontmatter Handling

### Agent Frontmatter Challenge

The `ProfileFrontmatter` struct only has `name`, `description`, `includes`, `inherits`, `type`. Agent files have additional fields (`model`, `skills`, `memory`, `color`) that aren't in this struct.

**Approach**: Parse agent frontmatter as raw `map[string]any` via `yaml.Unmarshal` to preserve all fields. Use the typed struct only for the output profile. For shim generation, serialize the raw map back to YAML with `allowed-tools` injected.

### Frontmatter Serialization

Need a `SerializeFrontmatter(fields map[string]any) string` helper that:
- Marshals to YAML
- Wraps in `---` delimiters
- Handles multi-line descriptions with `>` block scalar

This doesn't exist in the codebase today — `GenerateContent()` in `install.go` uses string templates. Follow the same template approach for consistency.

## File Copy Strategy

For skill directories with supporting files:
- Use `filepath.Walk` on source directory
- Skip `SKILL.md` (handled separately with transformation)
- Preserve relative path structure
- Use `io.Copy` for file contents
- Preserve file permissions via `os.Chmod`

## Symlink Resolution

```go
resolved, err := filepath.EvalSymlinks(path)
if err != nil {
    // broken symlink — skip with warning
    continue
}
info, err := os.Stat(resolved)
```

Both `~/.claude/skills` and `~/.claude/agents` directories themselves may be symlinks. Resolve at the directory level before walking.

## Shim Detection (FR-4)

```go
func isShim(body string) bool {
    return strings.Contains(body, "mcp__zombiekit__profile-compose")
}
```

Simple string match on the body content after frontmatter parsing.

## Error Strategy

- Invalid frontmatter → skip item, add to `Skipped` with reason
- Broken symlink → skip item, add to `Skipped` with reason
- Destination exists → return error (caller decides to overwrite or rename)
- File copy failure → return error (partial import is not acceptable for a single item)
- Name collision (skill vs agent) → return collision list in response (caller handles)
