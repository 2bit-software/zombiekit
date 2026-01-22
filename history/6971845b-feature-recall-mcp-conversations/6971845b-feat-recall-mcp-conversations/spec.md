# Feature Specification: Recall MCP Conversations

**Feature Branch**: `6971845b-feature-recall-mcp-conversations`
**Created**: 2026-01-21
**Status**: Draft
**Input**: MCP interface for listing and reading recent conversations through the recall plugin with pagination support

## User Scenarios & Testing

### User Story 1 - List Recent Conversations (Priority: P1)

As an LLM agent, I want to list recent conversations so I can identify which conversations to audit.

**Why this priority**: Foundation for all auditing workflows - without a list, you can't select what to read.

**Independent Test**: Can be fully tested by calling the tool and verifying it returns conversation summaries ordered by date.

**Acceptance Scenarios**:

1. **Given** conversations exist in the database, **When** I call `recall-list-conversations` with no parameters, **Then** I receive up to 20 conversations ordered by last message descending (latest first)
2. **Given** more than 20 conversations exist, **When** I call with `page=2`, **Then** I receive the next page of results with `has_more` indicating if more exist
3. **Given** conversations exist for multiple projects, **When** I call with `project=/path/to/project`, **Then** I only receive conversations where CWD starts with that prefix

---

### User Story 2 - Read Conversation Incrementally (Priority: P1)

As an LLM agent, I want to read through a conversation in pages so I can process it without overwhelming my context window.

**Why this priority**: Core functionality - the purpose is incremental consumption for auditing.

**Independent Test**: Can be fully tested by calling with a conversation ID and verifying paginated chunk retrieval.

**Acceptance Scenarios**:

1. **Given** a conversation with 50 chunks, **When** I call `recall-read-conversation` with `conversation_id=X` and `limit=10`, **Then** I receive the first 10 chunks in chronological order with `has_more=true`
2. **Given** a conversation with 50 chunks and I'm on page 5, **When** I call with `page=5, limit=10`, **Then** I receive chunks 41-50 with `has_more=false`
3. **Given** a non-existent conversation ID, **When** I call `recall-read-conversation`, **Then** I receive an error: `{"error": "conversation not found"}`

---

### Edge Cases

- No conversations exist → empty `items` array, `has_more=false`
- Page exceeds available data → empty `items` array, `has_more=false`
- Limit exceeds maximum (100) → silently cap at 100
- Limit=0 or negative → use default (20)
- Page < 1 → treat as page 1
- Invalid conversation_id (non-UUID) → validation error
- Conversation ID not found → `{"error": "conversation not found"}`

## Requirements

### Functional Requirements

- **FR-001**: System MUST provide MCP tool `recall-list-conversations` to list conversation summaries
- **FR-002**: System MUST provide MCP tool `recall-read-conversation` to read conversation chunks
- **FR-003**: List tool MUST return conversations ordered by last message descending (latest first)
- **FR-004**: Both tools MUST support pagination via `page` (1-indexed) and `limit` parameters
- **FR-005**: Both tools MUST return `has_more` boolean indicating additional pages exist
- **FR-006**: List tool MUST optionally filter by `project` path prefix (substring match: `/foo` matches `/foo/bar` but not `/foobar`)
- **FR-007**: Read tool MUST return chunks in chronological order (oldest first, secondary sort by chunk ID for identical timestamps)
- **FR-008**: Read tool MUST include chunk metadata: role, timestamp (ISO-8601), project (CWD), git_branch
- **FR-009**: System MUST enforce maximum limit of 100 items per page (silently cap, don't error)
- **FR-010**: System MUST default to limit=20 when not specified or invalid (0, negative)

### Key Entities

**ConversationSummary** (list tool response item):
```json
{
  "conversation_id": "uuid-string",
  "title": "First user message (truncated to 100 chars)",
  "message_count": 42,
  "first_message": "2026-01-15T10:30:00Z",
  "last_message": "2026-01-15T11:45:00Z",
  "source": "claude",
  "project": "/path/to/project"
}
```

**ConversationChunk** (read tool response item):
```json
{
  "id": "chunk-uuid",
  "content": "The actual message content...",
  "role": "user",
  "timestamp": "2026-01-15T10:30:00Z",
  "project": "/path/to/project",
  "git_branch": "main"
}
```

### MCP Tool Definitions

**Tool 1: `recall-list-conversations`**

| Parameter | Type | Required | Default | Validation |
|-----------|------|----------|---------|------------|
| page | integer | No | 1 | Must be >= 1 |
| limit | integer | No | 20 | Capped at 100 |
| project | string | No | "" | Empty returns all |

**Response**:
```json
{
  "page": 1,
  "limit": 20,
  "has_more": true,
  "items": [ConversationSummary, ...]
}
```

**Tool 2: `recall-read-conversation`**

| Parameter | Type | Required | Default | Validation |
|-----------|------|----------|---------|------------|
| conversation_id | string | Yes | - | Must be valid UUID |
| page | integer | No | 1 | Must be >= 1 |
| limit | integer | No | 20 | Capped at 100 |

**Response**:
```json
{
  "conversation_id": "uuid",
  "page": 1,
  "limit": 20,
  "has_more": true,
  "items": [ConversationChunk, ...]
}
```

**Error Response** (both tools):
```json
{
  "error": "descriptive error message"
}
```

### Implementation Notes

**Pagination Algorithm**:
```
offset = (page - 1) * limit
```

**has_more Detection**:
```
1. Fetch (limit + 1) items from storage
2. If returned count > limit:
   - Set has_more = true
   - Return only first limit items
3. Otherwise:
   - Set has_more = false
   - Return all items
```

**Storage Interface Extension Required**:
The current `GetByConversation(ctx, conversationID)` returns all chunks unbounded. A new method is needed:
```go
GetConversationChunks(ctx context.Context, conversationID string, limit, offset int) ([]Chunk, error)
```

## Success Criteria

### Measurable Outcomes

- **SC-001**: Both tools return valid JSON parseable by any LLM
- **SC-002**: Pagination metadata is accurate (has_more reflects actual remaining data)
- **SC-003**: LLM can page through a 100-chunk conversation in 10 calls with limit=10
- **SC-004**: Project filter correctly filters by CWD prefix

## Testing Requirements

### Test Strategy

Integration tests at the MCP tool level using PostgreSQL test infrastructure with seeded conversation data:
- 3+ conversations across 2+ projects
- Varying chunk counts (5, 25, 100+ chunks)
- Known timestamps for ordering verification

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | List tool returns ConversationSummary array |
| FR-002 | Integration | Read tool returns Chunk array for valid conversation ID |
| FR-003 | Integration | Verify ordering is last_message descending |
| FR-004 | Integration | Verify page/limit parameters affect results correctly |
| FR-005 | Integration | Verify has_more is true when more data exists, false otherwise |
| FR-006 | Integration | Verify project filter only returns matching conversations (prefix match) |
| FR-007 | Integration | Verify chunk ordering is chronological, then by ID |
| FR-008 | Integration | Verify metadata fields present and populated (ISO-8601 timestamps) |
| FR-009 | Integration | Verify limit > 100 is silently capped at 100 |
| FR-010 | Integration | Verify default limit is 20 when omitted or invalid |

### Error Case Tests

| Scenario | Expected Response |
|----------|-------------------|
| Invalid conversation_id (not UUID) | `{"error": "invalid conversation_id format"}` |
| Conversation not found | `{"error": "conversation not found"}` |
| No conversations exist | `{"page":1,"limit":20,"has_more":false,"items":[]}` |
| Page beyond data | Empty items array, has_more=false |

### Edge Case Coverage

- No conversations exist → empty array, has_more=false
- Page beyond data → empty array, has_more=false
- Limit=0 → use default (20)
- Negative page → treat as page 1
- Identical timestamps → secondary sort by chunk_id (deterministic)
