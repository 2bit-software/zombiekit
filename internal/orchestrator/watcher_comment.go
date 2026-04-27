package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/2bit-software/zombiekit/internal/github"
	"github.com/2bit-software/zombiekit/internal/state"
)

func (p *ProjectRunner) pollComments(ctx context.Context, dispatcher *CommentDispatcher, logger *slog.Logger) {
	prs, err := p.github.ListOpenPRs(ctx, p.cfg.TrackingLabel)
	if err != nil {
		logger.Error("failed to list open PRs", slog.String("err", err.Error()))
		return
	}

	activePRSet := make(map[int]bool, len(prs))

	for _, pr := range prs {
		activePRSet[pr.Number] = true
		p.pollPRComments(ctx, dispatcher, pr, logger)
	}

	// Reap queues for PRs no longer in the tracked set.
	for _, prNumber := range dispatcher.ActivePRs() {
		if !activePRSet[prNumber] {
			logger.Info("reaping stale PR queue", slog.Int("pr_number", prNumber))
			dispatcher.RemoveQueue(prNumber)
		}
	}
}

func (p *ProjectRunner) pollPRComments(ctx context.Context, dispatcher *CommentDispatcher, pr github.PRSummary, logger *slog.Logger) {
	prLog := logger.With(slog.Int("pr_number", pr.Number))

	job, err := p.store.GetJobByPR(ctx, p.id, int64(pr.Number))
	if err != nil {
		prLog.Error("failed to get job by PR", slog.String("err", err.Error()))
		return
	}
	if job == nil {
		return
	}

	if isTerminalStatus(job.Status) {
		return
	}

	filtered, err := p.fetchNewComments(ctx, pr, prLog)
	if err != nil || len(filtered) == 0 {
		return
	}

	q := p.ensurePRQueue(ctx, dispatcher, pr, job, prLog)
	enqueueComments(q, filtered, prLog)
}

// isTerminalStatus reports whether a job status means no further comment
// processing is needed.
func isTerminalStatus(status string) bool {
	switch status {
	case state.StatusComplete, state.StatusClosed, state.StatusNeedsAttention:
		return true
	default:
		return false
	}
}

// fetchNewComments retrieves review comments since the watermark and filters
// out those authored by the bot. Returns nil, nil when there are no new
// comments or an error occurred (already logged).
func (p *ProjectRunner) fetchNewComments(ctx context.Context, pr github.PRSummary, logger *slog.Logger) ([]github.PRComment, error) {
	watermark, err := p.store.GetCommentWatermark(ctx, p.id, int64(pr.Number))
	if err != nil {
		logger.Error("failed to get watermark", slog.String("err", err.Error()))
		return nil, err
	}

	comments, err := p.github.GetCommentsSince(ctx, pr.Number, github.CommentKindReview, watermark)
	if err != nil {
		logger.Error("failed to get comments", slog.String("err", err.Error()))
		return nil, err
	}

	filtered := filterBotComments(comments, p.cfg.BotUsername)
	return filtered, nil
}

// filterBotComments returns comments not authored by the given bot username.
func filterBotComments(comments []github.PRComment, botUsername string) []github.PRComment {
	var filtered []github.PRComment
	for _, c := range comments {
		if c.Author != botUsername {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// ensurePRQueue returns the existing queue for a PR or creates a new one and
// starts its processing goroutine.
func (p *ProjectRunner) ensurePRQueue(ctx context.Context, dispatcher *CommentDispatcher, pr github.PRSummary, job *state.Job, logger *slog.Logger) *prQueue {
	q := dispatcher.GetQueue(pr.Number)
	if q != nil {
		return q
	}
	prCtx, prCancel := context.WithCancel(ctx)
	q = dispatcher.CreateQueue(pr.Number, prCancel)
	go p.runPRQueue(prCtx, dispatcher, pr.Number, job, q, logger)
	return q
}

// enqueueComments sends comments to the queue, logging a warning when the
// buffer is full.
func enqueueComments(q *prQueue, comments []github.PRComment, logger *slog.Logger) {
	for _, c := range comments {
		select {
		case q.comments <- c:
		default:
			logger.Warn("PR comment queue full, skipping — will retry next poll",
				slog.Int64("comment_id", c.ID))
		}
	}
}

// runPRQueue processes comments serially for a single PR. It blocks on each
// session's completion signal before dispatching the next comment.
// job.TicketID and job.WorktreePath are immutable for the lifetime of a job.
func (p *ProjectRunner) runPRQueue(
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

			stop := p.handleQueuedComment(ctx, comment, job, prNumber, q, dispatcher, logger, &highestEnqueuedID)
			if stop {
				return
			}
		}
	}
}

// handleQueuedComment processes a single comment from the PR queue. Returns
// true when the queue should stop (PR closed/merged, or session failed).
func (p *ProjectRunner) handleQueuedComment(
	ctx context.Context,
	comment github.PRComment,
	job *state.Job,
	prNumber int,
	q *prQueue,
	dispatcher *CommentDispatcher,
	logger *slog.Logger,
	highestEnqueuedID *int64,
) bool {
	open, err := p.prStillOpen(ctx, prNumber, logger)
	if err != nil {
		return false
	}
	if !open {
		p.drainCommentChannel(q.comments)
		return true
	}

	result, err := p.processComment(ctx, comment, job, prNumber, dispatcher, logger)
	if err != nil {
		return false
	}

	if result.Kind == SessionFailed {
		logger.Info("session failed, draining queue")
		p.advanceWatermarkOnDrain(ctx, q.comments, prNumber, highestEnqueuedID, logger)
		return true
	}

	logger.Info("comment resolved", slog.Int64("comment_id", comment.ID))
	return false
}

// advanceWatermarkOnDrain drains all buffered comments and persists the
// highest observed comment ID as the watermark so they are not re-processed.
func (p *ProjectRunner) advanceWatermarkOnDrain(
	ctx context.Context,
	ch chan github.PRComment,
	prNumber int,
	highestEnqueuedID *int64,
	logger *slog.Logger,
) {
	drainedMax := p.drainCommentChannel(ch)
	if drainedMax > *highestEnqueuedID {
		*highestEnqueuedID = drainedMax
	}
	if err := p.store.SetCommentWatermark(ctx, p.id, int64(prNumber), *highestEnqueuedID); err != nil {
		logger.Error("failed to advance watermark on failure", slog.String("err", err.Error()))
	}
}

// prStillOpen reports whether the PR is still open (not merged or closed).
// Returns false with a nil error when the PR has been merged or closed.
func (p *ProjectRunner) prStillOpen(ctx context.Context, prNumber int, logger *slog.Logger) (bool, error) {
	merged, err := p.github.IsMerged(ctx, prNumber)
	if err != nil {
		logger.Error("IsMerged check failed", slog.String("err", err.Error()))
		return false, err
	}
	if merged {
		logger.Info("PR merged, aborting queue")
		return false, nil
	}

	closed, err := p.github.IsClosed(ctx, prNumber)
	if err != nil {
		logger.Error("IsClosed check failed", slog.String("err", err.Error()))
		return false, err
	}
	if closed {
		logger.Info("PR closed, aborting queue")
		return false, nil
	}

	return true, nil
}

// processComment handles the full lifecycle of a single comment: slot
// acquisition, writing the comment payload, spawning the AI session, and
// waiting for its result. It releases the slot on failure paths.
func (p *ProjectRunner) processComment(
	ctx context.Context,
	comment github.PRComment,
	job *state.Job,
	prNumber int,
	dispatcher *CommentDispatcher,
	logger *slog.Logger,
) (SessionResult, error) {
	if !p.acquireSlotBlocking(ctx, logger) {
		return SessionResult{}, ctx.Err()
	}

	if err := writeCommentJSON(job.WorktreePath, comment); err != nil {
		logger.Error("failed to write comment.json", slog.String("err", err.Error()))
		p.releaseSlotLogError(ctx, logger, "failed to release slot after write error")
		return SessionResult{}, err
	}

	done := dispatcher.RegisterSession(job.TicketID, prNumber)

	prompt := "Read .ai/comment.json — this is a PR review comment to resolve. Address the feedback."
	_, err := p.sessions.SpawnSession(ctx, job.TicketID, "comment-resolution", job.WorktreePath, nil, prompt)
	if err != nil {
		logger.Error("failed to spawn session", slog.String("err", err.Error()))
		p.releaseSlotLogError(ctx, logger, "failed to release slot after spawn error")
		return SessionResult{}, err
	}

	select {
	case <-ctx.Done():
		return SessionResult{}, ctx.Err()
	case result := <-done:
		return result, nil
	}
}

func (p *ProjectRunner) releaseSlotLogError(ctx context.Context, logger *slog.Logger, msg string) {
	if err := p.store.ReleaseSlot(ctx, p.id); err != nil {
		logger.Error(msg, slog.String("err", err.Error()))
	}
}

func (p *ProjectRunner) acquireSlotBlocking(ctx context.Context, logger *slog.Logger) bool {
	for {
		acquired, err := p.store.TryAcquireSlot(ctx, p.id, p.cfg.ConcurrencyLimit)
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
func (p *ProjectRunner) drainCommentChannel(ch chan github.PRComment) int64 {
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
