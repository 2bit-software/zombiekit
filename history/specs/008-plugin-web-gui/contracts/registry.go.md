# Contract: PluginRegistry

**Package**: `internal/web`

## Interface Definition

```go
package web

import "sync"

// PluginRegistry manages registered web plugins.
type PluginRegistry struct {
    plugins map[string]WebPlugin
    order   []string // maintains registration order
    mu      sync.RWMutex
}

// NewPluginRegistry creates a new plugin registry.
func NewPluginRegistry() *PluginRegistry

// Register adds a plugin to the registry.
// Returns an error if a plugin with the same ID already exists.
func (r *PluginRegistry) Register(p WebPlugin) error

// Get retrieves a plugin by ID.
// Returns the plugin and true if found, nil and false otherwise.
func (r *PluginRegistry) Get(id string) (WebPlugin, bool)

// All returns all registered plugins in registration order.
func (r *PluginRegistry) All() []WebPlugin

// SidebarItems returns all sidebar items from all plugins, sorted by Order.
func (r *PluginRegistry) SidebarItems() []SidebarItem
```

## Contract Guarantees

### Thread Safety
- All methods MUST be safe for concurrent access
- Registration typically happens at startup (single-threaded)
- Get, All, SidebarItems may be called concurrently during request handling

### Registration
- First registration with an ID wins
- Duplicate ID returns error, does not replace
- Registration order is preserved for iteration

### SidebarItems
- Aggregates items from all plugins
- Sorts by Order field (ascending)
- Stable sort preserves registration order for equal Order values

## Error Types

```go
// ErrDuplicatePlugin is returned when registering a plugin with an existing ID.
type ErrDuplicatePlugin struct {
    ID string
}

func (e *ErrDuplicatePlugin) Error() string {
    return fmt.Sprintf("plugin %q already registered", e.ID)
}
```
