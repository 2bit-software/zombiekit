# Feature Specification: .skill File Support

**Feature Branch**: `69ce9fad-feature-skill-file-support`  
**Created**: 2026-04-02  
**Status**: Revised (v3)

## Background

Claude web exports skills as `.skill` files (ZIP archives). Each ZIP contains a directory with
`SKILL.md` (the skill definition) and any number of supporting files (scripts, templates, data).
The SKILL.md references siblings by relative path. The agent reads those files at runtime using
its normal Read/Bash tools, given the base directory.

Claude Code CLI uses the same pattern: skills live as `skill-name/SKILL.md` directories, and the
base directory path is prepended to skill content at invocation so the agent can locate siblings.

The user workflow is: drop a `.skill` file into a profiles directory, start the MCP server, use
the skill. No import step, no tooling — just drop-and-go.

## User Scenarios & Testing

### User Story 1 — Auto-Extraction on Server Init (Priority: P1)

A user drops `.skill` ZIP files into `~/.brains/profiles/` (or `.brains/profiles/`). When the MCP
server starts, it automatically finds and extracts them into subdirectories. The skills are
immediately available via `profile-list` and `profile-compose` without any manual step.

**Why this priority**: This is the primary UX. User moves files, restarts server, skills work.

**Independent Test**: Drop `epic-planner.skill` into `~/.brains/profiles/`. Start the MCP server.
Call `profile-list` — confirm `epic-planner` appears. Call `profile-compose ["epic-planner"]` —
confirm it returns skill content with base directory prefix.

**Acceptance Scenarios**:

1. **Given** a `.skill` file in `~/.brains/profiles/`, **When** the MCP server initializes,
   **Then** the file is extracted to `~/.brains/profiles/epic-planner/` with all its contents
2. **Given** extraction already happened (directory exists), **When** the server initializes again,
   **Then** the existing directory is left untouched; no double-extraction
3. **Given** a corrupt `.skill` file, **When** the server initializes, **Then** extraction is
   skipped with a warning; all other profiles and skills load normally
4. **Given** a `.skill` file in a local `.brains/profiles/`, **When** the server initializes,
   **Then** it is extracted in that same local profiles directory

---

### User Story 2 — Load Directory-Layout Skills (Priority: P1)

The profile system loads skills from subdirectory layout (`profiles/skill-name/SKILL.md`)
identically to flat `.md` profiles, and prepends the base directory path so the agent can access
sibling files.

**Why this priority**: Core loader change. Everything else depends on this.

**Independent Test**: Place `epic-planner/SKILL.md` in `.brains/profiles/`. Call `profile-compose
["epic-planner"]` — confirm response starts with `Base directory for this skill: {abs-path}`.

**Acceptance Scenarios**:

1. **Given** `{profiles-dir}/epic-planner/SKILL.md`, **When** `profile-list` is called, **Then**
   `epic-planner` appears alongside flat `.md` profiles
2. **Given** a directory-layout skill, **When** `profile-compose ["epic-planner"]` is called,
   **Then** response begins with `Base directory for this skill: {absolute-path-to-dir}` on its own
   line, followed by the SKILL.md body
3. **Given** a flat `.md` profile, **When** composed, **Then** no base directory line is present —
   existing behavior unchanged
4. **Given** a skill directory with sibling files, **When** the agent receives the composed output,
   **Then** it can access siblings via `{base-dir}/script.sh` etc. using its Read/Bash tools

---

### User Story 3 — Skill Profiles as First-Class Citizens (Priority: P2)

Directory-layout skills participate fully in composition: `includes:` resolution, shadowing, and
precedence work identically to flat profiles.

**Acceptance Scenarios**:

1. **Given** a `.md` profile with `includes: ["epic-planner"]` and `epic-planner` as a directory
   skill, **When** `profile-compose ["my-profile"]` is called, **Then** output contains the base
   directory line and SKILL.md content at the correct position
2. **Given** a local and global version of the same skill name, **When** `profile-list` is called,
   **Then** local shadows global (standard precedence)

---

### Edge Cases

- Directory has no `SKILL.md` → skip silently; not treated as a skill
- `SKILL.md` has no frontmatter `name:` → derive from directory name (lowercase, normalize)
- Corrupt/malformed `.skill` ZIP → skip with warning on init; other profiles load normally
- `.skill` ZIP contains no `SKILL.md` → skip with warning; extracted dir is not created
- `.skill` file and already-extracted directory of same name both present → directory takes precedence; `.skill` file ignored
- Same name as flat `.md` profile in same dir → conflict; emit warning; first-found wins

## Requirements

### Functional Requirements

- **FR-001**: On MCP server init, ALL profiles directories MUST be scanned for `.skill` ZIP files and any found MUST be extracted into same-named subdirectories in that profiles directory
- **FR-002**: Extraction MUST be idempotent: if the target subdirectory already exists, skip extraction (no overwrite, no error)
- **FR-003**: Extraction MUST preserve all files from the ZIP, not just `SKILL.md`
- **FR-004**: Profile discovery MUST recognize `{profiles-dir}/{name}/SKILL.md` as a valid profile alongside `{profiles-dir}/{name}.md`
- **FR-005**: All `.brains/profiles/` discovery paths (local, parent dirs, global) MUST support directory-layout skills
- **FR-006**: When composing a directory-layout profile, response MUST be prefixed with `Base directory for this skill: {absolute-path}\n\n` before the SKILL.md content
- **FR-007**: Flat `.md` profiles MUST NOT receive a base directory prefix
- **FR-008**: `profile-list` MUST include directory-layout skills; source field MUST be `"skill"` for them
- **FR-009**: Directory-layout skills MUST participate in `includes:` resolution identically to flat profiles
- **FR-010**: Standard shadowing rules MUST apply to directory-layout skills

### Key Entities

- **SkillDirectory**: A directory containing `SKILL.md` within a profiles directory
- **SkillProfile**: A profile loaded from a SkillDirectory; carries `basePath` set to the directory's absolute path
- **SkillFile**: A `.skill` ZIP archive dropped into a profiles directory; auto-extracted on server init

## Success Criteria

- **SC-001**: Drop `.skill` files into `~/.brains/profiles/`, start server, skills are immediately composable — zero manual steps
- **SC-002**: `profile-list` and `profile-compose` work transparently for both flat and directory-layout profiles
- **SC-003**: Agent receives base directory path and can access skill sibling files via its normal tools
- **SC-004**: All existing `.md` profile loader tests pass (zero regressions)
- **SC-005**: HTTP proxy future: proxy syncs skill directories locally; agent-facing behavior unchanged

## Testing Requirements

### Test Strategy

Integration tests using temp directories at the profile loader boundary. Unit test for ZIP
extraction logic (idempotency, corrupt file handling).

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Profiles dir with `.skill` file → server init extracts it to subdir |
| FR-002 | Integration | Second init with existing subdir → no re-extraction |
| FR-003 | Integration | Extracted dir contains all ZIP files, not just SKILL.md |
| FR-004 | Integration | Loader finds `name/SKILL.md` as profile alongside `name.md` |
| FR-005 | Integration | Directory skills found in local, parent, and global profile dirs |
| FR-006 | Integration | Compose of directory skill → output starts with base dir line |
| FR-007 | Integration | Compose of flat `.md` → no base dir line |
| FR-008 | Integration | `profile-list` includes directory skills with `source: "skill"` |
| FR-009 | Integration | `.md` profile `includes: ["dir-skill"]` resolves correctly |
| FR-010 | Integration | Local dir skill shadows global dir skill of same name |

### Edge Case Coverage

- Corrupt ZIP → warning logged, init continues, other profiles load
- ZIP with no SKILL.md → warning, extracted dir not created
- Existing subdir present → skip silently
- Same name flat + dir in same profiles dir → warning, first-found wins
