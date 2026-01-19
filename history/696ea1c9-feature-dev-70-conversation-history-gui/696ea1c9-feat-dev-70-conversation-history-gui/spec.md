# Feature Specification: Conversation History Search GUI

**Feature Branch**: `morganhein/dev-70-conversation-history-search-gui-interface`
**Created**: 2026-01-19
**Status**: Draft
**Linear**: DEV-70

## User Scenarios & Testing

### User Story 1 - Search Past Conversations (Priority: P1)

A user wants to find a past conversation where they discussed a specific topic (e.g., "database migrations" or "authentication flow"). They enter a natural language query and see matching results with enough context to identify the right conversation.

**Why this priority**: Core value proposition - users need to find past context without remembering exact words or dates. This is the "what did we decide about X?" use case.

**Independent Test**: User searches for "authentication", sees list of matching conversations with snippets showing why each matched, clicks one to view full context.

**Acceptance Scenarios**:

1. **Given** the user is on the conversation history page, **When** they enter "database migrations" in the search box and press Enter, **Then** they see a list of conversations containing semantically related messages, each showing a snippet and similarity indicator.

2. **Given** search results are displayed, **When** the user clicks on a conversation, **Then** they navigate to the conversation detail view showing all messages in that conversation.

3. **Given** the user searches for a term with no matches, **When** results load, **Then** they see an empty state message "No matching conversations found."

---

### User Story 2 - Browse Recent Conversations (Priority: P1)

A user wants to see their recent conversation history without searching. They browse a chronologically ordered list and can page through older conversations.

**Why this priority**: Equal to search - browsing is the fallback when users don't know what to search for, and provides discovery of past work.

**Independent Test**: User navigates to conversation history, sees list of recent conversations with timestamps, clicks through to view any conversation.

**Acceptance Scenarios**:

1. **Given** the user navigates to the conversation history page, **When** the page loads, **Then** they see a paginated list of conversations ordered by most recent activity, showing title/summary, message count, and timestamp.

2. **Given** the list shows 20 conversations per page, **When** the user clicks "Next", **Then** the next 20 conversations load without full page refresh (HTMX).

3. **Given** a conversation in the list, **When** the user clicks it, **Then** they see the full conversation with all messages displayed chronologically.

---

### User Story 3 - View Conversation Detail (Priority: P1)

A user views a single conversation to read the full exchange, seeing messages attributed to user vs assistant with timestamps.

**Why this priority**: Essential complement to search/browse - without detail view, the other features are incomplete.

**Independent Test**: Given a conversation ID, user can view all messages in chronological order with role attribution and timestamps.

**Acceptance Scenarios**:

1. **Given** the user navigates to a conversation detail page, **When** the page loads, **Then** they see all messages in chronological order with role indicators (user/assistant), timestamps, and full content.

2. **Given** a long message (>500 chars), **When** displayed, **Then** it shows with proper formatting (whitespace preserved, code blocks if applicable).

3. **Given** a conversation with many messages, **When** viewing, **Then** the user can scroll through all messages (no pagination within conversation).

---

### User Story 4 - Filter by Project (Priority: P2)

A user wants to see only conversations from a specific project directory, filtering out conversations from other projects.

**Why this priority**: Important for power users with multiple projects, but core search/browse works without it.

**Independent Test**: User selects a project filter, list updates to show only conversations where working directory matches.

**Acceptance Scenarios**:

1. **Given** the user is on the conversation list, **When** they select a project from the filter dropdown, **Then** the list updates to show only conversations where messages originated from that project path.

2. **Given** a project filter is active, **When** the user searches, **Then** search results are scoped to that project only.

3. **Given** a project filter is active, **When** the user clears the filter, **Then** all conversations are shown again.

---

### User Story 5 - Global Search Integration (Priority: P3)

Conversation results appear in the global search bar, allowing users to find conversations from anywhere in the ZombieKit interface.

**Why this priority**: Convenience feature - users can already access dedicated search page. This is polish.

**Independent Test**: User types in global search bar, sees conversation results alongside memory/profile results.

**Acceptance Scenarios**:

1. **Given** the user types "authentication" in the global search bar, **When** results load, **Then** matching conversations appear under a "Conversations" section alongside other plugin results.

2. **Given** a conversation result in global search, **When** the user clicks it, **Then** they navigate to the conversation detail page.

---

### Edge Cases

- What happens when database connection fails? **Show error message, allow retry.**
- What happens when Ollama (embedder) is unavailable for search? **Show error: "Search unavailable - embedding service offline."**
- What happens with a conversation containing 0 messages? **Don't show in list (filter out empty conversations).**
- What happens with very long messages (>10KB)? **Display truncated with "Show more" expansion.**
- What happens with special characters in search query? **Pass through to semantic search (embeddings handle this).**

## Requirements

### Functional Requirements

- **FR-001**: System MUST display a paginated list of conversations ordered by most recent activity.
- **FR-002**: System MUST allow users to search conversations using natural language queries.
- **FR-003**: System MUST display conversation details showing all messages with role (user/assistant), timestamp, and content.
- **FR-004**: System MUST support filtering conversations by project directory.
- **FR-005**: Search results MUST show a relevance-ordered list with snippet previews.
- **FR-006**: System MUST integrate with the global search bar via the `Searchable` interface.
- **FR-007**: System MUST handle search service (Ollama) unavailability gracefully with user-facing error message.
- **FR-008**: Conversation list MUST show: derived title, message count, date range, and source indicator.

### Key Entities

- **Conversation**: A group of messages sharing the same `conversation_id`. Displayed with computed metadata (title from first user message or CWD, message count, date range).
- **Message (Chunk)**: Individual message within a conversation. Has role (user/assistant), timestamp, content, and optional metadata (git branch, CWD).
- **SearchResult**: A conversation matching a semantic query, with similarity score and snippet preview.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Users can find a specific past conversation within 30 seconds using search.
- **SC-002**: Conversation list loads within 2 seconds for up to 1000 conversations.
- **SC-003**: Search results return within 3 seconds (includes embedding generation).
- **SC-004**: Zero data loss - all imported conversations remain accessible.

## Testing Requirements

### Test Strategy

Integration tests at the handler level verifying HTTP responses. No unit tests for simple CRUD handlers. E2E tests for critical user journeys (search, browse, view).

Test frameworks: Go `testing` package with `httptest` for handler tests. Database tests use test PostgreSQL instance.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Handler returns paginated list with correct ordering and HTMX compatibility |
| FR-002 | Integration | Search handler returns semantically matched results |
| FR-003 | Integration | Detail handler returns full conversation with all messages |
| FR-004 | Integration | List handler filters by project query param |
| FR-005 | Integration | Search results include snippets and are relevance-ordered |
| FR-006 | Integration | Plugin implements Searchable interface correctly |
| FR-007 | Integration | Search returns graceful error when embedder unavailable |

### Edge Case Coverage

- Empty conversation list → Handler returns empty state response
- Embedder unavailable → Search returns 503 with error message
- Invalid conversation ID → Detail returns 404
- Invalid page/limit params → Defaults applied without error
