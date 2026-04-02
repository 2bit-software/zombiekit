# Implementation Plan: .skill File Support

## Dependency Order

```
Step 1: Profile struct + types
    ↓
Step 2: Skill loader (reads SKILL.md from subdirs)
    ↓
Step 3: ZIP extractor (extracts .skill → subdir)
    ↓
Step 4: Wire extraction into load path
    ↓
Step 5: Composer: base dir prefix for skills
    ↓
Step 6: profile-list: expose format field
    ↓
Step 7: Tests
```

---

## Step 1 — Profile Struct: Add `IsSkill` Field

**File:** `internal/profile/types.go`

Add `IsSkill bool` to the `Profile` struct. Set to `true` when loaded from a subdirectory layout.
Empty for all existing flat `.md` profiles — no behavior change.

```go
type Profile struct {
    // ... existing fields unchanged ...
    IsSkill bool // true if loaded from a skill-directory layout (name/SKILL.md)
}
```

Add `Format string` to `ListEntry` for JSON output:

```go
type ListEntry struct {
    // ... existing fields unchanged ...
    Format string `json:"format,omitempty"` // "skill" for directory-layout, omitted for flat
}
```

**Touches:** `types.go` only. No other files change in this step.

---

## Step 2 — Skill Loader

**New file:** `internal/profile/skill_loader.go`

Two functions:

```go
// IsSkillDirectory returns true if dir contains a SKILL.md file.
func IsSkillDirectory(dir string) bool

// LoadSkillProfile loads a Profile from a skill directory (dir/SKILL.md).
// Sets Profile.IsSkill = true and Profile.Path to the absolute SKILL.md path.
// Name comes from SKILL.md frontmatter; falls back to normalized directory name.
func LoadSkillProfile(dir string, source ProfileSource) (*Profile, error)
```

`LoadSkillProfile` reuses the existing `ParseFrontmatter()` for content parsing. The only
difference from flat profile loading is:
- Read from `filepath.Join(dir, "SKILL.md")`
- Set `IsSkill = true` on the returned Profile
- Derive name from directory name if frontmatter `name:` is absent

Name normalization (directory name → profile name): lowercase, replace spaces/underscores with
hyphens, collapse consecutive hyphens.

---

## Step 3 — ZIP Extractor

**New file:** `internal/profile/skill_extractor.go`

```go
// ExtractSkillFile extracts a .skill ZIP to targetDir.
// Creates targetDir; returns error if it already exists.
// Strips the common top-level directory prefix from ZIP entries (if present).
// Applies zip-slip mitigation on all entry paths.
func ExtractSkillFile(skillPath, targetDir string) error

// ExtractPendingSkills scans profilesDir for *.skill files and extracts
// any whose target subdirectory does not yet exist. Non-fatal: collects
// errors and returns them all, continuing on failure.
func ExtractPendingSkills(profilesDir string) []error
```

**ZIP entry handling:**
1. Open ZIP with `archive/zip.OpenReader()`
2. Detect common top-level prefix: if all entries share one top-level dir (e.g. `epic-planner/`), strip it
3. For each entry:
   - Compute target path: `filepath.Join(targetDir, strippedName)`
   - Zip-slip check: `filepath.Rel(targetDirAbs, targetPathAbs)` must not start with `..`
   - If directory entry: `os.MkdirAll()`
   - If file entry: create parent dirs, write contents, apply `file.Mode()`
4. Use temp-dir-then-rename pattern: extract to `targetDir + ".tmp"`, rename on success, clean up on failure

**Idempotency:** `ExtractPendingSkills` checks `os.Stat(targetDir)` before extracting. If the
directory exists (regardless of whether the `.skill` file is still present), skip.

---

## Step 4 — Wire into Load Path

**File:** `internal/profile/resolver.go`, function `loadProfilesFromDir()`

Current loop:
```go
for _, entry := range entries {
    if entry.IsDir() {
        continue  // ← change this
    }
    // .md loading...
}
```

Two-pass approach to handle extraction creating new subdirs during the same load cycle:

```go
func (r *Resolver) loadProfilesFromDir(dir ResolvedDirectory) (map[string]*Profile, error) {
    // Pass 1: extract any pending .skill files
    if errs := ExtractPendingSkills(dir.Path); len(errs) > 0 {
        for _, e := range errs {
            slog.Warn("skill extraction failed", "dir", dir.Path, "err", e)
        }
    }

    // Pass 2: re-read directory (picks up newly extracted subdirs)
    entries, err := os.ReadDir(dir.Path)
    // ...

    for _, entry := range entries {
        if entry.IsDir() {
            skillDir := filepath.Join(dir.Path, entry.Name())
            if IsSkillDirectory(skillDir) {
                p, err := LoadSkillProfile(skillDir, dir.Source)
                // handle err, add to profiles map
            }
            continue
        }
        if strings.HasSuffix(entry.Name(), ".skill") {
            continue  // already handled by ExtractPendingSkills above
        }
        if !strings.HasSuffix(entry.Name(), ".md") {
            continue
        }
        // existing .md loading unchanged...
    }
}
```

The two-pass approach (extract, then re-read) means newly extracted skill dirs are discovered in
the same `loadProfilesFromDir` call. No separate init required.

---

## Step 5 — Composer: Base Directory Prefix

**File:** `internal/profile/composer.go`, function `composeContent()`

In `resolveContent()` or in the content assembly loop — wherever per-profile content is assembled
before joining — prepend the base dir line for skill profiles:

```go
content, inherited, err := c.resolveContent(p)
// ...
if p.IsSkill && p.Path != "" {
    baseDir := filepath.Dir(p.Path)
    content = "Base directory for this skill: " + baseDir + "\n\n" + content
}
contentParts = append(contentParts, content)
```

This ensures the base dir line appears at the top of each skill's contribution to the composed
output, even when the skill is pulled in via `includes:`.

**Flat profiles:** `IsSkill` is `false` → no change whatsoever.

---

## Step 6 — profile-list: Format Field

**File:** `internal/profile/service.go`, function `List()`

Populate the new `Format` field on `ListEntry`:

```go
format := ""
if p.IsSkill {
    format = "skill"
}
entry := ListEntry{
    // ... existing fields ...
    Format: format,
}
```

The field is `omitempty` in JSON so flat profiles produce no extra field in the response.

---

## Step 7 — Tests

**File:** `internal/profile/skill_extractor_test.go` (new)
- Extract a valid `.skill` ZIP → verify directory created with correct files (not just SKILL.md)
- Extract with nested prefix dir → prefix stripped correctly
- Corrupt ZIP → error returned, no partial directory left
- ZIP with no SKILL.md → extraction fails, extracted dir cleaned up, error returned
- Idempotency: extract twice → second call is no-op (returns nil error)
- Zip-slip: ZIP with `../../etc/passwd` entry → error, no file created

**File:** `internal/profile/skill_loader_test.go` (new)
- Directory with `SKILL.md` (with frontmatter name) → profile loaded with correct name
- Directory with `SKILL.md` (no frontmatter name) → name derived from directory name
- Directory without `SKILL.md` → `IsSkillDirectory` returns false, not loaded
- `IsSkill` flag is set true; flat `.md` profiles have `IsSkill` false

**File:** `internal/profile/resolver_test.go` (extend existing)
- Profiles dir with skill subdir → discovered alongside flat profiles
- Profiles dir with `.skill` file → extracted and loaded in one `loadProfilesFromDir` call
- Skill profile in `profile-list` output has `format: "skill"`
- Flat profile in `profile-list` output has no `format` field
- Flat `.md` and directory skill with same name in same dir → warning logged, directory wins (processed first)
- Same skill name in local and global profiles dirs → local shadows global (standard precedence)
- Skill subdirs in local, parent, and global profile dirs → all discovered

**File:** `internal/profile/composer_test.go` (extend existing)
- Compose skill profile → output starts with `Base directory for this skill: {path}`
- Compose flat profile → no base dir line in output
- `.md` profile with `includes: ["skill-name"]` → base dir line present in output at skill's position

---

## Files Touched Summary

| File | Change |
|------|--------|
| `internal/profile/types.go` | Add `IsSkill bool` to `Profile`; add `Format string` to `ListEntry` |
| `internal/profile/skill_loader.go` | New — `IsSkillDirectory()`, `LoadSkillProfile()` |
| `internal/profile/skill_extractor.go` | New — `ExtractSkillFile()`, `ExtractPendingSkills()` |
| `internal/profile/resolver.go` | `loadProfilesFromDir()`: two-pass, call extractor, call skill loader for subdirs |
| `internal/profile/composer.go` | `composeContent()`: prepend base dir for `IsSkill` profiles |
| `internal/profile/service.go` | `List()`: populate `Format` field on `ListEntry` |
| `internal/profile/*_test.go` | New and extended tests |

**Not touched:** `mcp/server.go`, `mcp/tools/profile/tool.go`, `config/tools.go` — the MCP layer
needs no changes; extraction and loading are transparent to it.

---

## Risk & Uncertainty

- **ZIP prefix detection**: Most `.skill` files from Claude web have a single top-level directory.
  If a ZIP has entries at both root and in a subdirectory, prefix detection logic needs a clear
  rule (e.g. "extract everything relative to the most common prefix"). Low risk — Claude web
  consistently uses `{skill-name}/` as the single prefix.

- **Concurrent access**: If two processes extract the same `.skill` file simultaneously, both may
  attempt to create the same temp directory. Mitigation: use a unique temp dir name
  (`targetDir + ".tmp." + randomSuffix`) and ignore "already exists" errors on rename.

- **`.skill` file left after extraction**: The original ZIP remains in the profiles directory.
  `ExtractPendingSkills` skips it once the subdir exists. This is fine — user may want the
  original for backup or re-import.
