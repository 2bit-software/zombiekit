# Tasks: .skill File Support

**Total tasks:** 10  
**Complexity:** Medium (6 modified files, 4 new files, all within `internal/profile/`)  
**Critical path:** T001 → T002 → T004 → T009

---

## Implementation Tasks

- [ ] T001 [US1,US2] Add `IsSkill bool` to `Profile` struct and `Format string` to `ListEntry` struct in `internal/profile/types.go`
  - Add `IsSkill bool` field to `Profile` (after `RawContent []byte`)
  - Add `Format string \`json:"format,omitempty"\`` field to `ListEntry`
  - No other changes — these are additive only
  - Acceptance: file compiles, existing tests still pass

- [ ] T002 [P] [US1,US2] Create `internal/profile/skill_loader.go` with `IsSkillDirectory()` and `LoadSkillProfile()`
  - Depends on: T001
  - `IsSkillDirectory(dir string) bool` — checks `os.Stat(filepath.Join(dir, "SKILL.md"))`
  - `LoadSkillProfile(dir string, source ProfileSource) (*Profile, error)` — reads `SKILL.md`, calls `ParseFrontmatter()`, sets `IsSkill: true`, derives name from dir if frontmatter name absent
  - `normalizeSkillDirName(name string) string` — lowercase, spaces/underscores→hyphens, strip non-alphanumeric, collapse hyphens
  - See `technical-spec.md` for full implementation
  - Acceptance: unit-testable in isolation; returns correct Profile with IsSkill=true

- [ ] T003 [P] [US1] Create `internal/profile/skill_extractor.go` with `ExtractPendingSkills()` and `ExtractSkillFile()`
  - Depends on: T001 (for package context only — no direct dependency on new fields)
  - `ExtractPendingSkills(profilesDir string) []error` — scans for `*.skill`, skips if target subdir exists, calls `ExtractSkillFile` per file
  - `ExtractSkillFile(skillPath, targetDir string) error` — opens ZIP, detects top-level prefix, extracts to tmpDir, validates SKILL.md exists post-extract, renames to targetDir
  - `detectTopLevelPrefix()`, `extractFiles()`, `writeZipEntry()`, `validateHasSkillMD()` helpers
  - Zip-slip mitigation: verify all extracted paths are under targetDir
  - Atomic: extract to `targetDir + ".tmp.{rand}"`, rename on success, RemoveAll on failure
  - See `technical-spec.md` for full implementation
  - Acceptance: correctly extracts sample ZIP; returns error for corrupt/no-SKILL.md ZIPs; idempotent

- [ ] T004 [US1,US2,US3] Modify `loadProfilesFromDir()` in `internal/profile/resolver.go` to support skill directories and `.skill` auto-extraction
  - Depends on: T002, T003
  - Add `"log/slog"` to imports if absent
  - Insert Pass 1 after the initial `os.ReadDir()`: call `ExtractPendingSkills(dir.Path)`, log warnings
  - Insert Pass 2: re-read directory entries after extraction
  - In loop: when `entry.IsDir()`, call `IsSkillDirectory()` → if true, call `LoadSkillProfile()`, check for name conflict with existing entry (warn + skip if conflict), add to profiles map
  - Skip `.skill` files in the loop (already handled by Pass 1)
  - Leave all existing `.md` loading logic untouched
  - See `technical-spec.md` for full modified function
  - Acceptance: profiles dir with subdir/SKILL.md returns it in map; .skill file is extracted and loaded; flat .md profiles unchanged

- [ ] T005 [P] [US2] Modify `composeContent()` in `internal/profile/composer.go` to prepend base directory for skill profiles
  - Depends on: T001
  - Can run in parallel with T002/T003
  - After `c.resolveContent(p)` call (and its error handling), add:
    ```go
    if p.IsSkill && p.Path != "" {
        baseDir := filepath.Dir(p.Path)
        content = "Base directory for this skill: " + baseDir + "\n\n" + content
    }
    ```
  - Add `"path/filepath"` to imports if absent
  - Acceptance: skill profile compose output starts with base dir line; flat profile compose output is unchanged

- [ ] T006 [P] [US2] Modify `List()` in `internal/profile/service.go` to populate `Format` field on `ListEntry`
  - Depends on: T001
  - Can run in parallel with T002/T003
  - In `List()`, before constructing `ListEntry`, determine format:
    ```go
    format := ""
    if p.IsSkill {
        format = "skill"
    }
    ```
  - Add `Format: format` to the `ListEntry` literal
  - Acceptance: profile-list response includes `"format":"skill"` for dir-layout profiles; flat profiles have no format field in JSON

---

## Test Tasks

- [ ] T007 [P] Write unit + integration tests in `internal/profile/skill_extractor_test.go`
  - Depends on: T003
  - Test cases:
    - Valid `.skill` ZIP with nested prefix dir → extracted, prefix stripped, all files present
    - Valid `.skill` ZIP flat layout (no prefix) → extracted correctly
    - Corrupt ZIP → error returned, no directory created
    - ZIP with no SKILL.md → error returned, extracted dir cleaned up
    - Idempotency: call `ExtractPendingSkills` twice → second call is no-op
    - Zip-slip: ZIP with `../../escape` entry → error, no file written outside targetDir
    - Empty profiles dir → no errors

- [ ] T008 [P] Write unit tests in `internal/profile/skill_loader_test.go`
  - Depends on: T002
  - Test cases:
    - Directory with SKILL.md (name in frontmatter) → Profile with correct name, IsSkill=true
    - Directory with SKILL.md (no frontmatter name) → name derived from directory name
    - Directory without SKILL.md → `IsSkillDirectory` returns false
    - `normalizeSkillDirName`: spaces, underscores, mixed case, unicode stripping, consecutive hyphens
    - `LoadSkillProfile` for nonexistent dir → error returned

- [ ] T009 [P] Extend `internal/profile/resolver_test.go` with skill directory tests
  - Depends on: T004
  - Test cases:
    - Profiles dir with `epic-planner/SKILL.md` → discovered alongside flat `.md` profiles
    - Profiles dir with `epic-planner.skill` ZIP → extracted and loaded in one call
    - `profile-list` result for skill profile has `format: "skill"`
    - `profile-list` result for flat profile has no `format` field
    - Flat `.md` and directory skill with same name in same dir → warning logged, directory wins
    - Same skill name in local and global dirs → local shadows global
    - Skill subdirs present in local, parent, and global profile dirs → all discovered

- [ ] T010 [P] Extend `internal/profile/composer_test.go` with skill composition tests
  - Depends on: T005
  - Test cases:
    - Compose skill profile → output starts with `Base directory for this skill: {abs-path}\n\n`
    - Compose flat `.md` profile → no base dir line present
    - `.md` profile with `includes: ["epic-planner"]` where `epic-planner` is a skill → base dir line present in output at skill's position
    - Compose two skill profiles → each has its own base dir line

---

## Execution Order

**Wave 1** (sequential — unblocks everything):
- T001

**Wave 2** (parallel — can all run simultaneously after T001):
- T002, T003, T005, T006

**Wave 3** (sequential — requires T002 + T003):
- T004

**Wave 4** (parallel — all tests, after respective impl tasks):
- T007 (after T003), T008 (after T002), T009 (after T004), T010 (after T005)

---

## Traceability

| Task | Plan Step | FR |
|------|-----------|----|
| T001 | Step 1 | All |
| T002 | Step 2 | FR-004, FR-009 |
| T003 | Step 3 | FR-001, FR-002, FR-003 |
| T004 | Step 4 | FR-001, FR-002, FR-004, FR-005, FR-009, FR-010 |
| T005 | Step 5 | FR-006, FR-007 |
| T006 | Step 6 | FR-008 |
| T007 | Step 7 | FR-001, FR-002, FR-003 |
| T008 | Step 7 | FR-004 |
| T009 | Step 7 | FR-001, FR-004, FR-005, FR-008, FR-010 |
| T010 | Step 7 | FR-006, FR-007, FR-009 |
