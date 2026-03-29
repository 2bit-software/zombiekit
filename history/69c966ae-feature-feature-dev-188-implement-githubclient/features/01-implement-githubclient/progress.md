# Progress Log

## T001 - Add dependencies
- Status: Complete
- Files: `go.mod`, `go.sum`
- Notes: go-github v84 bumped go.mod to Go 1.25.0

## T002 - Create options.go
- Status: Complete
- Files: `internal/github/options.go`

## T003 - Create core infrastructure
- Status: Complete
- Files: `internal/github/http_client.go`
- Notes: struct, constructors, mapError, doWithRetry, checkRateLimit, retryDelay, isServerError

## T004-T009 - Implement interface methods
- Status: Complete
- Files: `internal/github/http_client.go`
- Notes: All 8 methods implemented in single file

## T010-T013 - Unit tests
- Status: Complete
- Files: `internal/github/http_client_test.go`
- Notes: 29 tests total (18 new + 11 existing mock tests)

## Discoveries During Implementation

### go-github v84 rate limit detection requires specific conditions
- `RateLimitError` only returned when `X-RateLimit-Remaining: 0` header present
- `AbuseRateLimitError` only when `documentation_url` matches rate-limit suffixes
- Plain `ErrorResponse` returned for 429 in other cases
- Fix: added 429 and 403+Retry-After handling in ErrorResponse path of mapError

### go-github CreateCommentInReplyTo URL
- Does NOT use `/comments/{id}/replies` path
- Uses `POST /pulls/{n}/comments` with `in_reply_to` in request body
- Test handler URL updated accordingly

### go-github internal rate limit pre-check
- go-github caches rate limit state and blocks requests when remaining==0
- Tests must set remaining>0 even on 429 responses to avoid blocking retries
- Production code handles this via gofri/go-github-ratelimit transport
