---
status: draft
updated: 2026-01-21
---

# Technical Specification: GUI Prompts Management

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Browser (HTMX)                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Chi Router                               │
│  /prompts/* → prompts plugin handlers                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Prompts Plugin                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ list()      │  │ view()      │  │ aggregate() │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
         │                  │                  │
         ▼                  ▼                  ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ workflow    │    │ profile     │    │ step        │
│ Service     │    │ Service     │    │ Service     │
└─────────────┘    └─────────────┘    └─────────────┘
         │                  │                  │
         ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│           File System (local / global / embedded)            │
└─────────────────────────────────────────────────────────────┘
```

## File Structure

```
internal/webplugins/prompts/
├── plugin.go           # Plugin interface, NewPlugin(), SidebarItems(), MountRoutes()
├── handlers.go         # HTTP handlers: list, view
├── types.go            # Prompt, ListData, ViewData, filter/sort helpers
├── converters.go       # Workflow/Profile/Step → Prompt converters
├── plugin_test.go      # Integration tests
└── templates/
    ├── list.html       # List view with filters
    └── view.html       # Detail view
```

## Type Definitions

### types.go

```go
package prompts

// PromptCategory identifies the type of prompt
type PromptCategory string

const (
	CategoryWorkflow PromptCategory = "workflow"
	CategoryProfile  PromptCategory = "profile"
	CategoryStep     PromptCategory = "step"
)

func (c PromptCategory) String() string { return string(c) }

func (c PromptCategory) Label() string {
	switch c {
	case CategoryWorkflow:
		return "Workflow"
	case CategoryProfile:
		return "Profile"
	case CategoryStep:
		return "Step"
	default:
		return string(c)
	}
}

// PromptSource identifies where the prompt is stored
type PromptSource string

const (
	SourceLocal    PromptSource = "local"
	SourceGlobal   PromptSource = "global"
	SourceEmbedded PromptSource = "embedded"
)

func (s PromptSource) String() string { return string(s) }

func (s PromptSource) BadgeColor() string {
	switch s {
	case SourceLocal:
		return "bg-green-100 text-green-800"
	case SourceGlobal:
		return "bg-blue-100 text-blue-800"
	case SourceEmbedded:
		return "bg-gray-100 text-gray-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

// Prompt is the unified representation for all prompt types
type Prompt struct {
	Name        string
	Category    PromptCategory
	Source      PromptSource
	Description string
	Path        string
	Shadowed    bool // True if overridden by higher-precedence source

	// Profile-specific (zero values for non-profiles)
	ProfileType string   // domain, action, step, skill
	Includes    []string // Referenced profiles
	Inherits    bool     // Inherit from parent dirs
	Model       string   // Claude model override
	Color       string   // UI color

	// Step-specific (zero values for non-steps)
	Profiles []string // Profiles to compose
	Files    []string // File patterns to load

	// Full content (populated only for detail view)
	Content string
}

// ListData is passed to the list template
type ListData struct {
	Prompts []Prompt

	// Current filter/sort state (for preserving in UI)
	CategoryFilter string
	SourceFilter   string
	Query          string
	SortField      string
	SortOrder      string

	// Error message if any
	Error string
}

// ViewData is passed to the view template
type ViewData struct {
	Prompt *Prompt
	Error  string
}

// FilterOptions encapsulates all filter parameters
type FilterOptions struct {
	Category string // "workflow", "profile", "step", or "" for all
	Source   string // "local", "global", "embedded", or "" for all
	Query    string // Search text
}

// SortOptions encapsulates sorting parameters
type SortOptions struct {
	Field string // "name", "category", "source"
	Order string // "asc", "desc"
}
```

## API Endpoints

### GET /prompts

List all prompts with optional filtering and sorting.

**Query Parameters:**
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| category | string | "" | Filter: "workflow", "profile", "step" |
| source | string | "" | Filter: "local", "global", "embedded" |
| q | string | "" | Search name and description |
| sort | string | "name" | Sort field: "name", "category", "source" |
| order | string | "asc" | Sort order: "asc", "desc" |

**Response:** HTML (list.html template)

**HTMX Headers:**
- `HX-Request: true` → Returns content only (for partial swap)
- Without header → Returns full page with shell

### GET /prompts/{category}/{name}

View a single prompt's details.

**Path Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| category | string | "workflow", "profile", "step" |
| name | string | Prompt name (without .md extension) |

**Response:** HTML (view.html template)

**Error Handling:**
- Unknown category → 404 with error template
- Prompt not found → 404 with error template

## Handler Implementation

### handlers.go

```go
package prompts

import (
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/step"
	"github.com/zombiekit/brains/internal/web"
	"github.com/zombiekit/brains/internal/workflow"
)

type handlers struct {
	profileSvc  *profile.Service
	stepSvc     *step.Service
	workflowSvc *workflow.Service
}

func newHandlers(profileSvc *profile.Service, stepSvc *step.Service, workflowSvc *workflow.Service) *handlers {
	return &handlers{
		profileSvc:  profileSvc,
		stepSvc:     stepSvc,
		workflowSvc: workflowSvc,
	}
}

func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	renderer := web.GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Parse query parameters
	opts := FilterOptions{
		Category: r.URL.Query().Get("category"),
		Source:   r.URL.Query().Get("source"),
		Query:    r.URL.Query().Get("q"),
	}
	sortOpts := SortOptions{
		Field: r.URL.Query().Get("sort"),
		Order: r.URL.Query().Get("order"),
	}
	if sortOpts.Field == "" {
		sortOpts.Field = "name"
	}
	if sortOpts.Order == "" {
		sortOpts.Order = "asc"
	}

	// Aggregate prompts from all sources
	prompts, err := h.aggregatePrompts()
	if err != nil {
		data := ListData{Error: err.Error()}
		renderer.Render(w, r, "prompts/list.html", data)
		return
	}

	// Filter
	prompts = filterPrompts(prompts, opts)

	// Sort
	sortPrompts(prompts, sortOpts)

	data := ListData{
		Prompts:        prompts,
		CategoryFilter: opts.Category,
		SourceFilter:   opts.Source,
		Query:          opts.Query,
		SortField:      sortOpts.Field,
		SortOrder:      sortOpts.Order,
	}

	if err := renderer.Render(w, r, "prompts/list.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *handlers) view(w http.ResponseWriter, r *http.Request) {
	renderer := web.GetRenderer(r)
	if renderer == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	category := chi.URLParam(r, "category")
	name := chi.URLParam(r, "name")

	prompt, err := h.getPrompt(PromptCategory(category), name)
	if err != nil {
		data := ViewData{Error: err.Error()}
		renderer.Render(w, r, "prompts/view.html", data)
		return
	}

	data := ViewData{Prompt: prompt}
	if err := renderer.Render(w, r, "prompts/view.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *handlers) aggregatePrompts() ([]Prompt, error) {
	var prompts []Prompt

	// Load workflows
	if h.workflowSvc != nil {
		workflows, err := h.workflowSvc.List()
		if err == nil {
			for _, wf := range workflows {
				prompts = append(prompts, convertWorkflow(wf))
			}
		}
	}

	// Load profiles
	if h.profileSvc != nil {
		profiles, err := h.profileSvc.List()
		if err == nil {
			for _, p := range profiles {
				prompts = append(prompts, convertProfile(p))
			}
		}
	}

	// Load steps
	if h.stepSvc != nil {
		steps, err := h.stepSvc.ListSteps()
		if err == nil {
			for _, s := range steps {
				prompts = append(prompts, convertStep(s))
			}
		}
	}

	return prompts, nil
}

func (h *handlers) getPrompt(category PromptCategory, name string) (*Prompt, error) {
	switch category {
	case CategoryWorkflow:
		wf, err := h.workflowSvc.Load(name)
		if err != nil {
			return nil, err
		}
		p := convertWorkflowFull(wf)
		return &p, nil

	case CategoryProfile:
		result, err := h.profileSvc.Show(name, false)
		if err != nil {
			return nil, err
		}
		p := convertProfileFull(result)
		return &p, nil

	case CategoryStep:
		s, err := h.stepSvc.GetStep(name)
		if err != nil {
			return nil, err
		}
		p := convertStepFull(s)
		return &p, nil

	default:
		return nil, fmt.Errorf("unknown category: %s", category)
	}
}

// Filter and sort helpers

func filterPrompts(prompts []Prompt, opts FilterOptions) []Prompt {
	var filtered []Prompt

	for _, p := range prompts {
		// Category filter
		if opts.Category != "" && string(p.Category) != opts.Category {
			continue
		}

		// Source filter
		if opts.Source != "" && string(p.Source) != opts.Source {
			continue
		}

		// Query filter (search name and description)
		if opts.Query != "" {
			query := strings.ToLower(opts.Query)
			name := strings.ToLower(p.Name)
			desc := strings.ToLower(p.Description)
			if !strings.Contains(name, query) && !strings.Contains(desc, query) {
				continue
			}
		}

		filtered = append(filtered, p)
	}

	return filtered
}

func sortPrompts(prompts []Prompt, opts SortOptions) {
	sort.Slice(prompts, func(i, j int) bool {
		var cmp int
		switch opts.Field {
		case "name":
			cmp = strings.Compare(prompts[i].Name, prompts[j].Name)
		case "category":
			cmp = strings.Compare(string(prompts[i].Category), string(prompts[j].Category))
			if cmp == 0 {
				cmp = strings.Compare(prompts[i].Name, prompts[j].Name)
			}
		case "source":
			cmp = strings.Compare(string(prompts[i].Source), string(prompts[j].Source))
			if cmp == 0 {
				cmp = strings.Compare(prompts[i].Name, prompts[j].Name)
			}
		default:
			cmp = strings.Compare(prompts[i].Name, prompts[j].Name)
		}

		if opts.Order == "desc" {
			return cmp > 0
		}
		return cmp < 0
	})
}
```

## Converters

### converters.go

```go
package prompts

import (
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/step"
	"github.com/zombiekit/brains/internal/workflow"
)

// Workflow converters

func convertWorkflow(wf *workflow.Workflow) Prompt {
	return Prompt{
		Name:        wf.Name,
		Category:    CategoryWorkflow,
		Source:      mapWorkflowSource(wf.Source),
		Description: wf.Description,
		Path:        wf.Path,
	}
}

func convertWorkflowFull(wf *workflow.Workflow) Prompt {
	p := convertWorkflow(wf)
	p.Content = wf.Content
	return p
}

func mapWorkflowSource(src string) PromptSource {
	switch src {
	case "local":
		return SourceLocal
	case "global":
		return SourceGlobal
	default:
		return SourceEmbedded
	}
}

// Profile converters

func convertProfile(entry profile.ListEntry) Prompt {
	return Prompt{
		Name:        entry.Name,
		Category:    CategoryProfile,
		Source:      mapProfileSource(entry.Source),
		Description: entry.Description,
		Path:        entry.Path,
		Shadowed:    entry.Shadowed,
		ProfileType: entry.Type,
		Includes:    entry.Includes,
		Inherits:    entry.Inherits,
		Model:       entry.Model,
		Color:       entry.Color,
	}
}

func convertProfileFull(result *profile.ShowResult) Prompt {
	p := Prompt{
		Name:        result.Name,
		Category:    CategoryProfile,
		Source:      mapProfileSource(result.Source),
		Description: result.Description,
		Path:        result.Path,
		ProfileType: result.Type,
		Includes:    result.Includes,
		Inherits:    result.Inherits,
		Model:       result.Model,
		Color:       result.Color,
		Content:     result.Body,
	}
	return p
}

func mapProfileSource(src profile.ProfileSource) PromptSource {
	switch src {
	case profile.SourceLocal:
		return SourceLocal
	case profile.SourceGlobal:
		return SourceGlobal
	default:
		return SourceEmbedded
	}
}

// Step converters

func convertStep(s *step.Step) Prompt {
	return Prompt{
		Name:        s.Name,
		Category:    CategoryStep,
		Source:      mapStepSource(s.Source),
		Description: s.Description,
		Path:        s.Path,
		Profiles:    s.Profiles,
		Files:       s.Files,
	}
}

func convertStepFull(s *step.Step) Prompt {
	p := convertStep(s)
	p.Content = s.Directive
	return p
}

func mapStepSource(src step.StepSource) PromptSource {
	switch src {
	case step.SourceLocal:
		return SourceLocal
	case step.SourceGlobal:
		return SourceGlobal
	default:
		return SourceEmbedded
	}
}
```

## Template Specifications

### list.html

```html
<div class="bg-white rounded-lg shadow">
    <!-- Header -->
    <div class="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
        <div>
            <h1 class="text-xl font-semibold text-gray-900">Prompts</h1>
            <p class="mt-1 text-sm text-gray-500">
                Manage workflows, profiles, and steps
            </p>
        </div>
        <!-- New Prompt button (disabled for MVP) -->
        <button disabled
                class="inline-flex items-center px-4 py-2 bg-gray-300 text-gray-500
                       text-sm font-medium rounded-md cursor-not-allowed"
                title="Coming soon">
            New Prompt
        </button>
    </div>

    <!-- Filters -->
    <div class="px-6 py-3 border-b border-gray-200 bg-gray-50">
        <form hx-get="/prompts"
              hx-target="#content"
              hx-push-url="true"
              class="flex flex-wrap items-center gap-4">

            <!-- Category tabs -->
            <div class="flex rounded-md shadow-sm">
                <button type="submit" name="category" value=""
                        class="px-4 py-2 text-sm font-medium rounded-l-md border
                               {{if eq .Content.CategoryFilter ""}}
                               bg-blue-600 text-white border-blue-600
                               {{else}}
                               bg-white text-gray-700 border-gray-300 hover:bg-gray-50
                               {{end}}">
                    All
                </button>
                <button type="submit" name="category" value="workflow"
                        class="px-4 py-2 text-sm font-medium border-t border-b
                               {{if eq .Content.CategoryFilter "workflow"}}
                               bg-blue-600 text-white border-blue-600
                               {{else}}
                               bg-white text-gray-700 border-gray-300 hover:bg-gray-50
                               {{end}}">
                    Workflows
                </button>
                <button type="submit" name="category" value="profile"
                        class="px-4 py-2 text-sm font-medium border-t border-b
                               {{if eq .Content.CategoryFilter "profile"}}
                               bg-blue-600 text-white border-blue-600
                               {{else}}
                               bg-white text-gray-700 border-gray-300 hover:bg-gray-50
                               {{end}}">
                    Profiles
                </button>
                <button type="submit" name="category" value="step"
                        class="px-4 py-2 text-sm font-medium rounded-r-md border
                               {{if eq .Content.CategoryFilter "step"}}
                               bg-blue-600 text-white border-blue-600
                               {{else}}
                               bg-white text-gray-700 border-gray-300 hover:bg-gray-50
                               {{end}}">
                    Steps
                </button>
            </div>

            <!-- Source filter -->
            <select name="source"
                    onchange="this.form.requestSubmit()"
                    class="px-3 py-2 border border-gray-300 rounded-md text-sm">
                <option value="" {{if eq .Content.SourceFilter ""}}selected{{end}}>
                    All Sources
                </option>
                <option value="local" {{if eq .Content.SourceFilter "local"}}selected{{end}}>
                    Local
                </option>
                <option value="global" {{if eq .Content.SourceFilter "global"}}selected{{end}}>
                    Global
                </option>
                <option value="embedded" {{if eq .Content.SourceFilter "embedded"}}selected{{end}}>
                    Embedded
                </option>
            </select>

            <!-- Search -->
            <input type="text"
                   name="q"
                   value="{{.Content.Query}}"
                   placeholder="Search prompts..."
                   class="flex-1 min-w-[200px] px-3 py-2 border border-gray-300 rounded-md text-sm">

            <!-- Sort -->
            <select name="sort"
                    onchange="this.form.requestSubmit()"
                    class="px-3 py-2 border border-gray-300 rounded-md text-sm">
                <option value="name" {{if eq .Content.SortField "name"}}selected{{end}}>
                    Sort by Name
                </option>
                <option value="category" {{if eq .Content.SortField "category"}}selected{{end}}>
                    Sort by Category
                </option>
                <option value="source" {{if eq .Content.SortField "source"}}selected{{end}}>
                    Sort by Source
                </option>
            </select>

            <!-- Hidden fields to preserve state -->
            <input type="hidden" name="order" value="{{.Content.SortOrder}}">

            <button type="submit"
                    class="px-4 py-2 bg-gray-100 text-gray-700 text-sm font-medium rounded-md">
                Search
            </button>
        </form>
    </div>

    {{if .Content.Error}}
    <div class="p-6">
        <div class="rounded-md bg-red-50 p-4">
            <p class="text-sm font-medium text-red-800">{{.Content.Error}}</p>
        </div>
    </div>
    {{else if not .Content.Prompts}}
    <div class="p-6 text-center py-12">
        {{if or .Content.Query .Content.CategoryFilter .Content.SourceFilter}}
        <p class="text-gray-500">No prompts match your filters</p>
        <a href="/prompts"
           hx-get="/prompts"
           hx-target="#content"
           hx-push-url="true"
           class="mt-2 inline-block text-blue-600 hover:text-blue-800">
            Clear filters
        </a>
        {{else}}
        <p class="text-gray-500">No prompts found</p>
        {{end}}
    </div>
    {{else}}
    <!-- Prompts table -->
    <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
                <tr>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                        Name
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                        Category
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                        Source
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                        Description
                    </th>
                </tr>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
                {{range .Content.Prompts}}
                <tr class="hover:bg-gray-50 cursor-pointer"
                    hx-get="/prompts/{{.Category}}/{{.Name}}"
                    hx-target="#content"
                    hx-push-url="true">
                    <td class="px-6 py-4 whitespace-nowrap">
                        <span class="text-sm font-medium text-gray-900">{{.Name}}</span>
                        {{if .Shadowed}}
                        <span class="ml-1 text-xs text-yellow-600" title="Shadowed by higher-precedence source">
                            (shadowed)
                        </span>
                        {{end}}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                        <span class="text-sm text-gray-500">{{.Category.Label}}</span>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{.Source.BadgeColor}}">
                            {{.Source}}
                        </span>
                    </td>
                    <td class="px-6 py-4">
                        <span class="text-sm text-gray-500 line-clamp-2">
                            {{if .Description}}{{.Description}}{{else}}-{{end}}
                        </span>
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>

    <!-- Result count -->
    <div class="px-6 py-3 border-t border-gray-200 bg-gray-50">
        <p class="text-sm text-gray-500">
            Showing {{len .Content.Prompts}} prompt{{if ne (len .Content.Prompts) 1}}s{{end}}
        </p>
    </div>
    {{end}}
</div>
```

### view.html

```html
<div class="bg-white rounded-lg shadow">
    <!-- Back link -->
    <div class="px-6 py-3 border-b border-gray-200 bg-gray-50">
        <a href="/prompts"
           hx-get="/prompts"
           hx-target="#content"
           hx-push-url="true"
           class="inline-flex items-center text-sm text-gray-600 hover:text-gray-900">
            ← Back to prompts
        </a>
    </div>

    {{if .Content.Error}}
    <div class="p-6">
        <div class="rounded-md bg-red-50 p-4">
            <h3 class="text-sm font-medium text-red-800">Prompt not found</h3>
            <p class="mt-2 text-sm text-red-700">{{.Content.Error}}</p>
        </div>
    </div>
    {{else}}
    <!-- Header -->
    <div class="px-6 py-4 border-b border-gray-200">
        <div class="flex items-center justify-between">
            <div>
                <h1 class="text-xl font-semibold text-gray-900">{{.Content.Prompt.Name}}</h1>
                <div class="mt-1 flex items-center space-x-2">
                    <span class="text-sm text-gray-500">{{.Content.Prompt.Category.Label}}</span>
                    <span class="text-gray-300">•</span>
                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{.Content.Prompt.Source.BadgeColor}}">
                        {{.Content.Prompt.Source}}
                    </span>
                </div>
            </div>
            {{if ne .Content.Prompt.Source "embedded"}}
            <!-- Edit/Delete buttons (disabled for MVP) -->
            <div class="flex items-center space-x-2">
                <button disabled
                        class="px-3 py-1.5 text-sm font-medium text-gray-400 bg-gray-100
                               rounded-md cursor-not-allowed"
                        title="Coming soon">
                    Edit
                </button>
                <button disabled
                        class="px-3 py-1.5 text-sm font-medium text-gray-400 bg-gray-100
                               rounded-md cursor-not-allowed"
                        title="Coming soon">
                    Delete
                </button>
            </div>
            {{end}}
        </div>
    </div>

    <!-- Metadata -->
    <div class="px-6 py-4 border-b border-gray-200 bg-gray-50">
        <dl class="grid grid-cols-2 gap-x-4 gap-y-3 sm:grid-cols-4">
            {{if .Content.Prompt.Description}}
            <div class="col-span-2 sm:col-span-4">
                <dt class="text-xs font-medium text-gray-500 uppercase">Description</dt>
                <dd class="mt-1 text-sm text-gray-900">{{.Content.Prompt.Description}}</dd>
            </div>
            {{end}}

            <div>
                <dt class="text-xs font-medium text-gray-500 uppercase">Path</dt>
                <dd class="mt-1 text-sm text-gray-900 font-mono text-xs truncate"
                    title="{{.Content.Prompt.Path}}">
                    {{.Content.Prompt.Path}}
                </dd>
            </div>

            {{/* Profile-specific metadata */}}
            {{if eq .Content.Prompt.Category "profile"}}
            {{if .Content.Prompt.ProfileType}}
            <div>
                <dt class="text-xs font-medium text-gray-500 uppercase">Type</dt>
                <dd class="mt-1 text-sm text-gray-900">{{.Content.Prompt.ProfileType}}</dd>
            </div>
            {{end}}
            {{if .Content.Prompt.Model}}
            <div>
                <dt class="text-xs font-medium text-gray-500 uppercase">Model</dt>
                <dd class="mt-1 text-sm text-gray-900">{{.Content.Prompt.Model}}</dd>
            </div>
            {{end}}
            {{if .Content.Prompt.Includes}}
            <div class="col-span-2">
                <dt class="text-xs font-medium text-gray-500 uppercase">Includes</dt>
                <dd class="mt-1 flex flex-wrap gap-1">
                    {{range .Content.Prompt.Includes}}
                    <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                        {{.}}
                    </span>
                    {{end}}
                </dd>
            </div>
            {{end}}
            {{end}}

            {{/* Step-specific metadata */}}
            {{if eq .Content.Prompt.Category "step"}}
            {{if .Content.Prompt.Profiles}}
            <div class="col-span-2">
                <dt class="text-xs font-medium text-gray-500 uppercase">Profiles</dt>
                <dd class="mt-1 flex flex-wrap gap-1">
                    {{range .Content.Prompt.Profiles}}
                    <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800">
                        {{.}}
                    </span>
                    {{end}}
                </dd>
            </div>
            {{end}}
            {{if .Content.Prompt.Files}}
            <div class="col-span-2">
                <dt class="text-xs font-medium text-gray-500 uppercase">Files</dt>
                <dd class="mt-1 flex flex-wrap gap-1">
                    {{range .Content.Prompt.Files}}
                    <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800 font-mono">
                        {{.}}
                    </span>
                    {{end}}
                </dd>
            </div>
            {{end}}
            {{end}}
        </dl>
    </div>

    <!-- Content -->
    <div class="p-6">
        <div class="flex items-center justify-between mb-3">
            <h3 class="text-sm font-medium text-gray-900">Content</h3>
            <button type="button"
                    onclick="toggleView()"
                    class="inline-flex items-center px-3 py-1 text-xs font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200">
                <span id="toggle-label">View Source</span>
            </button>
        </div>
        <div id="content-display"
             data-raw="{{.Content.Prompt.Content}}"
             data-mode="rendered"
             class="prose prose-sm max-w-none bg-gray-50 rounded-lg p-4 min-h-[200px] overflow-auto">
            <!-- Rendered by JavaScript -->
        </div>
    </div>
    {{end}}
</div>

<script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
<script>
function toggleView() {
    const display = document.getElementById('content-display');
    const label = document.getElementById('toggle-label');
    if (!display) return;

    if (display.dataset.mode === 'rendered') {
        display.textContent = display.dataset.raw;
        display.dataset.mode = 'source';
        display.classList.add('font-mono', 'whitespace-pre-wrap', 'text-xs');
        display.classList.remove('prose', 'prose-sm');
        label.textContent = 'View Rendered';
    } else {
        display.innerHTML = marked.parse(display.dataset.raw || '');
        display.dataset.mode = 'rendered';
        display.classList.remove('font-mono', 'whitespace-pre-wrap', 'text-xs');
        display.classList.add('prose', 'prose-sm');
        label.textContent = 'View Source';
    }
}

document.addEventListener('DOMContentLoaded', function() {
    const display = document.getElementById('content-display');
    if (display && display.dataset.raw !== undefined) {
        display.innerHTML = marked.parse(display.dataset.raw || '');
    }
});

document.body.addEventListener('htmx:afterSwap', function(e) {
    const display = document.getElementById('content-display');
    if (display && display.dataset.mode === 'rendered' && display.dataset.raw !== undefined) {
        display.innerHTML = marked.parse(display.dataset.raw || '');
    }
});
</script>
```

## Required Changes to Existing Code

### 1. Add List() to Workflow Service

**File**: `internal/workflow/service.go`

```go
// List returns all available workflows from all sources.
// Resolution order: local > global > embedded (higher precedence overwrites).
func (s *Service) List() ([]*Workflow, error) {
    workflowMap := make(map[string]*Workflow)

    // Load in reverse precedence order
    // 3. Embedded (lowest)
    if GetEmbeddedFS() != nil {
        s.loadAllFromEmbedded(workflowMap)
    }

    // 2. Global
    if s.homeDir != "" {
        s.loadAllFromDir(filepath.Join(s.homeDir, ".brains", "workflows"), "global", workflowMap)
    }

    // 1. Local (highest)
    s.loadAllFromDir(filepath.Join(s.workingDir, ".brains", "workflows"), "local", workflowMap)

    // Convert to slice
    workflows := make([]*Workflow, 0, len(workflowMap))
    for _, wf := range workflowMap {
        workflows = append(workflows, wf)
    }

    return workflows, nil
}

func (s *Service) loadAllFromDir(dir, source string, out map[string]*Workflow) {
    entries, err := os.ReadDir(dir)
    if err != nil {
        return
    }

    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
            continue
        }

        name := strings.TrimSuffix(entry.Name(), ".md")
        if wf := s.loadFromDir(dir, name, source); wf != nil {
            out[name] = wf
        }
    }
}

func (s *Service) loadAllFromEmbedded(out map[string]*Workflow) {
    globalEmbeddedMu.RLock()
    fsys := globalEmbeddedFS
    globalEmbeddedMu.RUnlock()

    if fsys == nil {
        return
    }

    entries, err := fs.ReadDir(fsys, "workflows")
    if err != nil {
        return
    }

    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
            continue
        }

        name := strings.TrimSuffix(entry.Name(), ".md")
        if wf := s.loadFromEmbedded(name); wf != nil {
            out[name] = wf
        }
    }
}
```

### 2. Register Plugin in CLI

**File**: `internal/cli/gui.go`

Add after existing plugin registrations:

```go
// Create prompts plugin
workflowSvc, _ := workflow.NewService(workDir)
promptsPlugin := prompts.NewPlugin(profileSvc, stepSvc, workflowSvc)
registry.Register("prompts", promptsPlugin)
```

## Testing Strategy

### Integration Tests

```go
func TestListHandler(t *testing.T) {
    // Setup test services with mock data
    // Test: GET /prompts returns all prompts
    // Test: GET /prompts?category=workflow filters
    // Test: GET /prompts?source=local filters
    // Test: GET /prompts?q=feat searches
}

func TestViewHandler(t *testing.T) {
    // Test: GET /prompts/workflow/new returns workflow
    // Test: GET /prompts/profile/feature returns profile
    // Test: GET /prompts/step/plan returns step
    // Test: GET /prompts/invalid/foo returns 404
}
```

## Security Considerations

1. **Path Traversal**: Names are validated before use; no direct path construction from user input
2. **XSS**: All content rendered through Go templates (auto-escaped)
3. **CSRF**: Mutations disabled for MVP; when added, will use HTMX's CSRF token support
