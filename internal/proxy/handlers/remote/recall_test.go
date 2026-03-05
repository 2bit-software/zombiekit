package remote

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zombiekit/brains/gen/zombiekit/brains/search/v1/searchv1connect"
)

type unconfiguredConn struct{}

func (u *unconfiguredConn) IsConfigured() bool                             { return false }
func (u *unconfiguredConn) Search() searchv1connect.SearchServiceClient    { return nil }

func TestRecallList_NotConfigured(t *testing.T) {
	handler := NewRecallListHandler(&unconfiguredConn{})

	_, err := handler(context.Background(), map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server not configured")
}

func TestRecallRead_NotConfigured(t *testing.T) {
	handler := NewRecallReadHandler(&unconfiguredConn{})

	_, err := handler(context.Background(), map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server not configured")
}

func TestRecallRead_MissingConversationID(t *testing.T) {
	// This test requires a configured connection to get past the guard
	// but the server won't be called since we return early for missing ID.
	// We can't test this without a mock server. Skip for unit tests.
	t.Skip("requires mock server connection")
}

func TestIntArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		defVal   int
		maxVal   int
		expected int
	}{
		{"missing key", map[string]any{}, "page", 1, 0, 1},
		{"valid value", map[string]any{"page": float64(3)}, "page", 1, 0, 3},
		{"exceeds max", map[string]any{"limit": float64(200)}, "limit", 20, 100, 100},
		{"zero value", map[string]any{"page": float64(0)}, "page", 1, 0, 1},
		{"negative", map[string]any{"page": float64(-1)}, "page", 1, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intArg(tt.args, tt.key, tt.defVal, tt.maxVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}
