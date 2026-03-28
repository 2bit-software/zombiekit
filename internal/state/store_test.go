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

	expected := []string{"ticket_id", "worktree_path", "cmux_session", "pr_number", "status", "created_at", "updated_at"}
	for _, col := range expected {
		assert.True(t, columns[col], "jobs table should have column %s", col)
	}
}

func TestNewSQLiteStore_IdempotentRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	ctx := context.Background()

	// First open — creates tables
	store1, err := NewSQLiteStore(ctx, dbPath)
	require.NoError(t, err)

	// Insert a row to verify data survives restart
	_, err = store1.DB().ExecContext(ctx,
		"INSERT INTO jobs (ticket_id, worktree_path, cmux_session, status) VALUES (?, ?, ?, ?)",
		"DEV-999", "/tmp/wt", "session-1", "queued",
	)
	require.NoError(t, err)
	require.NoError(t, store1.Close())

	// Second open — should not error, data preserved
	store2, err := NewSQLiteStore(ctx, dbPath)
	require.NoError(t, err)
	defer store2.Close()

	var ticketID string
	err = store2.DB().QueryRowContext(ctx,
		"SELECT ticket_id FROM jobs WHERE ticket_id = ?", "DEV-999",
	).Scan(&ticketID)
	require.NoError(t, err)
	assert.Equal(t, "DEV-999", ticketID)
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

	// Migrations already ran in constructor. Run again explicitly.
	err := store.Migrate(ctx)
	require.NoError(t, err)

	// Verify schema_migrations has exactly one entry
	var count int
	err = store.DB().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM schema_migrations",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// --- Job CRUD tests ---

func TestCreateJob_AndGetJob(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	before := time.Now().Add(-time.Second)
	err := store.CreateJob(ctx, "DEV-100", "/tmp/worktree", "session-abc")
	require.NoError(t, err)

	job, err := store.GetJob(ctx, "DEV-100")
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Equal(t, "DEV-100", job.TicketID)
	assert.Equal(t, "/tmp/worktree", job.WorktreePath)
	assert.Equal(t, "session-abc", job.CmuxSession)
	assert.Nil(t, job.PRNumber)
	assert.Equal(t, "queued", job.Status)
	assert.True(t, job.CreatedAt.After(before))
	assert.True(t, job.UpdatedAt.After(before))
}

func TestCreateJob_Duplicate_ReturnsErrJobExists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.CreateJob(ctx, "DEV-100", "/tmp/wt1", "s1")
	require.NoError(t, err)

	err = store.CreateJob(ctx, "DEV-100", "/tmp/wt2", "s2")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobExists))

	// Original record unchanged
	job, err := store.GetJob(ctx, "DEV-100")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/wt1", job.WorktreePath)
}

func TestGetJob_NonExistent_ReturnsNil(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	job, err := store.GetJob(ctx, "DOES-NOT-EXIST")
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestSetPR_UpdatesJob(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.CreateJob(ctx, "DEV-100", "/tmp/wt", "s1")
	require.NoError(t, err)

	jobBefore, err := store.GetJob(ctx, "DEV-100")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // ensure updated_at advances

	err = store.SetPR(ctx, "DEV-100", 42)
	require.NoError(t, err)

	job, err := store.GetJob(ctx, "DEV-100")
	require.NoError(t, err)
	require.NotNil(t, job.PRNumber)
	assert.Equal(t, int64(42), *job.PRNumber)
	assert.True(t, job.UpdatedAt.After(jobBefore.UpdatedAt))
}

func TestSetPR_NonExistent_ReturnsErrJobNotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.SetPR(ctx, "GHOST", 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobNotFound))
}

// --- Watermark tests ---

func TestGetCommentWatermark_Untracked_ReturnsZero(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	wm, err := store.GetCommentWatermark(ctx, 999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), wm)
}

func TestSetCommentWatermark_RoundTrip(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.SetCommentWatermark(ctx, 42, 100)
	require.NoError(t, err)

	wm, err := store.GetCommentWatermark(ctx, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(100), wm)
}

func TestSetCommentWatermark_Overwrite(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.SetCommentWatermark(ctx, 42, 100))
	require.NoError(t, store.SetCommentWatermark(ctx, 42, 200))

	wm, err := store.GetCommentWatermark(ctx, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(200), wm)

	// Overwrite with lower value also works (pure persistence)
	require.NoError(t, store.SetCommentWatermark(ctx, 42, 50))
	wm, err = store.GetCommentWatermark(ctx, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(50), wm)
}

// --- Slot tests ---

func TestTryAcquireSlot_AutoCreatesProject(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	acquired, err := store.TryAcquireSlot(ctx, "proj-1", 3)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Verify the row was created with correct values
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

	// Fill 2 slots
	acquired, err := store.TryAcquireSlot(ctx, "proj-1", 2)
	require.NoError(t, err)
	assert.True(t, acquired)

	acquired, err = store.TryAcquireSlot(ctx, "proj-1", 2)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Third should fail
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

	// Create a project row with 0 active
	_, err := store.TryAcquireSlot(ctx, "proj-1", 1)
	require.NoError(t, err)
	err = store.ReleaseSlot(ctx, "proj-1")
	require.NoError(t, err)

	// Release again -- should stay at 0
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

// --- Cross-cutting tests ---

func TestPersistence_AcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	ctx := context.Background()

	store1, err := NewSQLiteStore(ctx, dbPath)
	require.NoError(t, err)

	// Create job, set PR, set watermark, acquire slot
	require.NoError(t, store1.CreateJob(ctx, "DEV-200", "/tmp/wt", "s1"))
	require.NoError(t, store1.SetPR(ctx, "DEV-200", 55))
	require.NoError(t, store1.SetCommentWatermark(ctx, 55, 999))
	_, err = store1.TryAcquireSlot(ctx, "proj-persist", 5)
	require.NoError(t, err)

	require.NoError(t, store1.Close())

	// Reopen and verify everything survived
	store2, err := NewSQLiteStore(ctx, dbPath)
	require.NoError(t, err)
	defer store2.Close()

	job, err := store2.GetJob(ctx, "DEV-200")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "DEV-200", job.TicketID)
	require.NotNil(t, job.PRNumber)
	assert.Equal(t, int64(55), *job.PRNumber)

	wm, err := store2.GetCommentWatermark(ctx, 55)
	require.NoError(t, err)
	assert.Equal(t, int64(999), wm)

	var activeCount int
	err = store2.DB().QueryRowContext(ctx,
		"SELECT active_count FROM concurrency_slots WHERE project_id = ?", "proj-persist",
	).Scan(&activeCount)
	require.NoError(t, err)
	assert.Equal(t, 1, activeCount)
}
