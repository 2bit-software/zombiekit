package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/2bit-software/zombiekit/internal/callback"
	"github.com/2bit-software/zombiekit/internal/github"
	"github.com/2bit-software/zombiekit/internal/linear"
	"github.com/2bit-software/zombiekit/internal/state"
)

// --- test helpers ---

type routerFixture struct {
	events   chan callback.Event
	store    *routerMockStore
	gh       *github.MockClient
	lc       *linear.MockClient
	archiver *mockArchiver
	auditor  *mockAuditor
	cfg      *Config
	router   *Router
	worktree string
}

func newRouterFixture(t *testing.T) *routerFixture {
	t.Helper()
	worktree := t.TempDir()
	events := make(chan callback.Event, 8)
	store := &routerMockStore{}
	gh := &github.MockClient{}
	lc := &linear.MockClient{}
	arch := &mockArchiver{}
	aud := &mockAuditor{}
	cfg := &Config{
		ProjectID:     "test-project",
		BaseBranch:    "main",
		TrackingLabel: "ai-managed",
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	wt := &stubWorktree{basePath: t.TempDir()}
	r := NewRouter(events, store, gh, lc, wt, arch, aud, nil, cfg, logger)

	return &routerFixture{
		events:   events,
		store:    store,
		gh:       gh,
		lc:       lc,
		archiver: arch,
		auditor:  aud,
		cfg:      cfg,
		router:   r,
		worktree: worktree,
	}
}

func (f *routerFixture) writePRDescription(t *testing.T, content string) {
	t.Helper()
	aiDir := filepath.Join(f.worktree, ".ai")
	require.NoError(t, os.MkdirAll(aiDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(aiDir, "pr-description.md"), []byte(content), 0o644))
}

func (f *routerFixture) runSingleEvent(t *testing.T, evt callback.Event) {
	t.Helper()
	f.events <- evt
	close(f.events)
	err := f.router.Run(context.Background())
	require.NoError(t, err)
}

func prNum(n int64) *int64 { return &n }

// --- mock store for router tests ---

type routerMockStore struct {
	getJobFn          func(ctx context.Context, ticketID string) (*state.Job, error)
	setJobStatusFn    func(ctx context.Context, ticketID string, status string) error
	setPRFn           func(ctx context.Context, ticketID string, prNumber int64) error
	setCommentWaterFn func(ctx context.Context, prNumber int64, commentID int64) error
	releaseSlotFn     func(ctx context.Context, projectID string) error
	calls             []string
}

func (m *routerMockStore) Migrate(_ context.Context) error                      { return nil }
func (m *routerMockStore) Close() error                                         { return nil }
func (m *routerMockStore) CreateJob(_ context.Context, _, _, _, _ string) error { return nil }
func (m *routerMockStore) ListJobsByStatus(_ context.Context, _ ...string) ([]state.Job, error) {
	return nil, nil
}
func (m *routerMockStore) TryAcquireSlot(_ context.Context, _ string, _ int) (bool, error) {
	return true, nil
}
func (m *routerMockStore) ResetAllSlots(_ context.Context) (int, error)       { return 0, nil }
func (m *routerMockStore) ListAllJobs(_ context.Context) ([]state.Job, error) { return nil, nil }
func (m *routerMockStore) DeleteJob(_ context.Context, _ string) error        { return nil }
func (m *routerMockStore) ListSlots(_ context.Context) ([]state.ConcurrencySlot, error) {
	return nil, nil
}
func (m *routerMockStore) GetJobByPR(_ context.Context, _ int64) (*state.Job, error) {
	return nil, nil
}
func (m *routerMockStore) GetCommentWatermark(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

func (m *routerMockStore) GetJob(ctx context.Context, ticketID string) (*state.Job, error) {
	m.calls = append(m.calls, "GetJob")
	if m.getJobFn != nil {
		return m.getJobFn(ctx, ticketID)
	}
	return nil, nil
}

func (m *routerMockStore) SetJobStatus(ctx context.Context, ticketID string, status string) error {
	m.calls = append(m.calls, "SetJobStatus")
	if m.setJobStatusFn != nil {
		return m.setJobStatusFn(ctx, ticketID, status)
	}
	return nil
}

func (m *routerMockStore) SetPR(ctx context.Context, ticketID string, prNumber int64) error {
	m.calls = append(m.calls, "SetPR")
	if m.setPRFn != nil {
		return m.setPRFn(ctx, ticketID, prNumber)
	}
	return nil
}

func (m *routerMockStore) SetCommentWatermark(ctx context.Context, prNumber int64, commentID int64) error {
	m.calls = append(m.calls, "SetCommentWatermark")
	if m.setCommentWaterFn != nil {
		return m.setCommentWaterFn(ctx, prNumber, commentID)
	}
	return nil
}

func (m *routerMockStore) ReleaseSlot(ctx context.Context, projectID string) error {
	m.calls = append(m.calls, "ReleaseSlot")
	if m.releaseSlotFn != nil {
		return m.releaseSlotFn(ctx, projectID)
	}
	return nil
}

// --- mock archiver/auditor ---

type mockArchiver struct {
	calls []string
	err   error
}

func (m *mockArchiver) Archive(_ context.Context, ticketID string, _ callback.EventKind) error {
	m.calls = append(m.calls, "Archive:"+ticketID)
	return m.err
}

type mockAuditor struct {
	calls []string
	err   error
}

func (m *mockAuditor) Audit(_ context.Context, ticketID string, _ callback.EventKind) error {
	m.calls = append(m.calls, "Audit:"+ticketID)
	return m.err
}

// Compile-time interface checks
var (
	_ Archiver         = (*mockArchiver)(nil)
	_ Auditor          = (*mockAuditor)(nil)
	_ state.StateStore = (*routerMockStore)(nil)
)

// --- CompletionEvent tests ---

func TestRouter_Complete_HappyPath(t *testing.T) {
	f := newRouterFixture(t)
	f.writePRDescription(t, "PR body here")

	f.store.getJobFn = func(_ context.Context, _ string) (*state.Job, error) {
		return &state.Job{TicketID: "DEV-100", WorktreePath: f.worktree, Status: state.StatusInProgress}, nil
	}
	f.lc.GetTicketFn = func(_ context.Context, _ string) (*linear.Ticket, error) {
		return &linear.Ticket{Identifier: "DEV-100", Title: "Do stuff"}, nil
	}
	f.gh.CreatePRFn = func(_ context.Context, input github.CreatePRInput) (int, error) {
		assert.Equal(t, "DEV-100: Do stuff", input.Title)
		assert.Equal(t, "PR body here", input.Body)
		assert.Equal(t, "feat/branch", input.Head)
		assert.Equal(t, "main", input.Base)
		return 42, nil
	}
	f.gh.ApplyLabelFn = func(_ context.Context, pr int, label string) error {
		assert.Equal(t, 42, pr)
		assert.Equal(t, "ai-managed", label)
		return nil
	}

	f.runSingleEvent(t, callback.Event{
		Kind:     callback.EventComplete,
		TicketID: "DEV-100",
		Branch:   "feat/branch",
	})

	assert.Contains(t, f.store.calls, "SetPR")
	assert.Equal(t, []string{"Archive:DEV-100"}, f.archiver.calls)
	assert.Equal(t, []string{"Audit:DEV-100"}, f.auditor.calls)
}

func TestRouter_Complete_MissingPRDescription(t *testing.T) {
	f := newRouterFixture(t)
	// No pr-description.md written

	f.store.getJobFn = func(_ context.Context, _ string) (*state.Job, error) {
		return &state.Job{TicketID: "DEV-100", WorktreePath: f.worktree, Status: state.StatusInProgress}, nil
	}
	f.lc.SetTicketStatusFn = func(_ context.Context, _ string, status string) error {
		assert.Equal(t, "needs-attention", status)
		return nil
	}

	f.runSingleEvent(t, callback.Event{
		Kind:     callback.EventComplete,
		TicketID: "DEV-100",
		Branch:   "feat/branch",
	})

	assert.Contains(t, f.store.calls, "SetJobStatus")
	assert.Contains(t, f.lc.Calls[0].Method, "SetTicketStatus")
}

func TestRouter_Complete_UnknownTicket(t *testing.T) {
	f := newRouterFixture(t)
	// store returns nil, nil for unknown ticket (default behavior)

	f.runSingleEvent(t, callback.Event{
		Kind:     callback.EventComplete,
		TicketID: "DEV-999",
		Branch:   "feat/branch",
	})

	// No PR creation, no needs-attention, just discarded
	assert.Empty(t, f.gh.Calls)
	assert.Empty(t, f.lc.Calls)
	assert.Empty(t, f.archiver.calls)
}

func TestRouter_Complete_CreatePRFailure(t *testing.T) {
	f := newRouterFixture(t)
	f.writePRDescription(t, "body")

	f.store.getJobFn = func(_ context.Context, _ string) (*state.Job, error) {
		return &state.Job{TicketID: "DEV-100", WorktreePath: f.worktree}, nil
	}
	f.lc.GetTicketFn = func(_ context.Context, _ string) (*linear.Ticket, error) {
		return &linear.Ticket{Identifier: "DEV-100", Title: "Stuff"}, nil
	}
	f.gh.CreatePRFn = func(_ context.Context, _ github.CreatePRInput) (int, error) {
		return 0, fmt.Errorf("branch not found")
	}
	f.lc.SetTicketStatusFn = func(_ context.Context, _ string, _ string) error { return nil }

	f.runSingleEvent(t, callback.Event{
		Kind:     callback.EventComplete,
		TicketID: "DEV-100",
		Branch:   "feat/gone",
	})

	assert.Contains(t, f.store.calls, "SetJobStatus")
}

// --- FailureEvent tests ---

func TestRouter_Failed_HappyPath(t *testing.T) {
	f := newRouterFixture(t)

	f.store.getJobFn = func(_ context.Context, _ string) (*state.Job, error) {
		return &state.Job{TicketID: "DEV-100", WorktreePath: f.worktree}, nil
	}
	f.lc.SetTicketStatusFn = func(_ context.Context, _ string, _ string) error { return nil }
	f.lc.PostCommentFn = func(_ context.Context, id string, body string) error {
		assert.Equal(t, "DEV-100", id)
		assert.Equal(t, "tests failing", body)
		return nil
	}

	f.runSingleEvent(t, callback.Event{
		Kind:     callback.EventFailed,
		TicketID: "DEV-100",
		Reason:   "tests failing",
	})

	assert.Contains(t, f.store.calls, "SetJobStatus")
	assert.Contains(t, f.store.calls, "ReleaseSlot")
	assert.Equal(t, []string{"Archive:DEV-100"}, f.archiver.calls)
	assert.Empty(t, f.auditor.calls) // No audit for failures
}

func TestRouter_Failed_UnknownTicket(t *testing.T) {
	f := newRouterFixture(t)
	// store returns nil, nil (default)

	f.lc.SetTicketStatusFn = func(_ context.Context, _ string, _ string) error { return nil }
	f.lc.PostCommentFn = func(_ context.Context, _ string, _ string) error { return nil }

	f.runSingleEvent(t, callback.Event{
		Kind:     callback.EventFailed,
		TicketID: "DEV-999",
		Reason:   "boom",
	})

	// SetJobStatus should NOT be called (no job)
	assert.NotContains(t, f.store.calls, "SetJobStatus")
	// Slot should still be released
	assert.Contains(t, f.store.calls, "ReleaseSlot")
}

func TestRouter_Failed_LinearAPIFailure_SlotStillReleased(t *testing.T) {
	f := newRouterFixture(t)

	f.store.getJobFn = func(_ context.Context, _ string) (*state.Job, error) {
		return &state.Job{TicketID: "DEV-100", WorktreePath: f.worktree}, nil
	}
	f.lc.SetTicketStatusFn = func(_ context.Context, _ string, _ string) error {
		return fmt.Errorf("linear unavailable")
	}
	f.lc.PostCommentFn = func(_ context.Context, _ string, _ string) error {
		return fmt.Errorf("linear unavailable")
	}

	f.runSingleEvent(t, callback.Event{
		Kind:     callback.EventFailed,
		TicketID: "DEV-100",
		Reason:   "tests failing",
	})

	// Slot must still be released despite Linear failures
	assert.Contains(t, f.store.calls, "ReleaseSlot")
}

// --- CommentResolvedEvent tests ---

func TestRouter_CommentResolved_HappyPath(t *testing.T) {
	f := newRouterFixture(t)
	f.writePRDescription(t, "Updated PR body")

	f.store.getJobFn = func(_ context.Context, _ string) (*state.Job, error) {
		return &state.Job{TicketID: "DEV-100", WorktreePath: f.worktree, PRNumber: prNum(42)}, nil
	}
	f.gh.UpdatePRBodyFn = func(_ context.Context, pr int, body string) error {
		assert.Equal(t, 42, pr)
		assert.Equal(t, "Updated PR body", body)
		return nil
	}
	f.gh.PostCommentReplyFn = func(_ context.Context, pr int, kind github.CommentKind, cid int64, body string) (int64, error) {
		assert.Equal(t, 42, pr)
		assert.Equal(t, github.CommentKindReview, kind)
		assert.Equal(t, int64(777), cid)
		assert.Equal(t, "Fixed the issue", body)
		return 888, nil
	}

	f.runSingleEvent(t, callback.Event{
		Kind:       callback.EventCommentResolved,
		TicketID:   "DEV-100",
		CommentID:  "777",
		Resolution: "Fixed the issue",
	})

	assert.Contains(t, f.store.calls, "SetCommentWatermark")
	assert.Equal(t, []string{"Archive:DEV-100"}, f.archiver.calls)
	assert.Equal(t, []string{"Audit:DEV-100"}, f.auditor.calls)
}

func TestRouter_CommentResolved_NilPRNumber(t *testing.T) {
	f := newRouterFixture(t)

	f.store.getJobFn = func(_ context.Context, _ string) (*state.Job, error) {
		return &state.Job{TicketID: "DEV-100", WorktreePath: f.worktree, PRNumber: nil}, nil
	}
	f.lc.SetTicketStatusFn = func(_ context.Context, _ string, _ string) error { return nil }

	f.runSingleEvent(t, callback.Event{
		Kind:       callback.EventCommentResolved,
		TicketID:   "DEV-100",
		CommentID:  "777",
		Resolution: "Fixed",
	})

	assert.Contains(t, f.store.calls, "SetJobStatus")
	assert.Empty(t, f.gh.Calls) // No GitHub calls
}

func TestRouter_CommentResolved_InvalidCommentID(t *testing.T) {
	f := newRouterFixture(t)

	f.store.getJobFn = func(_ context.Context, _ string) (*state.Job, error) {
		return &state.Job{TicketID: "DEV-100", WorktreePath: f.worktree, PRNumber: prNum(42)}, nil
	}
	f.lc.SetTicketStatusFn = func(_ context.Context, _ string, _ string) error { return nil }

	f.runSingleEvent(t, callback.Event{
		Kind:       callback.EventCommentResolved,
		TicketID:   "DEV-100",
		CommentID:  "not-a-number",
		Resolution: "Fixed",
	})

	assert.Contains(t, f.store.calls, "SetJobStatus")
	assert.Empty(t, f.gh.Calls)
}

// --- Lifecycle tests ---

func TestRouter_ChannelClosed_ReturnsNil(t *testing.T) {
	f := newRouterFixture(t)
	close(f.events)

	err := f.router.Run(context.Background())
	assert.NoError(t, err)
}

func TestRouter_ContextCancelled_ReturnsNil(t *testing.T) {
	f := newRouterFixture(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- f.router.Run(ctx) }()

	// Give router time to start
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("router did not stop after context cancellation")
	}
}
