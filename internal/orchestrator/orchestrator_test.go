package orchestrator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(t *testing.T) *Config {
	t.Helper()
	return &Config{
		LinearAPIKey:     "test-key",
		GitHubToken:      "test-token",
		CallbackPort:     0, // port 0 = OS picks a free port (avoid conflicts)
		WorktreesRoot:    t.TempDir(),
		DBPath:           t.TempDir() + "/state.db",
		ConcurrencyLimit: 1,
		PollInterval:     100 * time.Millisecond,
		LogLevel:         "debug",
		ShutdownTimeout:  5 * time.Second,
		BotUsername:      "test-bot",
		TrackingLabel:    "ai-managed",
	}
}

func TestRun_Deprecated(t *testing.T) {
	cfg := testConfig(t)
	orch := New(cfg, nil, nil, nil, nil, nil)

	err := orch.Run()

	require.Error(t, err)
	assert.ErrorContains(t, err, "deprecated")
}
