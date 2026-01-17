# Plan Highlights: Semantic Search Foundation

**Status**: APPROVED
**Created**: 2026-01-17

---

## Approved Decisions

### 1. Duplicate Detection Behavior (BR-008)

**Decision**: Truly silent — no output, no error on duplicate content.

### 2. PostgreSQL-Only Implementation

**Decision**: PostgreSQL required. Clear error on SQLite.

### 3. Fixed Embedding Dimensions

**Decision**: Schema hardcodes `vector(768)` for nomic-embed-text. Validates at startup.

### 4. Command Naming

**Decision**: `brains recall save` (not "ingest")

---

## Final Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CLI namespace | `brains recall` | Separate from `brains memory` |
| Save command | `brains recall save` | User preference |
| Duplicate handling | Silent no-op | User preference |
| Database | PostgreSQL only | pgvector required |
| Dimensions | Fixed 768 | nomic-embed-text |
| Duplicate detection | SHA-256 content hash | Exact match, O(1) lookup |
| Index type | HNSW | Fast queries |
| Distance metric | Cosine | Standard for semantic similarity |

---

## Artifacts

```
history/696be2a1-feature-semantic-search-foundation/
  696be2a1-feat-semantic-search-foundation/
    business-spec.md
    technical-requirements-research.md
    research-summary.md
    implementation-plan.md
    technical-spec.md
    highlights.md  ← You are here
```

---

## Next Step

```
/brains.tasks
```
