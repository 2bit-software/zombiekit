package proxy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_Dispatch(t *testing.T) {
	r := NewRouter()
	r.Register("test-tool", func(_ context.Context, args map[string]any) (string, error) {
		return "ok", nil
	})

	result, err := r.Dispatch(context.Background(), "test-tool", nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestRouter_DispatchUnknownTool(t *testing.T) {
	r := NewRouter()

	_, err := r.Dispatch(context.Background(), "nonexistent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestRouter_DuplicateRegistrationPanics(t *testing.T) {
	r := NewRouter()
	r.Register("test-tool", func(_ context.Context, _ map[string]any) (string, error) {
		return "", nil
	})

	assert.Panics(t, func() {
		r.Register("test-tool", func(_ context.Context, _ map[string]any) (string, error) {
			return "", nil
		})
	})
}
