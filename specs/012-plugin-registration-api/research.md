# Research: Simplified Plugin Registration API

**Feature**: 012-plugin-registration-api
**Date**: 2025-12-22

## Research Areas

### 1. Plugin Name Validation Pattern

**Decision**: Use regex validation for URL-safe names (alphanumeric + hyphens)

**Rationale**: Plugin names become URL path segments, so they must be URL-safe without encoding. The pattern `^[a-z0-9]+(-[a-z0-9]+)*$` ensures:
- Lowercase only (consistent, case-insensitive URLs)
- No leading/trailing hyphens
- No consecutive hyphens
- No special characters requiring URL encoding

**Alternatives Considered**:
1. Allow uppercase (rejected: URLs are case-insensitive by convention)
2. Allow underscores (rejected: hyphens are more URL-conventional)
3. No validation (rejected: could break routing)

---

### 2. Registration Panic vs Error Return

**Decision**: Panic on registration failure (per spec clarification)

**Rationale**: Registration errors are programmer errors (configuration mistakes) not runtime errors. Panicking at startup:
- Fails fast before the server starts accepting requests
- Makes configuration errors immediately obvious
- Is the standard Go pattern for initialization-time errors

**Alternatives Considered**:
1. Return error (rejected: caller would likely panic anyway, extra boilerplate)
2. Log and skip (rejected: silent failure leads to confusing "missing plugin" bugs)

---

### 3. URL Prefix Detection for Double-Prefix Prevention

**Decision**: Check if URL already starts with `/{pluginName}/` before prefixing

**Rationale**: When a plugin registered as "foo" returns URL "/foo/bar", the system should not produce "/foo/foo/bar". Simple prefix check handles this case.

**Implementation**:
```go
func prefixURL(pluginName, url string) string {
    prefix := "/" + pluginName
    if strings.HasPrefix(url, prefix+"/") || url == prefix {
        return url // Already prefixed
    }
    if !strings.HasPrefix(url, "/") {
        url = "/" + url
    }
    return prefix + url
}
```

**Alternatives Considered**:
1. Require plugins to never include prefix (rejected: migration burden, easy to forget)
2. Strip all prefixes and rebuild (rejected: over-complicated for rare edge case)

---

### 4. Absolute URL Detection

**Decision**: URLs starting with `http://` or `https://` are passed through unchanged

**Rationale**: External links must not be prefixed. Simple prefix check is sufficient.

**Implementation**:
```go
func isAbsoluteURL(url string) bool {
    return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
```

**Alternatives Considered**:
1. Use net/url.Parse (rejected: overkill for this check, slower)
2. Support other schemes like `mailto:` (rejected: not needed per spec)

---

### 5. Plugin Name Injection Pattern

**Decision**: Constructor injection - pass name when creating plugin instance (per spec clarification)

**Rationale**: Plugins that need their registered name (e.g., for search result metadata) receive it via constructor. This:
- Makes the dependency explicit
- Keeps the WebPlugin interface minimal
- Follows standard dependency injection patterns

**Implementation Example**:
```go
// Before (if plugin needed to know its name, it couldn't)
plugin := memory.NewPlugin(storage)
registry.Register(plugin) // plugin.ID() was "memory"

// After
plugin := memory.NewPlugin(storage, "memory") // name injected
webgui.Register("memory", plugin)
```

**Alternatives Considered**:
1. Context parameter per call (rejected: repetitive, complicates interface)
2. Plugin queries registry (rejected: circular dependency)

---

### 6. Sidebar Path Prefixing Location

**Decision**: Prefix sidebar paths when aggregating in PluginRegistry.SidebarItems()

**Rationale**: Centralizing the prefixing logic in the registry:
- Plugins remain unaware of their mount location
- Single place to apply/modify prefixing logic
- Consistent with route mounting pattern

**Implementation**:
```go
func (r *PluginRegistry) SidebarItems() []SidebarItem {
    var items []SidebarItem
    for name, p := range r.plugins {
        for _, item := range p.SidebarItems() {
            item.Path = prefixURL(name, item.Path)
            items = append(items, item)
        }
    }
    // ... sort by order ...
    return items
}
```

---

### 7. Searchable Interface URL Handling

**Decision**: Search results return plugin-relative URLs; aggregator prefixes them

**Rationale**: The Searchable interface (in `internal/search/`) is independent of web plugins. When a WebPlugin also implements Searchable:
- The plugin returns relative URLs (e.g., "/notes")
- The search aggregator (part of webgui-search feature) prefixes URLs when combining results from multiple plugins

**Note**: This feature (012) prepares the foundation. The search aggregator will be implemented in the webgui-search feature (011).

---

### 8. Logging Pattern

**Decision**: Log at Info level with structured fields (per spec clarification)

**Rationale**: Using slog (already in use per server.go):
```go
logger.Info("registered plugin", "name", name, "path", "/"+name)
```

This provides:
- Startup diagnostics
- Troubleshooting capability
- No performance impact (once at startup)

---

## Migration Strategy

### Existing Plugin Changes

1. **Remove ID() method** from Plugin struct
2. **Update constructor** to accept name parameter (for plugins needing it)
3. **Update SidebarItems()** to return relative paths (e.g., "/" instead of "/memory")
4. **Update Search()** to return relative URLs (e.g., "/notes" instead of "/memory/notes")
5. **Update handlers** if any construct absolute redirect URLs

### Registration Site Changes

The registration call changes from:
```go
// Before
registry.Register(memory.NewPlugin(storage))

// After
webgui.Register("memory", memory.NewPlugin(storage))
```

### Breaking Change Communication

- Update any documentation referencing the old pattern
- The change is internal to this project (no external consumers)
- Both existing plugins (memory, profiles) migrate in this feature
