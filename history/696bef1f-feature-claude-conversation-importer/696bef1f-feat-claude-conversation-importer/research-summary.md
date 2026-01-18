# Research Summary: Claude Conversation Importer

**Feature**: conversation-importer
**Linear Ticket**: DEV-69

---

## Claude Code History File Format

### Location
- Project-specific: `~/.claude/projects/<encoded-path>/<session-id>.jsonl`
- Path encoding: Slashes replaced with dashes (e.g., `/Users/morgan/Projects/foo` → `-Users-morgan-Projects-foo`)
- Each session is a separate `.jsonl` file named by UUID

### File Structure
JSONL format - one JSON object per line. Key record types:

#### 1. User Messages
```json
{
  "type": "user",
  "uuid": "a1e68cfd-0ab5-4428-8b83-62f9b9c6d98b",
  "parentUuid": null,
  "sessionId": "c6ec5e1f-61e5-4644-a75c-8732dd2b840e",
  "timestamp": "2026-01-17T19:27:09.743Z",
  "cwd": "/Users/morgan/Projects/personal/zombiekit",
  "gitBranch": "main",
  "version": "2.1.8",
  "message": {
    "role": "user",
    "content": "the user's message here"
  },
  "isMeta": false,
  "isSidechain": false
}
```

#### 2. Assistant Messages
```json
{
  "type": "assistant",
  "uuid": "...",
  "parentUuid": "...",
  "sessionId": "...",
  "timestamp": "...",
  "message": {
    "role": "assistant",
    "content": "..." // can be string or array of content blocks
  }
}
```

#### 3. Summary Records
```json
{
  "type": "summary",
  "summary": "Brief description of conversation topic",
  "leafUuid": "..."
}
```

#### 4. File History Snapshots
```json
{
  "type": "file-history-snapshot",
  "messageId": "...",
  "snapshot": {...},
  "isSnapshotUpdate": false
}
```

### Key Fields
- `uuid`: Unique identifier for the message
- `parentUuid`: Links to parent message (for threading), null for conversation start
- `sessionId`: Groups messages into a conversation
- `timestamp`: ISO-8601 timestamp
- `isMeta`: True for system/meta messages (should skip)
- `isSidechain`: True for branched/alternate responses (import - see Decision 7)

### Filtering Strategy
- Import: `type === "user"` or `type === "assistant"` where `isMeta === false`
- **Sidechain messages**: Import (Decision 7) - valuable alternate conversation branches
- Use `summary` records for conversation-level metadata
- Skip `file-history-snapshot` and other internal records

---

## Existing Codebase Patterns

### Recall Package
- `recall.Storage` interface: `Save(ctx, content, embedding)`, `List`, `Search`, `Close`
- `recall.Embedder` interface: `Embed(ctx, text, purpose)`
- Postgres storage implementation with pgvector
- Duplicate detection via content hash (SHA-256)

### CLI Patterns
- Uses `urfave/cli/v2`
- Subcommands defined in `newRecallCommand()` → Subcommands slice
- Each subcommand has: Name, Usage, Flags, Action function
- Action signature: `func(c *cli.Context) error`

### Long-Running Command Pattern (from serve.go)
```go
done := make(chan os.Signal, 1)
signal.Notify(done, os.Interrupt, syscall.SIGTERM)

go func() {
    <-done
    // graceful shutdown
}()
```

### Helper Functions Available
- `getRecallStorage(ctx, cfg)` - returns postgres storage
- `getEmbedder(cfg)` - returns OllamaEmbedder
- `config.LoadStorageConfigFromEnv()` - loads config

---

## Proposed Command Structure

Based on user requirement for bucketed "watch" commands:

```
brains recall watch claude [flags]
```

Flags:
- `--once`: Import once and exit (vs continuous watch)
- `--path <dir>`: Override Claude history path (default: ~/.claude)
- `--project <path>`: Import only specific project's history

Future extensibility:
- `brains recall watch slack`
- `brains recall watch notion`
- etc.

---

## Technical Decisions Needed

1. **Chunking Strategy**: Import entire messages vs chunk long messages?
2. **Conversation Grouping**: Store conversation ID for context retrieval?
3. **Watch Mechanism**: fsnotify for file watching?
4. **Progress Tracking**: How to track what's already imported?

---

## Dependencies

- DEV-72: RAG Core Infrastructure (COMPLETED)
- `github.com/fsnotify/fsnotify` for file watching (needs to be added)
