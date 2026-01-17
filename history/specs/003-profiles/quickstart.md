# Quickstart: Profile Composition System

**Date**: 2025-12-22
**Feature**: 003-profiles

## Prerequisites

- Go 1.24+
- `brains` CLI installed (`go install ./cmd/brains`)

## Setup

### 1. Initialize profiles in your project

```bash
cd /path/to/your/project

# Create local .brains/profiles/ directory
brains init

# Optionally, also set up global profiles
brains init --global
```

### 2. Create your first profile

```bash
# Create a new profile
brains profile create database

# This creates .brains/profiles/database.md with template content
```

Edit `.brains/profiles/database.md`:

```markdown
---
name: database
description: SQL and database design expertise
includes: []
inherits: true
---

You are an expert database architect. You provide guidance on:

- Schema design and normalization
- Query optimization
- Index strategies
- Data modeling best practices

Always consider performance implications and suggest appropriate indexes.
```

### 3. Create a profile that includes others

Create `.brains/profiles/full-stack.md`:

```markdown
---
name: full-stack
description: Full stack development guidance
includes:
  - database
  - security
---

You are a full-stack developer. In addition to the included specialties,
you also understand frontend frameworks and API design.
```

## Usage

### List available profiles

```bash
# Human-readable
brains profile list

# JSON format
brains profile list --format json
```

### Show a profile

```bash
# Show with inheritance resolved
brains profile show database

# Show raw file content
brains profile show database --raw
```

### Compose profiles

```bash
# Compose multiple profiles
brains profile compose database,security

# Or with spaces
brains profile compose database security

# Get JSON output with metadata
brains profile compose database security --format json
```

### Validate profiles

```bash
# Check for errors
brains profile validate
```

## Profile Hierarchy

Profiles are resolved from multiple locations with precedence:

1. **Local** (highest): `.brains/profiles/` in current project
2. **Parent directories**: Any `.brains/profiles/` walking up to git root
3. **Global** (lowest): `~/.brains/profiles/`

When the same profile name exists in multiple locations, the local version wins.

### Inheritance

With `inherits: true` (the default), content from the same-named profile in parent directories is prepended:

```
~/.brains/profiles/database.md      ← Global base content
/project/.brains/profiles/database.md  ← Project-specific additions
```

Composed output: Global content first, then local content appended.

## MCP Integration

When running the MCP server, profile tools are available:

```bash
brains serve
```

Tools available to AI clients:
- `profile-compose` - Compose profiles into merged content
- `profile-list` - List available profiles
- `profile-show` - Show a specific profile
- `profile-validate` - Validate configuration

## Common Patterns

### Base + Specialty Pattern

Create a global base profile, then specialize per-project:

```bash
# Global: ~/.brains/profiles/base.md
---
name: base
---
You are a helpful coding assistant. Be concise and accurate.

# Project: .brains/profiles/base.md (inherits global)
---
name: base
inherits: true
---
This project uses Go 1.24 and follows standard Go conventions.
```

### Composition Pattern

Create focused profiles and compose them as needed:

```bash
# Small, focused profiles
.brains/profiles/sql-basics.md
.brains/profiles/security.md
.brains/profiles/testing.md

# Compose for specific tasks
brains profile compose sql-basics,security  # For database security review
brains profile compose testing,security     # For security test writing
```

## Troubleshooting

### Profile not found

```bash
# Check what's available
brains profile list

# Validate for typos in includes
brains profile validate
```

### Circular dependency

```
✗ security -> auth -> security: circular dependency detected
```

Remove the circular reference in your `includes` fields.

### Global profiles not loading

```bash
# Ensure global directory exists
brains init --global

# Check global profiles
ls ~/.brains/profiles/
```
