package orchestrator

import (
	"context"
	"log/slog"
	"time"

	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/shutdown"
	"github.com/zombiekit/brains/internal/state"
)

const (
	ticketStatusDone = "done"
)

// NewPRWatcher returns a ServiceFunc that polls for merged or closed PRs
// and performs cleanup: worktree deletion, Linear ticket status update,
// job status transition, and concurrency slot release.
func (o *Orchestrator) NewPRWatcher() shutdown.ServiceFunc {
	return func(ctx context.Context) error {
		logger := logging.Logger().With(slog.String("watcher", WatcherPRWatcher))
		logger.Info("pr watcher started", slog.Duration("poll_interval", o.cfg.PollInterval))

		ticker := time.NewTicker(o.cfg.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("pr watcher stopping")
				return nil
			case <-ticker.C:
				o.pollPRLifecycle(ctx, logger)
			}
		}
	}
}

// pollPRLifecycle runs one poll cycle: fetch queued jobs with PR numbers,
// check each PR's merged/closed status, and clean up terminal PRs.
func (o *Orchestrator) pollPRLifecycle(ctx context.Context, logger *slog.Logger) {
	jobs, err := o.store.ListJobsByStatus(ctx, state.StatusQueued)
	if err != nil {
		logger.Error("failed to list jobs", slog.String("err", err.Error()))
		return
	}

	for _, job := range jobs {
		if ctx.Err() != nil {
			return
		}

		if job.PRNumber == nil {
			continue
		}

		prNumber := int(*job.PRNumber)

		merged, err := o.github.IsMerged(ctx, prNumber)
		if err != nil {
			logger.Error("failed to check merge status",
				slog.String("ticket", job.TicketID),
				slog.Int("pr", prNumber),
				slog.String("err", err.Error()))
			continue
		}
		if merged {
			o.cleanupPR(ctx, job, ticketStatusDone, logger)
			continue
		}

		closed, err := o.github.IsClosed(ctx, prNumber)
		if err != nil {
			logger.Error("failed to check closed status",
				slog.String("ticket", job.TicketID),
				slog.Int("pr", prNumber),
				slog.String("err", err.Error()))
			continue
		}
		if closed {
			o.cleanupPR(ctx, job, o.cfg.ClosedPRTicketStatus, logger)
			continue
		}
	}
}

// cleanupPR performs best-effort cleanup for a merged or closed PR.
// Each step is independent — failure at any step does not prevent
// subsequent steps from executing.
func (o *Orchestrator) cleanupPR(_ context.Context, job state.Job, ticketStatus string, logger *slog.Logger) {
	logger = logger.With(
		slog.String("ticket", job.TicketID),
		slog.Int64("pr", *job.PRNumber),
		slog.String("ticket_status", ticketStatus),
	)
	logger.Info("cleaning up PR")

	// Use a detached context so mid-shutdown cleanup steps complete.
	cleanCtx := context.Background()

	if err := o.worktrees.DeleteWorktree(cleanCtx, job.WorktreePath); err != nil {
		logger.Error("failed to delete worktree",
			slog.String("path", job.WorktreePath),
			slog.String("err", err.Error()))
	}

	if err := o.linear.SetTicketStatus(cleanCtx, job.TicketID, ticketStatus); err != nil {
		logger.Error("failed to set ticket status", slog.String("err", err.Error()))
	}

	if err := o.store.SetJobStatus(cleanCtx, job.TicketID, state.StatusClosed); err != nil {
		logger.Error("failed to set job status", slog.String("err", err.Error()))
	}

	if err := o.store.ReleaseSlot(cleanCtx, o.cfg.ProjectID); err != nil {
		logger.Error("failed to release slot", slog.String("err", err.Error()))
	}

	logger.Info("PR cleanup complete")
}
