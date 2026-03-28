# Technical Requirements: DEV-157

## Implementation Preferences (from ticket)

- Linear uses GraphQL, not REST
- Use existing Go Linear client if well-maintained; otherwise hand-roll minimal queries
- Don't build a general-purpose GraphQL client
- Verify combined GraphQL filter syntax against Linear docs before writing the query
- If combined filters aren't supported natively, add client-side filter step

## Research Outcomes

- **No usable Go client**: `guillermo/linear` is 1-commit, 3-star, v0.0.0 -- hand-roll confirmed
- **Combined filters work**: `labels: { name: { eq: $label } }` + `description: { null: false }` are implicitly ANDed server-side
- **Client-side filter still needed**: `null: false` doesn't catch empty string -- add `len(description) > 0`

## Technical Constraints

- Auth header: `Authorization: <API_KEY>` (no Bearer prefix for API keys)
- Rate limit detection: HTTP 400 + `RATELIMITED` error code in response body (not HTTP 429)
- Rate limit headers: `X-RateLimit-Requests-Reset` provides reset time in UTC epoch ms
- Pagination: `first: 50` with cursor-based `after` parameter
- `issue(id:)` accepts both UUIDs and identifier strings like "DEV-157"

## Env Var Convention

Follow existing `BRAINS_*` pattern: `BRAINS_LINEAR_API_KEY`
