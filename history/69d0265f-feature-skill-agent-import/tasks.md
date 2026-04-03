# Tasks: Skill/Agent Import

## Task List

- [ ] T001 Add `IsShim()` helper to `internal/skill/install.go`
  - Add `func IsShim(body string) bool` that checks for `mcp__zombiekit__profile-compose` in body
  - Traces to: FR-4
  - AC: `IsShim("Call mcp__zombiekit__profile-compose...")` returns true; `IsShim("regular content")` returns false

- [ ] T002 Create discovery functions in `internal/skill/discover.go`
  - Define `DiscoverableItem` struct
  - Implement `DiscoverSkills(workingDir)`, `DiscoverAgents(workingDir)`, `DiscoverAll(workingDir)`
  - Use `recall/claude.DefaultClaudePath()` for home resolution
  - Resolve symlinks, skip broken ones, exclude shims via `IsShim()`
  - Detect name collisions between skills and agents
  - Traces to: FR-1, FR-2, FR-3, FR-4, FR-5, FR-24, FR-25
  - AC: Discovers skills from `~/.claude/skills/`, agents from `~/.claude/agents/`, excludes shims, skips broken symlinks, reports collisions
  - Depends on: T001

- [ ] T003 Create import types and helpers in `internal/skill/import.go`
  - Define `ImportResult`, `ImportedItem`, `SkippedItem`, `ShimItem`, `ImportOptions` structs
  - Implement `scopeDir()`, `copyDirContents()`, `transformSkillFrontmatter()`, `transformAgentFrontmatter()`, `generateAgentShim()`
  - Traces to: FR-6, FR-9, FR-10, FR-13, FR-19
  - AC: `scopeDir("global", "")` returns `~/.brains/profiles`; `copyDirContents` copies all files except excluded; agent shim preserves full original frontmatter

- [ ] T004 Implement `Import()` function in `internal/skill/import.go`
  - Skill import: transform frontmatter, write SKILL.md, copy supporting files
  - Agent import: transform frontmatter, prepend skills comment, write SKILL.md
  - Shim generation: skill shims via `GenerateContent()`+`WriteSkill()`, agent shims via `generateAgentShim()`
  - Skip items with invalid frontmatter (add to Skipped)
  - Check destination collision (return error)
  - Traces to: FR-7, FR-8, FR-9, FR-10, FR-11, FR-12, FR-13, FR-14, FR-15, FR-16, FR-17, FR-18, FR-19, FR-20, FR-23
  - AC: Skill with scripts imports complete; agent with skills field gets HTML comment; invalid frontmatter is skipped; collision detected
  - Depends on: T002, T003

- [ ] T005 [P] Create MCP tool handlers in `internal/mcp/tools/skillimport/tool.go`
  - `Tool` struct with `NewTool()`, `ExecuteList()`, `ExecuteImport()`
  - `ExecuteList`: calls `DiscoverAll()`, returns JSON array
  - `ExecuteImport`: parses args, calls `DiscoverAll()` then `Import()`, returns JSON result
  - Traces to: FR-5, FR-21, FR-22
  - AC: List returns JSON with name/type/description/source_path; Import returns JSON with imported/skipped/shims
  - Depends on: T004

- [ ] T006 [P] Wire tools into MCP server in `internal/mcp/server.go`
  - Import `skillimporttool` package
  - Add `skillImportTool` field to Server struct
  - Instantiate in `NewServer`
  - Register `skill-import-list` and `skill-import` tools with handler methods
  - Add `handleSkillImportList` and `handleSkillImport` handler methods
  - AC: Both tools appear in MCP tool list and dispatch correctly
  - Depends on: T005

- [ ] T007 Write discovery tests in `internal/skill/discover_test.go`
  - Test `DiscoverSkills` with fixture temp directory containing real SKILL.md files
  - Test `DiscoverAgents` with fixture temp directory containing real agent .md files
  - Test shim exclusion (FR-4)
  - Test broken symlink skip (FR-24)
  - Test name collision detection (FR-25)
  - Test `IsShim()` helper
  - AC: All tests pass, cover happy path + edge cases
  - Depends on: T002

- [ ] T008 Write import tests in `internal/skill/import_test.go`
  - Test skill import with supporting files → all files copied
  - Test agent import → SKILL.md created with correct frontmatter
  - Test agent with `skills:` field → HTML comment in body
  - Test skill shim generation → delegating content
  - Test agent shim generation → full original frontmatter preserved
  - Test skip on invalid frontmatter
  - Test destination collision detection
  - AC: All tests pass, cover all FR acceptance criteria
  - Depends on: T004

## Dependency Graph

```
T001 → T002 → T004 → T005 → T006
              ↗
T003 ────────┘

T002 → T007 (parallel with T004+)
T004 → T008 (parallel with T005+)
```

## Execution Order

1. **Wave 1**: T001, T003 (parallel — no dependencies)
2. **Wave 2**: T002 (depends on T001)
3. **Wave 3**: T004, T007 (parallel — T004 depends on T002+T003, T007 depends on T002)
4. **Wave 4**: T005, T008 (parallel — T005 depends on T004, T008 depends on T004)
5. **Wave 5**: T006 (depends on T005)

## Summary

- **Total tasks**: 8
- **Parallel opportunities**: 3 waves have parallel tasks
- **Complexity**: Simple (5 new files, 1 modified)
- **Critical path**: T001 → T002 → T004 → T005 → T006
