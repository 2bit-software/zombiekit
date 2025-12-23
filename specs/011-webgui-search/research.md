# Research: Web GUI Search Bar

**Date**: 2025-12-22
**Feature**: 011-webgui-search

## Research Tasks

### 1. HTMX Search Pattern Best Practices

**Task**: Find best practices for implementing search with HTMX

**Decision**: Use `hx-trigger="keyup changed delay:300ms"` for debounced search

**Rationale**:
- HTMX natively supports debouncing with `delay:Nms` modifier
- The `changed` modifier prevents requests if the input value hasn't changed
- Eliminates need for custom JavaScript debounce implementation
- Integrates cleanly with existing HTMX patterns in codebase

**Alternatives Considered**:
- Custom JavaScript debounce function: More code, harder to maintain
- Vanilla JS with fetch(): Would duplicate HTMX functionality already in project
- Server-side throttling: Doesn't reduce network traffic

**Implementation Pattern**:
```html
<input type="text" name="q"
       hx-get="/search"
       hx-trigger="keyup changed delay:300ms"
       hx-target="#search-results">
```

### 2. Search Result Dropdown Positioning

**Task**: Find best practices for dropdown positioning with Tailwind CSS

**Decision**: Use absolute positioning relative to search container with Tailwind utilities

**Rationale**:
- `relative` on container + `absolute` on dropdown is standard pattern
- `w-full` ensures dropdown matches input width
- `z-50` ensures dropdown appears above other content
- `shadow-lg` provides visual separation

**Alternatives Considered**:
- Fixed positioning: Breaks on scroll
- CSS Grid/Flexbox complex layouts: Overkill for this use case
- Third-party dropdown library: Unnecessary dependency

**Implementation Pattern**:
```html
<div class="relative">
  <input type="text" ... />
  <div id="search-results" class="absolute w-full z-50 bg-white shadow-lg rounded-md mt-1">
    <!-- Results rendered here -->
  </div>
</div>
```

### 3. Keyboard Navigation Implementation

**Task**: Find best practices for keyboard navigation in search dropdowns

**Decision**: Vanilla JavaScript with data attributes for result tracking

**Rationale**:
- Minimal JavaScript aligned with HTMX philosophy
- Use data-selected attribute to track highlighted result
- Arrow keys update selection, Enter navigates using HTMX
- Works with screen readers via ARIA attributes

**Alternatives Considered**:
- Full JS framework (Alpine.js): Additional dependency
- Pure CSS :focus states: Insufficient for custom keyboard behavior
- HTMX extensions: None suitable for keyboard navigation

**Implementation Pattern**:
```javascript
document.addEventListener('keydown', function(e) {
  const results = document.querySelectorAll('[data-search-result]');
  if (!results.length) return;

  if (e.key === 'ArrowDown') { /* select next */ }
  if (e.key === 'ArrowUp') { /* select previous */ }
  if (e.key === 'Enter') { /* navigate via htmx.trigger */ }
  if (e.key === 'Escape') { /* close dropdown */ }
});
```

### 4. Go Handler Pattern for Aggregated Search

**Task**: Find best practices for querying multiple Searchable implementations

**Decision**: Iterate PluginRegistry.All() with type assertion to Searchable interface

**Rationale**:
- Follows existing pattern in codebase (render.go uses similar iteration)
- Type assertion `plugin.(search.Searchable)` is idiomatic Go
- Errors from individual plugins are logged but don't fail entire search
- Results grouped by plugin ID for clear attribution

**Alternatives Considered**:
- Separate SearchRegistry: Duplicates PluginRegistry functionality
- Channel-based concurrent search: Overkill for local single-user tool
- Global search service: Adds unnecessary abstraction layer

**Implementation Pattern**:
```go
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    var results []PluginSearchResult

    for _, rp := range s.registry.All() {
        searchable, ok := rp.Plugin().(search.Searchable)
        if !ok {
            continue
        }
        items, err := searchable.Search(query, 3, search.SortRelevance)
        if err != nil {
            s.logger.Error("search failed", "plugin", rp.Name(), "error", err)
            continue
        }
        // Prefix URLs with plugin name
        for i := range items {
            items[i].URL = PrefixURL(rp.Name(), items[i].URL)
        }
        results = append(results, PluginSearchResult{
            PluginID:   rp.Name(),
            PluginName: /* from sidebar label */,
            Items:      items,
        })
    }
    // Render partial template
}
```

### 5. Dropdown Close Behavior

**Task**: Determine how to close dropdown when clicking outside or navigating away

**Decision**: Use HTMX `hx-on:htmx:after-swap` to clear results + click-outside handler

**Rationale**:
- After navigation, clear the search input and hide results
- Click-outside detection prevents dropdown persisting incorrectly
- Escape key already handled in keyboard navigation

**Alternatives Considered**:
- Focus/blur events only: Unreliable with HTMX content swaps
- Always visible empty state: Clutters UI
- Timer-based auto-close: Poor UX

**Implementation Pattern**:
```javascript
document.addEventListener('click', function(e) {
  const searchContainer = document.getElementById('search-container');
  if (!searchContainer.contains(e.target)) {
    document.getElementById('search-results').innerHTML = '';
  }
});
```

### 6. Plugin Label Resolution

**Task**: Determine how to get human-readable plugin names for search results

**Decision**: Use first SidebarItem label from plugin's SidebarItems() method

**Rationale**:
- SidebarItem already contains human-readable Label
- Consistent with sidebar navigation display
- No additional metadata required on plugins

**Alternatives Considered**:
- Add DisplayName() to WebPlugin interface: Breaking change
- Hardcoded plugin name map: Not extensible
- Use plugin ID directly: Less user-friendly ("memory" vs "Memory")

## Summary

All research complete. No NEEDS CLARIFICATION items remain.

Key decisions:
1. Use HTMX native debounce (`delay:300ms`) - no custom JS debounce needed
2. Absolute positioning with Tailwind for dropdown
3. Vanilla JavaScript for keyboard navigation with data attributes
4. Type assertion pattern for Searchable discovery
5. Click-outside handler + HTMX events for dropdown close
6. SidebarItem.Label for human-readable plugin names
