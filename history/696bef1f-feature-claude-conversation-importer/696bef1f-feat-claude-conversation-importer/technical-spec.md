# Technical Specification: Claude Conversation Importer

**Feature**: conversation-importer
**Linear Ticket**: DEV-69

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Command                              │
│                   brains recall watch claude                     │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Import Coordinator                          │
│  - Discovery: Find .jsonl files                                  │
│  - Polling: Check for changes                                    │
│  - Progress: Track and report                                    │
└─────────────────────────────────────────────────────────────────┘
                                │
          ┌─────────────────────┼─────────────────────┐
          ▼                     ▼                     ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Claude Parser  │  │    Chunker      │  │    Embedder     │
│  - Parse JSONL  │  │  - Split large  │  │  - Ollama API   │
│  - Filter types │  │    messages     │  │  - nomic-embed  │
│  - Extract text │  │  - 8000 char    │  │                 │
└─────────────────┘  └─────────────────┘  └─────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Recall Storage                              │
│  - PostgreSQL + pgvector                                         │
│  - SaveWithSource (duplicate detection by source_id)             │
│  - GetByConversation (conversation retrieval)                    │
└─────────────────────────────────────────────────────────────────┘
```

---

## Database Schema

### Migration 003: Source Tracking Columns

```sql
-- Add source tracking columns
ALTER TABLE recall_chunks
  ADD COLUMN IF NOT EXISTS source TEXT,
  ADD COLUMN IF NOT EXISTS source_id TEXT,
  ADD COLUMN IF NOT EXISTS conversation_id TEXT,
  ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Unique index for duplicate detection: (source, source_id)
-- Allows same source_id from different sources
CREATE UNIQUE INDEX IF NOT EXISTS idx_recall_chunks_source_id
  ON recall_chunks(source, source_id)
  WHERE source_id IS NOT NULL;

-- Index for conversation retrieval
CREATE INDEX IF NOT EXISTS idx_recall_chunks_conversation
  ON recall_chunks(conversation_id)
  WHERE conversation_id IS NOT NULL;

-- Index for source filtering
CREATE INDEX IF NOT EXISTS idx_recall_chunks_source
  ON recall_chunks(source)
  WHERE source IS NOT NULL;
```

### Column Definitions

| Column | Type | Description |
|--------|------|-------------|
| source | TEXT | Source identifier: "claude", "slack", etc. |
| source_id | TEXT | Original message UUID from source |
| conversation_id | TEXT | Groups messages into conversations |
| metadata | JSONB | Source-specific data (role, timestamp, etc.) |

---

## Type Definitions

### Chunk (Extended)

```go
// internal/recall/types.go

type Chunk struct {
    ID             string    `json:"id"`
    Content        string    `json:"content"`
    CreatedAt      time.Time `json:"created_at"`
    Source         string    `json:"source,omitempty"`
    SourceID       string    `json:"source_id,omitempty"`
    ConversationID string    `json:"conversation_id,omitempty"`
    Metadata       *Metadata `json:"metadata,omitempty"`
}

type Metadata struct {
    Role      string    `json:"role,omitempty"`       // "user" or "assistant"
    Timestamp time.Time `json:"timestamp,omitempty"`  // Original message timestamp
    GitBranch string    `json:"git_branch,omitempty"` // Git branch at time of message
    CWD       string    `json:"cwd,omitempty"`        // Working directory
    ParentID  string    `json:"parent_id,omitempty"`  // Parent message UUID for threading
}

type ChunkInput struct {
    Content        string
    Source         string
    SourceID       string
    ConversationID string
    Metadata       *Metadata
}
```

### Claude History Types

```go
// internal/recall/claude/types.go

type HistoryEntry struct {
    Type        string          `json:"type"`        // "user", "assistant", "summary", etc.
    UUID        string          `json:"uuid"`        // Unique message ID
    ParentUUID  *string         `json:"parentUuid"`  // Parent message for threading
    SessionID   string          `json:"sessionId"`   // Conversation/session ID
    Timestamp   time.Time       `json:"timestamp"`   // Message timestamp
    Message     *MessageContent `json:"message,omitempty"`
    IsMeta      bool            `json:"isMeta"`      // Skip if true
    IsSidechain bool            `json:"isSidechain"` // Alternate branch, may skip
    CWD         string          `json:"cwd,omitempty"`
    GitBranch   string          `json:"gitBranch,omitempty"`
    Version     string          `json:"version,omitempty"`
}

type MessageContent struct {
    Role    string      `json:"role"`    // "user" or "assistant"
    Content interface{} `json:"content"` // string OR []ContentBlock
}

type ContentBlock struct {
    Type string `json:"type"` // "text", "tool_use", etc.
    Text string `json:"text,omitempty"`
}
```

---

## Storage Interface Extension

```go
// internal/recall/storage.go

type Storage interface {
    // Existing methods
    Save(ctx context.Context, content string, embedding []float32) (id string, created bool, err error)
    List(ctx context.Context, limit int) ([]Chunk, error)
    Search(ctx context.Context, embedding []float32, limit int) ([]SearchResult, error)
    Close() error

    // New methods for source tracking
    SaveWithSource(ctx context.Context, input ChunkInput, embedding []float32) (id string, created bool, err error)
    ExistsBySourceID(ctx context.Context, source, sourceID string) (bool, error)
    GetByConversation(ctx context.Context, conversationID string) ([]Chunk, error)
}
```

### Implementation Notes

**ExistsBySourceID** (called first, before embedding):
- Fast lookup using the `idx_recall_chunks_source_id` index
- Returns true if exact (source, source_id) pair exists
- **Purpose**: Short-circuit expensive embedding generation for duplicates

**SaveWithSource**:
- Uses `ON CONFLICT (source, source_id) DO NOTHING` as safety net
- Falls back to content_hash if source_id is empty
- Returns `created=false` if duplicate (race condition handling)

**GetByConversation**:
- Returns all chunks with matching conversation_id
- Ordered by metadata->timestamp ASC for conversation flow

---

## Claude Parser

### File Discovery

```go
// internal/recall/claude/discovery.go

const DefaultClaudePath = "~/.claude"

// DiscoverHistoryFiles finds all JSONL history files
func DiscoverHistoryFiles(claudePath string) ([]string, error) {
    projectsDir := filepath.Join(claudePath, "projects")
    var files []string

    err := filepath.WalkDir(projectsDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil { return err }
        if !d.IsDir() && strings.HasSuffix(path, ".jsonl") {
            files = append(files, path)
        }
        return nil
    })
    return files, err
}

// DiscoverProjectFiles finds history files for a specific project
func DiscoverProjectFiles(claudePath, projectPath string) ([]string, error) {
    encoded := EncodeProjectPath(projectPath)
    projectDir := filepath.Join(claudePath, "projects", encoded)
    // ... return .jsonl files in that directory
}

// EncodeProjectPath converts /Users/foo/bar to -Users-foo-bar
func EncodeProjectPath(path string) string {
    return strings.ReplaceAll(path, "/", "-")
}
```

### JSONL Parser

```go
// internal/recall/claude/parser.go

// ParseFile reads a JSONL file and returns history entries
func ParseFile(path string) ([]HistoryEntry, error) {
    file, err := os.Open(path)
    if err != nil { return nil, err }
    defer file.Close()

    var entries []HistoryEntry
    scanner := bufio.NewScanner(file)
    scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line

    for scanner.Scan() {
        line := scanner.Bytes()
        if len(line) == 0 { continue }

        var entry HistoryEntry
        if err := json.Unmarshal(line, &entry); err != nil {
            // Skip malformed lines (graceful degradation)
            continue
        }
        entries = append(entries, entry)
    }
    return entries, scanner.Err()
}

// FilterImportable returns only user/assistant messages that aren't meta.
// Note: isSidechain messages ARE imported (see Decision 7 in highlights.md).
func FilterImportable(entries []HistoryEntry) []HistoryEntry {
    var result []HistoryEntry
    for _, e := range entries {
        if e.IsMeta { continue }
        if e.Type != "user" && e.Type != "assistant" { continue }
        if e.Message == nil { continue }
        // Sidechain messages are included - they represent valid conversation branches
        result = append(result, e)
    }
    return result
}

// ExtractContent handles both string and []ContentBlock content
// See Content Block Handling table below for type-specific behavior.
func ExtractContent(entry HistoryEntry) string {
    if entry.Message == nil { return "" }

    switch c := entry.Message.Content.(type) {
    case string:
        return c
    case []interface{}:
        var texts []string
        for _, block := range c {
            if m, ok := block.(map[string]interface{}); ok {
                blockType, _ := m["type"].(string)
                switch blockType {
                case "text":
                    if t, ok := m["text"].(string); ok {
                        texts = append(texts, t)
                    }
                case "thinking":
                    // Include thinking blocks - valuable for search
                    if t, ok := m["thinking"].(string); ok {
                        texts = append(texts, t)
                    }
                // tool_use, tool_result: skip (not searchable prose)
                }
            }
        }
        return strings.Join(texts, "\n")
    default:
        return fmt.Sprintf("%v", c)
    }
}

/*
Content Block Handling:

| Block Type   | Handling                                      |
|--------------|-----------------------------------------------|
| text         | Extract text field → include in corpus        |
| thinking     | Extract thinking field → include (valuable)   |
| tool_use     | Skip (function calls, not searchable prose)   |
| tool_result  | Skip (structured output, not prose)           |
| image        | Skip (binary data, not searchable)            |
*/
```

---

## Chunker

```go
// internal/recall/claude/chunker.go

const MaxChunkSize = 8000

// ChunkMessage splits long messages at sentence boundaries
func ChunkMessage(content string) []string {
    if len(content) <= MaxChunkSize {
        return []string{content}
    }

    var chunks []string
    remaining := content

    for len(remaining) > MaxChunkSize {
        // Find last sentence boundary before MaxChunkSize
        cutPoint := findSentenceBoundary(remaining[:MaxChunkSize])
        if cutPoint == 0 {
            cutPoint = MaxChunkSize // Force cut if no boundary found
        }

        chunks = append(chunks, strings.TrimSpace(remaining[:cutPoint]))
        remaining = strings.TrimSpace(remaining[cutPoint:])
    }

    if len(remaining) > 0 {
        chunks = append(chunks, remaining)
    }

    return chunks
}

func findSentenceBoundary(text string) int {
    // Look for ". " or ".\n" from the end
    for i := len(text) - 1; i > 0; i-- {
        if text[i] == '.' && i+1 < len(text) && (text[i+1] == ' ' || text[i+1] == '\n') {
            return i + 1
        }
    }
    return 0
}
```

---

## CLI Command Structure

```go
// In internal/cli/recall.go

func newRecallCommand() *cli.Command {
    return &cli.Command{
        Name:  "recall",
        Usage: "Semantic memory storage and retrieval",
        Subcommands: []*cli.Command{
            // Existing: save, list, search
            {
                Name:  "watch",
                Usage: "Watch and import content from external sources",
                Subcommands: []*cli.Command{
                    {
                        Name:   "claude",
                        Usage:  "Import Claude Code conversation history",
                        Action: recallWatchClaudeAction,
                        Flags: []cli.Flag{
                            &cli.BoolFlag{
                                Name:  "once",
                                Usage: "Import once and exit (no continuous watch)",
                            },
                            &cli.StringFlag{
                                Name:  "path",
                                Usage: "Path to Claude config directory",
                                Value: defaultClaudePath(),
                            },
                            &cli.StringFlag{
                                Name:  "project",
                                Usage: "Filter to specific project path",
                            },
                            &cli.BoolFlag{
                                Name:    "verbose",
                                Aliases: []string{"v"},
                                Usage:   "Show detailed import progress",
                            },
                            &cli.DurationFlag{
                                Name:  "interval",
                                Usage: "Poll interval for watch mode",
                                Value: 30 * time.Second,
                            },
                        },
                    },
                },
            },
            {
                Name:      "conversation",
                Usage:     "View all messages in a conversation",
                ArgsUsage: "<conversation-id>",
                Action:    recallConversationAction,
            },
        },
    }
}
```

---

## Import Flow

```
1. Initialize
   ├── Load config
   ├── Connect to storage
   └── Connect to embedder

2. Discover Files
   ├── Find all .jsonl files in ~/.claude/projects/
   └── Filter by --project if specified

3. For Each File
   ├── Parse JSONL
   ├── Filter to user/assistant messages (skip isMeta)
   └── For each message:
       │
       ├── ExistsBySourceID? ──YES──► Skip (no embedding cost)
       │         │
       │        NO
       │         │
       │         ▼
       ├── Extract content
       ├── Chunk if > 8000 chars
       ├── For each chunk:
       │   ├── Generate embedding (expensive)
       │   └── SaveWithSource
       └── Increment counters (new/skipped)

4. Report Summary
   └── "Imported X new messages (Y already existed)"

5. Watch Mode (if not --once)
   ├── Wait for interval
   ├── Re-scan files for new content
   └── Loop to step 3
```

---

## Error Handling

| Scenario | Handling |
|----------|----------|
| Malformed JSON line | Skip line, continue parsing |
| File read error | Log warning, continue to next file |
| Embedding API error | Retry with backoff, fail after 3 attempts |
| Database error | Return error, abort import |
| Partial line at EOF | Skip (JSONL append pattern) |

---

## Progress Reporting

### Normal Mode
```
Importing Claude history...
  Scanning: ~/.claude/projects/-Users-morgan-Projects-foo/
  Found 5 history files
  Importing: abc123.jsonl (24 messages)
  Importing: def456.jsonl (156 messages)
Done. Imported 180 new messages (0 duplicates skipped).
```

### Verbose Mode (-v)
```
Importing Claude history...
  File: abc123.jsonl
    [user] How do I implement... (1,234 chars) → imported
    [assistant] You can use... (5,678 chars) → imported
    [user] Thanks! (12 chars) → already exists, skipped
  ...
Done. Imported 180 new messages (15 duplicates skipped).
```

---

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| BRAINS_BACKEND | - | Must be "postgres" |
| BRAINS_POSTGRES_URL | - | PostgreSQL connection string |
| BRAINS_OLLAMA_URL | http://localhost:11434 | Ollama API URL |
| BRAINS_EMBEDDING_MODEL | nomic-embed-text | Embedding model |

---

## Future Extensibility

The `watch` command structure supports future sources:

```
brains recall watch claude   # This feature
brains recall watch slack    # Future
brains recall watch notion   # Future
brains recall watch files    # Future: watch directory for text files
```

Each source implements its own parser and discovery logic, but shares:
- Storage interface (SaveWithSource)
- Embedding generation (OllamaEmbedder)
- CLI patterns (flags, progress)
