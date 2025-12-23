# Data Model: Simplified Plugin Registration API

**Feature**: 012-plugin-registration-api
**Date**: 2025-12-22

## Overview

This feature modifies the plugin registration system. The primary changes are to interfaces and structs, not to persistent storage. All entities exist in-memory during server runtime.

## Entities

### WebPlugin (Modified Interface)

The core interface that plugins implement. The `ID() string` method is **removed**.

```go
// Before
type WebPlugin interface {
    ID() string                          // REMOVED
    SidebarItems() []SidebarItem
    MountRoutes(r chi.Router)
}

// After
type WebPlugin interface {
    SidebarItems() []SidebarItem         // Returns relative paths
    MountRoutes(r chi.Router)            // Router already scoped to /{name}/
}
```

**Field Changes**:
| Field | Change | Notes |
|-------|--------|-------|
| ID() | Removed | Name now provided at registration time |
| SidebarItems() | Modified semantics | Must return relative paths (e.g., "/" not "/memory") |
| MountRoutes() | Unchanged | Router remains pre-scoped |

---

### TemplatePlugin (Unchanged)

```go
type TemplatePlugin interface {
    WebPlugin
    Templates() fs.FS
}
```

No changes required. Extends WebPlugin.

---

### SidebarItem (Semantics Changed)

```go
type SidebarItem struct {
    ID       string        // Unique identifier
    Label    string        // Display text
    Path     string        // NOW: Relative path (e.g., "/", "/settings")
    Icon     string        // Icon name
    Order    int           // Sort order
    Badge    string        // Optional badge text
    Children []SidebarItem // Nested items
}
```

**Semantic Change**:
- `Path` field now contains **plugin-relative** paths
- The registry automatically prefixes with the plugin name when aggregating

---

### PluginRegistry (Modified)

```go
// Before
type PluginRegistry struct {
    mu      sync.RWMutex
    plugins []WebPlugin
    byID    map[string]WebPlugin  // Key came from plugin.ID()
}

// After
type PluginRegistry struct {
    mu      sync.RWMutex
    plugins []registeredPlugin    // Ordered list for iteration
    byName  map[string]WebPlugin  // Key comes from Register() call
    logger  *slog.Logger          // For registration logging
}

type registeredPlugin struct {
    name   string
    plugin WebPlugin
}
```

**Field Changes**:
| Field | Change | Notes |
|-------|--------|-------|
| byID | Renamed to byName | Semantic clarity: name from registration, not plugin |
| plugins | Type changed | Now stores name alongside plugin for iteration |
| logger | Added | For Info-level registration logging |

---

### DuplicatePluginError (Unchanged)

```go
type DuplicatePluginError struct {
    ID string  // Could rename to Name for consistency
}
```

Consider renaming `ID` field to `Name` for consistency, but not strictly required.

---

## New Functions

### Register (Package-level)

```go
// Package web

// Register adds a plugin to the global registry with the given name.
// Panics if:
//   - name is empty
//   - name contains invalid characters (must match [a-z0-9]+(-[a-z0-9]+)*)
//   - name is already registered
//
// Logs plugin registration at Info level.
func Register(name string, plugin WebPlugin)
```

**Validation Rules**:
- Name must match regex: `^[a-z0-9]+(-[a-z0-9]+)*$`
- Name must not be empty
- Name must not be duplicate

---

### Helper Functions

```go
// PrefixURL prepends the plugin name to a relative URL.
// Absolute URLs (http://, https://) are returned unchanged.
// Already-prefixed URLs are returned unchanged.
func PrefixURL(pluginName, url string) string

// ValidatePluginName checks if a name is valid for plugin registration.
// Returns an error describing the validation failure, or nil if valid.
func ValidatePluginName(name string) error
```

---

## State Transitions

No persistent state. Plugins are registered once at startup and remain registered for the server lifetime.

```
Application Start
    │
    ▼
Register("memory", plugin1)  ──► Log: "registered plugin" name=memory path=/memory
    │
    ▼
Register("profiles", plugin2) ──► Log: "registered plugin" name=profiles path=/profiles
    │
    ▼
Server.Start()  ──► Plugins mounted at /{name}/...
    │
    ▼
Running (plugins immutable)
    │
    ▼
Shutdown
```

---

## Relationships

```
PluginRegistry
    │
    ├── 1:N ── registeredPlugin
    │              │
    │              ├── name: string
    │              │
    │              └── plugin: WebPlugin
    │                      │
    │                      ├── SidebarItems() []SidebarItem
    │                      │
    │                      └── MountRoutes(chi.Router)
    │
    └── SidebarItems() ──► Aggregates + prefixes all plugin sidebar items
```

---

## Validation Rules

| Entity | Field | Rule |
|--------|-------|------|
| Register() | name | Must match `^[a-z0-9]+(-[a-z0-9]+)*$` |
| Register() | name | Must not be empty |
| Register() | name | Must not duplicate existing registration |
| Register() | plugin | Must not be nil |
| SidebarItem | Path | Should be relative (no validation, but prefixing assumes relative) |
| SearchResult | URL | Should be relative (no validation, but prefixing assumes relative) |

---

## Impact on Existing Code

### memory.Plugin
- Remove `ID() string` method
- Change `SidebarItems()` Path from `/memory` to `/`
- Change `Search()` URL from `/memory/{name}` to `/{name}`

### profiles.Plugin
- Remove `ID() string` method
- Change `SidebarItems()` Path from `/profiles` to `/`
- Change `Search()` URL from `/profiles/{name}` to `/{name}`

### web.Server
- Update `setupRouter()` to use registry's stored names instead of `plugin.ID()`

### Registration site (cmd or main)
- Change from `registry.Register(plugin)` to `webgui.Register("name", plugin)`
