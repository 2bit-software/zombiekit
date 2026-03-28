package linear

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*httpClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient("test-api-key",
		WithEndpoint(srv.URL),
		WithRetryTiming(10*time.Millisecond, 5*time.Millisecond),
	)
	require.NoError(t, err)
	return c, srv
}

func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func gqlSuccess(data any) map[string]any {
	raw, _ := json.Marshal(data)
	return map[string]any{"data": json.RawMessage(raw)}
}

func gqlError(code, message string) map[string]any {
	return map[string]any{
		"errors": []map[string]any{
			{
				"message":    message,
				"extensions": map[string]any{"code": code},
			},
		},
	}
}

// --- Constructor tests ---

func TestNewClient_MissingAPIKey(t *testing.T) {
	_, err := NewClient("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key must not be empty")
}

func TestNewClient_Success(t *testing.T) {
	c, err := NewClient("my-key")
	require.NoError(t, err)
	assert.Equal(t, "my-key", c.apiKey)
	assert.Equal(t, defaultEndpoint, c.endpoint)
}

func TestNewClientFromEnv(t *testing.T) {
	t.Setenv("BRAINS_LINEAR_API_KEY", "env-key")
	c, err := NewClientFromEnv()
	require.NoError(t, err)
	assert.Equal(t, "env-key", c.apiKey)
}

func TestNewClientFromEnv_Missing(t *testing.T) {
	t.Setenv("BRAINS_LINEAR_API_KEY", "")
	_, err := NewClientFromEnv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BRAINS_LINEAR_API_KEY")
}

// --- Auth header test ---

func TestAuthHeader_NoBearer(t *testing.T) {
	var gotAuth string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		jsonResponse(w, 200, gqlSuccess(map[string]any{"issues": map[string]any{"nodes": []any{}, "pageInfo": map[string]any{"hasNextPage": false}}}))
	})

	_, err := c.PollReadyTickets(context.Background(), "ai-ready")
	require.NoError(t, err)
	assert.Equal(t, "test-api-key", gotAuth)
}

// --- PollReadyTickets tests ---

func TestPollReadyTickets_Success(t *testing.T) {
	desc := "Build the thing"
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, gqlSuccess(map[string]any{
			"issues": map[string]any{
				"nodes": []map[string]any{
					{
						"id":          "uuid-1",
						"identifier":  "DEV-100",
						"title":       "Test ticket",
						"description": desc,
						"url":         "https://linear.app/test/DEV-100",
						"priority":    2.0,
						"state":       map[string]any{"name": "In Progress"},
						"labels":      map[string]any{"nodes": []map[string]any{{"name": "ai-ready"}}},
					},
				},
				"pageInfo": map[string]any{"hasNextPage": false},
			},
		}))
	})

	tickets, err := c.PollReadyTickets(context.Background(), "ai-ready")
	require.NoError(t, err)
	require.Len(t, tickets, 1)

	assert.Equal(t, "uuid-1", tickets[0].ID)
	assert.Equal(t, "DEV-100", tickets[0].Identifier)
	assert.Equal(t, "Test ticket", tickets[0].Title)
	assert.Equal(t, "Build the thing", tickets[0].Description)
	assert.Equal(t, "https://linear.app/test/DEV-100", tickets[0].URL)
	assert.Equal(t, 2, tickets[0].Priority)
	assert.Equal(t, "In Progress", tickets[0].Status)
	assert.Equal(t, []string{"ai-ready"}, tickets[0].Labels)
}

func TestPollReadyTickets_EmptyResult(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, gqlSuccess(map[string]any{
			"issues": map[string]any{
				"nodes":    []any{},
				"pageInfo": map[string]any{"hasNextPage": false},
			},
		}))
	})

	tickets, err := c.PollReadyTickets(context.Background(), "ai-ready")
	require.NoError(t, err)
	assert.Empty(t, tickets)
	assert.NotNil(t, tickets)
}

func TestPollReadyTickets_FiltersEmptyDescription(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, gqlSuccess(map[string]any{
			"issues": map[string]any{
				"nodes": []map[string]any{
					{
						"id":          "uuid-1",
						"identifier":  "DEV-100",
						"title":       "Has description",
						"description": "content",
						"url":         "https://linear.app/test/DEV-100",
						"priority":    1.0,
						"state":       map[string]any{"name": "Todo"},
						"labels":      map[string]any{"nodes": []any{}},
					},
					{
						"id":          "uuid-2",
						"identifier":  "DEV-101",
						"title":       "Empty description",
						"description": "",
						"url":         "https://linear.app/test/DEV-101",
						"priority":    1.0,
						"state":       map[string]any{"name": "Todo"},
						"labels":      map[string]any{"nodes": []any{}},
					},
					{
						"id":          "uuid-3",
						"identifier":  "DEV-102",
						"title":       "Null description",
						"description": nil,
						"url":         "https://linear.app/test/DEV-102",
						"priority":    1.0,
						"state":       map[string]any{"name": "Todo"},
						"labels":      map[string]any{"nodes": []any{}},
					},
				},
				"pageInfo": map[string]any{"hasNextPage": false},
			},
		}))
	})

	tickets, err := c.PollReadyTickets(context.Background(), "ai-ready")
	require.NoError(t, err)
	require.Len(t, tickets, 1)
	assert.Equal(t, "DEV-100", tickets[0].Identifier)
}

func TestPollReadyTickets_Pagination(t *testing.T) {
	var callCount atomic.Int32
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		call := callCount.Add(1)
		cursor := "cursor-page-2"
		if call == 1 {
			jsonResponse(w, 200, gqlSuccess(map[string]any{
				"issues": map[string]any{
					"nodes": []map[string]any{
						{"id": "uuid-1", "identifier": "DEV-1", "title": "Page 1", "description": "desc1", "url": "", "priority": 0.0, "state": map[string]any{"name": "Todo"}, "labels": map[string]any{"nodes": []any{}}},
					},
					"pageInfo": map[string]any{"hasNextPage": true, "endCursor": cursor},
				},
			}))
		} else {
			jsonResponse(w, 200, gqlSuccess(map[string]any{
				"issues": map[string]any{
					"nodes": []map[string]any{
						{"id": "uuid-2", "identifier": "DEV-2", "title": "Page 2", "description": "desc2", "url": "", "priority": 0.0, "state": map[string]any{"name": "Todo"}, "labels": map[string]any{"nodes": []any{}}},
					},
					"pageInfo": map[string]any{"hasNextPage": false},
				},
			}))
		}
	})

	tickets, err := c.PollReadyTickets(context.Background(), "ai-ready")
	require.NoError(t, err)
	require.Len(t, tickets, 2)
	assert.Equal(t, "DEV-1", tickets[0].Identifier)
	assert.Equal(t, "DEV-2", tickets[1].Identifier)
	assert.Equal(t, int32(2), callCount.Load())
}

// --- GetTicket tests ---

func TestGetTicket_Success(t *testing.T) {
	desc := "Full description"
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, gqlSuccess(map[string]any{
			"issue": map[string]any{
				"id":          "uuid-157",
				"identifier":  "DEV-157",
				"title":       "Implement polling",
				"description": desc,
				"url":         "https://linear.app/test/DEV-157",
				"priority":    2.0,
				"state":       map[string]any{"name": "In Progress"},
				"labels":      map[string]any{"nodes": []map[string]any{{"name": "ai-ready"}, {"name": "backend"}}},
			},
		}))
	})

	ticket, err := c.GetTicket(context.Background(), "DEV-157")
	require.NoError(t, err)
	assert.Equal(t, "uuid-157", ticket.ID)
	assert.Equal(t, "DEV-157", ticket.Identifier)
	assert.Equal(t, "Implement polling", ticket.Title)
	assert.Equal(t, "Full description", ticket.Description)
	assert.Equal(t, "In Progress", ticket.Status)
	assert.Equal(t, []string{"ai-ready", "backend"}, ticket.Labels)
	assert.Equal(t, 2, ticket.Priority)
	assert.Equal(t, "https://linear.app/test/DEV-157", ticket.URL)
}

func TestGetTicket_NotFound(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, gqlError("", "Entity not found"))
	})

	_, err := c.GetTicket(context.Background(), "DEV-99999")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

// --- Retry tests ---

func TestRetry_RateLimitThenSuccess(t *testing.T) {
	var callCount atomic.Int32
	desc := "success"
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		call := callCount.Add(1)
		if call == 1 {
			jsonResponse(w, 400, gqlError("RATELIMITED", "Too many requests"))
			return
		}
		jsonResponse(w, 200, gqlSuccess(map[string]any{
			"issue": map[string]any{
				"id": "uuid-1", "identifier": "DEV-1", "title": "t", "description": desc,
				"url": "", "priority": 0.0, "state": map[string]any{"name": "Todo"},
				"labels": map[string]any{"nodes": []any{}},
			},
		}))
	})

	ticket, err := c.GetTicket(context.Background(), "DEV-1")
	require.NoError(t, err)
	assert.Equal(t, "DEV-1", ticket.Identifier)
	assert.GreaterOrEqual(t, callCount.Load(), int32(2))
}

func TestRetry_RateLimitExhausted(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 400, gqlError("RATELIMITED", "Too many requests"))
	})

	_, err := c.GetTicket(context.Background(), "DEV-1")
	require.Error(t, err)
	assert.True(t, IsRateLimited(err))
}

func TestRetry_UsesResetHeader(t *testing.T) {
	var callCount atomic.Int32
	resetTime := time.Now().Add(100 * time.Millisecond)
	desc := "ok"
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		call := callCount.Add(1)
		if call == 1 {
			w.Header().Set("X-RateLimit-Requests-Reset", fmt.Sprintf("%d", resetTime.UnixMilli()))
			jsonResponse(w, 400, gqlError("RATELIMITED", "Too many requests"))
			return
		}
		jsonResponse(w, 200, gqlSuccess(map[string]any{
			"issue": map[string]any{
				"id": "uuid-1", "identifier": "DEV-1", "title": "t", "description": desc,
				"url": "", "priority": 0.0, "state": map[string]any{"name": "Todo"},
				"labels": map[string]any{"nodes": []any{}},
			},
		}))
	})

	start := time.Now()
	ticket, err := c.GetTicket(context.Background(), "DEV-1")
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, "DEV-1", ticket.Identifier)
	// Should have waited roughly until the reset time, not the full 1s base delay
	assert.Less(t, elapsed, 1*time.Second)
}

// --- Error mapping tests ---

func TestDo_HTTPError500(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
	})

	_, err := c.GetTicket(context.Background(), "DEV-1")
	require.Error(t, err)
	assert.True(t, IsNetworkError(err))
}

func TestDo_NonJSONResponse(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte("<html>Bad Gateway</html>"))
	})

	_, err := c.GetTicket(context.Background(), "DEV-1")
	require.Error(t, err)
	assert.True(t, IsNetworkError(err))
}

func TestDo_ContextCancelled(t *testing.T) {
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-done:
		}
	}))
	defer srv.Close()
	defer close(done)

	c, err := NewClient("test-api-key",
		WithEndpoint(srv.URL),
		WithRetryTiming(10*time.Millisecond, 5*time.Millisecond),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = c.GetTicket(ctx, "DEV-1")
	require.Error(t, err)
	assert.True(t, IsNetworkError(err))
}

func TestDo_Unauthorized(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error": "unauthorized"}`))
	})

	_, err := c.GetTicket(context.Background(), "DEV-1")
	require.Error(t, err)
	assert.True(t, IsAPIError(err))
}

// --- Unimplemented methods test ---

func TestUnimplemented_Methods(t *testing.T) {
	c, _ := NewClient("key")

	ctx := context.Background()

	err := c.SetTicketStatus(ctx, "id", "status")
	assert.ErrorContains(t, err, "not implemented")

	err = c.ApplyLabel(ctx, "id", "label")
	assert.ErrorContains(t, err, "not implemented")

	err = c.RemoveLabel(ctx, "id", "label")
	assert.ErrorContains(t, err, "not implemented")

	_, err = c.CreateTicket(ctx, CreateTicketInput{})
	assert.ErrorContains(t, err, "not implemented")

	err = c.UploadAttachment(ctx, "id", AttachmentInput{})
	assert.ErrorContains(t, err, "not implemented")
}
