# Research: Profile Composition System

**Date**: 2025-12-22
**Feature**: 003-profiles

## 1. OS-Level File Locking (Registry)

### Decision
Use `github.com/gofrs/flock` for cross-platform file locking.

### Rationale
- Most widely-used cross-platform file locking library (1,400+ projects)
- Thread-safe, works on Linux, macOS, and Windows
- Provides `TryLock()` for non-blocking attempts
- BSD 3-Clause license

### Implementation Pattern
```go
import "github.com/gofrs/flock"

// Use separate lock file rather than locking data file directly
fileLock := flock.New("/path/to/registry.json.lock")
if err := fileLock.Lock(); err != nil {
    return err
}
defer fileLock.Unlock()

// Read-modify-write cycle here
```

### Cross-Platform Considerations
- Library uses `syscall.Flock()` on Linux/macOS, `windows.LockFileEx()` on Windows
- Use separate `.lock` file alongside the JSON file
- For atomic writes, write to temp file then rename (use `github.com/natefinch/atomic` for Windows compatibility if needed)

### Alternatives Considered
- `syscall.Flock()` directly: Not cross-platform (fails on Windows)
- Go internal `lockedfile`: Not public API
- `sync.Mutex`: Only protects within single process, not across concurrent CLI invocations

## 2. YAML Frontmatter Parsing

### Decision
Use `github.com/adrg/frontmatter` with `gopkg.in/yaml.v3` for YAML parsing.

### Rationale
- Gracefully handles missing frontmatter (returns original content unchanged)
- Supports standard `---` delimiter format
- Type-safe decoding via struct tags
- Well-maintained, MIT licensed

### Implementation Pattern
```go
import (
    "github.com/adrg/frontmatter"
)

type ProfileFrontmatter struct {
    Name        string   `yaml:"name"`
    Description string   `yaml:"description"`
    Includes    []string `yaml:"includes"`
    Inherits    *bool    `yaml:"inherits"` // pointer to detect missing vs false
}

func ParseProfile(content []byte) (ProfileFrontmatter, string, error) {
    var fm ProfileFrontmatter
    rest, err := frontmatter.Parse(bytes.NewReader(content), &fm)
    if err != nil {
        return ProfileFrontmatter{}, "", fmt.Errorf("parsing frontmatter: %w", err)
    }
    body, _ := io.ReadAll(rest)
    return fm, string(body), nil
}
```

### Error Handling
- Missing frontmatter: No error, struct has zero values, body contains full content
- Malformed YAML: Returns error with context
- Use `*bool` for `inherits` to distinguish "not set" (default true) from "explicitly false"

### Alternatives Considered
- Manual parsing with regex: Error-prone, harder to maintain
- `goldmark/frontmatter`: Only useful if already using goldmark for markdown rendering
- `github.com/gernest/front`: Less type-safe (returns `map[string]interface{}`)

## 3. DAG Cycle Detection

### Decision
Use DFS with path tracking (two-map approach).

### Rationale
- O(V + E) time complexity
- O(V) space complexity
- Clear separation between "in current path" and "fully processed"
- Easy to generate clear error messages with full cycle path

### Implementation Pattern
```go
func (c *Composer) detectCycle(name string, visited, pathSet map[string]bool, path []string) error {
    pathSet[name] = true
    path = append(path, name)

    profile := c.profiles[name]
    for _, included := range profile.Includes {
        if pathSet[included] {
            // Cycle detected - include the full path in error
            return fmt.Errorf("cycle detected: %s -> %s",
                strings.Join(path, " -> "), included)
        }
        if visited[included] {
            continue // Already processed in another branch
        }
        if err := c.detectCycle(included, visited, pathSet, path); err != nil {
            return err
        }
    }

    delete(pathSet, name)
    visited[name] = true
    return nil
}
```

### Key Properties
- `pathSet`: Tracks current root-to-leaf path only - detects cycles
- `visited`: Tracks all fully-processed profiles - enables deduplication across branches
- Validate on load (DAG building phase), not during composition

### Alternatives Considered
- Three-color marking (White/Gray/Black): Equivalent but less explicit about intent
- Tarjan's algorithm: Overkill for simple cycle detection
- Depth limit: Rejected per spec clarification - use proper cycle detection instead

## 4. Directory Walking for Profile Resolution

### Decision
Use `filepath.Walk` with git root detection.

### Rationale
- Standard library, no dependencies
- Can stop at git root boundary
- Walk up from CWD collecting `.brains/` directories

### Implementation Pattern
```go
func findBrainsDirs(startDir string) ([]string, error) {
    var dirs []string
    current := startDir
    gitRoot := findGitRoot(startDir)

    for {
        brainsPath := filepath.Join(current, ".brains", "profiles")
        if info, err := os.Stat(brainsPath); err == nil && info.IsDir() {
            dirs = append(dirs, brainsPath)
        }

        parent := filepath.Dir(current)
        if parent == current || current == gitRoot {
            break
        }
        current = parent
    }

    // Add global last
    if home, err := os.UserHomeDir(); err == nil {
        globalPath := filepath.Join(home, ".brains", "profiles")
        if info, err := os.Stat(globalPath); err == nil && info.IsDir() {
            dirs = append(dirs, globalPath)
        }
    }

    return dirs, nil
}
```

### Order
1. CWD `.brains/profiles/` (highest precedence)
2. Walk up to git root, collecting any `.brains/profiles/` directories
3. `~/.brains/profiles/` (global, lowest precedence)

For inheritance with `inherits: true`, prepend content from global first, then git root level down to local.

## Dependencies Summary

**New dependencies to add:**
- `github.com/gofrs/flock` - OS-level file locking
- `github.com/adrg/frontmatter` - YAML frontmatter parsing

**Already present:**
- `gopkg.in/yaml.v3` - YAML parsing (transitive dependency)
