---
status: complete
updated: 2026-04-02
---

# Research: .skill File Support

## Executive Summary

`.skill` files are ZIP archives created by Claude web (claude.ai) that contain structured markdown
defining reusable AI skills. The zombiekit profile system currently only loads `.md` files. Adding
`.skill` support requires two things: native loading of `.skill` ZIPs in profile directories, and a
mechanism to import/move skill files into the correct profiles location.

## Findings

### Codebase Context

**Profile loading pipeline** (`internal/profile/`):
- `resolver.go`: Walks CWD → git root collecting `.brains/profiles/` dirs; also checks `~/.brains/profiles/` global
- `loader.go` (inferred): Reads `.md` files from profile dirs; calls `ParseFrontmatter()`
- `frontmatter.go`: Parses YAML frontmatter + markdown body from `.md` files
- `claude_frontmatter.go`: Claude-specific frontmatter (model, color fields)
- `importer.go`: Existing pattern for importing Claude agents → brains format (copy + convert)
- `source.go`: `ProfileSourceInterface` abstraction — `BrainsSource` vs `ClaudeSource`
- `composer.go`: Compose + include resolution; purely functional

**Key constraint**: File loading is currently hardcoded to `.md` extension only. Extension must be
added to whatever file glob/scan is used in the loader.

**Existing import pattern** (`importer.go`): Takes a source dir (`.claude/agents/`), reads all
profiles, converts frontmatter, writes to target dir. Same pattern applies for `.skill` import.

**Profile name derivation**: Comes from YAML `name:` field or filename (minus extension). This works
naturally for `.skill` files where `SKILL.md` frontmatter has the canonical name.

**Template sub-profiles**: No existing concept of "sub-profiles" from a single file. Template files
inside a `.skill` ZIP are a new concept that must be introduced.

### .skill File Format

- **Format**: ZIP archive (deflate compression)
- **Directory structure inside ZIP**:
  ```
  {skill-name}/
  ├── SKILL.md         # Required: YAML frontmatter (name, description) + skill body
  ├── TEMPLATE.md      # Optional: output scaffold
  └── TEMPLATE-*.md    # Optional: named template variants
  ```
- **SKILL.md frontmatter**: `name:` and `description:` fields — matches brains profile format
- **No binaries or scripts**: Content is pure markdown only
- **Size**: 3–21 KB typical

### Decision Points

- **D1**: How to handle `TEMPLATE*.md` files from inside a `.skill` ZIP?
  - Option A: Load only `SKILL.md`; ignore templates (simple, loses template value)
  - Option B: Register templates as additional profiles named `{skill}-template`, `{skill}-template-{variant}` (full value, more complex)
  - Option C: Concatenate all files into one profile (loses separation)
  - **Recommendation**: Option B — templates are useful artifacts worth preserving as composable profiles

- **D2**: Import behavior — copy ZIP vs extract?
  - Option A: Copy `.skill` file as-is to profiles dir (requires runtime ZIP extraction)
  - Option B: Extract contents on import, write as a subdirectory (no runtime overhead)
  - **Recommendation**: Option B — extract on import. Matches Claude Code's bundled skill pattern
    (extracted to temp dir at first use). Files are on disk, agent accesses them natively.

- **D3**: How does agent access sibling files?
  - Option A: Filesystem path prepended to compose output (Claude Code's approach)
  - Option B: MCP resource protocol
  - Option C: Inline all referenced content at compose time
  - **Recommendation**: Option A — prepend `Base directory for this skill: {path}`. Matches Claude
    Code exactly. Compatible with HTTP proxy model (proxy syncs files locally first).

- **D4**: Target profiles directory
  - `~/Projects/personal/ai/profiles` is symlinked to `.brains/profiles/` — zombiekit doesn't
    need to know about this. Target is `~/.brains/profiles/` (global) or `.brains/profiles/` (local).

## Recommendations

1. **Extend profile loader** to recognize `.skill` files in profile dirs alongside `.md` files
2. **ZIP extraction**: Use Go's `archive/zip` stdlib; extract `SKILL.md` for main profile, `TEMPLATE*.md` for sub-profiles
3. **Sub-profile naming**: `{skill-name}-template` (default TEMPLATE.md), `{skill-name}-template-{variant}` (TEMPLATE-{variant}.md)
4. **Import MCP tool** (`profile-import-skill`): Takes source path (file or directory), copies `.skill` files to target profiles dir (default: global `~/.brains/profiles/`)
5. **`profile-list` extension**: Show skill-file-sourced profiles with `source: "skill"` distinction

## Sources

- Codebase exploration: `internal/profile/` package
- `.skill` file inspection: `~/Downloads/*.skill` (epic-planner, ticket-decomposer, tech-dependency-audit, sdlc-orchestrator)
