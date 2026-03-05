package proxy

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProxy_LocalOnly(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := &ProxyConfig{}

	p, err := NewProxy(cfg, logger)
	require.NoError(t, err)
	assert.Nil(t, p.Connection())
}

func TestNewProxy_WithServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := &ProxyConfig{
		ServerURL: "http://localhost:8080",
	}

	p, err := NewProxy(cfg, logger)
	require.NoError(t, err)
	assert.NotNil(t, p.Connection())
	assert.True(t, p.Connection().IsConfigured())
}

func TestNewProxy_RegistersAllTools(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := &ProxyConfig{}

	p, err := NewProxy(cfg, logger)
	require.NoError(t, err)

	expectedTools := []string{
		"code-reasoning",
		"workflow-compose",
		"initiative",
		"profile-save",
		"brains-connection-status",
		"recall-list-conversations",
		"recall-read-conversation",
		"profile-compose",
		"profile-list",
	}

	for _, name := range expectedTools {
		_, dispatchErr := p.router.Dispatch(t.Context(), name, map[string]any{})
		// Should not return "unknown tool" error -- handler exists
		if dispatchErr != nil {
			assert.NotContains(t, dispatchErr.Error(), "unknown tool",
				"tool %q should be registered", name)
		}
	}
}

func TestNewProxy_NoStickyMemory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := &ProxyConfig{}

	p, err := NewProxy(cfg, logger)
	require.NoError(t, err)

	_, err = p.router.Dispatch(t.Context(), "stickymemory", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}
