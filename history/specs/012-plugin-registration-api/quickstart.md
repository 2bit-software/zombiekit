# Quickstart: Plugin Registration API

**Feature**: 012-plugin-registration-api
**Date**: 2025-12-22

## Overview

This guide shows how to register plugins with the new simplified API and how to migrate existing plugins.

## Before & After

### Registration (at startup)

```go
// BEFORE
registry := web.NewPluginRegistry()
registry.Register(memory.NewPlugin(storage))  // Plugin provided its own ID via ID() method
registry.Register(profiles.NewPlugin(svc))

// AFTER
registry := web.NewPluginRegistry(logger)
registry.Register("memory", memory.NewPlugin(storage))   // Name provided here
registry.Register("profiles", profiles.NewPlugin(svc))
```

### Plugin Implementation

```go
// BEFORE
type Plugin struct { ... }

func (p *Plugin) ID() string {
    return "memory"  // Plugin knew its own name
}

func (p *Plugin) SidebarItems() []web.SidebarItem {
    return []web.SidebarItem{{
        Path: "/memory",  // Full path
    }}
}

func (p *Plugin) Search(...) ([]search.SearchResult, error) {
    return []search.SearchResult{{
        URL: "/memory/my-note",  // Full path
    }}, nil
}

// AFTER
type Plugin struct { ... }

// No ID() method!

func (p *Plugin) SidebarItems() []web.SidebarItem {
    return []web.SidebarItem{{
        Path: "/",  // Relative path - system prefixes automatically
    }}
}

func (p *Plugin) Search(...) ([]search.SearchResult, error) {
    return []search.SearchResult{{
        URL: "/my-note",  // Relative path - system prefixes automatically
    }}, nil
}
```

## Migration Checklist

### For Each Plugin

- [ ] Remove `ID() string` method
- [ ] Update `SidebarItems()` to return relative paths
  - `"/memory"` → `"/"`
  - `"/memory/settings"` → `"/settings"`
- [ ] Update `Search()` to return relative URLs
  - `"/memory/note-name"` → `"/note-name"`
- [ ] Check handlers for hardcoded redirects (update to relative)
- [ ] Update tests that check for full paths

### At Registration Site

- [ ] Update `registry.Register(plugin)` to `registry.Register("name", plugin)`
- [ ] Ensure plugin name is valid: lowercase, alphanumeric, hyphens only

## Valid Plugin Names

| Valid | Invalid | Reason |
|-------|---------|--------|
| `memory` | `Memory` | Must be lowercase |
| `my-plugin` | `my_plugin` | Use hyphens, not underscores |
| `plugin2` | `2plugin` | Can't start with number (per regex) |
| `a-b-c` | `-abc` | Can't start with hyphen |
| `abc` | `abc-` | Can't end with hyphen |
| `my-cool-plugin` | `my--plugin` | No consecutive hyphens |

## Testing Your Migration

```go
func TestPluginReturnsRelativePaths(t *testing.T) {
    plugin := NewPlugin(mockStorage)

    items := plugin.SidebarItems()
    for _, item := range items {
        if strings.HasPrefix(item.Path, "/memory") {
            t.Errorf("SidebarItem path should be relative, got: %s", item.Path)
        }
    }

    results, _ := plugin.Search("test", 10, search.SortRelevance)
    for _, result := range results {
        if strings.HasPrefix(result.URL, "/memory") {
            t.Errorf("Search URL should be relative, got: %s", result.URL)
        }
    }
}
```

## Common Mistakes

### 1. Forgetting to Remove ID()

```go
// Compiler will catch this - interface no longer requires ID()
// But if you have ID() it won't break, just unused
```

### 2. Still Using Full Paths

```go
// WRONG
return []web.SidebarItem{{Path: "/memory"}}

// RIGHT
return []web.SidebarItem{{Path: "/"}}
```

### 3. Hardcoded Redirects in Handlers

```go
// WRONG
http.Redirect(w, r, "/memory/"+name, http.StatusSeeOther)

// RIGHT
http.Redirect(w, r, "/"+name, http.StatusSeeOther)
```

## URL Prefixing Behavior

The system automatically prefixes URLs:

| Plugin Name | Input URL | Output URL |
|-------------|-----------|------------|
| `memory` | `/` | `/memory/` |
| `memory` | `/notes` | `/memory/notes` |
| `memory` | `notes` | `/memory/notes` |
| `memory` | `/memory/notes` | `/memory/notes` (no double prefix) |
| `memory` | `https://example.com` | `https://example.com` (unchanged) |
