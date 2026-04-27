package orchestrator

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/2bit-software/zombiekit/internal/callback"
	"github.com/2bit-software/zombiekit/internal/linear"
	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test doubles ---

type stubLinear struct {
	mu             sync.Mutex
	calls          []string
	tickets        []linear.Ticket
	pollErr        error
	setStatusErr   error
	removeLabelErr error
	applyLabelErr  error
}

func (s *stubLinear) record(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, method)
}

func (s *stubLinear) getCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.calls))
	copy(out, s.calls)
	return out
}

func (s *stubLinear) PollReadyTickets(_ context.Context, _, _ string) ([]linear.Ticket, error) {
	s.record("PollReadyTickets")
	return s.tickets, s.pollErr
}
func (s *stubLinear) GetTicket(_ context.Context, _ string) (*linear.Ticket, error) {
	s.record("GetTicket")
	return nil, nil
}
func (s *stubLinear) SetTicketStatus(_ context.Context, _, _ string) error {
	s.record("SetTicketStatus")
	return s.setStatusErr
}
func (s *stubLinear) ApplyLabel(_ context.Context, _, _ string) error {
	s.record("ApplyLabel")
	return s.applyLabelErr
}
func (s *stubLinear) RemoveLabel(_ context.Context, _, _ string) error {
	s.record("RemoveLabel")
	return s.removeLabelErr
}
func (s *stubLinear) CreateTicket(_ context.Context, _ linear.CreateTicketInput) (*linear.Ticket, error) {
	s.record("CreateTicket")
	return nil, nil
}
func (s *stubLinear) UploadAttachment(_ context.Context, _ string, _ linear.AttachmentInput) error {
	s.record("UploadAttachment")
	return nil
}
func (s *stubLinear) PostComment(_ context.Context, _, _ string) error {
	s.record("PostComment")
	return nil
}

type stubWorktree struct {
	mu        sync.Mutex
	calls     []string
	basePath  string
	createErr error
	deleteErr error
}

func (s *stubWorktree) record(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, method)
}

func (s *stubWorktree) getCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.calls))
	copy(out, s.calls)
	return out
}

func (s *stubWorktree) CreateWorktree(_ context.Context, ticketID, _ string) (string, error) {
	s.record("CreateWorktree")
	if s.createErr != nil {
		return "", s.createErr
	}
	path := filepath.Join(s.basePath, ticketID)
	_ = os.MkdirAll(path, 0o755)
	return path, nil
}
func (s *stubWorktree) DeleteWorktree(_ context.Context, _ string) error {
	s.record("DeleteWorktree")
	return s.deleteErr
}
func (s *stubWorktree) CleanBranch(_ context.Context, _ string) error {
	s.record("CleanBranch")
	return nil
}
func (s *stubWorktree) PushBranch(_ context.Context, _, _ string) error {
	s.record("PushBranch")
	return nil
}

type stubSession struct {
	mu       sync.Mutex
	calls    []string
	spawnErr error
	killErr  error
}

func (s *stubSession) record(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, method)
}

func (s *stubSession) getCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.calls))
	copy(out, s.calls)
	return out
}

func (s *stubSession) SpawnSession(_ context.Context, ticketID, _, _ string, _ map[string]string, _ string) (string, error) {
	s.record("SpawnSession")
	if s.spawnErr != nil {
		return "", s.spawnErr
	}
	return "session-" + ticketID, nil
}
func (s *stubSession) KillSession(_ context.Context, _ string) error {
	s.record("KillSession")
	return s.killErr
}
func (s *stubSession) SessionExists(_ context.Context, _ string) (bool, error) {
	s.record("SessionExists")
	return false, nil
}

type stubState struct {
	mu             sync.Mutex
	calls          []string
	jobs           map[string]*state.Job
	acquireResult  bool
	acquireErr     error
	createJobErr   error
	releaseSlotErr error
}

func newStubState() *stubState {
	return &stubState{
		jobs:          make(map[string]*state.Job),
		acquireResult: true,
	}
}

func (s *stubState) record(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, method)
}

func (s *stubState) getCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.calls))
	copy(out, s.calls)
	return out
}

func (s *stubState) Migrate(_ context.Context) error { return nil }
func (s *stubState) Close() error                    { return nil }
func (s *stubState) GetJob(_ context.Context, _, ticketID string) (*state.Job, error) {
	s.record("GetJob")
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[ticketID], nil
}
func (s *stubState) CreateJob(_ context.Context, ticketID, worktreePath, session, _ string) error {
	s.record("CreateJob")
	if s.createJobErr != nil {
		return s.createJobErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[ticketID] = &state.Job{TicketID: ticketID, WorktreePath: worktreePath, CmuxSession: session}
	return nil
}
func (s *stubState) ListJobsByStatus(_ context.Context, _ string, _ ...string) ([]state.Job, error) {
	return nil, nil
}
func (s *stubState) SetJobStatus(_ context.Context, _, _, _ string) error { return nil }
func (s *stubState) SetPR(_ context.Context, _, _ string, _ int64) error  { return nil }
func (s *stubState) GetJobByPR(_ context.Context, _ string, _ int64) (*state.Job, error) {
	return nil, nil
}
func (s *stubState) GetCommentWatermark(_ context.Context, _ string, _ int64) (int64, error) {
	return 0, nil
}
func (s *stubState) SetCommentWatermark(_ context.Context, _ string, _ int64, _ int64) error {
	return nil
}
func (s *stubState) TryAcquireSlot(_ context.Context, _ string, _ int) (bool, error) {
	s.record("TryAcquireSlot")
	return s.acquireResult, s.acquireErr
}
func (s *stubState) ReleaseSlot(_ context.Context, _ string) error {
	s.record("ReleaseSlot")
	return s.releaseSlotErr
}
func (s *stubState) ResetAllSlots(_ context.Context) (int, error)                 { return 0, nil }
func (s *stubState) ListAllJobs(_ context.Context) ([]state.Job, error)           { return nil, nil }
func (s *stubState) DeleteJob(_ context.Context, _, _ string) error               { return nil }
func (s *stubState) ListSlots(_ context.Context) ([]state.ConcurrencySlot, error) { return nil, nil }

// capturingSessionManager captures SpawnSession args for assertion.
type capturingSessionManager struct {
	stubSession
	onSpawn func(env map[string]string, prompt string)
}

func (c *capturingSessionManager) SpawnSession(_ context.Context, ticketID, _, _ string, env map[string]string, prompt string) (string, error) {
	if c.onSpawn != nil {
		c.onSpawn(env, prompt)
	}
	return "session-" + ticketID, nil
}

// --- Helpers ---

func testTicket(id, identifier, title, desc string) linear.Ticket {
	return linear.Ticket{
		ID: id, Identifier: identifier, Title: title, Description: desc,
		Status: "Todo", Labels: []string{labelAIReady}, TeamID: "team-1",
	}
}

func buildRunner(t *testing.T, sl *stubLinear, sw *stubWorktree, ss *stubSession, st *stubState) *ProjectRunner {
	t.Helper()
	cfg := ProjectConfig{
		ID:               "test-project",
		CallbackPort:     9999,
		PollInterval:     Duration{50 * time.Millisecond},
		ConcurrencyLimit: 1,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	events := make(chan callback.Event, 8)
	return NewProjectRunner(cfg, st, sl, nil, sw, ss, events, false, sandbox.Config{}, logger)
}

// --- T007: Happy path and concurrency tests ---

func TestLinearPoller_SingleTicket(t *testing.T) {
	sl := &stubLinear{tickets: []linear.Ticket{testTicket("uuid-1", "DEV-100", "Do stuff", "ticket body")}}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	p.pollAndProcess(context.Background())

	// Verify full pipeline call order
	assert.Equal(t, []string{"PollReadyTickets"}, sl.getCalls()[:1])
	assert.Contains(t, st.getCalls(), "GetJob")
	assert.Contains(t, st.getCalls(), "TryAcquireSlot")
	assert.Contains(t, sw.getCalls(), "CreateWorktree")
	assert.Contains(t, ss.getCalls(), "SpawnSession")
	assert.Contains(t, st.getCalls(), "CreateJob")
	assert.Contains(t, sl.getCalls(), "SetTicketStatus")
	assert.Contains(t, sl.getCalls(), "RemoveLabel")

	// Verify job was created
	assert.NotNil(t, st.jobs["DEV-100"])
}

func TestLinearPoller_TicketFileWritten(t *testing.T) {
	desc := "## Implement feature X\n\nDo the thing."
	sl := &stubLinear{tickets: []linear.Ticket{testTicket("uuid-1", "DEV-101", "Feature X", desc)}}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	p.pollAndProcess(context.Background())

	ticketFile := filepath.Join(sw.basePath, "DEV-101", ".ai", "ticket.md")
	content, err := os.ReadFile(ticketFile)
	require.NoError(t, err)
	assert.Equal(t, desc, string(content))
}

func TestLinearPoller_CallbackURL(t *testing.T) {
	// Use a capturing session manager to verify the env map
	var capturedEnv map[string]string
	cs := &capturingSessionManager{
		onSpawn: func(env map[string]string, _ string) {
			capturedEnv = env
		},
	}
	sl := &stubLinear{tickets: []linear.Ticket{testTicket("uuid-1", "DEV-102", "Test", "body")}}
	sw := &stubWorktree{basePath: t.TempDir()}
	st := newStubState()

	cfg := ProjectConfig{
		ID:               "test-project",
		CallbackPort:     9999,
		PollInterval:     Duration{50 * time.Millisecond},
		ConcurrencyLimit: 1,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	events := make(chan callback.Event, 8)
	p := NewProjectRunner(cfg, st, sl, nil, sw, cs, events, false, sandbox.Config{}, logger)
	p.pollAndProcess(context.Background())

	require.NotNil(t, capturedEnv)
	assert.Equal(t, "http://localhost:9999/project/test-project/DEV-102", capturedEnv["WORK_CALLBACK_URL"])
}

func TestLinearPoller_ConcurrencyLimit(t *testing.T) {
	tickets := []linear.Ticket{
		testTicket("uuid-1", "DEV-200", "Task A", "body A"),
		testTicket("uuid-2", "DEV-201", "Task B", "body B"),
	}
	sl := &stubLinear{tickets: tickets}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()

	// After first ticket acquires, second should fail
	st.acquireResult = true // first call succeeds

	p := buildRunner(t, sl, sw, ss, st)
	// Process first ticket
	err := p.processTicket(context.Background(), tickets[0])
	require.NoError(t, err)
	assert.NotNil(t, st.jobs["DEV-200"])

	// Now set slot to full
	st.acquireResult = false
	err = p.processTicket(context.Background(), tickets[1])
	require.NoError(t, err) // not an error, just deferred
	assert.Nil(t, st.jobs["DEV-201"])
}

func TestLinearPoller_ConcurrencyMultiPoll(t *testing.T) {
	ticket1 := testTicket("uuid-1", "DEV-300", "Task A", "body A")
	ticket2 := testTicket("uuid-2", "DEV-301", "Task B", "body B")

	sl := &stubLinear{tickets: []linear.Ticket{ticket1, ticket2}}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	// First poll: only 1 slot, so first ticket gets it
	st.acquireResult = true
	err := p.processTicket(context.Background(), ticket1)
	require.NoError(t, err)
	assert.NotNil(t, st.jobs["DEV-300"])

	// Second ticket deferred (slot full)
	st.acquireResult = false
	err = p.processTicket(context.Background(), ticket2)
	require.NoError(t, err)
	assert.Nil(t, st.jobs["DEV-301"])

	// Simulate slot release
	st.acquireResult = true
	err = p.processTicket(context.Background(), ticket2)
	require.NoError(t, err)
	assert.NotNil(t, st.jobs["DEV-301"])
}

func TestLinearPoller_SkipExistingJob(t *testing.T) {
	ticket := testTicket("uuid-1", "DEV-400", "Already running", "body")
	sl := &stubLinear{tickets: []linear.Ticket{ticket}}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	// Pre-populate a job
	st.jobs["DEV-400"] = &state.Job{TicketID: "DEV-400"}

	p := buildRunner(t, sl, sw, ss, st)
	err := p.processTicket(context.Background(), ticket)

	require.NoError(t, err)
	// Should NOT have called TryAcquireSlot or anything downstream
	assert.NotContains(t, st.getCalls(), "TryAcquireSlot")
	assert.Empty(t, sw.getCalls())
	assert.Empty(t, ss.getCalls())
}

// --- T008: Rollback and failure tests ---

func TestLinearPoller_RollbackOnSpawnFailure(t *testing.T) {
	ticket := testTicket("uuid-1", "DEV-500", "Spawn fails", "body")
	sl := &stubLinear{tickets: []linear.Ticket{ticket}}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{spawnErr: assert.AnError}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	err := p.processTicket(context.Background(), ticket)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "spawn session")

	// Rollback: worktree deleted, slot released, no session to kill
	assert.Contains(t, sw.getCalls(), "DeleteWorktree")
	assert.Contains(t, st.getCalls(), "ReleaseSlot")
	// No KillSession because SpawnSession failed (sessionRef is empty)
	assert.NotContains(t, ss.getCalls(), "KillSession")

	// SetTicketStatus should NOT have been called
	assert.NotContains(t, sl.getCalls(), "SetTicketStatus")

	// FR-013: needs-attention should be applied
	assert.Contains(t, sl.getCalls(), "ApplyLabel")
}

func TestLinearPoller_RollbackOnCreateJobFailure(t *testing.T) {
	ticket := testTicket("uuid-1", "DEV-501", "Job create fails", "body")
	sl := &stubLinear{tickets: []linear.Ticket{ticket}}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	st.createJobErr = assert.AnError
	p := buildRunner(t, sl, sw, ss, st)

	err := p.processTicket(context.Background(), ticket)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create job")

	// Rollback: session killed, worktree deleted, slot released
	assert.Contains(t, ss.getCalls(), "KillSession")
	assert.Contains(t, sw.getCalls(), "DeleteWorktree")
	assert.Contains(t, st.getCalls(), "ReleaseSlot")

	// SetTicketStatus should NOT have been called
	assert.NotContains(t, sl.getCalls(), "SetTicketStatus")
}

func TestLinearPoller_RollbackOnWorktreeFailure(t *testing.T) {
	ticket := testTicket("uuid-1", "DEV-502", "Worktree fails", "body")
	sl := &stubLinear{tickets: []linear.Ticket{ticket}}
	sw := &stubWorktree{basePath: t.TempDir(), createErr: assert.AnError}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	err := p.processTicket(context.Background(), ticket)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create worktree")

	// Rollback: slot released, NO worktree delete (creation failed), no session
	assert.Contains(t, st.getCalls(), "ReleaseSlot")
	assert.NotContains(t, sw.getCalls(), "DeleteWorktree")
	assert.Empty(t, ss.getCalls())
}

func TestLinearPoller_NeedsAttentionOnFailure(t *testing.T) {
	ticket := testTicket("uuid-1", "DEV-503", "Needs attention", "body")
	sl := &stubLinear{tickets: []linear.Ticket{ticket}}
	sw := &stubWorktree{basePath: t.TempDir(), createErr: assert.AnError}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	_ = p.processTicket(context.Background(), ticket)

	// FR-013: needs-attention label applied, ai-ready removed
	calls := sl.getCalls()
	assert.Contains(t, calls, "ApplyLabel")
	assert.Contains(t, calls, "RemoveLabel")
}

func TestLinearPoller_LinearFailureAfterJob(t *testing.T) {
	ticket := testTicket("uuid-1", "DEV-504", "Linear fails after", "body")
	sl := &stubLinear{
		tickets:      []linear.Ticket{ticket},
		setStatusErr: assert.AnError,
	}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	err := p.processTicket(context.Background(), ticket)

	// Should NOT return error -- job is running
	require.NoError(t, err)
	// Job should exist
	assert.NotNil(t, st.jobs["DEV-504"])
	// SetTicketStatus was called (and failed, but we continued)
	assert.Contains(t, sl.getCalls(), "SetTicketStatus")
}

func TestLinearPoller_RemoveLabelFailureAfterJob(t *testing.T) {
	ticket := testTicket("uuid-1", "DEV-505", "RemoveLabel fails", "body")
	sl := &stubLinear{
		tickets:        []linear.Ticket{ticket},
		removeLabelErr: assert.AnError,
	}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	err := p.processTicket(context.Background(), ticket)

	// Should NOT return error -- job is running
	require.NoError(t, err)
	assert.NotNil(t, st.jobs["DEV-505"])
}

// --- T009: Edge case and shutdown tests ---

func TestLinearPoller_GracefulShutdown(t *testing.T) {
	// Two tickets, cancel context after first processes
	ticket1 := testTicket("uuid-1", "DEV-600", "First", "body")
	ticket2 := testTicket("uuid-2", "DEV-601", "Second", "body")
	sl := &stubLinear{tickets: []linear.Ticket{ticket1, ticket2}}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	ctx, cancel := context.WithCancel(context.Background())

	// Process first ticket
	err := p.processTicket(ctx, ticket1)
	require.NoError(t, err)
	assert.NotNil(t, st.jobs["DEV-600"])

	// Cancel before second ticket
	cancel()

	// pollAndProcess checks context between tickets
	// Since we already processed ticket1 manually, verify that
	// pollAndProcess with cancelled context skips work
	st2 := newStubState()
	sl2 := &stubLinear{tickets: []linear.Ticket{ticket2}}
	p2 := buildRunner(t, sl2, sw, ss, st2)
	p2.pollAndProcess(ctx)

	// With cancelled context, PollReadyTickets is still called but
	// the ticket loop checks ctx.Err() and returns
	assert.Nil(t, st2.jobs["DEV-601"])
}

func TestLinearPoller_ShutdownBetweenPolls(t *testing.T) {
	sl := &stubLinear{tickets: []linear.Ticket{testTicket("uuid-1", "DEV-700", "Never runs", "body")}}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := p.linearPoller(ctx)

	assert.NoError(t, err)
	// No polling should have occurred
	assert.Empty(t, sl.getCalls())
}

func TestLinearPoller_EmptyPoll(t *testing.T) {
	sl := &stubLinear{tickets: []linear.Ticket{}}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	p.pollAndProcess(context.Background())

	assert.Equal(t, []string{"PollReadyTickets"}, sl.getCalls())
	assert.Empty(t, sw.getCalls())
	assert.Empty(t, ss.getCalls())
	// Only GetJob-related calls should NOT appear (no tickets to check)
	assert.NotContains(t, st.getCalls(), "TryAcquireSlot")
}

func TestLinearPoller_PollError(t *testing.T) {
	sl := &stubLinear{pollErr: assert.AnError}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	// Should not panic
	p.pollAndProcess(context.Background())

	assert.Equal(t, []string{"PollReadyTickets"}, sl.getCalls())
	assert.Empty(t, sw.getCalls())
}

func TestLinearPoller_EmptyDescription(t *testing.T) {
	ticket := testTicket("uuid-1", "DEV-800", "Empty desc", "")
	sl := &stubLinear{tickets: []linear.Ticket{ticket}}
	sw := &stubWorktree{basePath: t.TempDir()}
	ss := &stubSession{}
	st := newStubState()
	p := buildRunner(t, sl, sw, ss, st)

	err := p.processTicket(context.Background(), ticket)
	require.NoError(t, err)

	ticketFile := filepath.Join(sw.basePath, "DEV-800", ".ai", "ticket.md")
	content, err := os.ReadFile(ticketFile)
	require.NoError(t, err)
	assert.Equal(t, "", string(content))
}

// --- shortTitle tests ---

func TestShortTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Watcher 1 — ready ticket pickup", "watcher-1-ready-ticket-pickup"},
		{"Simple", "simple"},
		{"UPPERCASE STUFF", "uppercase-stuff"},
		{"special!@#chars", "special-chars"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, shortTitle(tt.input))
		})
	}
}
