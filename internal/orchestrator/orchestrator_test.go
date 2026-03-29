package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zombiekit/brains/internal/github"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/state"
)

// mockStore records calls and can simulate errors.
type mockStore struct {
	calls        []string
	listJobsErr  error
	setStatusErr error
}

func (m *mockStore) Migrate(_ context.Context) error { return nil }
func (m *mockStore) Close() error                    { return nil }

func (m *mockStore) CreateJob(_ context.Context, _, _, _ string) error { return nil }
func (m *mockStore) GetJob(_ context.Context, _ string) (*state.Job, error) {
	return nil, nil
}

func (m *mockStore) ListJobsByStatus(_ context.Context, _ ...string) ([]state.Job, error) {
	m.calls = append(m.calls, "ListJobsByStatus")
	if m.listJobsErr != nil {
		return nil, m.listJobsErr
	}
	return []state.Job{}, nil
}

func (m *mockStore) SetJobStatus(_ context.Context, _ string, _ string) error {
	m.calls = append(m.calls, "SetJobStatus")
	return m.setStatusErr
}

func (m *mockStore) SetPR(_ context.Context, _ string, _ int64) error { return nil }
func (m *mockStore) GetJobByPR(_ context.Context, _ int64) (*state.Job, error) {
	return nil, nil
}

func (m *mockStore) GetCommentWatermark(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

func (m *mockStore) SetCommentWatermark(_ context.Context, _ int64, _ int64) error {
	return nil
}

func (m *mockStore) TryAcquireSlot(_ context.Context, _ string, _ int) (bool, error) {
	return true, nil
}

func (m *mockStore) ReleaseSlot(_ context.Context, _ string) error { return nil }

func (m *mockStore) ResetAllSlots(_ context.Context) (int, error) {
	m.calls = append(m.calls, "ResetAllSlots")
	return 0, nil
}

func setupLogger(t *testing.T) {
	t.Helper()
	logging.ResetLogger()
	logging.InitLogger("debug", false, nil)
	t.Cleanup(logging.ResetLogger)
}

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

func TestRun_ReconciliationRuns(t *testing.T) {
	setupLogger(t)
	store := &mockStore{}
	gh := &github.MockClient{
		ListOpenPRsFn: func(_ context.Context, _ string) ([]github.PRSummary, error) {
			return nil, nil
		},
	}
	orch := New(testConfig(t), store, &stubLinear{}, gh, &stubWorktree{basePath: t.TempDir()}, &stubSession{})

	// Run in a goroutine and send SIGINT to stop quickly
	// Since we can't easily send signals in tests, we use a port-0 callback
	// server which will start, and then we verify reconciliation ran.
	// We need to stop the orchestrator — the simplest approach is to verify
	// the mock was called, since reconciliation runs before services.
	errCh := make(chan error, 1)
	go func() { errCh <- orch.Run() }()

	// Give services time to start
	time.Sleep(200 * time.Millisecond)

	// Verify reconciliation ran
	assert.Contains(t, store.calls, "ListJobsByStatus")
}

func TestRun_ReconciliationFailure_PreventsServices(t *testing.T) {
	setupLogger(t)
	store := &mockStore{
		listJobsErr: fmt.Errorf("database locked"),
	}
	cfg := testConfig(t)
	orch := New(cfg, store, nil, nil, nil, nil)

	err := orch.Run()

	require.Error(t, err)
	assert.ErrorContains(t, err, "reconciliation")
	assert.ErrorContains(t, err, "database locked")
}
