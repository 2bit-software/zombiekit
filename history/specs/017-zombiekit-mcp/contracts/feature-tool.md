# MCP Tool Contract: feature

**Feature**: 017-zombiekit-mcp
**Date**: 2025-12-23

## Tool Definition

```json
{
  "name": "feature",
  "description": "Returns the contents of the step feature template file (~/.brains/templates/step.feature.md)"
}
```

## Input Schema

No parameters required for MVP.

```json
{
  "type": "object",
  "properties": {},
  "required": []
}
```

## Output Schema

### Success Response

```json
{
  "type": "text",
  "content": "<file contents as string>"
}
```

### Error Response

```json
{
  "type": "text",
  "content": "Error: <reason> (path: <resolved-path>)",
  "isError": true
}
```

## Examples

### Success Case

**Request**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "feature",
    "arguments": {}
  }
}
```

**Response**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "# Step Feature Template\n\n## Given...\n..."
    }
  ]
}
```

### Error Case (file not found)

**Request**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "feature",
    "arguments": {}
  }
}
```

**Response**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "Error: file not found (path: /Users/example/.brains/templates/step.feature.md)"
    }
  ],
  "isError": true
}
```

## MCP Discovery

The tool appears in `tools/list` response:

```json
{
  "tools": [
    {
      "name": "feature",
      "description": "Returns the contents of the step feature template file (~/.brains/templates/step.feature.md)",
      "inputSchema": {
        "type": "object",
        "properties": {},
        "required": []
      }
    }
  ]
}
```
