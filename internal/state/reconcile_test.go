package state

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- PlanReconciliation unit tests (pure function, no DB) ---

func TestPlanReconciliation(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		jobs           []Job
		wantOrphaned   int
		wantTicketIDs  []string
		wantHasFindings bool
	}{
		{
			name:            "empty input returns empty plan",
			jobs:            []Job{},
			wantOrphaned:    0,
			wantHasFindings: false,
		},
		{
			name: "all terminal jobs returns empty plan",
			jobs: []Job{
				{TicketID: "DEV-1", Status: StatusComplete, UpdatedAt: now.Add(-time.Hour)},
				{TicketID: "DEV-2", Status: StatusClosed, UpdatedAt: now.Add(-time.Hour)},
			},
			wantOrphaned:    0,
			wantHasFindings: false,
		},
		{
			name: "single in-progress job detected",
			jobs: []Job{
				{TicketID: "DEV-1", Status: StatusInProgress, WorktreePath: "/tmp/wt1", UpdatedAt: now.Add(-time.Hour)},
			},
			wantOrphaned:    1,
			wantTicketIDs:   []string{"DEV-1"},
			wantHasFindings: true,
		},
		{
			name: "multiple in-progress jobs all detected",
			jobs: []Job{
				{TicketID: "DEV-1", Status: StatusInProgress, WorktreePath: "/tmp/wt1", UpdatedAt: now.Add(-time.Hour)},
				{TicketID: "DEV-2", Status: StatusInProgress, WorktreePath: "/tmp/wt2", UpdatedAt: now.Add(-2 * time.Hour)},
			},
			wantOrphaned:    2,
			wantTicketIDs:   []string{"DEV-1", "DEV-2"},
			wantHasFindings: true,
		},
		{
			name: "mixed statuses only flags in-progress",
			jobs: []Job{
				{TicketID: "DEV-1", Status: StatusQueued, UpdatedAt: now.Add(-time.Hour)},
				{TicketID: "DEV-2", Status: StatusInProgress, WorktreePath: "/tmp/wt2", UpdatedAt: now.Add(-time.Hour)},
				{TicketID: "DEV-3", Status: StatusComplete, UpdatedAt: now.Add(-time.Hour)},
				{TicketID: "DEV-4", Status: StatusClosed, UpdatedAt: now.Add(-time.Hour)},
				{TicketID: "DEV-5", Status: StatusNeedsAttention, UpdatedAt: now.Add(-time.Hour)},
			},
			wantOrphaned:    1,
			wantTicketIDs:   []string{"DEV-2"},
			wantHasFindings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := PlanReconciliation(tt.jobs, now)
			assert.Equal(t, tt.wantOrphaned, len(plan.Orphaned))
			assert.Equal(t, tt.wantHasFindings, plan.HasFindings())

			if tt.wantTicketIDs != nil {
				var gotIDs []string
				for _, o := range plan.Orphaned {
					gotIDs = append(gotIDs, o.TicketID)
				}
				assert.ElementsMatch(t, tt.wantTicketIDs, gotIDs)
			}
		})
	}
}

func TestPlanReconciliation_NilPRNumber(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	plan := PlanReconciliation([]Job{
		{TicketID: "DEV-1", Status: StatusInProgress, WorktreePath: "/tmp/wt1", UpdatedAt: now.Add(-time.Hour)},
	}, now)

	require.Len(t, plan.Orphaned, 1)
	assert.Nil(t, plan.Orphaned[0].PRNumber)
	assert.Equal(t, "DEV-1", plan.Orphaned[0].TicketID)
}

func TestPlanReconciliation_WithPRNumber(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	pr42 := int64(42)

	plan := PlanReconciliation([]Job{
		{TicketID: "DEV-1", Status: StatusInProgress, WorktreePath: "/tmp/wt1", PRNumber: &pr42, UpdatedAt: now.Add(-time.Hour)},
	}, now)

	require.Len(t, plan.Orphaned, 1)
	require.NotNil(t, plan.Orphaned[0].PRNumber)
	assert.Equal(t, int64(42), *plan.Orphaned[0].PRNumber)
}

func TestPlanReconciliation_StaleDuration(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	updatedAt := now.Add(-3 * time.Hour)

	plan := PlanReconciliation([]Job{
		{TicketID: "DEV-1", Status: StatusInProgress, UpdatedAt: updatedAt},
	}, now)

	require.Len(t, plan.Orphaned, 1)
	assert.Equal(t, 3*time.Hour, plan.Orphaned[0].StaleDuration)
}

func TestPlanReconciliation_OrphanedJobFields(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	plan := PlanReconciliation([]Job{
		{
			TicketID:     "DEV-1",
			Status:       StatusInProgress,
			WorktreePath: "/home/user/worktrees/dev-1",
			UpdatedAt:    now.Add(-30 * time.Minute),
		},
	}, now)

	require.Len(t, plan.Orphaned, 1)
	orphan := plan.Orphaned[0]
	assert.Equal(t, "DEV-1", orphan.TicketID)
	assert.Equal(t, StatusInProgress, orphan.PreviousStatus)
	assert.Equal(t, "/home/user/worktrees/dev-1", orphan.WorktreePath)
	assert.Equal(t, 30*time.Minute, orphan.StaleDuration)
}

// --- ApplyReconciliation integration tests ---

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestApplyReconciliation_CleanState(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := ApplyReconciliation(ctx, store, testLogger())
	require.NoError(t, err)
}

func TestApplyReconciliation_SingleOrphanedJob(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.CreateJob(ctx, "DEV-1", "/tmp/wt1", "s1"))
	require.NoError(t, store.SetJobStatus(ctx, "DEV-1", StatusInProgress))

	err := ApplyReconciliation(ctx, store, testLogger())
	require.NoError(t, err)

	job, err := store.GetJob(ctx, "DEV-1")
	require.NoError(t, err)
	assert.Equal(t, StatusNeedsAttention, job.Status)
}

func TestApplyReconciliation_MultipleOrphanedJobs(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	for _, id := range []string{"DEV-1", "DEV-2", "DEV-3"} {
		require.NoError(t, store.CreateJob(ctx, id, "/tmp/"+id, "s-"+id))
		require.NoError(t, store.SetJobStatus(ctx, id, StatusInProgress))
	}

	err := ApplyReconciliation(ctx, store, testLogger())
	require.NoError(t, err)

	for _, id := range []string{"DEV-1", "DEV-2", "DEV-3"} {
		job, err := store.GetJob(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, StatusNeedsAttention, job.Status, "job %s should be needs-attention", id)
	}
}

func TestApplyReconciliation_MixedStatuses(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.CreateJob(ctx, "DEV-1", "/tmp/wt1", "s1"))
	require.NoError(t, store.SetJobStatus(ctx, "DEV-1", StatusInProgress))

	require.NoError(t, store.CreateJob(ctx, "DEV-2", "/tmp/wt2", "s2"))
	require.NoError(t, store.SetJobStatus(ctx, "DEV-2", StatusComplete))

	require.NoError(t, store.CreateJob(ctx, "DEV-3", "/tmp/wt3", "s3"))
	// DEV-3 stays queued

	err := ApplyReconciliation(ctx, store, testLogger())
	require.NoError(t, err)

	job1, _ := store.GetJob(ctx, "DEV-1")
	assert.Equal(t, StatusNeedsAttention, job1.Status)

	job2, _ := store.GetJob(ctx, "DEV-2")
	assert.Equal(t, StatusComplete, job2.Status)

	job3, _ := store.GetJob(ctx, "DEV-3")
	assert.Equal(t, StatusQueued, job3.Status)
}

func TestApplyReconciliation_ResetsSlots(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.CreateJob(ctx, "DEV-1", "/tmp/wt1", "s1"))
	require.NoError(t, store.SetJobStatus(ctx, "DEV-1", StatusInProgress))

	_, err := store.TryAcquireSlot(ctx, "proj-1", 5)
	require.NoError(t, err)
	_, err = store.TryAcquireSlot(ctx, "proj-1", 5)
	require.NoError(t, err)

	err = ApplyReconciliation(ctx, store, testLogger())
	require.NoError(t, err)

	var activeCount int
	err = store.DB().QueryRowContext(ctx,
		"SELECT active_count FROM concurrency_slots WHERE project_id = ?", "proj-1",
	).Scan(&activeCount)
	require.NoError(t, err)
	assert.Equal(t, 0, activeCount)
}

func TestApplyReconciliation_DBErrorOnQuery(t *testing.T) {
	store := setupTestStore(t)
	store.Close() // force DB error

	err := ApplyReconciliation(context.Background(), store, testLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reconciliation")
}
