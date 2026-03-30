package admin

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/2bit-software/zombiekit/internal/state"
)

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

	createTestJob(t, store, "DEV-1", "proj-1")
	createTestJob(t, store, "DEV-2", "proj-1")
	createTestJob(t, store, "DEV-3", "proj-1")

	jobs, err := svc.ListJobs(ctx, JobFilter{})
	require.NoError(t, err)
	assert.Len(t, jobs, 3)
}

func TestListJobs_FilterByStatus(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	createTestJob(t, store, "DEV-1", "proj-1")
	createTestJob(t, store, "DEV-2", "proj-1")
	require.NoError(t, store.SetJobStatus(ctx, "DEV-2", state.StatusInProgress))

	jobs, err := svc.ListJobs(ctx, JobFilter{Statuses: []string{state.StatusQueued}})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "DEV-1", jobs[0].TicketID)
}

func TestListJobs_FilterMultipleStatuses(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	createTestJob(t, store, "DEV-1", "proj-1")
	createTestJob(t, store, "DEV-2", "proj-1")
	createTestJob(t, store, "DEV-3", "proj-1")
	require.NoError(t, store.SetJobStatus(ctx, "DEV-2", state.StatusInProgress))
	require.NoError(t, store.SetJobStatus(ctx, "DEV-3", state.StatusClosed))

	jobs, err := svc.ListJobs(ctx, JobFilter{Statuses: []string{state.StatusQueued, state.StatusInProgress}})
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
}

func TestListJobs_Empty(t *testing.T) {
	svc, _ := setupTestService(t)

	jobs, err := svc.ListJobs(context.Background(), JobFilter{})
	require.NoError(t, err)
	assert.Empty(t, jobs)
	assert.NotNil(t, jobs)
}

func TestGetJob_Exists(t *testing.T) {
	svc, store := setupTestService(t)
	createTestJob(t, store, "DEV-100", "proj-1")

	job, err := svc.GetJob(context.Background(), "DEV-100")
	require.NoError(t, err)
	assert.Equal(t, "DEV-100", job.TicketID)
	assert.Equal(t, "proj-1", job.ProjectID)
	assert.Equal(t, state.StatusQueued, job.Status)
}

func TestGetJob_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.GetJob(context.Background(), "DEV-999")
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrJobNotFound))
}

func TestDeleteJob_Success(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()
	createTestJob(t, store, "DEV-100", "proj-1")

	result, err := svc.DeleteJob(ctx, "DEV-100")
	require.NoError(t, err)
	assert.Equal(t, "DEV-100", result.Job.TicketID)

	// Verify job is gone
	job, err := store.GetJob(ctx, "DEV-100")
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestDeleteJob_ReleasesSlot(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	createTestJob(t, store, "DEV-100", "proj-1")
	acquired, err := store.TryAcquireSlot(ctx, "proj-1", 1)
	require.NoError(t, err)
	require.True(t, acquired)

	result, err := svc.DeleteJob(ctx, "DEV-100")
	require.NoError(t, err)
	assert.True(t, result.SlotReleased)

	// Verify slot was released — can acquire again
	acquired, err = store.TryAcquireSlot(ctx, "proj-1", 1)
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestDeleteJob_NoSlotRelease_EmptyProjectID(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	createTestJob(t, store, "DEV-100", "")

	result, err := svc.DeleteJob(ctx, "DEV-100")
	require.NoError(t, err)
	assert.False(t, result.SlotReleased)
}

func TestDeleteJob_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.DeleteJob(context.Background(), "DEV-999")
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrJobNotFound))
}

func TestSetJobStatus_Valid(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()
	createTestJob(t, store, "DEV-100", "proj-1")

	for _, status := range state.ValidStatuses {
		err := svc.SetJobStatus(ctx, "DEV-100", status)
		require.NoError(t, err, "status: %s", status)

		job, err := store.GetJob(ctx, "DEV-100")
		require.NoError(t, err)
		assert.Equal(t, status, job.Status)
	}
}

func TestSetJobStatus_Invalid(t *testing.T) {
	svc, store := setupTestService(t)
	createTestJob(t, store, "DEV-100", "proj-1")

	err := svc.SetJobStatus(context.Background(), "DEV-100", "banana")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
	assert.Contains(t, err.Error(), "banana")
}

func TestSetJobStatus_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.SetJobStatus(context.Background(), "DEV-999", state.StatusQueued)
	require.Error(t, err)
	assert.True(t, errors.Is(err, state.ErrJobNotFound))
}

func TestListSlots(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	_, err := store.TryAcquireSlot(ctx, "proj-1", 2)
	require.NoError(t, err)

	slots, err := svc.ListSlots(ctx)
	require.NoError(t, err)
	require.Len(t, slots, 1)
	assert.Equal(t, "proj-1", slots[0].ProjectID)
	assert.Equal(t, 1, slots[0].ActiveCount)
	assert.Equal(t, 2, slots[0].SlotLimit)
}

func TestResetSlots(t *testing.T) {
	svc, store := setupTestService(t)
	ctx := context.Background()

	_, err := store.TryAcquireSlot(ctx, "proj-1", 2)
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
