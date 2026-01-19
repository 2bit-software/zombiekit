# Spike Results: DEV-70

**Created**: 2026-01-19

## Spike 1: SQL Query Validation

**Question**: Can we efficiently aggregate conversation metadata with a single SQL query?

**Finding**: Yes. The PostgreSQL query using CTE + GROUP BY is efficient:
- CTE fetches first user message per conversation (for title)
- Main query aggregates counts and timestamps
- HAVING clause filters empty conversations
- Existing index on `conversation_id` supports the query

**Validation**: Query pattern follows existing `GetByConversation` method structure. No spike code needed - pattern is proven in production.

## Spike 2: Embedder Availability Check

**Question**: How do we detect when Ollama is unavailable?

**Finding**: The embedder returns an error when Ollama is offline. Current pattern:
1. Call `embedder.Embed(ctx, query)`
2. If error, treat as unavailable
3. For global search, return empty results silently
4. For dedicated search page, show user-facing error

**No spike code needed** - error handling is standard Go pattern.

## Spike 3: URL Prefixing for Searchable

**Question**: Do we need to prefix URLs with plugin name in Searchable.Search()?

**Finding**: No. The web server automatically prefixes URLs with plugin name via `PrefixURL()` (see `/internal/web/search.go:71`).

**Correction**: Initial review suggested manual prefixing, but the framework handles this.

**Validated in**: `internal/web/search.go` line 71:
```go
items[i].URL = PrefixURL(rp.Name(), items[i].URL)
```

## Audit Findings Addressed

The plan audit identified several gaps. Resolutions:

| Gap | Resolution |
|-----|------------|
| Global search URL | **No change needed** - framework prefixes URLs automatically |
| Project filter in search | Added to Phase 5 - filter post-query |
| Similarity indicator | Added as optional display in search results |
| "Show more" for long messages | Documented as P2 enhancement, acceptable for MVP |
| Distinct projects query | Added to Phase 6 steps |
| Missing test cases | Added to Phase 8 |
