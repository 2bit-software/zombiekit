---
status: draft
updated: 2026-01-21
---

# Implementation Plan: GUI Prompts Management

## Overview

Create a new `prompts` GUI plugin that provides a unified view for managing workflows, profiles, and steps with filtering, searching, and viewing capabilities.

**MVP Scope**: P1 only (List, Filter, Search, View)

## Prerequisites

- Existing plugin patterns in `internal/webplugins/` (memory, profiles)
- Existing services: `profile.Service`, `step.Service`
- Missing: `workflow.Service.List()` method - must be added

## Implementation Steps

### Phase 1: Backend Service Enhancement

#### 1.1 Add List Method to Workflow Service

**File**: `internal/workflow/service.go`

Add a `List()` method following the same pattern as step loader:
- Scan local, global, and embedded directories
- Return all workflows with source metadata
- Higher precedence sources shadow lower ones

```go
// List returns all available workflows from all sources.
func (s *Service) List() ([]*Workflow, error)
```

**Dependencies**: None
**Estimated effort**: Small

### Phase 2: Plugin Infrastructure

#### 2.1 Create Plugin Directory Structure

Create new plugin at `internal/webplugins/prompts/`:

```
internal/webplugins/prompts/
├── plugin.go          # Plugin interface implementation
├── handlers.go        # HTTP handlers
├── types.go           # Data types for templates
├── plugin_test.go     # Tests
└── templates/
    ├── list.html      # Unified list view with filters
    └── view.html      # Detail view for single prompt
```

**Dependencies**: None
**Estimated effort**: Small (scaffolding)

#### 2.2 Implement Plugin Interface

**File**: `internal/webplugins/prompts/plugin.go`

```go
type Plugin struct {
    profileSvc  *profile.Service
    stepSvc     *step.Service
    workflowSvc *workflow.Service
}

func (p *Plugin) SidebarItems() []web.SidebarItem {
    return []web.SidebarItem{{
        ID:    "prompts",
        Label: "Prompts",
        Path:  "/",
        Order: 15,  // After profiles (10), before memory (20)
    }}
}

func (p *Plugin) MountRoutes(r chi.Router) {
    r.Get("/", h.list)
    r.Get("/{category}/{name}", h.view)
}
```

**Dependencies**: 2.1
**Estimated effort**: Small

### Phase 3: Data Layer

#### 3.1 Define Unified Prompt Type

**File**: `internal/webplugins/prompts/types.go`

Create a unified data structure that normalizes workflows, profiles, and steps:

```go
// PromptCategory represents the type of prompt
type PromptCategory string

const (
    CategoryWorkflow PromptCategory = "workflow"
    CategoryProfile  PromptCategory = "profile"
    CategoryStep     PromptCategory = "step"
)

// PromptSource represents where the prompt came from
type PromptSource string

const (
    SourceLocal    PromptSource = "local"
    SourceGlobal   PromptSource = "global"
    SourceEmbedded PromptSource = "embedded"
)

// Prompt is the unified representation for display
type Prompt struct {
    Name        string
    Category    PromptCategory
    Source      PromptSource
    Description string
    Path        string
    Shadowed    bool   // True if a higher-precedence version exists

    // Profile-specific fields (nil for workflows/steps)
    ProfileType string   // domain, action, step, skill
    Includes    []string
    Inherits    bool
    Model       string
    Color       string

    // Step-specific fields (nil for workflows/profiles)
    Profiles []string // Profile names to compose
    Files    []string // File patterns

    // Full content for view
    Content string
}
```

**Dependencies**: None
**Estimated effort**: Small

#### 3.2 Implement Prompt Aggregation

**File**: `internal/webplugins/prompts/handlers.go`

Create helper function to aggregate prompts from all services:

```go
func (h *handlers) aggregatePrompts() ([]Prompt, error) {
    var prompts []Prompt

    // Load workflows
    workflows, err := h.workflowSvc.List()
    if err == nil {
        for _, w := range workflows {
            prompts = append(prompts, convertWorkflow(w))
        }
    }

    // Load profiles
    profiles, err := h.profileSvc.List()
    if err == nil {
        for _, p := range profiles {
            prompts = append(prompts, convertProfile(p))
        }
    }

    // Load steps
    steps, err := h.stepSvc.ListSteps()
    if err == nil {
        for _, s := range steps {
            prompts = append(prompts, convertStep(s))
        }
    }

    return prompts, nil
}
```

**Dependencies**: 1.1, 3.1
**Estimated effort**: Medium

### Phase 4: HTTP Handlers

#### 4.1 Implement List Handler

**File**: `internal/webplugins/prompts/handlers.go`

Handle query parameters:
- `category` - filter by workflow/profile/step
- `source` - filter by local/global/embedded
- `q` - search query (name and description)
- `sort` - sort field (name, category, source)
- `order` - sort order (asc, desc)

```go
func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
    renderer := web.GetRenderer(r)

    // Parse filters from query params
    categoryFilter := r.URL.Query().Get("category")
    sourceFilter := r.URL.Query().Get("source")
    query := r.URL.Query().Get("q")
    sortField := r.URL.Query().Get("sort")
    sortOrder := r.URL.Query().Get("order")

    // Aggregate and filter prompts
    prompts, err := h.aggregatePrompts()
    if err != nil { ... }

    // Apply filters
    prompts = h.filterPrompts(prompts, categoryFilter, sourceFilter, query)

    // Apply sorting
    prompts = h.sortPrompts(prompts, sortField, sortOrder)

    // Render
    data := ListData{
        Prompts:        prompts,
        CategoryFilter: categoryFilter,
        SourceFilter:   sourceFilter,
        Query:          query,
        SortField:      sortField,
        SortOrder:      sortOrder,
    }

    renderer.Render(w, r, "prompts/list.html", data)
}
```

**Dependencies**: 3.2
**Estimated effort**: Medium

#### 4.2 Implement View Handler

**File**: `internal/webplugins/prompts/handlers.go`

```go
func (h *handlers) view(w http.ResponseWriter, r *http.Request) {
    renderer := web.GetRenderer(r)

    category := chi.URLParam(r, "category")
    name := chi.URLParam(r, "name")

    var prompt *Prompt
    var err error

    switch category {
    case "workflow":
        wf, err := h.workflowSvc.Load(name)
        if err == nil { prompt = convertWorkflowFull(wf) }
    case "profile":
        p, err := h.profileSvc.Show(name, false)
        if err == nil { prompt = convertProfileFull(p) }
    case "step":
        s, err := h.stepSvc.GetStep(name)
        if err == nil { prompt = convertStepFull(s) }
    }

    data := ViewData{Prompt: prompt, Error: err}
    renderer.Render(w, r, "prompts/view.html", data)
}
```

**Dependencies**: 3.1
**Estimated effort**: Small

### Phase 5: Templates

#### 5.1 Create List Template

**File**: `internal/webplugins/prompts/templates/list.html`

Key components:
- Header with "New Prompt" button (disabled for MVP - P2 feature)
- Category filter tabs: All | Workflows | Profiles | Steps
- Source filter dropdown: All | Local | Global | Embedded
- Search input
- Sort controls
- Table with columns: Name, Category, Source (badge), Description
- Row click navigates to detail view

HTMX patterns:
- Filter changes use `hx-get` with query params
- `hx-push-url="true"` for browser history
- `hx-target="#content"` for SPA-like navigation

**Dependencies**: 4.1
**Estimated effort**: Medium

#### 5.2 Create View Template

**File**: `internal/webplugins/prompts/templates/view.html`

Key components:
- Back link to list
- Header with prompt name
- Metadata section (source badge, category, path)
- For profiles: includes list, type, model, color
- For steps: profiles list, files list
- Content section with markdown rendering (same pattern as memory)
- Toggle between rendered/source view

**Dependencies**: 4.2
**Estimated effort**: Medium

### Phase 6: Integration

#### 6.1 Register Plugin with Server

**File**: `internal/cli/gui.go`

Add plugin registration:

```go
// Create prompts plugin
promptsPlugin := prompts.NewPlugin(profileSvc, stepSvc, workflowSvc)
registry.Register("prompts", promptsPlugin)
```

**Dependencies**: 2.2, 4.1, 4.2, 5.1, 5.2
**Estimated effort**: Small

### Phase 7: Testing

#### 7.1 Integration Tests

**File**: `internal/webplugins/prompts/plugin_test.go`

Test coverage:
- `GET /prompts` returns all prompts
- `GET /prompts?category=workflow` filters correctly
- `GET /prompts?source=local` filters correctly
- `GET /prompts?q=feat` searches correctly
- `GET /prompts/profile/feature` returns profile detail
- `GET /prompts/workflow/new` returns workflow detail
- `GET /prompts/step/plan` returns step detail

**Dependencies**: 6.1
**Estimated effort**: Medium

## Dependency Graph

```
1.1 (workflow.List)
    ↓
3.2 (aggregatePrompts) ← 3.1 (types)
    ↓
4.1 (list handler)
    ↓
5.1 (list template)
    ↓
6.1 (register plugin)
    ↓
7.1 (tests)

Parallel: 2.1, 2.2, 3.1, 4.2, 5.2 can start immediately
```

## Technical Decisions

1. **Unified Prompt Type**: Create a normalized `Prompt` struct that works for all three types, with optional fields for type-specific data. This simplifies templates and filtering.

2. **Direct Service Calls**: Use Go service calls directly rather than MCP tools. The GUI is internal and doesn't need the MCP abstraction layer.

3. **Server-Side Filtering**: All filtering, searching, and sorting happens server-side. Given the expected data volume (<100 prompts typically), this is simpler than client-side JS filtering.

4. **Consistent Patterns**: Follow existing memory plugin patterns exactly - same template structure, HTMX patterns, and Tailwind styling.

5. **No Pagination for MVP**: With typical prompt counts under 100, pagination adds complexity without benefit. Can add later if needed.

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Missing workflow.List() | Simple addition, follows existing step loader pattern |
| Service initialization order | Register plugin last to ensure services are ready |
| Template rendering performance | Pre-aggregate prompts, minimal template logic |

## Out of Scope (P2/P3)

- Create/Edit/Delete operations
- Copy/Fork embedded prompts
- Date sorting (file mtime)
- Markdown preview in list
- Bulk operations
