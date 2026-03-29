package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/zombiekit/brains/internal/github"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/shutdown"
	"github.com/zombiekit/brains/internal/state"
)

// NewCommentWatcher returns a ServiceFunc that polls tracked PRs for new
// review comments and dispatches them to per-PR goroutines for serial
// processing via AI sessions.
func (o *Orchestrator) NewCommentWatcher(dispatcher *CommentDispatcher) shutdown.ServiceFunc {
	return func(ctx context.Context) error {
		logger := logging.Logger().With(slog.String("watcher", WatcherCommentWatcher))
		logger.Info("comment watcher started", slog.Duration("poll_interval", o.cfg.PollInterval))

		ticker := time.NewTicker(o.cfg.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("comment watcher stopping")
				return nil
			case <-ticker.C:
				o.pollComments(ctx, dispatcher, logger)
			}
		}
	}
}

func (o *Orchestrator) pollComments(ctx context.Context, dispatcher *CommentDispatcher, logger *slog.Logger) {
	prs, err := o.github.ListOpenPRs(ctx, o.cfg.TrackingLabel)
	if err != nil {
		logger.Error("failed to list open PRs", slog.String("err", err.Error()))
		return
	}

	activePRSet := make(map[int]bool, len(prs))

	for _, pr := range prs {
		activePRSet[pr.Number] = true
		o.pollPRComments(ctx, dispatcher, pr, logger)
	}

	// Reap queues for PRs no longer in the tracked set.
	for _, prNumber := range dispatcher.ActivePRs() {
		if !activePRSet[prNumber] {
			logger.Info("reaping stale PR queue", slog.Int("pr_number", prNumber))
			dispatcher.RemoveQueue(prNumber)
		}
	}
}

func (o *Orchestrator) pollPRComments(ctx context.Context, dispatcher *CommentDispatcher, pr github.PRSummary, logger *slog.Logger) {
	prLog := logger.With(slog.Int("pr_number", pr.Number))

	job, err := o.store.GetJobByPR(ctx, int64(pr.Number))
	if err != nil {
		prLog.Error("failed to get job by PR", slog.String("err", err.Error()))
		return
	}
	if job == nil {
		return
	}

	switch job.Status {
	case state.StatusComplete, state.StatusClosed, state.StatusNeedsAttention:
		return
	}

	watermark, err := o.store.GetCommentWatermark(ctx, int64(pr.Number))
	if err != nil {
		prLog.Error("failed to get watermark", slog.String("err", err.Error()))
		return
	}

	comments, err := o.github.GetCommentsSince(ctx, pr.Number, github.CommentKindReview, watermark)
	if err != nil {
		prLog.Error("failed to get comments", slog.String("err", err.Error()))
		return
	}

	var filtered []github.PRComment
	for _, c := range comments {
		if c.Author == o.cfg.BotUsername {
			continue
		}
		filtered = append(filtered, c)
	}

	if len(filtered) == 0 {
		return
	}

	q := dispatcher.GetQueue(pr.Number)
	if q == nil {
		prCtx, prCancel := context.WithCancel(ctx)
		q = dispatcher.CreateQueue(pr.Number, prCancel)
		go o.runPRQueue(prCtx, dispatcher, pr.Number, job, q, prLog)
	}

	for _, c := range filtered {
		select {
		case q.comments <- c:
		default:
			prLog.Warn("PR comment queue full, skipping — will retry next poll",
				slog.Int64("comment_id", c.ID))
		}
	}
}

// runPRQueue processes comments serially for a single PR. It blocks on each
// session's completion signal before dispatching the next comment.
// job.TicketID and job.WorktreePath are immutable for the lifetime of a job.
func (o *Orchestrator) runPRQueue(
	ctx context.Context,
	dispatcher *CommentDispatcher,
	prNumber int,
	job *state.Job,
	q *prQueue,
	logger *slog.Logger,
) {
	defer dispatcher.RemoveQueue(prNumber)
	logger.Info("per-PR queue started")

	var highestEnqueuedID int64

	for {
		select {
		case <-ctx.Done():
			logger.Info("per-PR queue stopping (context cancelled)")
			return

		case comment, ok := <-q.comments:
			if !ok {
				return
			}

			if comment.ID > highestEnqueuedID {
				highestEnqueuedID = comment.ID
			}

			merged, err := o.github.IsMerged(ctx, prNumber)
			if err != nil {
				logger.Error("IsMerged check failed", slog.String("err", err.Error()))
				continue
			}
			if merged {
				logger.Info("PR merged, aborting queue")
				o.drainCommentChannel(q.comments)
				return
			}

			closed, err := o.github.IsClosed(ctx, prNumber)
			if err != nil {
				logger.Error("IsClosed check failed", slog.String("err", err.Error()))
				continue
			}
			if closed {
				logger.Info("PR closed, aborting queue")
				o.drainCommentChannel(q.comments)
				return
			}

			if !o.acquireSlotBlocking(ctx, logger) {
				return
			}

			if err := writeCommentJSON(job.WorktreePath, comment); err != nil {
				logger.Error("failed to write comment.json", slog.String("err", err.Error()))
				if releaseErr := o.store.ReleaseSlot(ctx, o.cfg.ProjectID); releaseErr != nil {
					logger.Error("failed to release slot after write error", slog.String("err", releaseErr.Error()))
				}
				continue
			}

			done := dispatcher.RegisterSession(job.TicketID, prNumber)

			_, err = o.sessions.SpawnSession(ctx, job.TicketID, "comment-resolution", job.WorktreePath, nil)
			if err != nil {
				logger.Error("failed to spawn session", slog.String("err", err.Error()))
				if releaseErr := o.store.ReleaseSlot(ctx, o.cfg.ProjectID); releaseErr != nil {
					logger.Error("failed to release slot after spawn error", slog.String("err", releaseErr.Error()))
				}
				continue
			}

			select {
			case <-ctx.Done():
				return
			case result := <-done:
				if result.Kind == SessionFailed {
					logger.Info("session failed, draining queue")
					drainedMax := o.drainCommentChannel(q.comments)
					if drainedMax > highestEnqueuedID {
						highestEnqueuedID = drainedMax
					}
					if err := o.store.SetCommentWatermark(ctx, int64(prNumber), highestEnqueuedID); err != nil {
						logger.Error("failed to advance watermark on failure", slog.String("err", err.Error()))
					}
					return
				}
				logger.Info("comment resolved", slog.Int64("comment_id", comment.ID))
			}
		}
	}
}

func (o *Orchestrator) acquireSlotBlocking(ctx context.Context, logger *slog.Logger) bool {
	for {
		acquired, err := o.store.TryAcquireSlot(ctx, o.cfg.ProjectID, o.cfg.ConcurrencyLimit)
		if err != nil {
			logger.Error("slot acquisition error", slog.String("err", err.Error()))
		}
		if acquired {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(5 * time.Second):
		}
	}
}

// drainCommentChannel reads and discards all buffered comments, returning the
// highest comment ID seen. Used to advance the watermark past drained comments.
func (o *Orchestrator) drainCommentChannel(ch chan github.PRComment) int64 {
	var maxID int64
	for {
		select {
		case c, ok := <-ch:
			if !ok {
				return maxID
			}
			if c.ID > maxID {
				maxID = c.ID
			}
		default:
			return maxID
		}
	}
}

func writeCommentJSON(worktreePath string, comment github.PRComment) error {
	payload := map[string]any{
		"id":        comment.ID,
		"author":    comment.Author,
		"body":      comment.Body,
		"path":      comment.Path,
		"diff_hunk": comment.DiffHunk,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal comment: %w", err)
	}
	dir := filepath.Join(worktreePath, ".ai")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create .ai dir: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "comment.json"), data, 0o644)
}
