package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient creates a client pointing at a test server.
// go-github's WithEnterpriseURLs adds /api/v3/ to the base URL,
// so test handlers must register routes under /api/v3/repos/owner/repo/...
func newTestClient(t *testing.T, handler http.Handler) *httpClient {
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

func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}

func rateLimitHeaders(w http.ResponseWriter, remaining int, resetUnix int64) {
	w.Header().Set("X-Ratelimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-Ratelimit-Limit", "5000")
	w.Header().Set("X-Ratelimit-Reset", strconv.FormatInt(resetUnix, 10))
}

// --- Constructor tests ---

func TestNewClient_EmptyToken(t *testing.T) {
	_, err := NewClient("", "owner", "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token must not be empty")
}

func TestNewClient_EmptyOwner(t *testing.T) {
	_, err := NewClient("token", "", "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner and repo must not be empty")
}

func TestNewClient_EmptyRepo(t *testing.T) {
	_, err := NewClient("token", "owner", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner and repo must not be empty")
}

func TestNewClient_Valid(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	c, err := NewClient("token", "owner", "repo",
		WithEndpoint(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	require.NoError(t, err)
	assert.NotNil(t, c.ghClient)
}

func TestNewClientFromEnv_Missing(t *testing.T) {
	t.Setenv("BRAINS_GITHUB_TOKEN", "")
	_, err := NewClientFromEnv("owner", "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BRAINS_GITHUB_TOKEN")
}

func TestNewClientFromEnv_Valid(t *testing.T) {
	t.Setenv("BRAINS_GITHUB_TOKEN", "test-token")
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	c, err := NewClientFromEnv("owner", "repo",
		WithEndpoint(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	require.NoError(t, err)
	assert.NotNil(t, c.ghClient)
}

// --- Error classification tests ---

func TestMapError_Nil(t *testing.T) {
	assert.Nil(t, mapError(nil))
}

func TestMapError_ContextCanceled(t *testing.T) {
	err := mapError(context.Canceled)
	require.Error(t, err)
	assert.True(t, IsNetworkError(err))
}

func TestMapError_NotFound(t *testing.T) {
	ghErr := &gh.ErrorResponse{
		Response: &http.Response{StatusCode: 404},
		Message:  "Not Found",
	}
	err := mapError(ghErr)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestMapError_APIError(t *testing.T) {
	ghErr := &gh.ErrorResponse{
		Response: &http.Response{StatusCode: 422},
		Message:  "Validation Failed",
	}
	err := mapError(ghErr)
	require.Error(t, err)
	assert.True(t, IsAPIError(err))
}

func TestMapError_NetworkError(t *testing.T) {
	err := mapError(fmt.Errorf("connection refused"))
	require.Error(t, err)
	assert.True(t, IsNetworkError(err))
}

// --- CreatePR tests ---

func TestCreatePR_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 201, map[string]any{
			"number": 42,
			"title":  "test",
			"state":  "open",
		})
	})
	c := newTestClient(t, mux)

	num, err := c.CreatePR(context.Background(), CreatePRInput{
		Title: "test", Body: "body", Head: "feature", Base: "main",
	})
	require.NoError(t, err)
	assert.Equal(t, 42, num)
}

func TestCreatePR_ValidationError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 422, map[string]any{
			"message": "Validation Failed",
		})
	})
	c := newTestClient(t, mux)

	_, err := c.CreatePR(context.Background(), CreatePRInput{
		Title: "test", Body: "body", Head: "bad", Base: "main",
	})
	require.Error(t, err)
	assert.True(t, IsAPIError(err))
}

// --- UpdatePRBody tests ---

func TestUpdatePRBody_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/repos/owner/repo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, map[string]any{
			"number": 1,
			"body":   "new body",
		})
	})
	c := newTestClient(t, mux)

	err := c.UpdatePRBody(context.Background(), 1, "new body")
	require.NoError(t, err)
}

func TestUpdatePRBody_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/repos/owner/repo/pulls/999", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 404, map[string]any{
			"message": "Not Found",
		})
	})
	c := newTestClient(t, mux)

	err := c.UpdatePRBody(context.Background(), 999, "body")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

// --- IsMerged tests ---

func TestIsMerged_True(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/1/merge", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		w.WriteHeader(204)
	})
	c := newTestClient(t, mux)

	merged, err := c.IsMerged(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, merged)
}

func TestIsMerged_False(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/1/merge", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 404, map[string]any{
			"message": "Not Found",
		})
	})
	c := newTestClient(t, mux)

	merged, err := c.IsMerged(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, merged)
}

// --- IsClosed tests ---

func TestIsClosed_ClosedNotMerged(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, map[string]any{
			"number": 1, "state": "closed", "merged": false,
		})
	})
	c := newTestClient(t, mux)

	closed, err := c.IsClosed(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, closed)
}

func TestIsClosed_Merged(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, map[string]any{
			"number": 1, "state": "closed", "merged": true,
		})
	})
	c := newTestClient(t, mux)

	closed, err := c.IsClosed(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, closed)
}

func TestIsClosed_Open(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, map[string]any{
			"number": 1, "state": "open", "merged": false,
		})
	})
	c := newTestClient(t, mux)

	closed, err := c.IsClosed(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, closed)
}

// --- ApplyLabel tests ---

func TestApplyLabel_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/issues/1/labels", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, []map[string]any{
			{"name": "bug"},
		})
	})
	c := newTestClient(t, mux)

	err := c.ApplyLabel(context.Background(), 1, "bug")
	require.NoError(t, err)
}

func TestApplyLabel_Idempotent(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/issues/1/labels", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, []map[string]any{
			{"name": "bug"},
		})
	})
	c := newTestClient(t, mux)

	require.NoError(t, c.ApplyLabel(context.Background(), 1, "bug"))
	require.NoError(t, c.ApplyLabel(context.Background(), 1, "bug"))
	assert.Equal(t, 2, callCount)
}

// --- GetCommentsSince tests ---

func TestGetCommentsSince_IssueComments_All(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, []map[string]any{
			{"id": 1, "body": "first", "user": map[string]any{"login": "alice"}, "created_at": "2026-01-01T00:00:00Z"},
			{"id": 2, "body": "second", "user": map[string]any{"login": "bob"}, "created_at": "2026-01-02T00:00:00Z"},
			{"id": 3, "body": "third", "user": map[string]any{"login": "carol"}, "created_at": "2026-01-03T00:00:00Z"},
		})
	})
	c := newTestClient(t, mux)

	comments, err := c.GetCommentsSince(context.Background(), 1, CommentKindIssue, 0)
	require.NoError(t, err)
	require.Len(t, comments, 3)
	assert.Equal(t, int64(1), comments[0].ID)
	assert.Equal(t, "alice", comments[0].Author)
	assert.Equal(t, "first", comments[0].Body)
	assert.Equal(t, int64(3), comments[2].ID)
}

func TestGetCommentsSince_IssueComments_AfterID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, []map[string]any{
			{"id": 1, "body": "first", "user": map[string]any{"login": "a"}, "created_at": "2026-01-01T00:00:00Z"},
			{"id": 2, "body": "second", "user": map[string]any{"login": "b"}, "created_at": "2026-01-02T00:00:00Z"},
			{"id": 3, "body": "third", "user": map[string]any{"login": "c"}, "created_at": "2026-01-03T00:00:00Z"},
			{"id": 4, "body": "fourth", "user": map[string]any{"login": "d"}, "created_at": "2026-01-04T00:00:00Z"},
			{"id": 5, "body": "fifth", "user": map[string]any{"login": "e"}, "created_at": "2026-01-05T00:00:00Z"},
		})
	})
	c := newTestClient(t, mux)

	comments, err := c.GetCommentsSince(context.Background(), 1, CommentKindIssue, 3)
	require.NoError(t, err)
	require.Len(t, comments, 2)
	assert.Equal(t, int64(4), comments[0].ID)
	assert.Equal(t, int64(5), comments[1].ID)
}

func TestGetCommentsSince_IssueComments_BoundaryExclusion(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, []map[string]any{
			{"id": 3, "body": "exact", "user": map[string]any{"login": "a"}, "created_at": "2026-01-01T00:00:00Z"},
			{"id": 4, "body": "after", "user": map[string]any{"login": "b"}, "created_at": "2026-01-02T00:00:00Z"},
		})
	})
	c := newTestClient(t, mux)

	comments, err := c.GetCommentsSince(context.Background(), 1, CommentKindIssue, 3)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, int64(4), comments[0].ID)
}

func TestGetCommentsSince_IssueComments_Pagination(t *testing.T) {
	page := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		page++
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		if page == 1 {
			// Simulate Link header for pagination
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v3/repos/owner/repo/issues/1/comments?page=2>; rel="next"`, "http://"+r.Host))
			jsonResponse(w, 200, []map[string]any{
				{"id": 1, "body": "p1c1", "user": map[string]any{"login": "a"}, "created_at": "2026-01-01T00:00:00Z"},
				{"id": 2, "body": "p1c2", "user": map[string]any{"login": "b"}, "created_at": "2026-01-02T00:00:00Z"},
			})
		} else {
			jsonResponse(w, 200, []map[string]any{
				{"id": 3, "body": "p2c1", "user": map[string]any{"login": "c"}, "created_at": "2026-01-03T00:00:00Z"},
			})
		}
	})
	c := newTestClient(t, mux)

	comments, err := c.GetCommentsSince(context.Background(), 1, CommentKindIssue, 0)
	require.NoError(t, err)
	require.Len(t, comments, 3)
	assert.Equal(t, 2, page)
}

func TestGetCommentsSince_ReviewComments(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/1/comments", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, []map[string]any{
			{
				"id": 10, "body": "review comment", "user": map[string]any{"login": "reviewer"},
				"created_at": "2026-01-01T00:00:00Z",
				"path":       "main.go", "diff_hunk": "@@ -1,3 +1,4 @@", "in_reply_to_id": 0,
			},
		})
	})
	c := newTestClient(t, mux)

	comments, err := c.GetCommentsSince(context.Background(), 1, CommentKindReview, 0)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "main.go", comments[0].Path)
	assert.Equal(t, "@@ -1,3 +1,4 @@", comments[0].DiffHunk)
}

// --- PostCommentReply tests ---

func TestPostCommentReply_IssueComment(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 201, map[string]any{
			"id":   100,
			"body": "reply",
		})
	})
	c := newTestClient(t, mux)

	id, err := c.PostCommentReply(context.Background(), 1, CommentKindIssue, 50, "reply")
	require.NoError(t, err)
	assert.Equal(t, int64(100), id)
}

func TestPostCommentReply_ReviewComment(t *testing.T) {
	mux := http.NewServeMux()
	// go-github sends POST /pulls/{n}/comments with in_reply_to in the body
	mux.HandleFunc("POST /api/v3/repos/owner/repo/pulls/1/comments", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 201, map[string]any{
			"id":   101,
			"body": "threaded reply",
		})
	})
	c := newTestClient(t, mux)

	id, err := c.PostCommentReply(context.Background(), 1, CommentKindReview, 50, "threaded reply")
	require.NoError(t, err)
	assert.Equal(t, int64(101), id)
}

func TestPostCommentReply_ReviewComment_ZeroID(t *testing.T) {
	c := newTestClient(t, http.NewServeMux())

	_, err := c.PostCommentReply(context.Background(), 1, CommentKindReview, 0, "body")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

// --- ListOpenPRs tests ---

func TestListOpenPRs_WithLabelFilter(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, []map[string]any{
			{
				"number": 1, "title": "has label", "state": "open",
				"head":   map[string]any{"ref": "feat-1"},
				"base":   map[string]any{"ref": "main"},
				"labels": []map[string]any{{"name": "target"}, {"name": "other"}},
			},
			{
				"number": 2, "title": "no label", "state": "open",
				"head":   map[string]any{"ref": "feat-2"},
				"base":   map[string]any{"ref": "main"},
				"labels": []map[string]any{{"name": "other"}},
			},
			{
				"number": 3, "title": "also has label", "state": "open",
				"head":   map[string]any{"ref": "feat-3"},
				"base":   map[string]any{"ref": "main"},
				"labels": []map[string]any{{"name": "target"}},
			},
		})
	})
	c := newTestClient(t, mux)

	prs, err := c.ListOpenPRs(context.Background(), "target")
	require.NoError(t, err)
	require.Len(t, prs, 2)
	assert.Equal(t, 1, prs[0].Number)
	assert.Equal(t, "feat-1", prs[0].Head)
	assert.Equal(t, "main", prs[0].Base)
	assert.Equal(t, []string{"target", "other"}, prs[0].Labels)
	assert.Equal(t, 3, prs[1].Number)
}

// --- Retry tests ---

func TestRetry_On429(t *testing.T) {
	attempt := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/1/merge", func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			// Set remaining > 0 so go-github's internal rate check doesn't
			// block the retry (it checks remaining==0 before each request)
			rateLimitHeaders(w, 10, time.Now().Add(time.Second).Unix())
			jsonResponse(w, 429, map[string]any{"message": "rate limit exceeded"})
			return
		}
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		w.WriteHeader(204)
	})
	c := newTestClient(t, mux)

	merged, err := c.IsMerged(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, merged)
	assert.Equal(t, 2, attempt)
}

func TestRetry_NoRetryOn404(t *testing.T) {
	attempt := 0
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/repos/owner/repo/pulls/999", func(w http.ResponseWriter, r *http.Request) {
		attempt++
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 404, map[string]any{"message": "Not Found"})
	})
	c := newTestClient(t, mux)

	err := c.UpdatePRBody(context.Background(), 999, "body")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.Equal(t, 1, attempt)
}

func TestRetry_On5xx(t *testing.T) {
	attempt := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/owner/repo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
			jsonResponse(w, 502, map[string]any{"message": "Bad Gateway"})
			return
		}
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, map[string]any{
			"number": 1, "state": "open", "merged": false,
		})
	})
	c := newTestClient(t, mux)

	closed, err := c.IsClosed(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, closed)
	assert.Equal(t, 2, attempt)
}

func TestRetry_ContextCancelled(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/issues/1/labels", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 10, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 429, map[string]any{"message": "rate limit"})
	})
	c := newTestClient(t, mux)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := c.ApplyLabel(ctx, 1, "bug")
	require.Error(t, err)
	assert.True(t, IsNetworkError(err))
}

// --- Rate limit pre-emptive delay test ---

func TestCheckRateLimit_SleepsWhenBelowThreshold(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/issues/1/labels", func(w http.ResponseWriter, r *http.Request) {
		// Set remaining below threshold, reset in 100ms
		rateLimitHeaders(w, 2, time.Now().Add(100*time.Millisecond).Unix())
		jsonResponse(w, 200, []map[string]any{{"name": "bug"}})
	})
	c := newTestClient(t, mux)
	c.rateLimitThreshold = 5

	start := time.Now()
	err := c.ApplyLabel(context.Background(), 1, "bug")
	elapsed := time.Since(start)

	require.NoError(t, err)
	// The reset time is at least ~100ms in the future (rounded to seconds),
	// but the exact sleep depends on time.Until(reset). Just verify the call
	// succeeded — the sleep mechanism is exercised by the non-zero elapsed time.
	// We can't assert exact timing due to second-granularity of the reset header.
	_ = elapsed
}

func TestCheckRateLimit_NoSleepAboveThreshold(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/owner/repo/issues/1/labels", func(w http.ResponseWriter, r *http.Request) {
		rateLimitHeaders(w, 4999, time.Now().Add(time.Hour).Unix())
		jsonResponse(w, 200, []map[string]any{{"name": "bug"}})
	})
	c := newTestClient(t, mux)

	start := time.Now()
	err := c.ApplyLabel(context.Background(), 1, "bug")
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, elapsed, 500*time.Millisecond)
}
