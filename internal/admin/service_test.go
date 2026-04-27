package admin

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/2bit-software/zombiekit/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSessionManager implements cmux.SessionManager for tests.
type stubSessionManager struct {
	killErr error
	killed  []string
}

func (s *stubSessionManager) SpawnSession(_ context.Context, _, _, _ string, _ map[string]string, _ string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (s *stubSessionManager) KillSession(_ context.Context, ticketID string) error {
	s.killed = append(s.killed, ticketID)
	return s.killErr
}

func (s *stubSessionManager) SessionExists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// stubWorktreeManager implements worktree.Manager for tests.
type stubWorktreeManager struct {
	deleteErr error
	deleted   []string
}

func (s *stubWorktreeManager) CreateWorktree(_ context.Context, _, _ string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (s *stubWorktreeManager) DeleteWorktree(_ context.Context, path string) error {
	s.deleted = append(s.deleted, path)
	return s.deleteErr
}

func (s *stubWorktreeManager) CleanBranch(_ context.Context, _ string) error {
	return nil
}

func (s *stubWorktreeManager) PushBranch(_ context.Context, _, _ string) error {
	return nil
}

const testProj = "test-proj"

func setupTestService(t *testing.T) (*Service, *state.SQLiteStore) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := state.NewSQLiteStore(context.Background(), dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return New(store), store
}

func createTestJob(t *testing.T, store *state.SQLiteStore, ticketID, projectID string) {
	t.Helper()
	err := store.CreateJob(context.Background(), ticketID, "/tmp/wt/"+ticketID, "session-"+ticketID, projectID)
	require.NoError(t, err)
}

func TestListJobs_All(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	createTestJob(t, store, "DEV-1", testProj)
	createTestJob(t, store, "DEV-2", testProj)
	createTestJob(t, store, "DEV-3", testProj)

	jobs, err := svc.ListJobs(ctx, testProj, JobFilter{})
	require.NoError(t, err)
	assert.Len(t, jobs, 3)
}

func TestListJobs_FilterByStatus(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	createTestJob(t, store, "DEV-1", testProj)
	createTestJob(t, store, "DEV-2", testProj)
	require.NoError(t, store.SetJobStatus(ctx, testProj, "DEV-2", state.StatusInProgress))

	jobs, err := svc.ListJobs(ctx, testProj, JobFilter{Statuses: []string{state.StatusQueued}})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "DEV-1", jobs[0].TicketID)
}

func TestListJobs_FilterMultipleStatuses(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	createTestJob(t, store, "DEV-1", testProj)
	createTestJob(t, store, "DEV-2", testProj)
	createTestJob(t, store, "DEV-3", testProj)
	require.NoError(t, store.SetJobStatus(ctx, testProj, "DEV-2", state.StatusInProgress))
	require.NoError(t, store.SetJobStatus(ctx, testProj, "DEV-3", state.StatusClosed))

	jobs, err := svc.ListJobs(ctx, testProj, JobFilter{Statuses: []string{state.StatusQueued, state.StatusInProgress}})
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
}

func TestListJobs_Empty(t *testing.T) {
	svc, _ := setupTestService(t)

	jobs, err := svc.ListJobs(context.Background(), testProj, JobFilter{})
	require.NoError(t, err)
	assert.Empty(t, jobs)
	assert.NotNil(t, jobs)
}

func TestGetJob_Exists(t *testing.T) {
	svc, store := setupTestService(t)
	createTestJob(t, store, "DEV-100", testProj)

	job, err := svc.GetJob(context.Background(), testProj, "DEV-100")
	require.NoError(t, err)
	assert.Equal(t, "DEV-100", job.TicketID)
	assert.Equal(t, testProj, job.ProjectID)
	assert.Equal(t, state.StatusQueued, job.Status)
}

func TestGetJob_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.GetJob(context.Background(), testProj, "DEV-999")
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrJobNotFound))
}

func TestDeleteJob_Success(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()
	createTestJob(t, store, "DEV-100", testProj)

	result, err := svc.DeleteJob(ctx, testProj, "DEV-100")
	require.NoError(t, err)
	assert.Equal(t, "DEV-100", result.Job.TicketID)

	// Verify job is gone
	job, err := store.GetJob(ctx, testProj, "DEV-100")
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestDeleteJob_ReleasesSlot(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	createTestJob(t, store, "DEV-100", testProj)
	acquired, err := store.TryAcquireSlot(ctx, testProj, 1)
	require.NoError(t, err)
	require.True(t, acquired)

	result, err := svc.DeleteJob(ctx, testProj, "DEV-100")
	require.NoError(t, err)
	assert.True(t, result.SlotReleased)

	// Verify slot was released — can acquire again
	acquired, err = store.TryAcquireSlot(ctx, testProj, 1)
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestDeleteJob_NoSlotRelease_EmptyProjectID(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	createTestJob(t, store, "DEV-100", "")

	result, err := svc.DeleteJob(ctx, "", "DEV-100")
	require.NoError(t, err)
	assert.False(t, result.SlotReleased)
}

func TestDeleteJob_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.DeleteJob(context.Background(), testProj, "DEV-999")
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrJobNotFound))
}

func TestSetJobStatus_Valid(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()
	createTestJob(t, store, "DEV-100", testProj)

	for _, status := range state.ValidStatuses {
		err := svc.SetJobStatus(ctx, testProj, "DEV-100", status)
		require.NoError(t, err, "status: %s", status)

		job, err := store.GetJob(ctx, testProj, "DEV-100")
		require.NoError(t, err)
		assert.Equal(t, status, job.Status)
	}
}

func TestSetJobStatus_Invalid(t *testing.T) {
	svc, store := setupTestService(t)
	createTestJob(t, store, "DEV-100", testProj)

	err := svc.SetJobStatus(context.Background(), testProj, "DEV-100", "banana")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
	assert.Contains(t, err.Error(), "banana")
}

func TestSetJobStatus_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.SetJobStatus(context.Background(), testProj, "DEV-999", state.StatusQueued)
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrJobNotFound))
}

func TestListSlots(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	_, err := store.TryAcquireSlot(ctx, testProj, 2)
	require.NoError(t, err)

	slots, err := svc.ListSlots(ctx)
	require.NoError(t, err)
	require.Len(t, slots, 1)
	assert.Equal(t, testProj, slots[0].ProjectID)
	assert.Equal(t, 1, slots[0].ActiveCount)
	assert.Equal(t, 2, slots[0].SlotLimit)
}

func TestResetSlots(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	_, err := store.TryAcquireSlot(ctx, testProj, 2)
	require.NoError(t, err)

	n, err := svc.ResetSlots(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	slots, err := svc.ListSlots(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, slots[0].ActiveCount)
}

func TestResetSlots_AlreadyZero(t *testing.T) {
	svc, _ := setupTestService(t)

	n, err := svc.ResetSlots(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestDeleteJob_SessionCleanup(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()
	createTestJob(t, store, "DEV-100", testProj)

	sm := &stubSessionManager{}
	result, err := svc.DeleteJob(ctx, testProj, "DEV-100", WithSessionCleanup(sm))
	require.NoError(t, err)
	assert.True(t, result.SessionKilled)
	assert.Nil(t, result.SessionErr)
	assert.Equal(t, []string{"DEV-100"}, sm.killed)
}

func TestDeleteJob_WorktreeCleanup(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()
	createTestJob(t, store, "DEV-100", testProj)

	wm := &stubWorktreeManager{}
	result, err := svc.DeleteJob(ctx, testProj, "DEV-100", WithWorktreeCleanup(wm))
	require.NoError(t, err)
	assert.True(t, result.WorktreeDeleted)
	assert.Nil(t, result.WorktreeErr)
	assert.Equal(t, []string{"/tmp/wt/DEV-100"}, wm.deleted)
}

func TestDeleteJob_BothCleanups(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()
	createTestJob(t, store, "DEV-100", testProj)

	sm := &stubSessionManager{}
	wm := &stubWorktreeManager{}
	result, err := svc.DeleteJob(ctx, testProj, "DEV-100", WithSessionCleanup(sm), WithWorktreeCleanup(wm))
	require.NoError(t, err)
	assert.True(t, result.SessionKilled)
	assert.True(t, result.WorktreeDeleted)
	assert.Equal(t, []string{"DEV-100"}, sm.killed)
	assert.Equal(t, []string{"/tmp/wt/DEV-100"}, wm.deleted)
}

func TestDeleteJob_SessionCleanupFailure(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()
	createTestJob(t, store, "DEV-100", testProj)

	sm := &stubSessionManager{killErr: errors.New("cmux not running")}
	result, err := svc.DeleteJob(ctx, testProj, "DEV-100", WithSessionCleanup(sm))
	require.NoError(t, err)
	assert.False(t, result.SessionKilled)
	assert.EqualError(t, result.SessionErr, "cmux not running")

	// Job should still be deleted
	job, err := store.GetJob(ctx, testProj, "DEV-100")
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestDeleteJob_WorktreeCleanupFailure(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()
	createTestJob(t, store, "DEV-100", testProj)

	wm := &stubWorktreeManager{deleteErr: errors.New("worktree not found")}
	result, err := svc.DeleteJob(ctx, testProj, "DEV-100", WithWorktreeCleanup(wm))
	require.NoError(t, err)
	assert.False(t, result.WorktreeDeleted)
	assert.EqualError(t, result.WorktreeErr, "worktree not found")

	// Job should still be deleted
	job, err := store.GetJob(ctx, testProj, "DEV-100")
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestDeleteJob_SkipsCleanupWhenFieldsEmpty(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	// Create job with empty worktree path and cmux session
	err := store.CreateJob(ctx, "DEV-100", "", "", testProj)
	require.NoError(t, err)

	sm := &stubSessionManager{}
	wm := &stubWorktreeManager{}
	result, err := svc.DeleteJob(ctx, testProj, "DEV-100", WithSessionCleanup(sm), WithWorktreeCleanup(wm))
	require.NoError(t, err)

	// Cleanup should be skipped — managers never called
	assert.False(t, result.SessionKilled)
	assert.False(t, result.WorktreeDeleted)
	assert.Empty(t, sm.killed)
	assert.Empty(t, wm.deleted)
}
