# Implementation Plan: Skill/Agent Import

## Overview

Two new MCP tools: `skill-import-list` (discovery) and `skill-import` (import + optional shim). Follows the existing `skill-install` tool pattern.

## Step 1: Discovery Package

**Package**: `internal/skill/discover.go`

### Types

```go
type DiscoverableItem struct {
    Name        string // derived from directory/file name
    Type        string // "skill" or "agent"
    Description string // from frontmatter
    SourcePath  string // resolved absolute path to source file
    IsShim      bool   // true if body contains "mcp__zombiekit__profile-compose"
}
```

### Functions

```go
// DiscoverSkills finds all Claude Code skills in global and local dirs.
// Resolves symlinks. Returns only non-shim items (FR-4).
func DiscoverSkills(workingDir string) ([]DiscoverableItem, error)

// DiscoverAgents finds all Claude Code agents in global and local dirs.
// Resolves symlinks.
func DiscoverAgents(workingDir string) ([]DiscoverableItem, error)

// DiscoverAll combines skills and agents, sorted by name.
// Detects name collisions between skills and agents (FR-25).
func DiscoverAll(workingDir string) ([]DiscoverableItem, []string, error)
// Returns: items, name collision warnings, error
```

**Logic**:
- Use `recall/claude.DefaultClaudePath()` for `~/.claude` home resolution (reuse existing)
- Skills: walk `{claudeHome}/skills/` and `.claude/skills/`, look for `SKILL.md` in each subdirectory
- Agents: walk `{claudeHome}/agents/` and `.claude/agents/`, look for `*.md` files
- Use `os.Stat` after `filepath.EvalSymlinks` to resolve symlinks (FR-3)
- Skip broken symlinks with warning (FR-24)
- Parse frontmatter to extract name/description
- Use new `IsShim(body)` to detect shims (FR-4)

**Depends on**: `internal/profile/frontmatter.go` → `ParseFrontmatter()`, `internal/recall/claude` → `DefaultClaudePath()`

### New helper in `internal/skill/install.go`

```go
// IsShim returns true if the body delegates to profile-compose (i.e., is a zombiekit shim).
func IsShim(body string) bool
```

---

## Step 2: Import Package

**Package**: `internal/skill/import.go`

### Types

```go
type ImportResult struct {
    Imported []ImportedItem
    Skipped  []SkippedItem
    Shims    []ShimItem
}

type ImportedItem struct {
    Name string
    Type string // "skill" or "agent"
    Path string // destination path
}

type SkippedItem struct {
    Name   string
    Reason string
}

type ShimItem struct {
    Name string
    Path string
}

type ImportOptions struct {
    Names      []string
    Scope      string // "local" or "global"
    Shim       bool
    WorkingDir string
}
```

### Functions

```go
// Import imports the named skills/agents into zombiekit profiles.
func Import(opts ImportOptions, items []DiscoverableItem) (*ImportResult, error)
```

**Logic per item**:

1. Find the item in `items` by name
2. If not found or item has invalid frontmatter → add to `Skipped` (FR-23)
3. Determine destination: `{scope-dir}/.brains/profiles/{name}/`
4. Check if destination exists → return collision error (FR-7, handled by caller)
5. **Skill import** (FR-8 through FR-11):
   - Read source `SKILL.md`, parse frontmatter
   - Transform frontmatter: keep `name`, `description`, drop `allowed-tools`
   - Write new `SKILL.md` to destination with transformed frontmatter + original body
   - Copy all other files/subdirectories from source dir (FR-9)
6. **Agent import** (FR-12 through FR-15):
   - Read source `.md` file, parse frontmatter
   - Transform frontmatter: keep `name`, `description`, drop `model`, `skills`, `memory`, `color`
   - If `skills:` field was present, prepend HTML comment to body (FR-15)
   - Create destination directory, write `SKILL.md` with transformed content
7. If `Shim` is true:
   - **Skill shim** (FR-18): use existing `skill.GenerateContent()` + `skill.WriteSkill()` with `skill.TargetDir()` for path resolution
   - **Agent shim** (FR-19): generate agent shim with full original frontmatter + `allowed-tools`, write to original agent path. Follow `yaml.Marshal` + delimiter pattern from `profile/importer.go:convertClaudeToBrains()`

**Depends on**:
- `internal/skill/install.go` → `GenerateContent()`, `WriteSkill()`, `TargetDir()`, `IsShim()`
- `internal/profile/frontmatter.go` → `ParseFrontmatter()`
- `internal/recall/claude` → `DefaultClaudePath()`

### Helper Functions

```go
// transformSkillFrontmatter strips allowed-tools, preserves name/description.
func transformSkillFrontmatter(fm ProfileFrontmatter) ProfileFrontmatter

// transformAgentFrontmatter strips model/skills/memory/color, preserves name/description.
func transformAgentFrontmatter(fm map[string]any) map[string]any

// generateAgentShim creates an agent shim with full original frontmatter preserved.
func generateAgentShim(name string, originalFrontmatter map[string]any, description string) string

// copyDirContents copies all files/subdirs from src to dst, excluding excludeFiles.
func copyDirContents(src, dst string, excludeFiles []string) error

// scopeDir returns the base directory for the given scope.
func scopeDir(scope, workingDir string) (string, error)
```

**Note on agent frontmatter**: Agent frontmatter has fields not in `ProfileFrontmatter` (`model`, `skills`, `memory`, `color`). Use `map[string]any` from raw YAML parse for agent source files to preserve all fields for shim generation.

---

## Step 3: MCP Tool — `skill-import-list`

**Package**: `internal/mcp/tools/skillimport/tool.go`

### Tool Definition

```go
type Tool struct{}

func NewTool() *Tool

func (t *Tool) ExecuteList(ctx context.Context, args map[string]any) (string, error)
// Args: working_directory (optional)
// Returns: JSON array of {name, type, description, source_path}
```

### MCP Registration

```go
mcp.NewTool("skill-import-list",
    mcp.WithDescription("List Claude Code skills and agents available for import into zombiekit profiles"),
    mcp.WithString("working_directory", mcp.Description("Working directory for local discovery")),
)
```

---

## Step 4: MCP Tool — `skill-import`

**Package**: `internal/mcp/tools/skillimport/tool.go` (same file)

### Tool Definition

```go
func (t *Tool) ExecuteImport(ctx context.Context, args map[string]any) (string, error)
// Args: names ([]string, required), scope (string, required), shim (bool, optional), working_directory (optional)
// Returns: JSON ImportResult
```

### MCP Registration

```go
mcp.NewTool("skill-import",
    mcp.WithDescription("Import Claude Code skills and agents into zombiekit profiles"),
    mcp.WithArray("names", mcp.Required(), mcp.Description("Names of skills/agents to import")),
    mcp.WithString("scope", mcp.Required(), mcp.Description("Destination: 'local' or 'global'")),
    mcp.WithBoolean("shim", mcp.Description("Write shims in original Claude locations (default: false)")),
    mcp.WithString("working_directory", mcp.Description("Working directory for local scope resolution")),
)
```

---

## Step 5: Server Wiring

**File**: `internal/mcp/server.go`

1. Import `skillimporttool "github.com/2bit-software/zombiekit/internal/mcp/tools/skillimport"`
2. Add `skillImportTool *skillimporttool.Tool` field to `Server` struct
3. Instantiate in `NewServer`: `skillImportTool: skillimporttool.NewTool()`
4. Register both tools in `registerTools`:
   - `skill-import-list` → `s.handleSkillImportList`
   - `skill-import` → `s.handleSkillImport`
5. Add handler methods following existing pattern

---

## Step 6: Tests

**Files**:
- `internal/skill/discover_test.go` — test discovery with fixture directories
- `internal/skill/import_test.go` — test import transformation, copy, shim generation

**Test cases**:
- Discover skills from a temp directory with real SKILL.md files
- Discover agents from a temp directory with real .md files
- Exclude shim skills from discovery
- Skip broken symlinks
- Import a skill with supporting files → verify all files copied
- Import an agent → verify SKILL.md created with correct frontmatter
- Import an agent with `skills:` field → verify HTML comment in body
- Generate skill shim → verify delegating content
- Generate agent shim → verify full original frontmatter preserved
- Skip item with missing frontmatter → verify warning in result
- Detect name collision between skill and agent

---

## Dependency Order

```
Step 1 (discover) → Step 2 (import) → Step 3 (list tool) + Step 4 (import tool) → Step 5 (wiring) → Step 6 (tests)
```

Steps 3 and 4 can be done in parallel since they're in the same file.

## Files to Create

| File | Purpose |
|------|---------|
| `internal/skill/discover.go` | Discovery logic |
| `internal/skill/import.go` | Import + transformation + shim logic |
| `internal/mcp/tools/skillimport/tool.go` | MCP tool handlers |
| `internal/skill/discover_test.go` | Discovery tests |
| `internal/skill/import_test.go` | Import tests |

## Files to Modify

| File | Change |
|------|--------|
| `internal/mcp/server.go` | Register new tools + handler methods |
