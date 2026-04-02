# Technical Spec: .skill File Support

## Data Model Changes

### `internal/profile/types.go`

```go
// Profile — add one field
type Profile struct {
    Name        string
    Path        string
    Source      ProfileSource
    Description string
    Includes    []string
    Inherits    bool
    Model       string
    Color       string
    Type        string
    Body        string
    RawContent  []byte
    IsSkill     bool   // NEW: true if loaded from skill-directory layout (name/SKILL.md)
}

// ListEntry — add one field
type ListEntry struct {
    Name        string        `json:"name"`
    Source      ProfileSource `json:"-"`
    SourceStr   string        `json:"source"`
    Path        string        `json:"path"`
    Description string        `json:"description"`
    Includes    []string      `json:"includes"`
    Inherits    bool          `json:"inherits"`
    Shadowed    bool          `json:"shadowed,omitempty"`
    Model       string        `json:"model,omitempty"`
    Color       string        `json:"color,omitempty"`
    Type        string        `json:"type,omitempty"`
    Format      string        `json:"format,omitempty"` // NEW: "skill" for dir-layout, omitted for flat
}
```

---

## New File: `internal/profile/skill_loader.go`

```go
package profile

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

var nonAlphanumHyphen = regexp.MustCompile(`[^a-z0-9-]+`)
var multiHyphen       = regexp.MustCompile(`-{2,}`)

// IsSkillDirectory returns true if dir contains a SKILL.md file.
func IsSkillDirectory(dir string) bool {
    info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
    return err == nil && !info.IsDir()
}

// LoadSkillProfile loads a Profile from a skill directory (dir/SKILL.md).
func LoadSkillProfile(dir string, source ProfileSource) (*Profile, error) {
    skillMD := filepath.Join(dir, "SKILL.md")
    content, err := os.ReadFile(skillMD)
    if err != nil {
        return nil, fmt.Errorf("reading SKILL.md in %s: %w", dir, err)
    }

    fm, body, err := ParseFrontmatter(content)
    if err != nil {
        return nil, fmt.Errorf("parsing SKILL.md in %s: %w", dir, err)
    }

    name := fm.Name
    if name == "" {
        name = normalizeSkillDirName(filepath.Base(dir))
    }

    return &Profile{
        Name:        name,
        Path:        skillMD,
        Source:      source,
        Description: fm.Description,
        Includes:    fm.Includes,
        Inherits:    fm.Inherits,
        Body:        body,
        RawContent:  content,
        IsSkill:     true,
    }, nil
}

// normalizeSkillDirName converts a directory name to a valid profile name.
// e.g. "My Cool Skill" → "my-cool-skill", "EpicPlanner" → "epicplanner"
func normalizeSkillDirName(name string) string {
    name = strings.ToLower(name)
    name = strings.ReplaceAll(name, " ", "-")
    name = strings.ReplaceAll(name, "_", "-")
    name = nonAlphanumHyphen.ReplaceAllString(name, "")
    name = multiHyphen.ReplaceAllString(name, "-")
    name = strings.Trim(name, "-")
    return name
}
```

---

## New File: `internal/profile/skill_extractor.go`

```go
package profile

import (
    "archive/zip"
    "fmt"
    "io"
    "math/rand"
    "os"
    "path/filepath"
    "strings"
)

// ExtractPendingSkills scans profilesDir for *.skill ZIPs and extracts any
// whose target subdirectory does not yet exist. Returns all errors encountered
// (non-fatal — caller should log and continue).
func ExtractPendingSkills(profilesDir string) []error {
    entries, err := os.ReadDir(profilesDir)
    if err != nil {
        return []error{fmt.Errorf("reading profiles dir %s: %w", profilesDir, err)}
    }

    var errs []error
    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".skill") {
            continue
        }
        skillName := strings.TrimSuffix(entry.Name(), ".skill")
        targetDir := filepath.Join(profilesDir, skillName)

        if _, err := os.Stat(targetDir); err == nil {
            continue // already extracted
        }

        skillPath := filepath.Join(profilesDir, entry.Name())
        if err := ExtractSkillFile(skillPath, targetDir); err != nil {
            errs = append(errs, fmt.Errorf("extracting %s: %w", entry.Name(), err))
        }
    }
    return errs
}

// ExtractSkillFile extracts a .skill ZIP to targetDir.
// Uses a temp directory + rename for atomicity.
func ExtractSkillFile(skillPath, targetDir string) error {
    r, err := zip.OpenReader(skillPath)
    if err != nil {
        return fmt.Errorf("opening zip: %w", err)
    }
    defer r.Close()

    prefix := detectTopLevelPrefix(r.File)
    tmpDir := targetDir + fmt.Sprintf(".tmp.%d", rand.Int63())

    if err := os.MkdirAll(tmpDir, 0o755); err != nil {
        return fmt.Errorf("creating temp dir: %w", err)
    }

    if err := extractFiles(r.File, tmpDir, prefix, targetDir); err != nil {
        os.RemoveAll(tmpDir)
        return err
    }

    if err := os.Rename(tmpDir, targetDir); err != nil {
        os.RemoveAll(tmpDir)
        return fmt.Errorf("finalizing extraction: %w", err)
    }

    // Validate the ZIP actually contained a SKILL.md; clean up if not.
    if err := validateHasSkillMD(targetDir); err != nil {
        os.RemoveAll(targetDir)
        return err
    }
    return nil
}

// detectTopLevelPrefix returns the single common top-level directory prefix
// shared by all ZIP entries (e.g. "epic-planner/"). Returns "" if entries are
// flat or if multiple top-level dirs exist.
func detectTopLevelPrefix(files []*zip.File) string {
    var first string
    for _, f := range files {
        parts := strings.SplitN(f.Name, "/", 2)
        if len(parts) < 2 {
            return "" // flat entry — no common prefix
        }
        if first == "" {
            first = parts[0]
        } else if parts[0] != first {
            return "" // multiple top-level dirs
        }
    }
    if first != "" {
        return first + "/"
    }
    return ""
}

// validateHasSkillMD returns an error if the extracted directory has no SKILL.md.
// Caller should remove targetDir on error.
func validateHasSkillMD(targetDir string) error {
    if _, err := os.Stat(filepath.Join(targetDir, "SKILL.md")); os.IsNotExist(err) {
        return fmt.Errorf("no SKILL.md found in skill archive")
    }
    return nil
}

func extractFiles(files []*zip.File, tmpDir, prefix, targetDir string) error {
    targetAbs, err := filepath.Abs(targetDir)
    if err != nil {
        return err
    }

    for _, f := range files {
        name := strings.TrimPrefix(f.Name, prefix)
        if name == "" || name == "/" {
            continue
        }

        destPath := filepath.Join(tmpDir, filepath.FromSlash(name))
        destAbs, err := filepath.Abs(destPath)
        if err != nil {
            return fmt.Errorf("resolving path %s: %w", name, err)
        }

        // Zip-slip mitigation: dest must be under targetDir
        rel, err := filepath.Rel(targetAbs, destAbs)
        if err != nil || strings.HasPrefix(rel, "..") {
            return fmt.Errorf("unsafe path in zip: %s", f.Name)
        }

        if f.FileInfo().IsDir() {
            if err := os.MkdirAll(destPath, f.Mode()); err != nil {
                return fmt.Errorf("creating dir %s: %w", name, err)
            }
            continue
        }

        if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
            return fmt.Errorf("creating parent for %s: %w", name, err)
        }

        if err := writeZipEntry(f, destPath); err != nil {
            return fmt.Errorf("writing %s: %w", name, err)
        }
    }
    return nil
}

func writeZipEntry(f *zip.File, destPath string) error {
    rc, err := f.Open()
    if err != nil {
        return err
    }
    defer rc.Close()

    out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, rc)
    return err
}
```

---

## Modified: `internal/profile/resolver.go`

Add `"log/slog"` to imports if not already present.

In `loadProfilesFromDir()`, insert Pass 1 after the initial `os.ReadDir()` call and modify the loop body to handle directories and `.skill` files:

```go
// Existing: entries, err := os.ReadDir(dir.Path) — unchanged

// Pass 1: extract pending .skill files (idempotent)
if errs := ExtractPendingSkills(dir.Path); len(errs) > 0 {
    for _, e := range errs {
        slog.Warn("skill extraction warning", "dir", dir.Path, "err", e)
    }
}

// Pass 2: re-read to pick up newly extracted subdirs
entries, err = os.ReadDir(dir.Path)
if err != nil {
    return nil, fmt.Errorf("re-reading directory %s: %w", dir.Path, err)
}

for _, entry := range entries {
    if entry.IsDir() {
        skillDir := filepath.Join(dir.Path, entry.Name())
        if !IsSkillDirectory(skillDir) {
            continue
        }
        p, err := LoadSkillProfile(skillDir, dir.Source)
        if err != nil {
            slog.Warn("failed to load skill", "dir", skillDir, "err", err)
            continue
        }
        // Conflict: flat .md and directory skill share the same name in the same dir.
        // Directory is processed first (loop order); emit warning, first-found wins.
        if existing, ok := profiles[p.Name]; ok {
            slog.Warn("skill name conflict: directory and flat profile share name",
                "name", p.Name, "skill_dir", skillDir, "flat", existing.Path)
            continue
        }
        profiles[p.Name] = p
        continue
    }

    name := entry.Name()
    if strings.HasSuffix(name, ".skill") {
        continue // handled by ExtractPendingSkills above
    }
    if !strings.HasSuffix(name, ".md") {
        continue
    }
    // ... existing .md loading unchanged ...
}
```

---

## Modified: `internal/profile/composer.go`

In `composeContent()`, after calling `c.resolveContent(p)`:

```go
content, inherited, err := c.resolveContent(p)
if err != nil { /* ... */ }

if p.IsSkill && p.Path != "" {
    baseDir := filepath.Dir(p.Path)
    content = "Base directory for this skill: " + baseDir + "\n\n" + content
}

contentParts = append(contentParts, content)
```

---

## Modified: `internal/profile/service.go`

In `List()`, when building each `ListEntry`:

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

---

## Invariants

- Flat `.md` profiles: `IsSkill == false`, `Format == ""`, no base dir prefix in compose output
- Skill directory profiles: `IsSkill == true`, `Format == "skill"`, base dir prefix in compose output
- Extraction is idempotent: running `ExtractPendingSkills` N times on the same dir is identical to running it once
- The `.skill` ZIP file is left in place after extraction — it is skipped on subsequent loads
- Zip-slip: all extracted paths verified to be under `targetDir` before writing
