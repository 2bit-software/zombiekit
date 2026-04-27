package state

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// OrphanedJob describes a job that needs attention after a crash.
type OrphanedJob struct {
	ProjectID      string
	TicketID       string
	PreviousStatus string
	WorktreePath   string
	PRNumber       *int64
	StaleDuration  time.Duration
}

// ReconciliationPlan describes the actions the reconciler wants to take.
type ReconciliationPlan struct {
	Orphaned []OrphanedJob
}

// HasFindings returns true if the plan contains any orphaned jobs.
func (p ReconciliationPlan) HasFindings() bool {
	return len(p.Orphaned) > 0
}

// PlanReconciliation classifies jobs as orphaned based on their status.
// Any job with StatusInProgress is considered orphaned (interrupted by a crash).
// This is a pure function with no side effects.
func PlanReconciliation(jobs []Job, now time.Time) ReconciliationPlan {
	var plan ReconciliationPlan
	for _, job := range jobs {
		if job.Status == StatusInProgress {
			plan.Orphaned = append(plan.Orphaned, OrphanedJob{
				ProjectID:      job.ProjectID,
				TicketID:       job.TicketID,
				PreviousStatus: job.Status,
				WorktreePath:   job.WorktreePath,
				PRNumber:       job.PRNumber,
				StaleDuration:  now.Sub(job.UpdatedAt),
			})
		}
	}
	return plan
}

// ApplyReconciliation scans the state store for orphaned jobs, marks them
// as needing attention, and warns about jobs belonging to unconfigured
// projects. It must be called during startup before any watcher goroutines
// begin polling.
//
// configuredProjects lists the project IDs from the TOML config. Jobs
// with a project_id not in this set are logged as orphans and their
// slots are released. Pass nil to skip orphan-by-project detection
// (backwards compatibility).
func ApplyReconciliation(ctx context.Context, store StateStore, logger *slog.Logger, configuredProjects ...string) error {
	start := time.Now()

	jobs, err := store.ListAllJobs(ctx)
	if err != nil {
		return fmt.Errorf("reconciliation: list jobs: %w", err)
	}

	configured := make(map[string]bool, len(configuredProjects))
	for _, id := range configuredProjects {
		configured[id] = true
	}

	if len(configured) > 0 {
		detectOrphanedProjects(ctx, jobs, configured, store, logger)
	}

	plan := PlanReconciliation(jobs, time.Now())

	if !plan.HasFindings() {
		logger.Info("reconciliation complete: no orphaned jobs found")
	} else {
		for _, orphan := range plan.Orphaned {
			if err := store.SetJobStatus(ctx, orphan.ProjectID, orphan.TicketID, StatusNeedsAttention); err != nil {
				return fmt.Errorf("reconciliation: mark job %s/%s as needs-attention: %w", orphan.ProjectID, orphan.TicketID, err)
			}
			logger.Info("reconciliation: orphaned job detected",
				slog.String("project_id", orphan.ProjectID),
				slog.String("ticket_id", orphan.TicketID),
				slog.String("previous_status", orphan.PreviousStatus),
				slog.String("new_status", StatusNeedsAttention),
				slog.String("worktree_path", orphan.WorktreePath),
				slog.Duration("stale_duration", orphan.StaleDuration),
			)
		}
	}

	slotsReset, err := store.ResetAllSlots(ctx)
	if err != nil {
		return fmt.Errorf("reconciliation: reset slots: %w", err)
	}

	logger.Info("reconciliation complete",
		slog.Int("orphaned_count", len(plan.Orphaned)),
		slog.Int("slots_reset", slotsReset),
		slog.Duration("elapsed", time.Since(start)),
	)

	return nil
}

func detectOrphanedProjects(ctx context.Context, jobs []Job, configured map[string]bool, store StateStore, logger *slog.Logger) {
	for _, job := range jobs {
		if job.ProjectID == "" || configured[job.ProjectID] {
			continue
		}
		logger.Warn("reconciliation: job belongs to unconfigured project, releasing slot",
			slog.String("project_id", job.ProjectID),
			slog.String("ticket_id", job.TicketID),
			slog.String("status", job.Status),
		)
		if err := store.ReleaseSlot(ctx, job.ProjectID); err != nil {
			logger.Error("reconciliation: failed to release orphaned project slot",
				slog.String("project_id", job.ProjectID),
				slog.String("err", err.Error()),
			)
		}
	}
}
