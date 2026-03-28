package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zombiekit/brains/internal/logging"
)

func init() {
	logging.InitLogger("error", false, nil)
}

func startTestServer(t *testing.T, bufferSize ...int) (*CallbackServer, string) {
	t.Helper()

	port := freePort(t)
	srv := &CallbackServer{
		port:   port,
		events: make(chan Event, defaultBufferSize),
		mux:    http.NewServeMux(),
	}
	if len(bufferSize) > 0 {
		srv.events = make(chan Event, bufferSize[0])
	}
	srv.registerRoutes()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	require.Eventually(t, func() bool {
		resp, err := http.Get(baseURL + "/healthz")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 3*time.Second, 50*time.Millisecond, "server did not start in time")

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("server returned error: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("server did not shut down in time")
		}
	})

	return srv, baseURL
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

func drainEvent(t *testing.T, srv *CallbackServer) Event {
	t.Helper()
	select {
	case ev := <-srv.Events():
		return ev
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}

// --- Happy Path Tests ---

func TestHandleComplete(t *testing.T) {
	srv, baseURL := startTestServer(t)

	resp := postJSON(t, baseURL+"/DEV-123/complete", map[string]string{
		"status":    "complete",
		"ticket_id": "DEV-123",
		"branch":    "DEV-123/add-feature",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var body map[string]bool
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.True(t, body["ok"])

	ev := drainEvent(t, srv)
	assert.Equal(t, EventComplete, ev.Kind)
	assert.Equal(t, "DEV-123", ev.TicketID)
	assert.Equal(t, "DEV-123/add-feature", ev.Branch)
	assert.False(t, ev.Timestamp.IsZero())
}

func TestHandleCommentResolved(t *testing.T) {
	srv, baseURL := startTestServer(t)

	resp := postJSON(t, baseURL+"/DEV-789/comment-resolved", map[string]string{
		"status":     "comment-resolved",
		"ticket_id":  "DEV-789",
		"comment_id": "IC_def456",
		"resolution": "Added nil check as requested",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	ev := drainEvent(t, srv)
	assert.Equal(t, EventCommentResolved, ev.Kind)
	assert.Equal(t, "DEV-789", ev.TicketID)
	assert.Equal(t, "IC_def456", ev.CommentID)
	assert.Equal(t, "Added nil check as requested", ev.Resolution)
}

func TestHandleFailedWithoutCommentID(t *testing.T) {
	srv, baseURL := startTestServer(t)

	resp := postJSON(t, baseURL+"/DEV-456/failed", map[string]string{
		"status":    "failed",
		"ticket_id": "DEV-456",
		"reason":    "tests failing after 3 attempts",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	ev := drainEvent(t, srv)
	assert.Equal(t, EventFailed, ev.Kind)
	assert.Equal(t, "DEV-456", ev.TicketID)
	assert.Equal(t, "tests failing after 3 attempts", ev.Reason)
	assert.Empty(t, ev.CommentID)
}

func TestHandleFailedWithCommentID(t *testing.T) {
	srv, baseURL := startTestServer(t)

	resp := postJSON(t, baseURL+"/DEV-456/failed", map[string]string{
		"status":     "failed",
		"ticket_id":  "DEV-456",
		"comment_id": "IC_abc123",
		"reason":     "cannot resolve conflicting review feedback",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	ev := drainEvent(t, srv)
	assert.Equal(t, EventFailed, ev.Kind)
	assert.Equal(t, "IC_abc123", ev.CommentID)
	assert.Equal(t, "cannot resolve conflicting review feedback", ev.Reason)
}

// --- Validation Tests ---

func TestValidationErrors(t *testing.T) {
	_, baseURL := startTestServer(t)

	tests := []struct {
		name     string
		path     string
		body     any
		wantErr  string
	}{
		{
			name:    "complete missing branch",
			path:    "/DEV-1/complete",
			body:    map[string]string{"status": "complete", "ticket_id": "DEV-1"},
			wantErr: "missing required field: branch",
		},
		{
			name:    "complete missing ticket_id",
			path:    "/DEV-1/complete",
			body:    map[string]string{"status": "complete", "branch": "x"},
			wantErr: "missing required field: ticket_id",
		},
		{
			name:    "complete wrong status",
			path:    "/DEV-1/complete",
			body:    map[string]string{"status": "failed", "ticket_id": "DEV-1", "branch": "x"},
			wantErr: "status field must be 'complete' for this route",
		},
		{
			name:    "comment-resolved missing comment_id",
			path:    "/DEV-1/comment-resolved",
			body:    map[string]string{"status": "comment-resolved", "ticket_id": "DEV-1", "resolution": "fixed"},
			wantErr: "missing required field: comment_id",
		},
		{
			name:    "comment-resolved missing resolution",
			path:    "/DEV-1/comment-resolved",
			body:    map[string]string{"status": "comment-resolved", "ticket_id": "DEV-1", "comment_id": "IC_1"},
			wantErr: "missing required field: resolution",
		},
		{
			name:    "failed missing reason",
			path:    "/DEV-1/failed",
			body:    map[string]string{"status": "failed", "ticket_id": "DEV-1"},
			wantErr: "missing required field: reason",
		},
		{
			name:    "failed wrong status",
			path:    "/DEV-1/failed",
			body:    map[string]string{"status": "complete", "ticket_id": "DEV-1", "reason": "x"},
			wantErr: "status field must be 'failed' for this route",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := postJSON(t, baseURL+tt.path, tt.body)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			var errBody map[string]string
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&errBody))
			assert.Contains(t, errBody["error"], tt.wantErr)
		})
	}
}

func TestMalformedJSON(t *testing.T) {
	_, baseURL := startTestServer(t)

	resp, err := http.Post(baseURL+"/DEV-1/complete", "application/json", strings.NewReader("{invalid"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestEmptyBody(t *testing.T) {
	_, baseURL := startTestServer(t)

	resp, err := http.Post(baseURL+"/DEV-1/complete", "application/json", strings.NewReader(""))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUnknownFields(t *testing.T) {
	_, baseURL := startTestServer(t)

	resp := postJSON(t, baseURL+"/DEV-1/complete", map[string]string{
		"status":    "complete",
		"ticket_id": "DEV-1",
		"branch":    "x",
		"extra":     "should-reject",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOversizedBody(t *testing.T) {
	_, baseURL := startTestServer(t)

	bigBody := `{"status":"complete","ticket_id":"DEV-1","branch":"` + strings.Repeat("x", 70*1024) + `"}`
	resp, err := http.Post(baseURL+"/DEV-1/complete", "application/json", strings.NewReader(bigBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- Backpressure Test ---

func TestBackpressure(t *testing.T) {
	_, baseURL := startTestServer(t, 2)

	// Fill the buffer
	for i := range 2 {
		resp := postJSON(t, baseURL+fmt.Sprintf("/DEV-%d/complete", i), map[string]string{
			"status":    "complete",
			"ticket_id": fmt.Sprintf("DEV-%d", i),
			"branch":    "x",
		})
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Third should get 503
	resp := postJSON(t, baseURL+"/DEV-99/complete", map[string]string{
		"status":    "complete",
		"ticket_id": "DEV-99",
		"branch":    "x",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var errBody map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errBody))
	assert.Equal(t, "event queue full, retry later", errBody["error"])
}

// --- Concurrency Test ---

func TestConcurrentRequests(t *testing.T) {
	srv, baseURL := startTestServer(t)

	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		go func(id int) {
			defer wg.Done()
			resp := postJSON(t, baseURL+fmt.Sprintf("/TICKET-%d/complete", id), map[string]string{
				"status":    "complete",
				"ticket_id": fmt.Sprintf("TICKET-%d", id),
				"branch":    fmt.Sprintf("TICKET-%d/feature", id),
			})
			resp.Body.Close()
		}(i)
	}

	wg.Wait()

	received := make(map[string]bool)
	for range n {
		ev := drainEvent(t, srv)
		received[ev.TicketID] = true
		assert.Equal(t, EventComplete, ev.Kind)
	}

	assert.Len(t, received, n)
	for i := range n {
		assert.True(t, received[fmt.Sprintf("TICKET-%d", i)])
	}
}

// --- Shutdown Test ---

func TestGracefulShutdown(t *testing.T) {
	port := freePort(t)
	srv := New(port)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	require.Eventually(t, func() bool {
		resp, err := http.Get(baseURL + "/healthz")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 3*time.Second, 50*time.Millisecond)

	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("server did not shut down in time")
	}

	// Verify events channel is closed
	_, ok := <-srv.Events()
	assert.False(t, ok, "events channel should be closed after shutdown")
}

// --- Health Check Test ---

func TestHealthCheck(t *testing.T) {
	_, baseURL := startTestServer(t)

	resp, err := http.Get(baseURL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	assert.Equal(t, "ok", buf.String())
}

// --- Method Not Allowed Test ---

func TestMethodNotAllowed(t *testing.T) {
	_, baseURL := startTestServer(t)

	resp, err := http.Get(baseURL + "/DEV-1/complete")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// --- Ticket ID Mismatch Test ---

func TestTicketIDMismatch(t *testing.T) {
	srv, baseURL := startTestServer(t)

	resp := postJSON(t, baseURL+"/URL-TICKET/complete", map[string]string{
		"status":    "complete",
		"ticket_id": "BODY-TICKET",
		"branch":    "x",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	ev := drainEvent(t, srv)
	assert.Equal(t, "URL-TICKET", ev.TicketID, "URL path ticket ID should be authoritative")
}

// --- Duplicate Callback Test ---

func TestDuplicateCallback(t *testing.T) {
	srv, baseURL := startTestServer(t)

	for range 2 {
		resp := postJSON(t, baseURL+"/DEV-DUP/complete", map[string]string{
			"status":    "complete",
			"ticket_id": "DEV-DUP",
			"branch":    "DEV-DUP/feature",
		})
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	ev1 := drainEvent(t, srv)
	ev2 := drainEvent(t, srv)
	assert.Equal(t, "DEV-DUP", ev1.TicketID)
	assert.Equal(t, "DEV-DUP", ev2.TicketID)
}
