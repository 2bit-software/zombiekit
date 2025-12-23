# Research: Sticky Memory Web Plugin

**Feature**: 009-sticky-memory-plugin
**Date**: 2025-12-22

## Research Questions

### 1. Client-Side Markdown Rendering Library

**Decision**: marked.js via CDN

**Rationale**:
- Specified in clarifications (spec.md Session 2025-12-22)
- Consistent with existing CDN pattern (Tailwind, HTMX loaded via CDN)
- Lightweight (~28KB minified), no build step required
- Well-maintained, widely used library
- Supports GitHub-flavored markdown (headers, code blocks, links, lists, emphasis per SC-005)

**Alternatives Considered**:
- **markdown-it**: More extensible but heavier (~100KB), overkill for this use case
- **showdown**: Similar to marked.js but less active maintenance
- **Server-side rendering**: Would require Go markdown library, adds complexity, not needed for single-user local tool

**CDN URL**: `https://cdn.jsdelivr.net/npm/marked/marked.min.js`

**Integration Pattern**:
```html
<script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
<script>
  function renderMarkdown(elementId) {
    const el = document.getElementById(elementId);
    if (el.dataset.mode === 'rendered') {
      el.innerHTML = marked.parse(el.dataset.raw);
    } else {
      el.textContent = el.dataset.raw;
    }
  }
</script>
```

### 2. HTMX Patterns for CRUD Operations

**Decision**: Follow existing profiles plugin pattern with additions for forms

**Rationale**:
- Consistency with established codebase patterns
- Profiles plugin already demonstrates list/view navigation
- HTMX handles form submission natively with `hx-post`

**Patterns by Operation**:

| Operation | HTMX Attributes | Target |
|-----------|----------------|--------|
| List | `hx-get="/memory"` | `#content` |
| View | `hx-get="/memory/{name}"` | `#content` |
| Create Form | `hx-get="/memory/new"` | `#content` |
| Create Submit | `hx-post="/memory"` | `#content` |
| Edit Form | `hx-get="/memory/{name}/edit"` | `#content` |
| Edit Submit | `hx-put="/memory/{name}"` | `#content` |
| Delete Confirm | `hx-get="/memory/{name}/delete"` | Modal or `#content` |
| Delete Execute | `hx-delete="/memory/{name}"` | `#content` (redirect to list) |
| Search | `hx-get="/memory?q={query}"` | `#content` |

**Form Submission Pattern**:
```html
<form hx-post="/memory" hx-target="#content" hx-push-url="true">
  <input type="text" name="name" required>
  <textarea name="content"></textarea>
  <button type="submit">Save</button>
</form>
```

### 3. Pagination Implementation

**Decision**: Server-side pagination with query parameters

**Rationale**:
- Simple implementation, no additional JavaScript state
- Works with HTMX navigation pattern
- Configurable limits per FR-012 (10, 20, 50, 100)

**Query Parameters**:
- `page`: Current page number (1-indexed, default 1)
- `limit`: Items per page (default 20, options: 10, 20, 50, 100)
- `q`: Search query (optional)

**URL Examples**:
- `/memory` - First 20 memories
- `/memory?page=2` - Second page
- `/memory?limit=50` - 50 items per page
- `/memory?q=project&page=1` - Search with pagination

**Pagination Data Structure**:
```go
type PaginationData struct {
    CurrentPage  int
    TotalPages   int
    TotalItems   int
    Limit        int
    HasPrev      bool
    HasNext      bool
    PrevPage     int
    NextPage     int
    LimitOptions []int // [10, 20, 50, 100]
}
```

### 4. Form Handling and Validation

**Decision**: Server-side validation with error display

**Rationale**:
- Consistent with Go/HTMX patterns
- No client-side JavaScript validation needed
- Errors returned in response, displayed in form

**Validation Rules** (from spec):
- Name: Required, max 255 chars, sanitized to URL-safe (edge case handling)
- Content: Optional (empty allowed), max 1MB

**Error Display Pattern**:
```html
{{if .Error}}
<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
  {{.Error}}
</div>
{{end}}
```

**Form Data Structure**:
```go
type FormData struct {
    Name    string
    Content string
    Error   string
    IsEdit  bool
}
```

### 5. Markdown View Toggle Implementation

**Decision**: Client-side toggle with data attributes

**Rationale**:
- No server round-trip needed for view mode switch
- Raw content stored in data attribute
- Toggle button switches between rendered/source views

**Implementation**:
```html
<div id="content-display"
     data-raw="{{.Memory.Content}}"
     data-mode="rendered">
  <!-- Rendered content inserted by JavaScript -->
</div>

<button onclick="toggleView()">
  <span id="toggle-label">View Source</span>
</button>

<script>
function toggleView() {
  const display = document.getElementById('content-display');
  const label = document.getElementById('toggle-label');

  if (display.dataset.mode === 'rendered') {
    display.textContent = display.dataset.raw;
    display.dataset.mode = 'source';
    display.classList.add('font-mono', 'whitespace-pre-wrap');
    label.textContent = 'View Rendered';
  } else {
    display.innerHTML = marked.parse(display.dataset.raw);
    display.dataset.mode = 'rendered';
    display.classList.remove('font-mono', 'whitespace-pre-wrap');
    label.textContent = 'View Source';
  }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
  const display = document.getElementById('content-display');
  if (display) {
    display.innerHTML = marked.parse(display.dataset.raw);
  }
});

// Re-initialize after HTMX swap
document.body.addEventListener('htmx:afterSwap', (e) => {
  const display = document.getElementById('content-display');
  if (display && display.dataset.mode === 'rendered') {
    display.innerHTML = marked.parse(display.dataset.raw);
  }
});
</script>
```

### 6. Delete Confirmation Pattern

**Decision**: Inline confirmation with HTMX swap

**Rationale**:
- Simpler than modal (no additional JavaScript)
- Consistent with HTMX-first approach
- Replaces content with confirmation, cancel returns to view

**Flow**:
1. User clicks "Delete" button
2. HTMX loads confirmation template into `#content`
3. Confirmation shows memory name, "Confirm" and "Cancel" buttons
4. Cancel: `hx-get` back to memory view
5. Confirm: `hx-delete` executes deletion, redirects to list

### 7. Empty State and Error Handling

**Decision**: Dedicated template sections with consistent styling

**Patterns**:

**Empty State (no memories)**:
```html
<div class="text-center py-12">
  <h3 class="text-lg font-medium text-gray-900">No memories yet</h3>
  <p class="mt-2 text-sm text-gray-500">Create your first memory to get started.</p>
  <a href="/memory/new" hx-get="/memory/new" hx-target="#content"
     class="mt-4 inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded">
    New Memory
  </a>
</div>
```

**Empty Search Results**:
```html
<div class="text-center py-8">
  <p class="text-gray-500">No memories match "{{.Query}}"</p>
  <a href="/memory" hx-get="/memory" hx-target="#content">Clear search</a>
</div>
```

**Operation Error**:
```html
<div class="bg-red-50 border-l-4 border-red-400 p-4">
  <p class="text-red-700">{{.Error}}</p>
</div>
```

### 8. Size Formatting

**Decision**: Server-side formatting helper

**Rationale**:
- Consistent display across list and detail views
- Simple implementation in Go

**Implementation**:
```go
func FormatSize(bytes int) string {
    if bytes < 1024 {
        return fmt.Sprintf("%d B", bytes)
    } else if bytes < 1024*1024 {
        return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
    }
    return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
```

## Resolved Clarifications

All "NEEDS CLARIFICATION" items from Technical Context are resolved:

| Item | Resolution | Source |
|------|------------|--------|
| Markdown library | marked.js via CDN | Spec clarifications |
| Form handling | Server-side validation, HTMX form submission | Research |
| Pagination | Query params, server-side, configurable limits | FR-012 + research |
| Delete confirmation | Inline HTMX swap pattern | Research |
| View toggle | Client-side with data attributes | Research |

## Integration Points

### Existing Code to Reuse

1. **`internal/memory.Storage`**: All CRUD operations
2. **`internal/web.WebPlugin`**: Plugin interface
3. **`internal/web.TemplatePlugin`**: Template embedding
4. **`internal/web.Renderer`**: Dual-mode rendering
5. **`internal/web.SidebarItem`**: Navigation entry

### Modifications Required

1. **`internal/cli/gui.go`**: Add memory plugin registration
2. **`internal/web/static/js/app.js`**: May need HTMX afterSwap handler for markdown (or inline in template)

### New Files

1. `internal/webplugins/memory/plugin.go`
2. `internal/webplugins/memory/handlers.go`
3. `internal/webplugins/memory/handlers_test.go`
4. `internal/webplugins/memory/templates/*.html` (4 templates)
