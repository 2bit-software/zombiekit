## Problem

The zombiekit MCP server currently operates against a single implicit working directory. When working across multiple repositories or subdirectories, there's no way to specify which directory a git operation should target.

## Proposal

Add an optional `directory` parameter to the MCP server that allows callers to specify the working directory for git operations. When omitted, behavior stays the same (uses the default/current directory).