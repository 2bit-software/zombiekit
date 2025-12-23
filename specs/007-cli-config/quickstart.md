# Quickstart: CLI Configuration System

**Feature**: 007-cli-config

## Configuration File Locations

| Platform | Global Config Path |
|----------|-------------------|
| Linux | `~/.config/brains/config.toml` (or `$XDG_CONFIG_HOME/brains/config.toml`) |
| macOS | `~/.config/brains/config.toml` (or `$XDG_CONFIG_HOME/brains/config.toml`) |
| Windows | `%APPDATA%\brains\config.toml` |

Local config is always `.brains/config.toml` relative to the current working directory.

## Basic Usage

### Disable a tool globally

Create `~/.config/brains/config.toml`:

```toml
[tools.stickymemory]
enabled = false
```

### Disable a tool category

```toml
[tools.profile]
enabled = false
# Disables both profile-compose and profile-list
```

### Enable specific tool in disabled category

```toml
[tools.profile]
enabled = false

[tools.profile-list]
enabled = true
# profile-compose disabled, profile-list enabled
```

### Override via CLI

```bash
# Disable tool for this session only
brains serve --disable-tool=stickymemory

# Enable tool that's disabled in config
brains serve --enable-tool=code-reasoning

# Multiple tools
brains serve --disable-tool=stickymemory --disable-tool=code-reasoning
```

## Precedence Order

1. **CLI flags** (highest priority)
2. **Local config** (`.brains/config.toml`)
3. **Global config** (`~/.config/brains/config.toml`)
4. **Defaults** (all tools enabled)

## Debugging Configuration

Use debug log level to see which config files are loaded:

```bash
brains serve --log-level=debug
```

Output will show:
```
level=DEBUG msg="loaded global config" path=/home/user/.config/brains/config.toml
level=DEBUG msg="loaded local config" path=/project/.brains/config.toml
```

## Available Tools

| Tool Name | Category | Description |
|-----------|----------|-------------|
| stickymemory | stickymemory | Persistent memory storage |
| code-reasoning | code | Sequential thinking/reasoning |
| profile-compose | profile | Compose multiple profiles |
| profile-list | profile | List available profiles |

## Example: Project-Specific Configuration

```bash
# Create project config
mkdir -p .brains
cat > .brains/config.toml << 'EOF'
# Disable memory for this project
[tools.stickymemory]
enabled = false
EOF

# Verify
brains serve --log-level=debug
# Check that stickymemory is not in tool list
```
