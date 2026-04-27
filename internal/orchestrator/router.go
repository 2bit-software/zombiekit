package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/2bit-software/zombiekit/internal/callback"
	"github.com/2bit-software/zombiekit/internal/github"
	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/state"
)

func (p *ProjectRunner) handleEvent(ctx context.Context, evt callback.Event) {
	logger := p.logger.With(
		slog.String("ticket_id", evt.TicketID),
		slog.String("event_kind", string(evt.Kind)),
	)
	logger.Info("processing event")

	switch evt.Kind {
	case callback.EventComplete:
		p.handleComplete(ctx, evt, logger)
	case callback.EventFailed:
		p.handleFailed(ctx, evt, logger)
	case callback.EventCommentResolved:
		p.handleCommentResolved(ctx, evt, logger)
	default:
		logger.Warn("unknown event kind, discarding")
	}
}

func (p *ProjectRunner) handleComplete(ctx context.Context, evt callback.Event, logger *slog.Logger) {
	job, err := p.store.GetJob(ctx, p.id, evt.TicketID)
	if err != nil {
		logger.Error("failed to get job", slog.String("step", "GetJob"), slog.String("err", err.Error()))
		return
	}
	if job == nil {
		logger.Warn("no job found for ticket, discarding event")
		return
	}

	if err := p.worktrees.PushBranch(ctx, job.WorktreePath, evt.Branch); err != nil {
		logger.Error("failed to push branch", slog.String("step", "PushBranch"), slog.String("branch", evt.Branch), slog.String("err", err.Error()))
		p.markJobNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	prDescPath := filepath.Join(job.WorktreePath, ".ai", "pr-description.md")
	body, err := os.ReadFile(prDescPath)
	if err != nil {
		logger.Error("failed to read pr-description.md", slog.String("step", "ReadFile"), slog.String("err", err.Error()))
		p.markJobNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	ticket, err := p.linear.GetTicket(ctx, evt.TicketID)
	if err != nil {
		logger.Error("failed to get ticket for PR title", slog.String("step", "GetTicket"), slog.String("err", err.Error()))
		p.markJobNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	prNumber, err := p.github.CreatePR(ctx, github.CreatePRInput{
		Title: ticket.Identifier + ": " + ticket.Title,
		Body:  string(body),
		Head:  evt.Branch,
		Base:  p.cfg.BaseBranch,
	})
	if err != nil {
		logger.Error("failed to create PR", slog.String("step", "CreatePR"), slog.String("err", err.Error()))
		p.markJobNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	if err := p.store.SetPR(ctx, p.id, evt.TicketID, int64(prNumber)); err != nil {
		logger.Error("failed to store PR number", slog.String("step", "SetPR"), slog.String("err", err.Error()))
		p.markJobNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	if err := p.github.ApplyLabel(ctx, prNumber, p.cfg.TrackingLabel); err != nil {
		logger.Error("failed to apply label", slog.String("step", "ApplyLabel"), slog.String("err", err.Error()))
		p.markJobNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	if err := p.archiver.Archive(ctx, evt.TicketID, evt.Kind); err != nil {
		logger.Error("archival failed", slog.String("step", "Archive"), slog.String("err", err.Error()))
	}
	if err := p.auditor.Audit(ctx, evt.TicketID, evt.Kind); err != nil {
		logger.Error("audit failed", slog.String("step", "Audit"), slog.String("err", err.Error()))
	}

	if p.sandboxAvailable {
		sandbox.Cleanup(ctx, sandbox.Name(evt.TicketID))
	}

	logger.Info("completion processed", slog.Int("pr_number", prNumber))
}

func (p *ProjectRunner) handleFailed(ctx context.Context, evt callback.Event, logger *slog.Logger) {
	job, err := p.store.GetJob(ctx, p.id, evt.TicketID)
	if err != nil {
		logger.Error("failed to get job", slog.String("step", "GetJob"), slog.String("err", err.Error()))
	}

	// Slot release must always happen, even if other steps fail.
	defer func() {
		if releaseErr := p.store.ReleaseSlot(ctx, p.id); releaseErr != nil {
			logger.Error("failed to release slot", slog.String("step", "ReleaseSlot"), slog.String("err", releaseErr.Error()))
		}
	}()

	if statusErr := p.linear.SetTicketStatus(ctx, evt.TicketID, "needs-attention"); statusErr != nil {
		logger.Error("failed to set Linear status", slog.String("step", "SetTicketStatus"), slog.String("err", statusErr.Error()))
	}

	if job != nil {
		if statusErr := p.store.SetJobStatus(ctx, p.id, evt.TicketID, state.StatusNeedsAttention); statusErr != nil {
			logger.Error("failed to set job status", slog.String("step", "SetJobStatus"), slog.String("err", statusErr.Error()))
		}
	}

	if commentErr := p.linear.PostComment(ctx, evt.TicketID, evt.Reason); commentErr != nil {
		logger.Error("failed to post failure comment", slog.String("step", "PostComment"), slog.String("err", commentErr.Error()))
	}

	if archiveErr := p.archiver.Archive(ctx, evt.TicketID, evt.Kind); archiveErr != nil {
		logger.Error("archival failed", slog.String("step", "Archive"), slog.String("err", archiveErr.Error()))
	}

	if p.dispatcher != nil {
		p.dispatcher.NotifyResult(evt.TicketID, SessionResult{
			Kind:     SessionFailed,
			TicketID: evt.TicketID,
		})
	}

	if p.sandboxAvailable {
		sandbox.Cleanup(ctx, sandbox.Name(evt.TicketID))
	}

	logger.Info("failure processed", slog.String("reason", evt.Reason))
}

func (p *ProjectRunner) handleCommentResolved(ctx context.Context, evt callback.Event, logger *slog.Logger) {
	job, err := p.store.GetJob(ctx, p.id, evt.TicketID)
	if err != nil {
		logger.Error("failed to get job", slog.String("step", "GetJob"), slog.String("err", err.Error()))
		return
	}
	if job == nil {
		logger.Warn("no job found for ticket, discarding event")
		return
	}

	prNumber, commentID, err := p.resolveComment(ctx, evt, job, logger)
	if err != nil {
		logger.Error("comment resolution failed", slog.String("err", err.Error()))
		p.markJobNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	if err := p.archiver.Archive(ctx, evt.TicketID, evt.Kind); err != nil {
		logger.Error("archival failed", slog.String("step", "Archive"), slog.String("err", err.Error()))
	}
	if err := p.auditor.Audit(ctx, evt.TicketID, evt.Kind); err != nil {
		logger.Error("audit failed", slog.String("step", "Audit"), slog.String("err", err.Error()))
	}

	if err := p.store.ReleaseSlot(ctx, p.id); err != nil {
		logger.Error("failed to release slot", slog.String("step", "ReleaseSlot"), slog.String("err", err.Error()))
	}

	if p.dispatcher != nil {
		p.dispatcher.NotifyResult(evt.TicketID, SessionResult{
			Kind:     SessionResolved,
			TicketID: evt.TicketID,
			PRNumber: prNumber,
		})
	}

	logger.Info("comment resolution processed", slog.Int("pr_number", prNumber), slog.Int64("comment_id", commentID))
}

// resolveComment performs the core steps of comment resolution: parsing IDs,
// updating the PR body, posting the reply, and advancing the watermark.
func (p *ProjectRunner) resolveComment(ctx context.Context, evt callback.Event, job *state.Job, logger *slog.Logger) (int, int64, error) {
	if job.PRNumber == nil {
		return 0, 0, fmt.Errorf("job has no PR number")
	}
	prNumber := int(*job.PRNumber)

	commentID, err := strconv.ParseInt(evt.CommentID, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid comment ID %q: %w", evt.CommentID, err)
	}

	prDescPath := filepath.Join(job.WorktreePath, ".ai", "pr-description.md")
	body, err := os.ReadFile(prDescPath)
	if err != nil {
		return 0, 0, fmt.Errorf("read pr-description.md: %w", err)
	}

	if err := p.github.UpdatePRBody(ctx, prNumber, string(body)); err != nil {
		return 0, 0, fmt.Errorf("update PR body: %w", err)
	}

	if _, err := p.github.PostCommentReply(ctx, prNumber, github.CommentKindReview, commentID, evt.Resolution); err != nil {
		return 0, 0, fmt.Errorf("post comment reply: %w", err)
	}

	if err := p.store.SetCommentWatermark(ctx, p.id, *job.PRNumber, commentID); err != nil {
		return 0, 0, fmt.Errorf("set comment watermark: %w", err)
	}

	return prNumber, commentID, nil
}

// markJobNeedsAttention moves a ticket to needs-attention in both Linear and
// the local state store. Errors are logged but not returned since this is
// called from error-handling paths.
func (p *ProjectRunner) markJobNeedsAttention(ctx context.Context, ticketID string, job *state.Job, logger *slog.Logger) {
	if err := p.linear.SetTicketStatus(ctx, ticketID, "needs-attention"); err != nil {
		logger.Error("failed to set Linear needs-attention",
			slog.String("step", "markJobNeedsAttention.SetTicketStatus"),
			slog.String("err", err.Error()),
		)
	}
	if job != nil {
		if err := p.store.SetJobStatus(ctx, p.id, ticketID, state.StatusNeedsAttention); err != nil {
			logger.Error("failed to set job needs-attention",
				slog.String("step", "markJobNeedsAttention.SetJobStatus"),
				slog.String("err", fmt.Sprintf("%v", err)),
			)
		}
	}
}
