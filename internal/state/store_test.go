package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface compliance check.
var _ StateStore = (*SQLiteStore)(nil)

const testProj = "test-proj"

func setupTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(context.Background(), dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

func TestNewSQLiteStore_FirstRun(t *testing.T) {
	store := setupTestStore(t)

	expectedTables := []string{"jobs", "comment_watermarks", "concurrency_slots", "schema_migrations"}
	for _, table := range expectedTables {
		var count int
		err := store.DB().QueryRowContext(
			context.Background(),
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "table %s should exist", table)
	}
}

func TestNewSQLiteStore_JobsTableColumns(t *testing.T) {
	store := setupTestStore(t)

	rows, err := store.DB().QueryContext(context.Background(), "PRAGMA table_info(jobs)")
	require.NoError(t, err)
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt *string
		err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk)
		require.NoError(t, err)
		columns[name] = true
	}
	require.NoError(t, rows.Err())

	expected := []string{"ticket_id", "worktree_path", "cmux_session", "pr_number", "status", "created_at", "updated_at", "project_id"}
	for _, col := range expected {
		assert.True(t, columns[col], "jobs table should have column %s", col)
	}
}

func TestNewSQLiteStore_IdempotentRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	ctx := context.Background()

	store1, err := NewSQLiteStore(ctx, dbPath)
	require.NoError(t, err)

	require.NoError(t, store1.CreateJob(ctx, "DEV-999", "/tmp/wt", "session-1", testProj))
	require.NoError(t, store1.Close())

	store2, err := NewSQLiteStore(ctx, dbPath)
	require.NoError(t, err)
	defer store2.Close()

	job, err := store2.GetJob(ctx, testProj, "DEV-999")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "DEV-999", job.TicketID)
}

func TestNewSQLiteStore_EmptyPath(t *testing.T) {
	_, err := NewSQLiteStore(context.Background(), "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidDBPath))
}

func TestNewSQLiteStore_UnwritablePath(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping unwritable path test when running as root")
	}

	_, err := NewSQLiteStore(context.Background(), "/root/forbidden/test.db")
	require.Error(t, err)
}

func TestNewSQLiteStore_PragmasSet(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	var journalMode string
	err := store.DB().QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", journalMode)

	var foreignKeys int
	err = store.DB().QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys)
	require.NoError(t, err)
	assert.Equal(t, 1, foreignKeys)

	var synchronous int
	err = store.DB().QueryRowContext(ctx, "PRAGMA synchronous").Scan(&synchronous)
	require.NoError(t, err)
	assert.Equal(t, 1, synchronous) // NORMAL = 1
}

func TestRunMigrations_Idempotent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.Migrate(ctx)
	require.NoError(t, err)

	var count int
	err = store.DB().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM schema_migrations",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// --- Job CRUD tests ---

func TestCreateJob_AndGetJob(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	before := time.Now().Add(-time.Second)
	err := store.CreateJob(ctx, "DEV-100", "/tmp/worktree", "session-abc", testProj)
	require.NoError(t, err)

	job, err := store.GetJob(ctx, testProj, "DEV-100")
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Equal(t, "DEV-100", job.TicketID)
	assert.Equal(t, "/tmp/worktree", job.WorktreePath)
	assert.Equal(t, "session-abc", job.CmuxSession)
	assert.Equal(t, testProj, job.ProjectID)
	assert.Nil(t, job.PRNumber)
	assert.Equal(t, StatusQueued, job.Status)
	assert.True(t, job.CreatedAt.After(before))
	assert.True(t, job.UpdatedAt.After(before))
}

func TestCreateJob_Duplicate_ReturnsErrJobExists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.CreateJob(ctx, "DEV-100", "/tmp/wt1", "s1", testProj)
	require.NoError(t, err)

	err = store.CreateJob(ctx, "DEV-100", "/tmp/wt2", "s2", testProj)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobExists))

	job, err := store.GetJob(ctx, testProj, "DEV-100")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/wt1", job.WorktreePath)
}

func TestCreateJob_SameTicketDifferentProject(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.CreateJob(ctx, "DEV-100", "/tmp/wt1", "s1", "proj-a")
	require.NoError(t, err)
	err = store.CreateJob(ctx, "DEV-100", "/tmp/wt2", "s2", "proj-b")
	require.NoError(t, err)

	jobA, err := store.GetJob(ctx, "proj-a", "DEV-100")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/wt1", jobA.WorktreePath)

	jobB, err := store.GetJob(ctx, "proj-b", "DEV-100")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/wt2", jobB.WorktreePath)
}

func TestGetJob_NonExistent_ReturnsNil(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	job, err := store.GetJob(ctx, testProj, "DOES-NOT-EXIST")
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestSetPR_UpdatesJob(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.CreateJob(ctx, "DEV-100", "/tmp/wt", "s1", testProj)
	require.NoError(t, err)

	jobBefore, err := store.GetJob(ctx, testProj, "DEV-100")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	err = store.SetPR(ctx, testProj, "DEV-100", 42)
	require.NoError(t, err)

	job, err := store.GetJob(ctx, testProj, "DEV-100")
	require.NoError(t, err)
	require.NotNil(t, job.PRNumber)
	assert.Equal(t, int64(42), *job.PRNumber)
	assert.True(t, job.UpdatedAt.After(jobBefore.UpdatedAt))
}

func TestSetPR_NonExistent_ReturnsErrJobNotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.SetPR(ctx, testProj, "GHOST", 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobNotFound))
}

// --- Watermark tests ---

func TestGetCommentWatermark_Untracked_ReturnsZero(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	wm, err := store.GetCommentWatermark(ctx, testProj, 999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), wm)
}

func TestSetCommentWatermark_RoundTrip(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.SetCommentWatermark(ctx, testProj, 42, 100)
	require.NoError(t, err)

	wm, err := store.GetCommentWatermark(ctx, testProj, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(100), wm)
}

func TestSetCommentWatermark_Overwrite(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.SetCommentWatermark(ctx, testProj, 42, 100))
	require.NoError(t, store.SetCommentWatermark(ctx, testProj, 42, 200))

	wm, err := store.GetCommentWatermark(ctx, testProj, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(200), wm)

	require.NoError(t, store.SetCommentWatermark(ctx, testProj, 42, 50))
	wm, err = store.GetCommentWatermark(ctx, testProj, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(50), wm)
}

func TestSetCommentWatermark_CrossProjectIsolation(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.SetCommentWatermark(ctx, "proj-a", 42, 100))
	require.NoError(t, store.SetCommentWatermark(ctx, "proj-b", 42, 200))

	wmA, err := store.GetCommentWatermark(ctx, "proj-a", 42)
	require.NoError(t, err)
	assert.Equal(t, int64(100), wmA)

	wmB, err := store.GetCommentWatermark(ctx, "proj-b", 42)
	require.NoError(t, err)
	assert.Equal(t, int64(200), wmB)
}

// --- Slot tests ---

func TestTryAcquireSlot_AutoCreatesProject(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	acquired, err := store.TryAcquireSlot(ctx, "proj-1", 3)
	require.NoError(t, err)
	assert.True(t, acquired)

	var activeCount, slotLimit int
	err = store.DB().QueryRowContext(ctx,
		"SELECT active_count, slot_limit FROM concurrency_slots WHERE project_id = ?", "proj-1",
	).Scan(&activeCount, &slotLimit)
	require.NoError(t, err)
	assert.Equal(t, 1, activeCount)
	assert.Equal(t, 3, slotLimit)
}

func TestTryAcquireSlot_AtLimit_ReturnsFalse(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	acquired, err := store.TryAcquireSlot(ctx, "proj-1", 2)
	require.NoError(t, err)
	assert.True(t, acquired)

	acquired, err = store.TryAcquireSlot(ctx, "proj-1", 2)
	require.NoError(t, err)
	assert.True(t, acquired)

	acquired, err = store.TryAcquireSlot(ctx, "proj-1", 2)
	require.NoError(t, err)
	assert.False(t, acquired)
}

func TestReleaseSlot_Decrements(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.TryAcquireSlot(ctx, "proj-1", 2)
	require.NoError(t, err)
	_, err = store.TryAcquireSlot(ctx, "proj-1", 2)
	require.NoError(t, err)

	err = store.ReleaseSlot(ctx, "proj-1")
	require.NoError(t, err)

	var activeCount int
	err = store.DB().QueryRowContext(ctx,
		"SELECT active_count FROM concurrency_slots WHERE project_id = ?", "proj-1",
	).Scan(&activeCount)
	require.NoError(t, err)
	assert.Equal(t, 1, activeCount)
}

func TestReleaseSlot_ClampsToZero(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.TryAcquireSlot(ctx, "proj-1", 1)
	require.NoError(t, err)
	err = store.ReleaseSlot(ctx, "proj-1")
	require.NoError(t, err)

	err = store.ReleaseSlot(ctx, "proj-1")
	require.NoError(t, err)

	var activeCount int
	err = store.DB().QueryRowContext(ctx,
		"SELECT active_count FROM concurrency_slots WHERE project_id = ?", "proj-1",
	).Scan(&activeCount)
	require.NoError(t, err)
	assert.Equal(t, 0, activeCount)
}

func TestReleaseSlot_NonExistentProject_NoOp(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.ReleaseSlot(ctx, "does-not-exist")
	require.NoError(t, err)
}

func TestTryAcquireSlot_Concurrent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	const goroutines = 10
	const slotLimit = 3

	var wg sync.WaitGroup
	results := make(chan bool, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			acquired, err := store.TryAcquireSlot(ctx, "proj-concurrent", slotLimit)
			require.NoError(t, err)
			results <- acquired
		}()
	}

	wg.Wait()
	close(results)

	successes := 0
	for acquired := range results {
		if acquired {
			successes++
		}
	}
	assert.Equal(t, slotLimit, successes, "exactly %d goroutines should acquire slots", slotLimit)
}

// --- ListJobsByStatus tests ---

func TestListJobsByStatus_FiltersCorrectly(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.CreateJob(ctx, "DEV-1", "/tmp/wt1", "s1", testProj))
	require.NoError(t, store.SetJobStatus(ctx, testProj, "DEV-1", StatusInProgress))
	require.NoError(t, store.CreateJob(ctx, "DEV-2", "/tmp/wt2", "s2", testProj))

	jobs, err := store.ListJobsByStatus(ctx, testProj, StatusInProgress)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "DEV-1", jobs[0].TicketID)
}

func TestListJobsByStatus_MultipleStatuses(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.CreateJob(ctx, "DEV-1", "/tmp/wt1", "s1", testProj))
	require.NoError(t, store.SetJobStatus(ctx, testProj, "DEV-1", StatusInProgress))
	require.NoError(t, store.CreateJob(ctx, "DEV-2", "/tmp/wt2", "s2", testProj))
	require.NoError(t, store.SetJobStatus(ctx, testProj, "DEV-2", StatusComplete))
	require.NoError(t, store.CreateJob(ctx, "DEV-3", "/tmp/wt3", "s3", testProj))

	jobs, err := store.ListJobsByStatus(ctx, testProj, StatusInProgress, StatusComplete)
	require.NoError(t, err)
	require.Len(t, jobs, 2)

	ticketIDs := []string{jobs[0].TicketID, jobs[1].TicketID}
	assert.Contains(t, ticketIDs, "DEV-1")
	assert.Contains(t, ticketIDs, "DEV-2")
}

func TestListJobsByStatus_NoMatches(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.CreateJob(ctx, "DEV-1", "/tmp/wt1", "s1", testProj))

	jobs, err := store.ListJobsByStatus(ctx, testProj, StatusInProgress)
	require.NoError(t, err)
	assert.Empty(t, jobs)
	assert.NotNil(t, jobs)
}

func TestListJobsByStatus_EmptyStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	jobs, err := store.ListJobsByStatus(ctx, testProj, StatusInProgress)
	require.NoError(t, err)
	assert.Empty(t, jobs)
	assert.NotNil(t, jobs)
}

func TestListJobsByStatus_CrossProjectIsolation(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.CreateJob(ctx, "DEV-1", "/tmp/wt1", "s1", "proj-a"))
	require.NoError(t, store.SetJobStatus(ctx, "proj-a", "DEV-1", StatusInProgress))
	require.NoError(t, store.CreateJob(ctx, "DEV-2", "/tmp/wt2", "s2", "proj-b"))
	require.NoError(t, store.SetJobStatus(ctx, "proj-b", "DEV-2", StatusInProgress))

	jobsA, err := store.ListJobsByStatus(ctx, "proj-a", StatusInProgress)
	require.NoError(t, err)
	require.Len(t, jobsA, 1)
	assert.Equal(t, "DEV-1", jobsA[0].TicketID)

	jobsB, err := store.ListJobsByStatus(ctx, "proj-b", StatusInProgress)
	require.NoError(t, err)
	require.Len(t, jobsB, 1)
	assert.Equal(t, "DEV-2", jobsB[0].TicketID)
}

// --- SetJobStatus tests ---

func TestSetJobStatus_UpdatesStatusAndTimestamp(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.CreateJob(ctx, "DEV-1", "/tmp/wt1", "s1", testProj))
	jobBefore, err := store.GetJob(ctx, testProj, "DEV-1")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	require.NoError(t, store.SetJobStatus(ctx, testProj, "DEV-1", StatusInProgress))

	job, err := store.GetJob(ctx, testProj, "DEV-1")
	require.NoError(t, err)
	assert.Equal(t, StatusInProgress, job.Status)
	assert.True(t, job.UpdatedAt.After(jobBefore.UpdatedAt))
}

func TestSetJobStatus_NonExistent_ReturnsErrJobNotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.SetJobStatus(ctx, testProj, "GHOST", StatusInProgress)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobNotFound))
}

// --- ResetAllSlots tests ---

func TestResetAllSlots_ResetsActiveCounts(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.TryAcquireSlot(ctx, "proj-1", 5)
	require.NoError(t, err)
	_, err = store.TryAcquireSlot(ctx, "proj-1", 5)
	require.NoError(t, err)
	_, err = store.TryAcquireSlot(ctx, "proj-2", 3)
	require.NoError(t, err)

	n, err := store.ResetAllSlots(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	var count1, count2 int
	err = store.DB().QueryRowContext(ctx,
		"SELECT active_count FROM concurrency_slots WHERE project_id = ?", "proj-1",
	).Scan(&count1)
	require.NoError(t, err)
	assert.Equal(t, 0, count1)

	err = store.DB().QueryRowContext(ctx,
		"SELECT active_count FROM concurrency_slots WHERE project_id = ?", "proj-2",
	).Scan(&count2)
	require.NoError(t, err)
	assert.Equal(t, 0, count2)
}

func TestResetAllSlots_NoActiveSlots(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	n, err := store.ResetAllSlots(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

// --- GetJobByPR tests ---

func TestGetJobByPR_Found(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.CreateJob(ctx, "DEV-300", "/tmp/wt-pr", "session-pr", testProj))
	require.NoError(t, store.SetPR(ctx, testProj, "DEV-300", 77))

	job, err := store.GetJobByPR(ctx, testProj, 77)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Equal(t, "DEV-300", job.TicketID)
	assert.Equal(t, "/tmp/wt-pr", job.WorktreePath)
	assert.Equal(t, "session-pr", job.CmuxSession)
	require.NotNil(t, job.PRNumber)
	assert.Equal(t, int64(77), *job.PRNumber)
	assert.Equal(t, StatusQueued, job.Status)
}

func TestGetJobByPR_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	job, err := store.GetJobByPR(ctx, testProj, 9999)
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestGetJobByPR_NoPRSet(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.CreateJob(ctx, "DEV-301", "/tmp/wt-nopr", "session-nopr", testProj))

	job, err := store.GetJobByPR(ctx, testProj, 0)
	require.NoError(t, err)
	assert.Nil(t, job)
}

// --- Cross-cutting tests ---

func TestPersistence_AcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	ctx := context.Background()

	store1, err := NewSQLiteStore(ctx, dbPath)
	require.NoError(t, err)

	require.NoError(t, store1.CreateJob(ctx, "DEV-200", "/tmp/wt", "s1", testProj))
	require.NoError(t, store1.SetPR(ctx, testProj, "DEV-200", 55))
	require.NoError(t, store1.SetCommentWatermark(ctx, testProj, 55, 999))
	_, err = store1.TryAcquireSlot(ctx, "proj-persist", 5)
	require.NoError(t, err)

	require.NoError(t, store1.Close())

	store2, err := NewSQLiteStore(ctx, dbPath)
	require.NoError(t, err)
	defer store2.Close()

	job, err := store2.GetJob(ctx, testProj, "DEV-200")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "DEV-200", job.TicketID)
	require.NotNil(t, job.PRNumber)
	assert.Equal(t, int64(55), *job.PRNumber)

	wm, err := store2.GetCommentWatermark(ctx, testProj, 55)
	require.NoError(t, err)
	assert.Equal(t, int64(999), wm)

	var activeCount int
	err = store2.DB().QueryRowContext(ctx,
		"SELECT active_count FROM concurrency_slots WHERE project_id = ?", "proj-persist",
	).Scan(&activeCount)
	require.NoError(t, err)
	assert.Equal(t, 1, activeCount)
}
