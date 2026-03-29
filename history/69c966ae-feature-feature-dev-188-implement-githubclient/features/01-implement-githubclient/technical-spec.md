# Technical Specification: Implement GitHubClient

## Architecture

```
Caller → github.Client interface
           ↓
         httpClient (this implementation)
           ↓
         *github.Client (google/go-github v84)
           ↓
         Transport stack:
           auth (WithAuthToken) → secondary rate limiter → primary rate limiter → http.DefaultTransport
```

The `httpClient` wraps `*github.Client` from go-github, which handles JSON serialization, auth headers, and response parsing. Our layer adds:
1. Interface compliance with `github.Client`
2. Error mapping from go-github types to our `github.Error` types
3. Pre-emptive rate limit checking via `Response.Rate`
4. Type conversion from go-github structs to our domain types

## File: `internal/github/options.go`

```go
package github

import (
	"net/http"
	"time"
)

// Option configures an httpClient.
type Option func(*httpClient)

// WithEndpoint overrides the default GitHub API base URL.
// Useful for GitHub Enterprise or testing with httptest.
func WithEndpoint(url string) Option {
	return func(c *httpClient) { c.endpoint = url }
}

// WithHTTPClient overrides the default HTTP client.
// When set, the rate-limiter transport is NOT applied —
// the caller is responsible for any transport middleware.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *httpClient) {
		c.customHTTPClient = hc
	}
}

// WithRetryTiming overrides retry backoff timing (for tests).
func WithRetryTiming(base, maxJitter time.Duration) Option {
	return func(c *httpClient) {
		c.retryBase = base
		c.retryMaxJitter = maxJitter
	}
}

// WithRateLimitThreshold sets the low-water mark for pre-emptive
// rate limit delay. When X-RateLimit-Remaining drops below this
// value, the client sleeps before the next request. Default: 10.
func WithRateLimitThreshold(n int) Option {
	return func(c *httpClient) { c.rateLimitThreshold = n }
}
```

## File: `internal/github/http_client.go`

### Constants and struct

```go
package github

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
)

const (
	defaultTimeout         = 30 * time.Second
	defaultRateLimitThresh = 10
	maxRetries             = 3
	retryBaseDelay         = 1 * time.Second
	maxJitter              = 500 * time.Millisecond
	maxPreemptiveSleep     = 60 * time.Second
)

var _ Client = (*httpClient)(nil)

type httpClient struct {
	ghClient           *gh.Client
	owner              string
	repo               string
	endpoint           string
	customHTTPClient   *http.Client
	retryBase          time.Duration
	retryMaxJitter     time.Duration
	rateLimitThreshold int
}
```

### Constructors

```go
func NewClient(token, owner, repo string, opts ...Option) (*httpClient, error) {
	if token == "" {
		return nil, fmt.Errorf("github: token must not be empty")
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("github: owner and repo must not be empty")
	}

	c := &httpClient{
		owner:              owner,
		repo:               repo,
		retryBase:          retryBaseDelay,
		retryMaxJitter:     maxJitter,
		rateLimitThreshold: defaultRateLimitThresh,
	}
	for _, opt := range opts {
		opt(c)
	}

	var ghClient *gh.Client
	if c.customHTTPClient != nil {
		// Caller-provided HTTP client — skip rate limiter transport
		ghClient = gh.NewClient(c.customHTTPClient).WithAuthToken(token)
	} else {
		// Standard path: rate-limiter transport wrapping default transport
		rateLimiter := github_ratelimit.NewClient(nil)
		ghClient = gh.NewClient(rateLimiter).WithAuthToken(token)
	}

	if c.endpoint != "" {
		var err error
		ghClient, err = ghClient.WithEnterpriseURLs(c.endpoint, c.endpoint)
		if err != nil {
			return nil, fmt.Errorf("github: invalid endpoint URL: %w", err)
		}
	}

	c.ghClient = ghClient
	return c, nil
}

func NewClientFromEnv(owner, repo string, opts ...Option) (*httpClient, error) {
	token := os.Getenv("BRAINS_GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("github: BRAINS_GITHUB_TOKEN environment variable not set")
	}
	return NewClient(token, owner, repo, opts...)
}
```

### Pre-emptive rate limit check

```go
// checkRateLimit inspects the rate limit info from a response and sleeps
// if remaining requests are below the threshold. Returns an error only
// if the context is cancelled during the sleep.
func (c *httpClient) checkRateLimit(ctx context.Context, resp *gh.Response) error {
	if resp == nil || resp.Rate.Remaining >= c.rateLimitThreshold {
		return nil
	}

	sleepDuration := time.Until(resp.Rate.Reset.Time)
	if sleepDuration <= 0 {
		return nil
	}
	if sleepDuration > maxPreemptiveSleep {
		sleepDuration = maxPreemptiveSleep
	}

	select {
	case <-ctx.Done():
		return NewNetworkError("github: request cancelled during rate limit wait", ctx.Err())
	case <-time.After(sleepDuration):
		return nil
	}
}
```

### Error mapping

```go
// mapError converts go-github errors to our github.Error types.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	// Context cancellation/timeout
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return NewNetworkError("github: "+err.Error(), err)
	}

	// Primary rate limit
	var rateLimitErr *gh.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return NewRateLimitedError(
			fmt.Sprintf("github: primary rate limit exceeded, resets at %v", rateLimitErr.Rate.Reset),
			err,
		)
	}

	// Secondary (abuse) rate limit
	var abuseErr *gh.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		msg := "github: secondary rate limit exceeded"
		if abuseErr.RetryAfter != nil {
			msg = fmt.Sprintf("%s, retry after %v", msg, *abuseErr.RetryAfter)
		}
		return NewRateLimitedError(msg, err)
	}

	// GitHub API error with status code
	var ghErr *gh.ErrorResponse
	if errors.As(err, &ghErr) {
		switch ghErr.Response.StatusCode {
		case http.StatusNotFound:
			return NewNotFoundError(fmt.Sprintf("github: %s", ghErr.Message), err)
		default:
			return NewAPIError(fmt.Sprintf("github: %s (HTTP %d)", ghErr.Message, ghErr.Response.StatusCode), err)
		}
	}

	// Network-level errors (DNS, connection refused, etc.)
	return NewNetworkError(fmt.Sprintf("github: %s", err.Error()), err)
}
```

### Retry wrapper

```go
// doWithRetry wraps an operation with retry logic for rate-limited and 5xx errors.
func (c *httpClient) doWithRetry(ctx context.Context, op func() error) error {
	var lastErr error
	for attempt := range maxRetries + 1 {
		err := op()
		if err == nil {
			return nil
		}
		mapped := mapError(err)

		// Only retry rate limits and transient server errors
		if !IsRateLimited(mapped) && !isServerError(err) {
			return mapped
		}
		lastErr = mapped
		if attempt == maxRetries {
			break
		}

		delay := c.retryDelay(attempt)
		select {
		case <-ctx.Done():
			return NewNetworkError("github: request cancelled during retry", ctx.Err())
		case <-time.After(delay):
		}
	}
	return lastErr
}

// isServerError checks if a go-github error is a 5xx response.
func isServerError(err error) bool {
	var ghErr *gh.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response.StatusCode >= 500 {
		return true
	}
	return false
}

func (c *httpClient) retryDelay(attempt int) time.Duration {
	base := float64(c.retryBase) * math.Pow(2.0, float64(attempt))
	jitter := time.Duration(rand.Int64N(int64(c.retryMaxJitter) + 1))
	return time.Duration(base) + jitter
}
```

### Method implementations

#### CreatePR

```go
func (c *httpClient) CreatePR(ctx context.Context, input CreatePRInput) (int, error) {
	var prNumber int
	err := c.doWithRetry(ctx, func() error {
		pr, resp, err := c.ghClient.PullRequests.Create(ctx, c.owner, c.repo, &gh.NewPullRequest{
			Title: gh.Ptr(input.Title),
			Body:  gh.Ptr(input.Body),
			Head:  gh.Ptr(input.Head),
			Base:  gh.Ptr(input.Base),
		})
		if err != nil {
			return err
		}
		_ = c.checkRateLimit(ctx, resp)
		prNumber = pr.GetNumber()
		return nil
	})
	if err != nil {
		return 0, err // already mapped by doWithRetry
	}
	return prNumber, nil
}
```

#### UpdatePRBody

```go
func (c *httpClient) UpdatePRBody(ctx context.Context, prNumber int, body string) error {
	return c.doWithRetry(ctx, func() error {
		_, resp, err := c.ghClient.PullRequests.Edit(ctx, c.owner, c.repo, prNumber, &gh.PullRequest{
			Body: gh.Ptr(body),
		})
		if err != nil {
			return err
		}
		_ = c.checkRateLimit(ctx, resp)
		return nil
	})
}
```

#### GetCommentsSince

```go
func (c *httpClient) GetCommentsSince(ctx context.Context, prNumber int, kind CommentKind, afterID int64) ([]PRComment, error) {
	var comments []PRComment
	var fetchErr error

	switch kind {
	case CommentKindIssue:
		comments, fetchErr = c.getIssueCommentsSince(ctx, prNumber, afterID)
	case CommentKindReview:
		comments, fetchErr = c.getReviewCommentsSince(ctx, prNumber, afterID)
	default:
		return nil, NewAPIError(fmt.Sprintf("github: unknown comment kind %q", kind), nil)
	}

	if fetchErr != nil {
		return nil, fetchErr
	}
	return comments, nil
}

func (c *httpClient) getIssueCommentsSince(ctx context.Context, prNumber int, afterID int64) ([]PRComment, error) {
	var result []PRComment
	opts := &gh.IssueListCommentsOptions{
		Sort:      gh.Ptr("created"),
		Direction: gh.Ptr("asc"),
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		var comments []*gh.IssueComment
		var resp *gh.Response
		err := c.doWithRetry(ctx, func() error {
			var e error
			comments, resp, e = c.ghClient.Issues.ListComments(ctx, c.owner, c.repo, prNumber, opts)
			return e
		})
		if err != nil {
			return nil, err
		}

		for _, comment := range comments {
			if comment.GetID() > afterID {
				result = append(result, PRComment{
					ID:        comment.GetID(),
					Author:    comment.GetUser().GetLogin(),
					Body:      comment.GetBody(),
					CreatedAt: comment.GetCreatedAt().Time,
				})
			}
		}

		_ = c.checkRateLimit(ctx, resp)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return result, nil
}

func (c *httpClient) getReviewCommentsSince(ctx context.Context, prNumber int, afterID int64) ([]PRComment, error) {
	var result []PRComment
	opts := &gh.PullRequestListCommentsOptions{
		Sort:      "created",
		Direction: "asc",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		var comments []*gh.PullRequestComment
		var resp *gh.Response
		err := c.doWithRetry(ctx, func() error {
			var e error
			comments, resp, e = c.ghClient.PullRequests.ListComments(ctx, c.owner, c.repo, prNumber, opts)
			return e
		})
		if err != nil {
			return nil, err
		}

		for _, comment := range comments {
			if comment.GetID() > afterID {
				result = append(result, PRComment{
					ID:          comment.GetID(),
					Author:      comment.GetUser().GetLogin(),
					Body:        comment.GetBody(),
					CreatedAt:   comment.GetCreatedAt().Time,
					Path:        comment.GetPath(),
					DiffHunk:    comment.GetDiffHunk(),
					InReplyToID: comment.GetInReplyTo(),
				})
			}
		}

		_ = c.checkRateLimit(ctx, resp)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return result, nil
}
```

#### PostCommentReply

```go
func (c *httpClient) PostCommentReply(ctx context.Context, prNumber int, kind CommentKind, commentID int64, body string) (int64, error) {
	switch kind {
	case CommentKindIssue:
		var newID int64
		err := c.doWithRetry(ctx, func() error {
			comment, resp, err := c.ghClient.Issues.CreateComment(ctx, c.owner, c.repo, prNumber, &gh.IssueComment{
				Body: gh.Ptr(body),
			})
			if err != nil {
				return err
			}
			_ = c.checkRateLimit(ctx, resp)
			newID = comment.GetID()
			return nil
		})
		if err != nil {
			return 0, err
		}
		return newID, nil

	case CommentKindReview:
		if commentID == 0 {
			return 0, NewNotFoundError("github: commentID must be non-zero for review comment replies", nil)
		}
		var newID int64
		err := c.doWithRetry(ctx, func() error {
			comment, resp, err := c.ghClient.PullRequests.CreateCommentInReplyTo(ctx, c.owner, c.repo, prNumber, body, commentID)
			if err != nil {
				return err
			}
			_ = c.checkRateLimit(ctx, resp)
			newID = comment.GetID()
			return nil
		})
		if err != nil {
			return 0, err
		}
		return newID, nil

	default:
		return 0, NewAPIError(fmt.Sprintf("github: unknown comment kind %q", kind), nil)
	}
}
```

#### ApplyLabel

```go
func (c *httpClient) ApplyLabel(ctx context.Context, prNumber int, label string) error {
	return c.doWithRetry(ctx, func() error {
		_, resp, err := c.ghClient.Issues.AddLabelsToIssue(ctx, c.owner, c.repo, prNumber, []string{label})
		if err != nil {
			return err
		}
		_ = c.checkRateLimit(ctx, resp)
		return nil
	})
}
```

#### IsMerged

```go
func (c *httpClient) IsMerged(ctx context.Context, prNumber int) (bool, error) {
	var merged bool
	err := c.doWithRetry(ctx, func() error {
		m, resp, err := c.ghClient.PullRequests.IsMerged(ctx, c.owner, c.repo, prNumber)
		if err != nil {
			// go-github's IsMerged uses parseBoolResponse which converts
			// 404 to (false, nil). But guard defensively: if a 404 error
			// does come through, treat it as "not merged" per FR-7.
			mapped := mapError(err)
			if IsNotFound(mapped) {
				merged = false
				return nil
			}
			return err
		}
		_ = c.checkRateLimit(ctx, resp)
		merged = m
		return nil
	})
	if err != nil {
		return false, err
	}
	return merged, nil
}
```

#### IsClosed

```go
func (c *httpClient) IsClosed(ctx context.Context, prNumber int) (bool, error) {
	var closed bool
	err := c.doWithRetry(ctx, func() error {
		pr, resp, err := c.ghClient.PullRequests.Get(ctx, c.owner, c.repo, prNumber)
		if err != nil {
			return err
		}
		_ = c.checkRateLimit(ctx, resp)
		closed = pr.GetState() == "closed" && !pr.GetMerged()
		return nil
	})
	if err != nil {
		return false, err
	}
	return closed, nil
}
```

#### ListOpenPRs

```go
func (c *httpClient) ListOpenPRs(ctx context.Context, label string) ([]PRSummary, error) {
	var result []PRSummary
	opts := &gh.PullRequestListOptions{
		State:       "open",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		var prs []*gh.PullRequest
		var resp *gh.Response
		err := c.doWithRetry(ctx, func() error {
			var e error
			prs, resp, e = c.ghClient.PullRequests.List(ctx, c.owner, c.repo, opts)
			return e
		})
		if err != nil {
			return nil, err
		}

		for _, pr := range prs {
			if hasLabel(pr.Labels, label) {
				result = append(result, PRSummary{
					Number: pr.GetNumber(),
					Title:  pr.GetTitle(),
					Head:   pr.GetHead().GetRef(),
					Base:   pr.GetBase().GetRef(),
					Labels: labelNames(pr.Labels),
				})
			}
		}

		_ = c.checkRateLimit(ctx, resp)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return result, nil
}

func hasLabel(labels []*gh.Label, target string) bool {
	for _, l := range labels {
		if l.GetName() == target {
			return true
		}
	}
	return false
}

func labelNames(labels []*gh.Label) []string {
	names := make([]string, len(labels))
	for i, l := range labels {
		names[i] = l.GetName()
	}
	return names
}
```

## Testing Strategy

Unit tests use `httptest.NewServer` with `WithEndpoint(srv.URL)` and `WithHTTPClient` to bypass the rate-limiter transport. Test server handlers return canned JSON responses matching GitHub's REST API format.

### Test helper

```go
func newTestClient(t *testing.T, handler http.HandlerFunc) *httpClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient("test-token", "owner", "repo",
		WithEndpoint(srv.URL),
		WithHTTPClient(srv.Client()),
		WithRetryTiming(10*time.Millisecond, 5*time.Millisecond),
	)
	require.NoError(t, err)
	return c
}
```

### Test categories

1. **Constructor tests**: empty token, empty owner/repo, env var missing
2. **Happy path**: each method with valid response
3. **Error classification**: 404, 429, 403-rate-limit, 403-permission, 5xx, network error
4. **Pagination**: multi-page GetCommentsSince, multi-page ListOpenPRs
5. **ID filtering**: GetCommentsSince correctly filters by afterID boundary
6. **Kind dispatching**: PostCommentReply routes correctly for issue vs review
7. **Guards**: PostCommentReply with commentID=0 for review kind
8. **Retry**: verify retry on 429, verify no retry on 404
9. **Rate limit**: verify checkRateLimit sleeps when remaining < threshold

### Important: go-github test server URL format

go-github's `WithEnterpriseURLs` appends `/api/v3/` to the base URL. The test helper sets up the endpoint so that go-github's URL construction works correctly with `httptest.NewServer`. Test handlers must match the full path including `/api/v3/repos/{owner}/{repo}/...`.

Alternative: use `gh.NewClient(httpClient)` directly in tests and set `ghClient.BaseURL` manually to avoid the enterprise URL suffix.
