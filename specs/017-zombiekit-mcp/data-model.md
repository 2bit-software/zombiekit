# Data Model: ZombieKit MCP Tool

**Feature**: 017-zombiekit-mcp
**Date**: 2025-12-23

## Entities

### Tool (runtime)

The ZombieKit Tool struct manages the feature tool functionality.

| Field | Type | Description |
|-------|------|-------------|
| (none) | - | Stateless tool; no fields needed for MVP |

### ToolDefinition (configuration)

Describes the tool for MCP discovery.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | `"feature"` |
| Description | string | Tool description for MCP clients |

### ToolResponse (output)

Response structure returned by Execute().

| Field | Type | Description |
|-------|------|-------------|
| content | string | File contents (success) |
| error | string | Error message (failure) |
| path | string | Resolved file path (for debugging) |

## State Transitions

N/A - This is a stateless tool. Each invocation is independent.

## Validation Rules

1. **Home directory resolution**: Must successfully resolve `~` to user home directory
2. **File existence**: Target file must exist at resolved path
3. **File readability**: User must have read permissions on target file

## Relationships

```text
Server (1) ──registers──> (1) ZombieKit Tool
                               │
                               │ reads
                               ▼
                          Template File
                    (~/.brains/templates/step.feature.md)
```
