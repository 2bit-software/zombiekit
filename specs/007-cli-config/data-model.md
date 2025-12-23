# Data Model: CLI Configuration System

**Feature**: 007-cli-config
**Date**: 2025-12-22

## Entities

### Config

The root configuration structure representing merged settings from all sources.

| Field | Type | Description |
|-------|------|-------------|
| Tools | map[string]ToolConfig | Map of tool/category name to configuration |

**Validation Rules**:
- Tool names must be non-empty strings
- Unknown tool names generate warnings but don't cause errors

**State Transitions**: N/A (immutable after loading)

### ToolConfig

Configuration for a single tool or tool category.

| Field | Type | Description |
|-------|------|-------------|
| Enabled | *bool | Whether tool is enabled. nil = not set (inherit from category/default) |

**Validation Rules**:
- Enabled must be explicitly true or false when set
- nil means "not specified at this config level"

### ConfigSource

Represents a single configuration file with its origin.

| Field | Type | Description |
|-------|------|-------------|
| Path | string | Absolute path to the config file |
| Level | SourceLevel | Enum: Global, Local, CLI |
| Config | *Config | Parsed configuration (nil if load failed) |
| Error | error | Parse/load error if any |

**Source Levels** (precedence order, highest first):
1. `CLI` - Command line flags
2. `Local` - `.brains/config.toml` in working directory
3. `Global` - `~/.config/brains/config.toml` (or platform equivalent)

## Relationships

```
ConfigSource (Global) ─┐
                       ├─> Merger ─> Config (merged)
ConfigSource (Local)  ─┤
                       │
CLI Flags ─────────────┘

Config.Tools["profile"] ─────> affects ─> profile-compose, profile-list
Config.Tools["profile-list"] ─> overrides ─> Config.Tools["profile"]
```

## TOML Schema

```toml
# .brains/config.toml

[tools]
# Disable entire categories
[tools.profile]
enabled = false

# Or individual tools
[tools.stickymemory]
enabled = false

# Enable specific tool even if category disabled
[tools.profile-list]
enabled = true
```

## Tool Name Registry

Current known tools and their categories:

| Tool Name | Category | Default |
|-----------|----------|---------|
| stickymemory | stickymemory | enabled |
| code-reasoning | code | enabled |
| profile-compose | profile | enabled |
| profile-list | profile | enabled |

**Category Derivation Rule**: Category = substring before first hyphen, or full name if no hyphen.

## Merge Algorithm

```
function MergeConfigs(global, local, cli) -> Config:
    result = DefaultConfig()  // all tools enabled

    // Apply in precedence order (lowest first)
    if global.Config != nil:
        result.Merge(global.Config)

    if local.Config != nil:
        result.Merge(local.Config)

    // CLI flags applied last (highest precedence)
    for tool in cli.DisabledTools:
        result.Tools[tool].Enabled = false

    for tool in cli.EnabledTools:
        result.Tools[tool].Enabled = true

    return result
```

## IsToolEnabled Algorithm

```
function IsToolEnabled(config, toolName) -> bool:
    // 1. Check tool-specific setting
    if config.Tools[toolName].Enabled != nil:
        return *config.Tools[toolName].Enabled

    // 2. Check category setting
    category = ToolCategory(toolName)
    if category != toolName && config.Tools[category].Enabled != nil:
        return *config.Tools[category].Enabled

    // 3. Default to enabled
    return true
```
