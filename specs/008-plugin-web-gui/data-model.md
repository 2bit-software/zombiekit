# Data Model: Plugin-Style Web GUI Architecture

**Feature**: 008-plugin-web-gui
**Date**: 2025-12-22

## Overview

This feature introduces interfaces and value types for the web plugin architecture. No persistent storage is added; the example plugin uses the existing `profile.Service`.

## Core Entities

### SidebarItem

Navigation entry displayed in the sidebar.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| ID | string | Yes | Unique identifier for this item |
| Label | string | Yes | Display text in navigation |
| Path | string | Yes | URL path (e.g., "/profiles") |
| Icon | string | No | Icon name (for future use) |
| Order | int | Yes | Sort order (lower = earlier) |
| Badge | string | No | Optional badge text (e.g., count) |
| Children | []SidebarItem | No | Nested items for expandable menus |

**Validation Rules**:
- ID must be non-empty, unique within a plugin
- Path must start with "/"
- Order must be >= 0

### PageData

Data structure passed to all templates.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Title | string | No | Page title (appended to site name) |
| SidebarItems | []SidebarItem | Yes | All sidebar items from registry |
| ActivePath | string | Yes | Current URL path for highlighting |
| Content | any | No | Plugin-specific template data |
| IsHTMX | bool | Yes | True if partial update request |

### ServerConfig

Configuration for the web server.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| Port | int | No | 8080 | HTTP listen port |
| ReadTimeout | time.Duration | No | 15s | Max time to read request |
| WriteTimeout | time.Duration | No | 15s | Max time to write response |
| IdleTimeout | time.Duration | No | 60s | Max keep-alive idle time |

## Interfaces

### WebPlugin

Core interface that tools implement to participate in the web GUI.

```go
type WebPlugin interface {
    // ID returns a unique identifier for this plugin.
    // Routes are mounted at /{ID}/...
    ID() string

    // SidebarItems returns navigation entries for this plugin.
    SidebarItems() []SidebarItem

    // MountRoutes registers HTTP handlers on the router.
    // Router is already scoped to /{ID}/
    MountRoutes(r chi.Router)
}
```

### TemplatePlugin (Optional Extension)

For plugins that provide their own templates.

```go
type TemplatePlugin interface {
    WebPlugin

    // Templates returns an embed.FS with plugin templates.
    // Templates should be in a "templates/" subdirectory.
    Templates() fs.FS
}
```

### PluginRegistry

Central store for registered plugins.

```go
type PluginRegistry interface {
    // Register adds a plugin. Returns error if ID already exists.
    Register(p WebPlugin) error

    // Get retrieves a plugin by ID.
    Get(id string) (WebPlugin, bool)

    // All returns all plugins in registration order.
    All() []WebPlugin

    // SidebarItems returns aggregated, sorted sidebar items.
    SidebarItems() []SidebarItem
}
```

### Renderer

Template rendering service.

```go
type Renderer interface {
    // Render renders a template with HTMX-aware handling.
    // For HTMX requests: renders content template only.
    // For full page: renders shell with content embedded.
    Render(w http.ResponseWriter, r *http.Request, tmpl string, data any) error

    // RenderPartial always renders just the template (no shell).
    RenderPartial(w http.ResponseWriter, tmpl string, data any) error
}
```

## State Transitions

### Plugin Registration Flow

```
Unregistered → Register(plugin) → Registered
                    │
                    ├── Success: plugin added to registry
                    │
                    └── Error: duplicate ID (registry unchanged)
```

### Request Handling Flow

```
Request Received
    │
    ├── Static asset? → Serve from embed.FS
    │
    ├── Health check? → Return JSON {"status": "healthy"}
    │
    ├── Root path (/)? → Render dashboard
    │
    └── Plugin path (/{id}/...)?
            │
            ├── Plugin exists? → Route to plugin handler
            │       │
            │       └── Handler calls Render()
            │               │
            │               ├── HX-Request header? → Content only
            │               │
            │               └── No header → Shell + content
            │
            └── Plugin not found → 404 page
```

## Relationships

```
┌─────────────────┐       ┌─────────────────┐
│  PluginRegistry │◀──────│    Server       │
└────────┬────────┘       └────────┬────────┘
         │                         │
         │ contains                │ uses
         ▼                         ▼
┌─────────────────┐       ┌─────────────────┐
│   WebPlugin     │       │    Renderer     │
│   (interface)   │       │                 │
└────────┬────────┘       └─────────────────┘
         │
         │ implemented by
         ▼
┌─────────────────┐
│ profiles.Plugin │
│ (example)       │
└────────┬────────┘
         │
         │ uses
         ▼
┌─────────────────┐
│ profile.Service │
│ (existing)      │
└─────────────────┘
```

## Data Volumes

| Entity | Expected Count | Notes |
|--------|----------------|-------|
| Plugins | 1-10 | Typically few tools registered |
| SidebarItems | 1-20 | 1-3 per plugin |
| Concurrent Users | 1 | Local development tool |
| Templates | 10-30 | Shell + plugin templates |
