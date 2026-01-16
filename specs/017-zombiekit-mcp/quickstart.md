# Quickstart: ZombieKit Feature Tool

**Feature**: 017-zombiekit-mcp
**Date**: 2025-12-23

## Prerequisites

1. Go 1.24+ installed
2. brains CLI built and available
3. Template file exists at `~/.brains/templates/step.feature.md`

## Build

```bash
go build -o bin/brains ./cmd/brains
```

## Run MCP Server

```bash
# Start with stdio transport (default)
./bin/brains serve

# Or with specific transport
./bin/brains serve --transport stdio
./bin/brains serve --transport sse --port 8080
```

## Test the Feature Tool

### Using MCP Inspector

```bash
npx @anthropic-ai/mcp-inspector ./bin/brains serve
```

Then invoke the `feature` tool from the inspector UI.

### Direct JSON-RPC (stdio)

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"feature","arguments":{}}}' | ./bin/brains serve
```

## Enable/Disable Tool

```bash
# Disable feature tool
./bin/brains serve --disable-tool feature

# Enable only feature tool
./bin/brains serve --disable-tool stickymemory --disable-tool code-reasoning
```

## Verify Installation

1. Start the server
2. List tools - verify `feature` appears
3. Call `feature` tool - verify template contents returned
4. Delete template file - verify error message is descriptive
