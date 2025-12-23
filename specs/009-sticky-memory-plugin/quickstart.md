# Quickstart: Sticky Memory Web Plugin

**Feature**: 009-sticky-memory-plugin
**Date**: 2025-12-22

## Prerequisites

- Go 1.24.0+ installed
- zombiekit repository cloned
- Feature 008 (Plugin Web GUI) complete

## Development Setup

### 1. Build the Project

```bash
cd /Users/morgan/Projects/personal/zombiekit
go build -o bin/brains ./cmd/brains
```

### 2. Run the Web GUI

```bash
./bin/brains gui
```

Default: http://localhost:8080

### 3. Access Memory Plugin

Navigate to: http://localhost:8080/memory

## Implementation Order

### Phase 1: Plugin Skeleton (P0)

1. Create plugin directory structure:
```bash
mkdir -p internal/webplugins/memory/templates
```

2. Implement `plugin.go`:
   - Define `Plugin` struct with `memory.Storage` dependency
   - Implement `ID()` → "memory"
   - Implement `SidebarItems()` → single item at Order=20
   - Implement `MountRoutes()` → placeholder handlers
   - Implement `Templates()` → embedded filesystem

3. Register plugin in `internal/cli/gui.go`

4. Create minimal `list.html` template

5. Verify: Memory appears in sidebar, clicking shows empty list

### Phase 2: List View (P1 MVP)

1. Implement list handler with pagination
2. Complete `list.html` template with:
   - Table/cards for memories
   - Pagination controls
   - Search input
   - "New Memory" button
   - Empty state

3. Verify: Can browse paginated list, search works

### Phase 3: View Memory (P1 MVP)

1. Implement view handler
2. Create `view.html` template with:
   - Memory content display
   - Metadata section
   - Markdown rendering (marked.js)
   - View source toggle
   - Edit/Delete buttons

3. Verify: Can click memory to view content, toggle works

### Phase 4: Create Memory (P1 MVP)

1. Implement createForm and create handlers
2. Create `form.html` template with:
   - Name input (required)
   - Content textarea
   - Submit/Cancel buttons
   - Error display

3. Verify: Can create new memory, appears in list

### Phase 5: Edit Memory (P2)

1. Implement editForm and update handlers
2. Extend `form.html` for edit mode:
   - Pre-populated fields
   - Name field disabled
   - Version increment on save

3. Verify: Can edit memory, version increments

### Phase 6: Delete Memory (P2)

1. Implement deleteConfirm and delete handlers
2. Create `delete.html` template with:
   - Confirmation message
   - Memory name display
   - Confirm/Cancel buttons

3. Verify: Delete works with confirmation

### Phase 7: Polish (P3)

1. Add keyboard navigation (SC-006)
2. Improve error handling
3. Add loading states

## File Creation Order

```
1. internal/webplugins/memory/plugin.go
2. internal/webplugins/memory/templates/list.html
3. internal/webplugins/memory/handlers.go (list only)
4. internal/webplugins/memory/templates/view.html
5. internal/webplugins/memory/handlers.go (view added)
6. internal/webplugins/memory/templates/form.html
7. internal/webplugins/memory/handlers.go (create added)
8. internal/webplugins/memory/handlers.go (edit added)
9. internal/webplugins/memory/templates/delete.html
10. internal/webplugins/memory/handlers.go (delete added)
11. internal/webplugins/memory/handlers_test.go
```

## Testing Commands

```bash
# Run all tests
go test ./...

# Run memory plugin tests only
go test ./internal/webplugins/memory/...

# Run with verbose output
go test -v ./internal/webplugins/memory/...

# Run specific test
go test -v -run TestListHandler ./internal/webplugins/memory/...
```

## Verification Checklist

### P1 MVP Complete When:
- [ ] Memory sidebar item visible
- [ ] Empty state shown when no memories
- [ ] Can create a memory
- [ ] Memory appears in list after creation
- [ ] Can click to view memory content
- [ ] Markdown renders correctly
- [ ] Search filters list

### P2 Complete When:
- [ ] Can edit memory content
- [ ] Version increments on edit
- [ ] Can delete with confirmation
- [ ] Deleted memory no longer in list

### P3 Complete When:
- [ ] Keyboard navigation works
- [ ] View toggle (rendered/source) works
- [ ] Pagination respects limit selection

## Common Patterns

### Adding a New Handler

```go
func (h *handlers) newHandler(w http.ResponseWriter, r *http.Request) {
    renderer := web.GetRenderer(r)

    // Get data from storage
    result, err := h.storage.SomeMethod(r.Context(), params)
    if err != nil {
        data := someData{Error: err.Error()}
        renderer.Render(w, r, "memory/template.html", data)
        return
    }

    // Render success
    data := someData{...}
    renderer.Render(w, r, "memory/template.html", data)
}
```

### HTMX Link Pattern

```html
<a href="/memory/{{.Name}}"
   hx-get="/memory/{{.Name}}"
   hx-target="#content"
   hx-push-url="true"
   class="hover:underline">
   {{.Name}}
</a>
```

### Form Pattern

```html
<form hx-post="/memory"
      hx-target="#content"
      hx-push-url="true"
      class="space-y-4">
    <input type="text" name="name" required
           class="border rounded px-3 py-2 w-full">
    <textarea name="content"
              class="border rounded px-3 py-2 w-full h-64"></textarea>
    <button type="submit"
            class="bg-blue-600 text-white px-4 py-2 rounded">
        Save
    </button>
</form>
```

## Troubleshooting

### Plugin Not Showing in Sidebar

1. Check plugin is registered in `gui.go`
2. Verify `ID()` returns "memory"
3. Check `SidebarItems()` returns non-empty slice

### Templates Not Loading

1. Verify `//go:embed templates/*` directive exists
2. Check template files are in `templates/` subdirectory
3. Ensure template names match (e.g., "memory/list.html")

### HTMX Not Working

1. Check `hx-target="#content"` is correct
2. Verify no JavaScript errors in console
3. Ensure response returns HTML (not redirect for HTMX)

### Markdown Not Rendering

1. Check marked.js CDN is loaded
2. Verify content is in `data-raw` attribute
3. Check `htmx:afterSwap` listener is registered
