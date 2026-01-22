---
status: complete
updated: 2026-01-21
---

# Research: GUI Prompts Management

## Executive Summary

The ZombieKit GUI uses a Go templates + HTMX + Tailwind architecture with a plugin-based system. The prompts system (workflows, profiles, steps) uses file-based storage with YAML frontmatter across three resolution levels (local → global → embedded). Adding a "Prompts" section requires creating a new GUI plugin that leverages existing MCP tools for data operations.

## Findings

### Codebase Context

**GUI Architecture:**
- Server-side rendering with Go templates (`html/template`)
- HTMX 1.9.10 for interactivity (no React/Vue/Svelte)
- Tailwind CSS for styling
- Chi v5 router with plugin-based routing

**Navigation Structure:**
- Fixed left sidebar (264px) with dark theme
- Plugins register `SidebarItem` with ID, Label, Path, Order, Badge, Children
- Routes scoped per plugin: `/plugin-name/*`
- Active state managed server-side, updated via JS

**Existing CRUD Patterns (Memory Plugin):**
- List: GET `/memory` with query params (`?q=search&page=1&limit=20`)
- Create: GET `/new` → POST `/`
- View: GET `/{name}`
- Edit: GET `/{name}/edit` → PUT `/{name}`
- Delete: GET `/{name}/delete` → DELETE `/{name}`
- HTMX for partial updates, `HX-Redirect` for mutations

**Plugin Interface:**
```go
type WebPlugin interface {
    SidebarItems() []SidebarItem
    MountRoutes(r chi.Router)
}
type TemplatePlugin interface {
    WebPlugin
    Templates() fs.FS
}
type Searchable interface {
    Search(query, maxResults, sortOrder) ([]SearchResult, error)
}
```

### Prompts Data Models

**Workflows:**
- Location: `embed/workflows/`, `~/.brains/workflows/`, `.brains/workflows/`
- Structure: YAML frontmatter (name, description) + markdown body
- Operations: Load by name (read-only in embedded, writable elsewhere)

**Profiles (Domain Agents):**
- Location: `embed/profiles/`, `~/.brains/profiles/`, `.brains/profiles/`
- Structure: YAML frontmatter (name, description, type, includes, inherits, model, color) + markdown body
- Types: domain, action, step, skill
- Operations: Compose, List, Show, Save, Validate

**Steps:**
- Location: `embed/steps/`, `~/.brains/steps/`, `.brains/steps/`
- Structure: YAML frontmatter (name, description, profiles, files, type) + markdown body
- Operations: Execute (within initiative context)

**Common Attributes:**
- Name (derived from filename)
- Description
- Source (local, global, embedded)
- Path (absolute filesystem path)
- Body (markdown content)

### Domain Knowledge

**UX Patterns for Prompt Management:**
- Tabbed or segmented navigation for categories (workflows, profiles, steps)
- Card or table views for listing items
- Inline editing vs modal editing (existing codebase uses separate pages)
- Preview pane for markdown content
- Source indicators (local/global/embedded badges)
- Search and filter capabilities
- Sort by name, date created, date updated

**Similar Tools:**
- Langflow: Visual prompt builder with categories
- PromptLayer: List view with search, filter by tags
- Claude Code MCP: File-based profiles with compose operations

### Dependencies and Constraints

**Existing Services Available:**
- `internal/profile/service.go` - Profile list, compose, show, save
- `internal/workflow/service.go` - Workflow load, list
- `internal/step/service.go` - Step load, list, execute

**MCP Tools (can be called from GUI handlers):**
- `profile-compose`, `profile-list`, `profile-save`
- `workflow-compose`
- `step` (execute)

**Constraints:**
- Embedded assets are read-only (compile-time)
- Local/global paths can be written to
- File operations must handle missing directories
- YAML frontmatter parsing required

## Decision Points

- [x] **D1**: Navigation approach - Single top-level "Prompts" with sub-tabs/filters vs three separate sidebar items
  - **Decision**: Single "Prompts" top-level with category filter tabs (cleaner UX, avoids sidebar clutter)

- [x] **D2**: Display format - Card view vs Table view vs Hybrid
  - **Decision**: Table view with expandable rows (consistent with Memory plugin, better for many items)

- [x] **D3**: Edit approach - Inline editing vs Separate edit page
  - **Decision**: Separate edit/create pages (consistent with existing patterns, simpler HTMX implementation)

- [x] **D4**: Where to save new prompts - Local (.brains/) vs Global (~/.brains/)
  - **Decision**: User chooses on create, default to local (project-specific is most common case)

- [x] **D5**: Handle embedded prompts - Show but disable edit, or hide entirely
  - **Decision**: Show with "embedded" badge, view-only (transparency about what's available)

## Recommendations

1. **Create new `prompts` plugin** following existing plugin patterns (memory, profiles, recall)

2. **Unified list view** with category filter tabs (Workflows | Profiles | Steps) at top of content area

3. **Table columns**: Name, Type, Source (badge), Description, Actions (View/Edit/Delete)

4. **Filter/Sort controls**: Category filter, source filter (local/global/embedded), text search, sort dropdown

5. **CRUD operations**:
   - List: All categories with filters
   - View: Read-only display with syntax-highlighted markdown
   - Create: Form with YAML frontmatter fields + markdown editor
   - Edit: Same as create, pre-populated (only for local/global)
   - Delete: Confirmation dialog (only for local/global)

6. **Reuse existing services** via direct Go calls rather than MCP tool invocations (simpler for internal use)

## Sources

- `/Users/morgan/Projects/personal/zombiekit/internal/gui/server.go` - Server setup
- `/Users/morgan/Projects/personal/zombiekit/internal/gui/plugin.go` - Plugin interfaces
- `/Users/morgan/Projects/personal/zombiekit/internal/gui/plugins/memory/` - CRUD patterns
- `/Users/morgan/Projects/personal/zombiekit/internal/profile/service.go` - Profile operations
- `/Users/morgan/Projects/personal/zombiekit/internal/workflow/service.go` - Workflow operations
- `/Users/morgan/Projects/personal/zombiekit/internal/step/service.go` - Step operations
- `/Users/morgan/Projects/personal/zombiekit/embed/` - Embedded assets structure
