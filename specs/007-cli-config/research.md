# Research: CLI Configuration System

**Feature**: 007-cli-config
**Date**: 2025-12-22

## Research Topics

### 1. TOML Parsing in Go

**Decision**: Use `github.com/BurntSushi/toml` for TOML parsing

**Rationale**:
- Already an indirect dependency in go.mod (v1.5.0)
- De-facto standard for TOML in Go
- Supports TOML 1.0 specification
- Simple API: `toml.DecodeFile()` and `toml.Decode()`
- Good error messages with line numbers for parse failures

**Alternatives Considered**:
- `pelletier/go-toml/v2`: Faster but adds new dependency, overkill for small config files
- `encoding/json` with YAML: Different format than user requested
- Custom parser: Unnecessary complexity

**Implementation Pattern**:
```go
type Config struct {
    Tools map[string]ToolConfig `toml:"tools"`
}

type ToolConfig struct {
    Enabled *bool `toml:"enabled"` // Pointer to detect explicit false vs absent
}

var cfg Config
if _, err := toml.DecodeFile(path, &cfg); err != nil {
    // Log warning, continue with defaults
}
```

### 2. XDG Base Directory Specification

**Decision**: Implement XDG-compliant path resolution with platform fallbacks

**Rationale**:
- XDG is the standard on Linux and increasingly on macOS
- Windows has different conventions (`%APPDATA%`)
- Go stdlib `os.UserConfigDir()` handles this automatically

**Alternatives Considered**:
- Custom path logic: Error-prone, reinvents the wheel
- Third-party XDG library: Unnecessary, `os.UserConfigDir()` sufficient
- Hardcoded `~/.brains/`: Not XDG compliant, breaks expectations

**Implementation Pattern**:
```go
func GlobalConfigPath() (string, error) {
    // os.UserConfigDir() returns:
    // - Linux: $XDG_CONFIG_HOME or ~/.config
    // - macOS: ~/Library/Application Support (but we prefer ~/.config for CLI tools)
    // - Windows: %APPDATA%

    if runtime.GOOS == "darwin" {
        // Prefer XDG on macOS for CLI tools
        if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
            return filepath.Join(xdg, "brains", "config.toml"), nil
        }
        home, err := os.UserHomeDir()
        if err != nil {
            return "", err
        }
        return filepath.Join(home, ".config", "brains", "config.toml"), nil
    }

    configDir, err := os.UserConfigDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(configDir, "brains", "config.toml"), nil
}
```

**Note on macOS**: While `os.UserConfigDir()` returns `~/Library/Application Support` on macOS, CLI tools conventionally use `~/.config` for XDG compatibility. The implementation respects `$XDG_CONFIG_HOME` and falls back to `~/.config/brains/` on macOS.

### 3. Configuration Precedence Merging

**Decision**: Use nil-pointer pattern for optional boolean fields, merge in precedence order

**Rationale**:
- Need to distinguish "explicitly set to false" from "not set"
- Simple merge: later values overwrite earlier values
- No deep merge needed - tool settings are flat key-value pairs

**Alternatives Considered**:
- Bit flags for each setting: Complex, hard to extend
- Separate "override" vs "default" structs: Over-engineered
- JSON merge patch (RFC 7396): Overkill for this use case

**Implementation Pattern**:
```go
// Merge applies src settings over dst (src wins for set values)
func (dst *Config) Merge(src *Config) {
    for name, srcTool := range src.Tools {
        if dstTool, exists := dst.Tools[name]; exists {
            if srcTool.Enabled != nil {
                dstTool.Enabled = srcTool.Enabled
                dst.Tools[name] = dstTool
            }
        } else {
            dst.Tools[name] = srcTool
        }
    }
}
```

### 4. Tool Category Matching

**Decision**: Derive category from tool name prefix (hyphen-separated)

**Rationale**:
- Current tools: `stickymemory`, `code-reasoning`, `profile-compose`, `profile-list`
- Tools with hyphens: prefix before first hyphen is category
- Tools without hyphens: tool name is its own category
- Simple, no configuration needed, follows existing naming

**Alternatives Considered**:
- Explicit category mapping in config: Adds complexity
- Separate category config section: Over-engineered
- Tag-based categories: Requires schema changes

**Implementation Pattern**:
```go
func ToolCategory(toolName string) string {
    if idx := strings.Index(toolName, "-"); idx > 0 {
        return toolName[:idx]
    }
    return toolName
}

// IsToolEnabled checks tool-specific then category-level config
func (c *Config) IsToolEnabled(toolName string) bool {
    // Check tool-specific setting first
    if tool, ok := c.Tools[toolName]; ok && tool.Enabled != nil {
        return *tool.Enabled
    }
    // Check category setting
    category := ToolCategory(toolName)
    if cat, ok := c.Tools[category]; ok && cat.Enabled != nil {
        return *cat.Enabled
    }
    // Default: enabled
    return true
}
```

### 5. CLI Flag Handling for Tool Enable/Disable

**Decision**: Use urfave/cli StringSlice flags for `--enable-tool` and `--disable-tool`

**Rationale**:
- Already using urfave/cli/v2 for the CLI
- StringSlice allows multiple flag instances: `--disable-tool=foo --disable-tool=bar`
- Integrates naturally with existing serve command flags

**Alternatives Considered**:
- Comma-separated single flag: Less intuitive, harder to shell-escape
- Custom flag type: Unnecessary, StringSlice works
- Positional arguments: Conflicts with existing command structure

**Implementation Pattern**:
```go
&cli.StringSliceFlag{
    Name:  "enable-tool",
    Usage: "Enable specific MCP tool (can be repeated)",
},
&cli.StringSliceFlag{
    Name:  "disable-tool",
    Usage: "Disable specific MCP tool (can be repeated)",
},
```

### 6. Error Handling for Invalid Config

**Decision**: Log warnings and continue with defaults on config errors

**Rationale**:
- Graceful degradation preferred over hard failure
- User may not realize config exists or has errors
- MCP server should start even with config issues
- Warning includes file path and error for debugging

**Alternatives Considered**:
- Fail on invalid config: Too strict, breaks user workflow
- Silent fallback: User unaware of config issues
- Interactive prompt: Not appropriate for MCP server startup

**Implementation Pattern**:
```go
func LoadConfig(logger *slog.Logger) *Config {
    cfg := NewDefaultConfig()

    // Load global config
    globalPath, _ := GlobalConfigPath()
    if err := cfg.LoadFile(globalPath); err != nil && !os.IsNotExist(err) {
        logger.Warn("failed to load global config",
            "path", globalPath,
            "error", err.Error(),
        )
    } else if err == nil {
        logger.Debug("loaded global config", "path", globalPath)
    }

    // Similar for local config...
    return cfg
}
```

## Summary

All research topics resolved with clear decisions. No NEEDS CLARIFICATION items remain. The implementation will use:

1. **BurntSushi/toml** (existing dependency) for TOML parsing
2. **os.UserConfigDir()** with macOS XDG override for global config path
3. **Nil-pointer pattern** for optional boolean merge
4. **Prefix-based category** derived from tool names
5. **StringSlice CLI flags** for enable/disable
6. **Warning-and-continue** error handling strategy
