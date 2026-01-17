# Business Specification: Semantic Search Foundation

**Short Name**: `semantic-search-foundation`
**Created**: 2026-01-17
**Status**: Approved
**Linear Ticket**: DEV-72

---

## Problem Statement

ZombieKit needs a way to store and retrieve information based on meaning, not just keywords. Without this foundation, the Claude Conversation Importer (DEV-69) has no target system to import conversations into, blocking the broader goal of searchable project history.

## User Scenarios

### Scenario 1: Add Content to Memory (Priority: P1)

**As a** ZombieKit operator
**I need to** add text content to the system's memory
**So that** the information becomes searchable and retrievable later

**How we'll know it works:**
1. Given I have text content, when I add it to the system, then I receive confirmation that the content was stored
2. Given I add content, when I check what's stored, then my content appears in the list

---

### Scenario 2: Find Content by Meaning (Priority: P1)

**As a** ZombieKit operator
**I need to** search for stored content using natural language queries
**So that** I can find relevant information without remembering exact keywords

**How we'll know it works:**
1. Given I stored "The deployment failed because of memory limits", when I search for "out of memory errors", then the system returns the relevant content
2. Given I search with a query, when results are returned, then they are ranked by relevance to my query

---

### Scenario 3: Review Stored Content (Priority: P2)

**As a** ZombieKit operator
**I need to** view a list of all content stored in the system
**So that** I can understand what information is available and verify content was added correctly

**How we'll know it works:**
1. Given content exists in the system, when I request a list, then I see all stored entries
2. Given no content exists, when I request a list, then I receive a clear indication the system is empty

---

### Edge Cases

- What should happen when a user searches but no content has been stored yet?
- How should users be informed when their search returns no relevant matches?
- ~~What should happen if the same content is added multiple times?~~ **Resolved: Silent no-op, no duplicate created**

## Business Requirements

- **BR-001**: Operators can add arbitrary text content to the system's searchable memory
- **BR-002**: Operators can search for content using natural language queries
- **BR-003**: Operators can view all content currently stored in the system
- **BR-004**: The system confirms successful storage after content is added
- **BR-005**: Search results indicate relevance or similarity to the query
- **BR-006**: The system operates locally without requiring external services (local-only is the permanent model)
- **BR-007**: The system records when content was added and displays this timestamp in listings and search results
- **BR-008**: Adding duplicate content is silently ignored (no error, no duplicate entry created)

## Key Concepts

- **Memory/Storage**: The persistent repository where content is kept for later retrieval. Think of it as a searchable notebook that remembers everything added to it.
- **Semantic Search**: Finding content based on meaning rather than exact words. Searching for "deployment issues" can find content about "release failures" because the concepts are related.
- **Content/Entry**: A piece of text added to the system. Could be a conversation excerpt, a note, a document section, or any other text.
- **Relevance**: How closely a search result matches what the user is looking for. Higher relevance means the content is more likely to answer the user's question.

## Success Metrics

- **SM-001**: Operators can add content and retrieve it via search within a single session (round-trip validation)
- **SM-002**: Searches for semantically similar queries return expected content (e.g., "memory issues" finds "out of memory" content)

## Assumptions

- This is an operator-facing tool, not end-user facing; command-line interface is acceptable
- Local-only operation is sufficient for initial version
- The system will be extended later to support conversation import (DEV-69)
- A single user/workspace model is acceptable; multi-tenancy not required yet
- Ollama is running locally on the computer and managed by the user (not by this system)

## Out of Scope

- Web-based interface (future: DEV-71)
- Importing conversations from Claude (future: DEV-69)
- Multi-user access controls
- Cloud deployment or remote access
- Automatic content ingestion from external sources
- Deleting or updating stored content
- Content size limits
- Managing Ollama lifecycle (user's responsibility)

## Open Questions

*All questions resolved during review.*
