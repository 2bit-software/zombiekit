# Technical Requirements: .skill File Support

## Implementation Notes

Implementation preferences extracted from codebase conventions and user intent.
See `business-spec.md` for behavioral requirements.

---

## Design: Two Profile Layouts

The profile loader currently handles only flat files: `{profiles-dir}/name.md`.

After this feature, it handles two layouts:

| Layout | Path | Notes |
|--------|------|-------|
| Flat | `{profiles-dir}/name.md` | Existing, unchanged |
| Directory | `{profiles-dir}/name/SKILL.md` | New |

Both participate equally in all profile operations (list, compose, include resolution, shadowing).

---

## Base Directory Prefix

Mirroring Claude Code CLI behavior (`loadSkillsDir.ts` line 346):

When composing a **directory-layout** profile, prepend:
```
Base directory for this skill: {absolute-path-to-skill-dir}

{SKILL.md content}
```

**Do not** prepend for flat `.md` profiles — no behavior change there.

The absolute path allows the agent to `Read` sibling files using its normal tools. No special
protocol or resource abstraction needed.

---

## Profile Name Derivation

For directory-layout profiles, name comes from:
1. `name:` field in SKILL.md YAML frontmatter (preferred)
2. If absent: directory name, normalized (lowercase, spaces/underscores → hyphens, collapse hyphens)

Example: directory `My Cool Skill/` with no frontmatter name → profile name `my-cool-skill`

---

## ZIP Structure Handling

`.skill` ZIPs from Claude web use this layout:
```
{skill-name}/
  SKILL.md
  script.sh
  templates/output.md
  ...
```

Import extracts the inner directory to the target profiles dir, stripping the outer ZIP-name wrapper:
```
~/.brains/profiles/epic-planner/
  SKILL.md
  script.sh
  templates/output.md
```

Walk the ZIP entries; find the common top-level directory (if all entries share one). Extract its
contents to `{target}/{skill-name}/`. If entries are flat (no top-level dir), extract directly to
`{target}/{skill-name}/` using the ZIP filename (minus `.skill`) as the directory name.

---

## Files to Create / Modify

### New: `internal/profile/skill_loader.go`

Responsible for:
- `LoadSkillDirectory(dir string) (*Profile, error)` — reads `SKILL.md`, returns Profile with `BasePath` set
- `IsSkillDirectory(dir string) bool` — checks for presence of `SKILL.md`

### New: `internal/profile/skill_extractor.go`

Responsible for:
- `ExtractSkillFiles(profilesDir string) []ExtractError` — scans a profiles dir for `.skill` ZIPs, extracts each
- Idempotent: skip if target subdirectory already exists
- Returns non-fatal errors (warn + continue)
- Called once during MCP server init for each profiles directory

### Modified: `internal/profile/` loader scan

Where the loader currently globs `*.md` files, also walk subdirectories and call
`IsSkillDirectory()` for each. Construct profiles from both flat and directory entries.

### Modified: `internal/profile/composer.go` or content assembly

When assembling profile content, check if `Profile.BasePath != ""` and prepend the base directory
line before the body.

### Modified: MCP server init (wherever profiles are first initialized)

Call `ExtractSkillFiles(profilesDir)` for each profiles directory before profile loading begins.
Log any extraction errors as warnings; do not abort init.

---

## Profile Struct Change

Current `Profile` struct likely has: `Name`, `Description`, `Body`, `Source`, `Includes`, etc.

Add: `BasePath string` — absolute path to the skill directory. Empty string for flat profiles.

---


## `profile-list` Output Change

Add `source` field to list entries. Possible values:
- `"brains"` — flat `.md` profile (existing)
- `"skill"` — directory-layout profile (new)
- `"claude"` — from ClaudeSource (existing, if applicable)
- `"embedded"` — compiled-in profile (existing)

---

## Existing Pattern Reference

- `internal/profile/importer.go` — Claude agent → brains importer; similar copy-and-convert pattern
- `internal/profile/resolver.go` — walks dirs to find profile dirs; needs to also yield subdirs as potential skill directories
- Claude Code CLI `loadSkillsDir.ts` line 424-426: confirms directory-only check for skills (single `.md` files not supported in skills dir)

---

## Future HTTP Proxy Compatibility

This design is compatible with a local-sync proxy model:
- Proxy syncs skill directories from central server to `~/.brains/profiles/`
- Agent-facing behavior unchanged (base dir path in prompt, files on local filesystem)
- No changes needed to the loader or composition layer when proxy is introduced
- The proxy is purely a sync mechanism; zombiekit never knows files came from a remote source
