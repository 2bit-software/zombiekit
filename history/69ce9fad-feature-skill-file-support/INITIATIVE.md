# Initiative: skill-file-support

**Type**: feature
**Status**: completed
**Created**: 2026-04-02
**Completed**: 2026-04-02
**ID**: 69ce9fad-feature-skill-file-support

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-04-02 10:45 |
| plan | completed | 2026-04-02 11:30 |
| tasks | completed | 2026-04-02 11:45 |
| implement | completed | 2026-04-02 12:30 |

## Description

Add native support for `.skill` files (ZIP archives from Claude web) in the zombiekit profile
system. Drop a `.skill` file into any profiles directory; on MCP server init it is auto-extracted
into a subdirectory. Skills are then discoverable and composable identically to flat `.md` profiles,
with the base directory path prepended to composed content so the agent can access sibling files.

## Goals

1. MCP server init auto-extracts `.skill` ZIPs in profile directories (idempotent)
2. Profile loader discovers `name/SKILL.md` directory layout alongside `name.md` flat layout
3. Compose prepends `Base directory for this skill: {path}` for directory-layout profiles only
4. `profile-list` and `profile-compose` work transparently for both layouts
5. Zero regression in existing `.md` profile behavior

## Progress

- [x] Research: codebase profile architecture + `.skill` file format
- [x] Spec: business-spec.md + technical-requirements-research.md
- [x] Audit: completeness + AI-consumer audits passed (issues resolved)
- [x] Plan: implementation plan + technical spec
- [x] Implement: 6 files modified/created, 35 new tests, all passing

## Completion

**Completed**: 2026-04-02  
**Duration**: same day

### Outcomes

- **skill_loader.go** (new): `IsSkillDirectory()`, `LoadSkillProfile()`, `normalizeSkillDirName()`
- **skill_extractor.go** (new): `ExtractSkillFile()`, `ExtractPendingSkills()`, zip-slip protection
- **resolver.go** (modified): Two-pass load — auto-extract `.skill` ZIPs, discover `name/SKILL.md` subdirs
- **composer.go** (modified): Prepends `Base directory for this skill: {path}` for skill profiles
- **types.go** (modified): `IsSkill bool` on `Profile`, `Format string` on `ListEntry`
- **service.go** (modified): Populates `format: "skill"` in profile-list output
- **4 test files** (new): 35 tests covering extractor, loader, resolver, composer

### Notes

Key design decisions made during spec:
- Drop-and-go UX: user drops `.skill` into profiles dir, extraction happens automatically on first load
- Mirrors Claude Code CLI pattern: `Base directory for this skill: {path}` prepended to composed content
- HTTP proxy compatible: proxy syncs skill dirs locally, agent behavior unchanged
- One bug caught during tests: zip-slip check was against wrong directory (targetDir vs tmpDir)
