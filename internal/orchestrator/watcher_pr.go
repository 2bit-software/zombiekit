package orchestrator

import (
	"context"
	"log/slog"

	"github.com/2bit-software/zombiekit/internal/state"
)

const (
	ticketStatusDone = "done"
)

// pollPRLifecycle runs one poll cycle: fetch queued jobs with PR numbers,
// check each PR's merged/closed status, and clean up terminal PRs.
func (p *ProjectRunner) pollPRLifecycle(ctx context.Context, logger *slog.Logger) {
	jobs, err := p.store.ListJobsByStatus(ctx, p.id, state.StatusQueued)
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

		merged, err := p.github.IsMerged(ctx, prNumber)
		if err != nil {
			logger.Error("failed to check merge status",
				slog.String("ticket", job.TicketID),
				slog.Int("pr", prNumber),
				slog.String("err", err.Error()))
			continue
		}
		if merged {
			p.cleanupPR(ctx, job, ticketStatusDone, logger)
			continue
		}

		closed, err := p.github.IsClosed(ctx, prNumber)
		if err != nil {
			logger.Error("failed to check closed status",
				slog.String("ticket", job.TicketID),
				slog.Int("pr", prNumber),
				slog.String("err", err.Error()))
			continue
		}
		if closed {
			p.cleanupPR(ctx, job, p.cfg.ClosedPRStatus, logger)
			continue
		}
	}
}

// cleanupPR performs best-effort cleanup for a merged or closed PR.
// Each step is independent -- failure at any step does not prevent
// subsequent steps from executing.
func (p *ProjectRunner) cleanupPR(_ context.Context, job state.Job, ticketStatus string, logger *slog.Logger) {
	logger = logger.With(
		slog.String("ticket", job.TicketID),
		slog.Int64("pr", *job.PRNumber),
		slog.String("ticket_status", ticketStatus),
	)
	logger.Info("cleaning up PR")

	// Use a detached context so mid-shutdown cleanup steps complete.
	cleanCtx := context.Background()

	if err := p.worktrees.DeleteWorktree(cleanCtx, job.WorktreePath); err != nil {
		logger.Error("failed to delete worktree",
			slog.String("path", job.WorktreePath),
			slog.String("err", err.Error()))
	}

	if err := p.linear.SetTicketStatus(cleanCtx, job.TicketID, ticketStatus); err != nil {
		logger.Error("failed to set ticket status", slog.String("err", err.Error()))
	}

	if err := p.store.SetJobStatus(cleanCtx, p.id, job.TicketID, state.StatusClosed); err != nil {
		logger.Error("failed to set job status", slog.String("err", err.Error()))
	}

	if err := p.store.ReleaseSlot(cleanCtx, p.id); err != nil {
		logger.Error("failed to release slot", slog.String("err", err.Error()))
	}

	logger.Info("PR cleanup complete")
}
