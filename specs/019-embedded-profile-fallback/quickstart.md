# Quickstart: Embedded Profile Fallback

**Feature**: 019-embedded-profile-fallback
**Date**: 2025-12-23

## Overview

This feature embeds default profiles directly into the brains binary, enabling immediate use without configuration.

## Usage

### No Configuration Needed

After building brains with embedded profiles, all profile commands work out of the box:

```bash
# List available profiles (includes embedded ones)
brains profile list

# Compose an embedded profile
brains profile compose init

# Show embedded profile details
brains profile show research
```

### Overriding Embedded Profiles

Create a local profile with the same name to override:

```bash
# Create local override for the "init" profile
brains profile create init

# Your local version now takes precedence
brains profile compose init
```

### Inheritance from Embedded Profiles

Create a local profile that extends an embedded one:

```yaml
---
name: custom-init
inherits: true
---

# Additional content added after embedded base
```

## MCP Tool Usage

The MCP tools work identically with embedded profiles:

```json
// profile-list - returns embedded profiles
{
  "profiles": [
    {"name": "init", "source": "embedded"},
    {"name": "research", "source": "embedded"}
  ]
}

// profile-compose with embedded profile
{
  "profiles": ["init"]
}
// Returns content from embedded init.md
```

## For Developers

### Building with Embedded Profiles

No special build steps required:

```bash
go build -o brains ./cmd/brains
```

The `//go:embed profiles/*` directive in `cmd/brains/main.go` automatically includes all profile files.

### Testing

Unit tests can use a mock embed.FS:

```go
import "testing/fstest"

func TestWithMockEmbedded(t *testing.T) {
    mockFS := fstest.MapFS{
        "test.md": &fstest.MapFile{
            Data: []byte("---\nname: test\n---\nTest content"),
        },
    }
    profile.SetEmbeddedFS(mockFS)
    // ... run tests
}
```

### Adding New Embedded Profiles

1. Add `.md` file to `profiles/` directory
2. Ensure valid YAML frontmatter
3. Rebuild the binary

## Verification Checklist

After implementation, verify:

- [X] `brains profile list` shows embedded profiles with source "embedded"
- [X] `brains profile compose <embedded>` returns content
- [X] Local profile overrides embedded when same name
- [X] MCP profile-list includes embedded profiles
- [X] Binary works on fresh machine without `.brains/profiles/`
