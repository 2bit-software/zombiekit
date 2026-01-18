# Technical Requirements Research

**Feature**: Claude Conversation Importer
**Linear Ticket**: DEV-69

---

## User-Specified Technical Preferences

These preferences were explicitly stated by the user and should be honored:

### 1. CLI-Only Interface (No MCP)

> "I want this to start up when called as a specific command from the CLI. There should be NO mcp interface for this."

**Implication**: Implement as CLI subcommand only. Do not create MCP tools for this functionality.

### 2. Command Structure: Bucketed Under "watch"

> "We should bucket these 'import' commands, so like `brains recall watch claude` for example, so we can support 'watching/importing' from other places as well when we want to in the future"

**Implication**:
- Parent command: `brains recall watch`
- Subcommand for this feature: `brains recall watch claude`
- Future extensibility: `brains recall watch <source>` pattern (e.g., `brains recall watch slack`, `brains recall watch notion`)

### 3. Command Semantics

The term "watch" suggests:
- Long-running process that monitors for changes
- Automatic import when new content detected
- Could also support one-shot import mode (e.g., `--once` flag)

---

## Technical Stack (from DEV-69)

- PostgreSQL with pgvector (already implemented in DEV-72)
- Ollama instance for embeddings (already implemented in DEV-72)
- Existing `recall` package for storage and embedding

---

## Research Questions

1. What is the exact format of Claude Code history files?
2. Where are they located (global vs project-specific)?
3. How to detect new conversations efficiently?
4. What chunking strategy for conversation messages?
5. How to handle file locking during active Claude sessions?

---

## Dependencies

- DEV-72: RAG Core Infrastructure & CLI (COMPLETED - provides recall save/list/search)
- Existing `internal/recall` package
- Existing `internal/cli` patterns
