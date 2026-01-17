# CLI Contract: Profile Commands

**Date**: 2025-12-22
**Feature**: 003-profiles

## Command Structure

```
brains init [--global]
brains profile compose <profiles...> [--format json]
brains profile list [--format json]
brains profile show <name> [--raw] [--format json]
brains profile create <name> [--global]
brains profile validate [--format json]
```

## Commands

### brains init

Initialize `.brains/` directory structure.

**Arguments**: None

**Flags**:
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--global` | bool | false | Create in `~/.brains/` instead of current directory |

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Directory creation failed (permissions) |

**Output (success)**:
```
Initialized .brains/profiles/ in /path/to/project
```

**Output (already exists)**:
```
.brains/ already exists in /path/to/project
```

---

### brains profile compose

Compose one or more profiles into merged output.

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| `profiles` | Yes | Comma-separated or space-separated profile names |

**Flags**:
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | "text" | Output format: "text" or "json" |

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Profile not found |
| 2 | Circular dependency detected |
| 3 | Parse error in profile |

**Output (text)**:
```
<raw concatenated profile content>
```

**Output (json)**:
```json
{
  "content": "<concatenated content>",
  "profiles_used": ["base", "database", "security"],
  "character_count": 2450,
  "estimated_tokens": 612,
  "warnings": [],
  "resolution": [
    {"name": "base", "source": "global", "path": "~/.brains/profiles/base.md"},
    {"name": "database", "source": "local", "path": ".brains/profiles/database.md"},
    {"name": "security", "source": "local", "path": ".brains/profiles/security.md"}
  ]
}
```

**Error Output**:
```json
{
  "error": {
    "code": "PROFILE_NOT_FOUND",
    "message": "Profile 'databse' not found",
    "suggestions": ["database", "data-model"]
  }
}
```

---

### brains profile list

List all available profiles from all sources.

**Arguments**: None

**Flags**:
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | "text" | Output format: "text" or "json" |

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success (even if no profiles found) |

**Output (text)**:
```
PROFILE          SOURCE    DESCRIPTION
database         local     SQL and schema design guidance
security         local     Security best practices
base             global    Base system prompt
coding           global    General coding assistant
```

**Output (json)**:
```json
{
  "profiles": [
    {
      "name": "database",
      "source": "local",
      "path": ".brains/profiles/database.md",
      "description": "SQL and schema design guidance",
      "includes": ["sql-basics"],
      "inherits": true
    },
    {
      "name": "security",
      "source": "local",
      "path": ".brains/profiles/security.md",
      "description": "Security best practices",
      "includes": [],
      "inherits": true
    }
  ]
}
```

---

### brains profile show

Display a single profile's content.

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| `name` | Yes | Profile name to show |

**Flags**:
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--raw` | bool | false | Show raw file content without inheritance |
| `--format` | string | "text" | Output format: "text" or "json" |

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Profile not found |

**Output (text, default)**:
```
<resolved content with inherited content prepended>
```

**Output (text, --raw)**:
```
---
name: database
includes: [sql-basics]
---

You are a database expert...
```

**Output (json)**:
```json
{
  "name": "database",
  "source": "local",
  "path": ".brains/profiles/database.md",
  "description": "SQL and schema design guidance",
  "includes": ["sql-basics"],
  "inherits": true,
  "content": "<resolved content>",
  "raw_content": "<original file content>",
  "inherited_from": [
    {"source": "global", "path": "~/.brains/profiles/database.md"}
  ]
}
```

---

### brains profile create

Create a new profile with template frontmatter.

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| `name` | Yes | Profile name (will be normalized) |

**Flags**:
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--global` | bool | false | Create in `~/.brains/profiles/` instead |

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Profile already exists |
| 2 | Directory doesn't exist (run `brains init` first) |

**Output (success)**:
```
Created profile: .brains/profiles/my-profile.md
```

**Template Content**:
```markdown
---
name: my-profile
description:
includes: []
inherits: true
---

# My Profile

Add your profile content here.
```

---

### brains profile validate

Validate all profiles for errors.

**Arguments**: None

**Flags**:
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | "text" | Output format: "text" or "json" |

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | All profiles valid |
| 1 | Validation errors found |

**Output (text, success)**:
```
✓ All 5 profiles validated successfully
```

**Output (text, errors)**:
```
✗ Validation failed with 2 errors:

  database: includes non-existent profile 'sql-bsics' (did you mean 'sql-basics'?)
  security -> auth -> security: circular dependency detected
```

**Output (json)**:
```json
{
  "valid": false,
  "profiles_checked": 5,
  "errors": [
    {
      "profile": "database",
      "code": "MISSING_INCLUDE",
      "message": "includes non-existent profile 'sql-bsics'",
      "suggestions": ["sql-basics"]
    },
    {
      "profile": "security",
      "code": "CIRCULAR_DEPENDENCY",
      "message": "circular dependency detected",
      "cycle": ["security", "auth", "security"]
    }
  ]
}
```

## Error Codes

| Code | Description |
|------|-------------|
| `PROFILE_NOT_FOUND` | Requested profile does not exist |
| `MISSING_INCLUDE` | Profile includes non-existent profile |
| `CIRCULAR_DEPENDENCY` | Cycle detected in includes |
| `PARSE_ERROR` | Invalid YAML frontmatter |
| `PERMISSION_ERROR` | Cannot read/write file |
| `NOT_INITIALIZED` | .brains/ directory not found |
