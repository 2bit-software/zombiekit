package orchestrator

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zombiekit/brains/internal/github"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/state"
)

// --- PR watcher test doubles ---

// prStubState is a StateStore stub tailored for PR watcher tests. It supports
// configurable ListJobsByStatus results, records SetJobStatus and ReleaseSlot
// calls, and allows injecting errors per method.
type prStubState struct {
	stubState
	listJobs        []state.Job
	listJobsErr     error
	setJobStatusErr error

	setJobStatusCalls []string
	releaseSlotCalls  int
}

func (s *prStubState) ListJobsByStatus(_ context.Context, _ ...string) ([]state.Job, error) {
	s.record("ListJobsByStatus")
	return s.listJobs, s.listJobsErr
}

func (s *prStubState) SetJobStatus(_ context.Context, ticketID, status string) error {
	s.record("SetJobStatus")
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setJobStatusCalls = append(s.setJobStatusCalls, ticketID+":"+status)
	return s.setJobStatusErr
}

func (s *prStubState) ReleaseSlot(_ context.Context, _ string) error {
	s.record("ReleaseSlot")
	s.mu.Lock()
	defer s.mu.Unlock()
	s.releaseSlotCalls++
	return s.releaseSlotErr
}

// prStubLinear records SetTicketStatus calls with their arguments.
type prStubLinear struct {
	stubLinear
	setStatusCalls []string
}

func (s *prStubLinear) SetTicketStatus(_ context.Context, ticketID, status string) error {
	s.record("SetTicketStatus")
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setStatusCalls = append(s.setStatusCalls, ticketID+":"+status)
	return s.setStatusErr
}

// --- Helpers ---

func buildPRWatcherOrch(t *testing.T, gh *github.MockClient, wt *stubWorktree, lc *prStubLinear, st *prStubState) *Orchestrator {
	t.Helper()
	setupLogger(t)
	cfg := &Config{
		PollInterval:         50 * time.Millisecond,
		ProjectID:            "test-project",
		ClosedPRTicketStatus: "cancelled",
	}
	return New(cfg, st, lc, gh, wt, &stubSession{})
}

// --- Tests ---

func TestPRWatcher_MergedPR(t *testing.T) {
	gh := &github.MockClient{
		IsMergedFn: func(_ context.Context, _ int) (bool, error) { return true, nil },
	}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	st := &prStubState{
		stubState: *newStubState(),
		listJobs: []state.Job{
			{TicketID: "DEV-100", WorktreePath: "/tmp/wt/DEV-100", PRNumber: prNum(42), Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)

	o.pollPRLifecycle(context.Background(), setupTestLogger(t))

	assert.Contains(t, wt.getCalls(), "DeleteWorktree")
	assert.Equal(t, []string{"DEV-100:done"}, lc.setStatusCalls)
	assert.Equal(t, []string{"DEV-100:" + state.StatusClosed}, st.setJobStatusCalls)
	assert.Equal(t, 1, st.releaseSlotCalls)
}

func TestPRWatcher_ClosedPR(t *testing.T) {
	gh := &github.MockClient{
		IsMergedFn: func(_ context.Context, _ int) (bool, error) { return false, nil },
		IsClosedFn: func(_ context.Context, _ int) (bool, error) { return true, nil },
	}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	st := &prStubState{
		stubState: *newStubState(),
		listJobs: []state.Job{
			{TicketID: "DEV-200", WorktreePath: "/tmp/wt/DEV-200", PRNumber: prNum(99), Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)
	o.cfg.ClosedPRTicketStatus = "backlog"

	o.pollPRLifecycle(context.Background(), setupTestLogger(t))

	assert.Equal(t, []string{"DEV-200:backlog"}, lc.setStatusCalls)
	assert.Equal(t, []string{"DEV-200:" + state.StatusClosed}, st.setJobStatusCalls)
	assert.Equal(t, 1, st.releaseSlotCalls)
}

func TestPRWatcher_SkipNoPR(t *testing.T) {
	gh := &github.MockClient{}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	st := &prStubState{
		stubState: *newStubState(),
		listJobs: []state.Job{
			{TicketID: "DEV-300", WorktreePath: "/tmp/wt/DEV-300", PRNumber: nil, Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)

	o.pollPRLifecycle(context.Background(), setupTestLogger(t))

	assert.Empty(t, gh.Calls, "should not call GitHub for jobs without PR")
	assert.Empty(t, wt.getCalls())
	assert.Empty(t, lc.setStatusCalls)
	assert.Empty(t, st.setJobStatusCalls)
	assert.Equal(t, 0, st.releaseSlotCalls)
}

func TestPRWatcher_OpenPR(t *testing.T) {
	gh := &github.MockClient{
		IsMergedFn: func(_ context.Context, _ int) (bool, error) { return false, nil },
		IsClosedFn: func(_ context.Context, _ int) (bool, error) { return false, nil },
	}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	st := &prStubState{
		stubState: *newStubState(),
		listJobs: []state.Job{
			{TicketID: "DEV-400", WorktreePath: "/tmp/wt/DEV-400", PRNumber: prNum(10), Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)

	o.pollPRLifecycle(context.Background(), setupTestLogger(t))

	assert.Len(t, gh.Calls, 2, "should call IsMerged and IsClosed")
	assert.Empty(t, wt.getCalls(), "should not clean up open PR")
	assert.Empty(t, st.setJobStatusCalls)
}

func TestPRWatcher_PartialFailure_Worktree(t *testing.T) {
	gh := &github.MockClient{
		IsMergedFn: func(_ context.Context, _ int) (bool, error) { return true, nil },
	}
	wt := &stubWorktree{deleteErr: errors.New("worktree gone")}
	lc := &prStubLinear{}
	st := &prStubState{
		stubState: *newStubState(),
		listJobs: []state.Job{
			{TicketID: "DEV-500", WorktreePath: "/tmp/wt/DEV-500", PRNumber: prNum(50), Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)

	o.pollPRLifecycle(context.Background(), setupTestLogger(t))

	assert.Contains(t, wt.getCalls(), "DeleteWorktree")
	assert.Equal(t, []string{"DEV-500:done"}, lc.setStatusCalls, "ticket status should still be set")
	assert.Equal(t, []string{"DEV-500:" + state.StatusClosed}, st.setJobStatusCalls, "job status should still be set")
	assert.Equal(t, 1, st.releaseSlotCalls, "slot should still be released")
}

func TestPRWatcher_PartialFailure_Linear(t *testing.T) {
	gh := &github.MockClient{
		IsMergedFn: func(_ context.Context, _ int) (bool, error) { return true, nil },
	}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	lc.setStatusErr = errors.New("linear down")
	st := &prStubState{
		stubState: *newStubState(),
		listJobs: []state.Job{
			{TicketID: "DEV-600", WorktreePath: "/tmp/wt/DEV-600", PRNumber: prNum(60), Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)

	o.pollPRLifecycle(context.Background(), setupTestLogger(t))

	assert.Contains(t, wt.getCalls(), "DeleteWorktree")
	assert.Equal(t, []string{"DEV-600:" + state.StatusClosed}, st.setJobStatusCalls, "job status should still be set despite Linear failure")
	assert.Equal(t, 1, st.releaseSlotCalls, "slot should still be released despite Linear failure")
}

func TestPRWatcher_PartialFailure_SetJobStatus(t *testing.T) {
	gh := &github.MockClient{
		IsMergedFn: func(_ context.Context, _ int) (bool, error) { return true, nil },
	}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	st := &prStubState{
		stubState:       *newStubState(),
		setJobStatusErr: errors.New("db error"),
		listJobs: []state.Job{
			{TicketID: "DEV-700", WorktreePath: "/tmp/wt/DEV-700", PRNumber: prNum(70), Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)

	o.pollPRLifecycle(context.Background(), setupTestLogger(t))

	assert.Equal(t, 1, st.releaseSlotCalls, "slot should still be released despite SetJobStatus failure")
}

func TestPRWatcher_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	gh := &github.MockClient{}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	st := &prStubState{
		stubState: *newStubState(),
		listJobs: []state.Job{
			{TicketID: "DEV-800", WorktreePath: "/tmp/wt/DEV-800", PRNumber: prNum(80), Status: state.StatusQueued},
			{TicketID: "DEV-801", WorktreePath: "/tmp/wt/DEV-801", PRNumber: prNum(81), Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)

	// Cancel before poll starts
	cancel()

	o.pollPRLifecycle(ctx, setupTestLogger(t))

	assert.Empty(t, gh.Calls, "should not check any PRs after context cancellation")
}

func TestPRWatcher_MultiplePRs(t *testing.T) {
	gh := &github.MockClient{
		IsMergedFn: func(_ context.Context, prNumber int) (bool, error) {
			return prNumber == 11, nil // only PR 11 is merged
		},
		IsClosedFn: func(_ context.Context, prNumber int) (bool, error) {
			return prNumber == 12, nil // only PR 12 is closed
		},
	}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	st := &prStubState{
		stubState: *newStubState(),
		listJobs: []state.Job{
			{TicketID: "DEV-A", WorktreePath: "/tmp/wt/DEV-A", PRNumber: prNum(11), Status: state.StatusQueued},
			{TicketID: "DEV-B", WorktreePath: "/tmp/wt/DEV-B", PRNumber: prNum(12), Status: state.StatusQueued},
			{TicketID: "DEV-C", WorktreePath: "/tmp/wt/DEV-C", PRNumber: prNum(13), Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)

	o.pollPRLifecycle(context.Background(), setupTestLogger(t))

	require.Len(t, st.setJobStatusCalls, 2, "two PRs should be cleaned up")
	assert.Contains(t, st.setJobStatusCalls, "DEV-A:"+state.StatusClosed)
	assert.Contains(t, st.setJobStatusCalls, "DEV-B:"+state.StatusClosed)
	assert.Equal(t, 2, st.releaseSlotCalls)

	// Check ticket statuses: merge = "done", close = "cancelled"
	assert.Contains(t, lc.setStatusCalls, "DEV-A:done")
	assert.Contains(t, lc.setStatusCalls, "DEV-B:cancelled")
}

func TestPRWatcher_Idempotent(t *testing.T) {
	gh := &github.MockClient{
		IsMergedFn: func(_ context.Context, _ int) (bool, error) { return true, nil },
	}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	st := &prStubState{
		stubState: *newStubState(),
		listJobs: []state.Job{
			{TicketID: "DEV-900", WorktreePath: "/tmp/wt/DEV-900", PRNumber: prNum(90), Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)
	logger := setupTestLogger(t)

	// First poll — cleanup happens
	o.pollPRLifecycle(context.Background(), logger)
	assert.Equal(t, 1, st.releaseSlotCalls)

	// Second poll — no more queued jobs (simulate StatusClosed by returning empty list)
	st.listJobs = []state.Job{}
	st.setJobStatusCalls = nil
	st.releaseSlotCalls = 0
	lc.setStatusCalls = nil

	o.pollPRLifecycle(context.Background(), logger)

	assert.Empty(t, st.setJobStatusCalls, "no cleanup on second poll")
	assert.Equal(t, 0, st.releaseSlotCalls, "no slot release on second poll")
}

func TestPRWatcher_MergedAndClosed(t *testing.T) {
	gh := &github.MockClient{
		IsMergedFn: func(_ context.Context, _ int) (bool, error) { return true, nil },
		IsClosedFn: func(_ context.Context, _ int) (bool, error) { return true, nil },
	}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	st := &prStubState{
		stubState: *newStubState(),
		listJobs: []state.Job{
			{TicketID: "DEV-MC", WorktreePath: "/tmp/wt/DEV-MC", PRNumber: prNum(77), Status: state.StatusQueued},
		},
	}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)

	o.pollPRLifecycle(context.Background(), setupTestLogger(t))

	// IsMerged is checked first — merge path should be taken
	assert.Equal(t, []string{"DEV-MC:done"}, lc.setStatusCalls, "should use merge status, not close status")

	// IsClosed should NOT be called (short-circuited by merge detection)
	for _, call := range gh.Calls {
		assert.NotEqual(t, "IsClosed", call.Method, "IsClosed should not be called when IsMerged returns true")
	}
}

func TestPRWatcher_ServiceFunc(t *testing.T) {
	gh := &github.MockClient{
		IsMergedFn: func(_ context.Context, _ int) (bool, error) { return false, nil },
		IsClosedFn: func(_ context.Context, _ int) (bool, error) { return false, nil },
	}
	wt := &stubWorktree{}
	lc := &prStubLinear{}
	st := &prStubState{stubState: *newStubState()}
	o := buildPRWatcherOrch(t, gh, wt, lc, st)

	ctx, cancel := context.WithCancel(context.Background())
	svc := o.NewPRWatcher()

	done := make(chan error, 1)
	go func() { done <- svc(ctx) }()

	// Let it run for a tick
	time.Sleep(100 * time.Millisecond)
	cancel()

	err := <-done
	assert.NoError(t, err, "NewPRWatcher should return nil on context cancellation")
}

// --- Test logger helper ---

func setupTestLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return logging.Logger().With(slog.String("test", t.Name()))
}
