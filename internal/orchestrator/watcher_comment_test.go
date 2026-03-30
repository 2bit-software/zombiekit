package orchestrator

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/2bit-software/zombiekit/internal/github"
	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/2bit-software/zombiekit/internal/state"
)

// --- Comment watcher test doubles ---

type commentStubState struct {
	mu              sync.Mutex
	calls           []string
	jobsByPR        map[int64]*state.Job
	watermarks      map[int64]int64
	acquireResult   bool
	acquireErr      error
	releaseSlotErr  error
	setWatermarkErr error
}

func newCommentStubState() *commentStubState {
	return &commentStubState{
		jobsByPR:      make(map[int64]*state.Job),
		watermarks:    make(map[int64]int64),
		acquireResult: true,
	}
}

func (s *commentStubState) record(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, method)
}

func (s *commentStubState) getCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.calls))
	copy(out, s.calls)
	return out
}

func (s *commentStubState) Migrate(_ context.Context) error { return nil }
func (s *commentStubState) Close() error                    { return nil }
func (s *commentStubState) CreateJob(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (s *commentStubState) GetJob(_ context.Context, _ string) (*state.Job, error) {
	return nil, nil
}
func (s *commentStubState) ListJobsByStatus(_ context.Context, _ ...string) ([]state.Job, error) {
	return nil, nil
}
func (s *commentStubState) SetJobStatus(_ context.Context, _, _ string) error { return nil }
func (s *commentStubState) SetPR(_ context.Context, _ string, _ int64) error  { return nil }

func (s *commentStubState) GetJobByPR(_ context.Context, prNumber int64) (*state.Job, error) {
	s.record("GetJobByPR")
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobsByPR[prNumber], nil
}

func (s *commentStubState) GetCommentWatermark(_ context.Context, prNumber int64) (int64, error) {
	s.record("GetCommentWatermark")
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.watermarks[prNumber], nil
}

func (s *commentStubState) SetCommentWatermark(_ context.Context, prNumber int64, commentID int64) error {
	s.record("SetCommentWatermark")
	if s.setWatermarkErr != nil {
		return s.setWatermarkErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.watermarks[prNumber] = commentID
	return nil
}

func (s *commentStubState) TryAcquireSlot(_ context.Context, _ string, _ int) (bool, error) {
	s.record("TryAcquireSlot")
	return s.acquireResult, s.acquireErr
}

func (s *commentStubState) ReleaseSlot(_ context.Context, _ string) error {
	s.record("ReleaseSlot")
	return s.releaseSlotErr
}

func (s *commentStubState) ResetAllSlots(_ context.Context) (int, error)       { return 0, nil }
func (s *commentStubState) ListAllJobs(_ context.Context) ([]state.Job, error) { return nil, nil }
func (s *commentStubState) DeleteJob(_ context.Context, _ string) error        { return nil }
func (s *commentStubState) ListSlots(_ context.Context) ([]state.ConcurrencySlot, error) {
	return nil, nil
}

// commentTestFixture bundles all test dependencies for the comment watcher.
type commentTestFixture struct {
	cfg        *Config
	gh         *github.MockClient
	store      *commentStubState
	sessions   *commentStubSession
	dispatcher *CommentDispatcher
	orch       *Orchestrator
	logger     *slog.Logger
}

type commentStubSession struct {
	mu       sync.Mutex
	calls    []string
	spawnErr error
}

func (s *commentStubSession) record(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, method)
}

func (s *commentStubSession) getCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.calls))
	copy(out, s.calls)
	return out
}

func (s *commentStubSession) SpawnSession(_ context.Context, ticketID, _, _ string, _ map[string]string) (string, error) {
	s.record("SpawnSession")
	if s.spawnErr != nil {
		return "", s.spawnErr
	}
	return "session-" + ticketID, nil
}

func (s *commentStubSession) KillSession(_ context.Context, _ string) error {
	s.record("KillSession")
	return nil
}

func (s *commentStubSession) SessionExists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func newCommentFixture(t *testing.T) *commentTestFixture {
	t.Helper()
	setupLogger(t)

	cfg := &Config{
		CallbackPort:     9999,
		PollInterval:     50 * time.Millisecond,
		ConcurrencyLimit: 1,
		ProjectID:        "test-project",
		TrackingLabel:    "ai-managed",
		BotUsername:      "test-bot",
	}

	store := newCommentStubState()
	gh := &github.MockClient{}
	sess := &commentStubSession{}
	logger := logging.Logger()
	dispatcher := NewCommentDispatcher(logger)
	orch := New(cfg, store, nil, gh, nil, sess)

	return &commentTestFixture{
		cfg:        cfg,
		gh:         gh,
		store:      store,
		sessions:   sess,
		dispatcher: dispatcher,
		orch:       orch,
		logger:     logger,
	}
}

// --- Tests ---

func TestPollDetectsNewComments(t *testing.T) {
	f := newCommentFixture(t)

	prNum := 10
	prNumber64 := int64(prNum)

	f.store.jobsByPR[prNumber64] = &state.Job{
		TicketID:     "DEV-100",
		WorktreePath: t.TempDir(),
		Status:       state.StatusInProgress,
	}
	f.store.watermarks[prNumber64] = 0

	f.gh.ListOpenPRsFn = func(_ context.Context, _ string) ([]github.PRSummary, error) {
		return []github.PRSummary{{Number: prNum, Title: "Test PR", Labels: []string{"ai-managed"}}}, nil
	}

	comments := []github.PRComment{
		{ID: 101, Author: "alice", Body: "fix this"},
		{ID: 102, Author: "bob", Body: "and this too"},
	}
	f.gh.GetCommentsSinceFn = func(_ context.Context, pr int, kind github.CommentKind, afterID int64) ([]github.PRComment, error) {
		assert.Equal(t, prNum, pr)
		assert.Equal(t, github.CommentKindReview, kind)
		assert.Equal(t, int64(0), afterID)
		return comments, nil
	}

	f.orch.pollComments(context.Background(), f.dispatcher, f.logger)

	// GetCommentsSince was called with the right args (validated in the Fn above).
	var found bool
	for _, c := range f.gh.Calls {
		if c.Method == "GetCommentsSince" {
			found = true
			require.Len(t, c.Args, 3)
			assert.Equal(t, github.CommentKindReview, c.Args[1])
			assert.Equal(t, int64(0), c.Args[2])
		}
	}
	require.True(t, found, "GetCommentsSince was not called")

	// Comments should be enqueued in the dispatcher.
	q := f.dispatcher.GetQueue(prNum)
	require.NotNil(t, q, "queue should have been created for PR")
}

func TestBotCommentFiltered(t *testing.T) {
	f := newCommentFixture(t)

	prNum := 20
	prNumber64 := int64(prNum)

	f.store.jobsByPR[prNumber64] = &state.Job{
		TicketID:     "DEV-200",
		WorktreePath: t.TempDir(),
		Status:       state.StatusInProgress,
	}

	f.gh.ListOpenPRsFn = func(_ context.Context, _ string) ([]github.PRSummary, error) {
		return []github.PRSummary{{Number: prNum}}, nil
	}

	f.gh.GetCommentsSinceFn = func(_ context.Context, _ int, _ github.CommentKind, _ int64) ([]github.PRComment, error) {
		return []github.PRComment{
			{ID: 201, Author: "alice", Body: "please fix"},
			{ID: 202, Author: "test-bot", Body: "I already did"},
		}, nil
	}

	f.orch.pollComments(context.Background(), f.dispatcher, f.logger)

	q := f.dispatcher.GetQueue(prNum)
	require.NotNil(t, q, "queue should exist")

	// Only the non-bot comment should be in the queue.
	select {
	case c := <-q.comments:
		assert.Equal(t, int64(201), c.ID)
		assert.Equal(t, "alice", c.Author)
	default:
		t.Fatal("expected at least one comment in queue")
	}

	// No second comment should be present.
	select {
	case c := <-q.comments:
		t.Fatalf("unexpected second comment in queue: %+v", c)
	default:
		// good
	}
}

func TestTerminalJobSkipped(t *testing.T) {
	f := newCommentFixture(t)

	prNum := 30
	prNumber64 := int64(prNum)

	f.store.jobsByPR[prNumber64] = &state.Job{
		TicketID:     "DEV-300",
		WorktreePath: t.TempDir(),
		Status:       state.StatusNeedsAttention,
	}

	f.gh.ListOpenPRsFn = func(_ context.Context, _ string) ([]github.PRSummary, error) {
		return []github.PRSummary{{Number: prNum}}, nil
	}
	f.gh.GetCommentsSinceFn = func(_ context.Context, _ int, _ github.CommentKind, _ int64) ([]github.PRComment, error) {
		t.Fatal("GetCommentsSince should not be called for terminal job")
		return nil, nil
	}

	f.orch.pollComments(context.Background(), f.dispatcher, f.logger)

	// No calls to GetCommentsSince should have been made.
	for _, c := range f.gh.Calls {
		assert.NotEqual(t, "GetCommentsSince", c.Method)
	}
}

func TestPRReaping(t *testing.T) {
	f := newCommentFixture(t)

	// Pre-create a queue for PR #42.
	_, cancel := context.WithCancel(context.Background())
	f.dispatcher.CreateQueue(42, cancel)

	require.NotNil(t, f.dispatcher.GetQueue(42), "queue should exist before poll")

	// ListOpenPRs returns empty -- PR 42 is gone.
	f.gh.ListOpenPRsFn = func(_ context.Context, _ string) ([]github.PRSummary, error) {
		return []github.PRSummary{}, nil
	}

	f.orch.pollComments(context.Background(), f.dispatcher, f.logger)

	assert.Nil(t, f.dispatcher.GetQueue(42), "queue for PR 42 should have been reaped")
}

func TestSerialProcessing(t *testing.T) {
	f := newCommentFixture(t)

	prNum := 50
	prNumber64 := int64(prNum)
	worktree := t.TempDir()

	job := &state.Job{
		TicketID:     "DEV-500",
		WorktreePath: worktree,
		Status:       state.StatusInProgress,
	}
	f.store.jobsByPR[prNumber64] = job

	// Track spawn calls so we can verify ordering.
	var spawnMu sync.Mutex
	spawnCount := 0
	spawnCalled := make(chan struct{}, 2)

	// Replace sessions with a custom implementation that signals on spawn.
	customSess := &serialTestSession{
		spawnCalled: spawnCalled,
		spawnCount:  &spawnCount,
		spawnMu:     &spawnMu,
	}
	f.orch.sessions = customSess

	// Mock GitHub calls needed by runPRQueue.
	f.gh.IsMergedFn = func(_ context.Context, _ int) (bool, error) { return false, nil }
	f.gh.IsClosedFn = func(_ context.Context, _ int) (bool, error) { return false, nil }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := f.dispatcher.CreateQueue(prNum, cancel)

	// Enqueue 2 comments.
	q.comments <- github.PRComment{ID: 501, Author: "alice", Body: "first"}
	q.comments <- github.PRComment{ID: 502, Author: "bob", Body: "second"}

	go f.orch.runPRQueue(ctx, f.dispatcher, prNum, job, q, f.logger)

	// Wait for first spawn.
	select {
	case <-spawnCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first SpawnSession")
	}

	// At this point, the goroutine is blocking on the session result.
	// Second SpawnSession should NOT have been called yet.
	spawnMu.Lock()
	firstCount := spawnCount
	spawnMu.Unlock()
	assert.Equal(t, 1, firstCount, "only one SpawnSession should have been called before notification")

	// Notify the first session completed.
	f.dispatcher.NotifyResult("DEV-500", SessionResult{Kind: SessionResolved, TicketID: "DEV-500", PRNumber: prNum})

	// Wait for second spawn.
	select {
	case <-spawnCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second SpawnSession")
	}

	spawnMu.Lock()
	secondCount := spawnCount
	spawnMu.Unlock()
	assert.Equal(t, 2, secondCount, "second SpawnSession should be called after first session completes")

	// Notify second session completed to let goroutine finish cleanly.
	f.dispatcher.NotifyResult("DEV-500", SessionResult{Kind: SessionResolved, TicketID: "DEV-500", PRNumber: prNum})
}

// serialTestSession is a session manager stub that signals on each SpawnSession call.
type serialTestSession struct {
	spawnCalled chan struct{}
	spawnCount  *int
	spawnMu     *sync.Mutex
}

func (s *serialTestSession) SpawnSession(_ context.Context, ticketID, _, _ string, _ map[string]string) (string, error) {
	s.spawnMu.Lock()
	*s.spawnCount++
	s.spawnMu.Unlock()
	s.spawnCalled <- struct{}{}
	return "session-" + ticketID, nil
}

func (s *serialTestSession) KillSession(_ context.Context, _ string) error { return nil }

func (s *serialTestSession) SessionExists(_ context.Context, _ string) (bool, error) {
	return false, nil
}
