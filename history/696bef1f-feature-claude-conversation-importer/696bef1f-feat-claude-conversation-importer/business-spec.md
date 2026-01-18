# Business Specification: Claude Conversation Importer

**Short Name**: `conversation-importer`
**Linear Ticket**: DEV-69
**Status**: Approved

---

## Problem Statement

Users accumulate valuable knowledge, decisions, and context across many Claude conversations, but this information becomes difficult to find and reuse over time. Without a way to import and organize past conversations, users repeatedly lose access to insights they've already developed.

## User Scenarios

### Scenario 1: Import Conversation History (Priority: P1)

**As a** ZombieKit user
**I need to** import my Claude Code conversation history into the system
**So that** I can later search and reference past discussions, decisions, and insights

**How we'll know it works:**
1. Given a user has Claude Code history files available, when they initiate an import, then their conversations are added to the searchable corpus
2. Given conversations have been imported, when the user searches for a topic they discussed previously, then relevant conversations appear in results

---

### Scenario 2: Update Existing History (Priority: P1)

**As a** ZombieKit user
**I need to** re-import conversations without creating duplicates
**So that** I can keep my searchable history current while maintaining data integrity

**How we'll know it works:**
1. Given the user has previously imported conversations, when they import again, then new conversations are added without duplicating existing ones
2. Given duplicate detection is active, when the same conversation appears in multiple imports, then only one copy exists in the system

---

### Scenario 3: View Full Conversation from Search Result (Priority: P1)

**As a** ZombieKit user
**I need to** navigate from a search result to see the entire conversation it came from
**So that** I can understand the full context around the specific message I found

**How we'll know it works:**
1. Given a user finds a relevant message via search, when they request the full conversation, then all messages from that conversation are displayed in order
2. Given a user is viewing a message, when they look at conversation context, then they can see which conversation it belongs to and navigate the thread

---

### Scenario 4: Trigger Import on Demand (Priority: P2)

**As a** ZombieKit user
**I need to** manually trigger an import when I want fresh data
**So that** I control when my conversation history is updated

---

### Scenario 5: Watch for New Conversations (Priority: P2)

**As a** ZombieKit user
**I need to** have the system automatically detect and import new conversations
**So that** my searchable history stays current without manual intervention

---

## Business Requirements

- **BR-001**: Users can import their Claude conversation history into the searchable system
- **BR-002**: Users can re-import without creating duplicate entries for the same conversations
- **BR-003**: Users can manually trigger an import at any time
- **BR-004**: Users receive feedback on import progress and completion
- **BR-005**: Users can run a "watch" mode that automatically imports new conversations as they appear
- **BR-006**: The system preserves conversation context and structure for meaningful search results
- **BR-007**: ~~Users can view a history of past import operations~~ **DEFERRED** (see Out of Scope)
- **BR-008**: Each conversation has a unique identifier so users can reference and retrieve it
- **BR-009**: Users can retrieve all messages belonging to a specific conversation
- **BR-010**: Users can navigate from any message to view the full conversation it belongs to

## Key Concepts

- **Conversation**: A complete exchange between a user and Claude Code, consisting of multiple messages back and forth. Each conversation has a unique identifier.
- **Message**: A single contribution within a conversation (either from the user or from Claude). Each message belongs to exactly one conversation and can reference its parent message to show threading/sequence.
- **History File**: Claude Code's local storage of conversation history (history.jsonl), located in ~/.claude/ for global history or \<projectDir\>/.claude/ for project-specific history
- **Corpus**: The collection of all imported conversations available for searching
- **Watch Mode**: A long-running process that monitors history files for changes and imports new conversations automatically

## Assumptions

- Conversation history is sourced from Claude Code's local history files (~/.claude/projects/\<path\>/\<id\>.jsonl)
- Both user messages and Claude responses are imported (full conversation context)
- Conversations are retained indefinitely; date-range filtering will be handled at search time (future work)
- If a conversation is deleted from the source but exists locally, the local copy is preserved

## Out of Scope

- Importing from sources other than Claude Code history files (future: other sources via same command pattern)
- Selective import filtering (future work)
- Editing conversation content after import
- Sharing conversations between users
- Real-time synchronization
- Search functionality (covered by DEV-67, implemented in DEV-72)
- MCP tool interface (CLI only for this feature)
- **BR-007 (Deferred)**: Explicit import history tracking. Users can infer import timing from chunk `created_at` timestamps. A dedicated `import_runs` table can be added in a future iteration if explicit tracking is needed.
