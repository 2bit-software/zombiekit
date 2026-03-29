# Technical Requirements & Implementation Hints

Extracted from DEV-188 ticket — implementation preferences, not business requirements.

## Library Choice

- `google/go-github` or hand-rolled minimal REST calls — either acceptable
- "Don't over-engineer; the orchestrator only needs the 8 methods on the interface"

## Authentication

- GitHub personal access token via `BRAINS_GITHUB_TOKEN` env var (per project convention)
- `repo` scope covers all operations for private repos
- `public_repo` suffices for public repos
- Verify required scopes against GitHub REST API docs before implementing

## Rate Limiting Mechanism

- Read `X-RateLimit-Remaining` response header
- Pre-emptive slowdown before hitting zero (suggested threshold: 10 remaining)
- Backoff on HTTP 429 and 403 responses

## Pagination

- `GetCommentsSince` is the core of "Watcher 2"
- Pagination and watermark logic must be precise — off-by-one means duplicate or skipped comments downstream

## Testing

- Integration tests behind `//go:build integration` build tag
- Test against real GitHub API

## Spike Allowance

- A spike is acceptable if token scope requirements are unclear
