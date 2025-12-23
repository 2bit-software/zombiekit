# MCP Tools Contract: Profile Tools

**Date**: 2025-12-22
**Feature**: 003-profiles

## Tools Overview

| Tool Name | Description |
|-----------|-------------|
| `profile-compose` | Compose one or more profiles into merged content |
| `profile-list` | List all available profiles |
| `profile-show` | Show a single profile's content |
| `profile-validate` | Validate profile configuration |

## Tool Definitions

### profile-compose

Compose one or more profiles into merged content.

**Schema**:
```json
{
  "name": "profile-compose",
  "description": "Compose one or more profiles into merged prompt content. Profiles are resolved from local (.brains/profiles/) and global (~/.brains/profiles/) directories with local taking precedence.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "profiles": {
        "type": "array",
        "items": { "type": "string" },
        "description": "List of profile names to compose",
        "minItems": 1
      },
      "working_directory": {
        "type": "string",
        "description": "Working directory for profile resolution (defaults to CWD)"
      }
    },
    "required": ["profiles"]
  }
}
```

**Success Response**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "<composed profile content>"
    }
  ]
}
```

**Error Response**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "Error: Profile 'databse' not found. Did you mean 'database'?"
    }
  ],
  "isError": true
}
```

---

### profile-list

List all available profiles from all sources.

**Schema**:
```json
{
  "name": "profile-list",
  "description": "List all available profiles from local and global .brains/profiles/ directories.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "working_directory": {
        "type": "string",
        "description": "Working directory for profile resolution (defaults to CWD)"
      }
    }
  }
}
```

**Success Response**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "Available profiles:\n\n- database (local): SQL and schema design guidance\n- security (local): Security best practices\n- base (global): Base system prompt"
    }
  ]
}
```

---

### profile-show

Show a single profile's content.

**Schema**:
```json
{
  "name": "profile-show",
  "description": "Show the content of a specific profile, with inheritance resolved by default.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "description": "Name of the profile to show"
      },
      "raw": {
        "type": "boolean",
        "description": "If true, show raw file content without inheritance resolution",
        "default": false
      },
      "working_directory": {
        "type": "string",
        "description": "Working directory for profile resolution (defaults to CWD)"
      }
    },
    "required": ["name"]
  }
}
```

**Success Response**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "<profile content with inheritance resolved>"
    }
  ]
}
```

**Error Response**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "Error: Profile 'databse' not found. Did you mean 'database'?"
    }
  ],
  "isError": true
}
```

---

### profile-validate

Validate all profiles for errors.

**Schema**:
```json
{
  "name": "profile-validate",
  "description": "Validate all profiles for errors like circular dependencies or missing references.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "working_directory": {
        "type": "string",
        "description": "Working directory for profile resolution (defaults to CWD)"
      }
    }
  }
}
```

**Success Response (all valid)**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "All 5 profiles validated successfully."
    }
  ]
}
```

**Success Response (errors found)**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "Validation failed with 2 errors:\n\n- database: includes non-existent profile 'sql-bsics' (did you mean 'sql-basics'?)\n- security -> auth -> security: circular dependency detected"
    }
  ]
}
```

## Registration in MCP Server

The tools will be registered in `internal/mcp/server.go`:

```go
// In NewServer, after existing tool registrations:

profileTool := profile.NewTool()
s.registerProfileTools(profileTool)
```

```go
func (s *Server) registerProfileTools(pt *profile.Tool) {
    // profile-compose
    composeTool := mcp.NewTool("profile-compose",
        mcp.WithDescription("Compose one or more profiles into merged prompt content"),
        mcp.WithArray("profiles",
            mcp.Required(),
            mcp.Description("List of profile names to compose"),
            mcp.Items(map[string]interface{}{"type": "string"}),
        ),
        mcp.WithString("working_directory",
            mcp.Description("Working directory for profile resolution"),
        ),
    )
    s.mcpServer.AddTool(composeTool, s.handleProfileCompose)

    // profile-list
    listTool := mcp.NewTool("profile-list",
        mcp.WithDescription("List all available profiles"),
        mcp.WithString("working_directory",
            mcp.Description("Working directory for profile resolution"),
        ),
    )
    s.mcpServer.AddTool(listTool, s.handleProfileList)

    // profile-show
    showTool := mcp.NewTool("profile-show",
        mcp.WithDescription("Show a specific profile's content"),
        mcp.WithString("name",
            mcp.Required(),
            mcp.Description("Name of the profile to show"),
        ),
        mcp.WithBoolean("raw",
            mcp.Description("Show raw content without inheritance"),
        ),
        mcp.WithString("working_directory",
            mcp.Description("Working directory for profile resolution"),
        ),
    )
    s.mcpServer.AddTool(showTool, s.handleProfileShow)

    // profile-validate
    validateTool := mcp.NewTool("profile-validate",
        mcp.WithDescription("Validate profile configuration"),
        mcp.WithString("working_directory",
            mcp.Description("Working directory for profile resolution"),
        ),
    )
    s.mcpServer.AddTool(validateTool, s.handleProfileValidate)
}
```

## Error Handling

All MCP tools return errors via `mcp.NewToolResultError()`:

```go
func (s *Server) handleProfileCompose(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    args, ok := req.Params.Arguments.(map[string]interface{})
    if !ok {
        return mcp.NewToolResultError("invalid arguments format"), nil
    }

    profiles, ok := args["profiles"].([]interface{})
    if !ok || len(profiles) == 0 {
        return mcp.NewToolResultError("profiles array is required"), nil
    }

    // Convert to []string and call service
    result, err := s.profileService.Compose(ctx, profileNames, workingDir)
    if err != nil {
        return mcp.NewToolResultError(formatError(err)), nil
    }

    return mcp.NewToolResultText(result.Content), nil
}
```

## Working Directory Handling

When `working_directory` is not specified:
1. Use `os.Getwd()` as default
2. MCP server may be running from a different directory than the user's project
3. AI clients should pass the user's project directory when known
