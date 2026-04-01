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
	"github.com/2bit-software/zombiekit/internal/linear"
	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/state"
)

// Router consumes events from the callback server and dispatches them to
// typed handlers that coordinate post-session processing.
type Router struct {
	events     <-chan callback.Event
	store      state.StateStore
	github     github.Client
	linear     linear.Client
	archiver   Archiver
	auditor    Auditor
	dispatcher *CommentDispatcher
	cfg        *Config
	logger     *slog.Logger
}

// NewRouter creates a Router wired to the given dependencies.
func NewRouter(
	events <-chan callback.Event,
	store state.StateStore,
	gh github.Client,
	lc linear.Client,
	arch Archiver,
	aud Auditor,
	dispatcher *CommentDispatcher,
	cfg *Config,
	logger *slog.Logger,
) *Router {
	return &Router{
		events:     events,
		store:      store,
		github:     gh,
		linear:     lc,
		archiver:   arch,
		auditor:    aud,
		dispatcher: dispatcher,
		cfg:        cfg,
		logger:     logger,
	}
}

// Run implements shutdown.ServiceFunc. It processes events until the channel
// closes or the context is cancelled.
func (r *Router) Run(ctx context.Context) error {
	r.logger.Info("event router started")
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("event router stopped", slog.String("reason", "context cancelled"))
			return nil
		case evt, ok := <-r.events:
			if !ok {
				r.logger.Info("event router stopped", slog.String("reason", "channel closed"))
				return nil
			}
			r.handleEvent(ctx, evt)
		}
	}
}

func (r *Router) handleEvent(ctx context.Context, evt callback.Event) {
	logger := r.logger.With(
		slog.String("ticket_id", evt.TicketID),
		slog.String("event_kind", string(evt.Kind)),
	)
	logger.Info("processing event")

	switch evt.Kind {
	case callback.EventComplete:
		r.handleComplete(ctx, evt, logger)
	case callback.EventFailed:
		r.handleFailed(ctx, evt, logger)
	case callback.EventCommentResolved:
		r.handleCommentResolved(ctx, evt, logger)
	default:
		logger.Warn("unknown event kind, discarding")
	}
}

func (r *Router) handleComplete(ctx context.Context, evt callback.Event, logger *slog.Logger) {
	job, err := r.store.GetJob(ctx, evt.TicketID)
	if err != nil {
		logger.Error("failed to get job", slog.String("step", "GetJob"), slog.String("err", err.Error()))
		return
	}
	if job == nil {
		logger.Warn("no job found for ticket, discarding event")
		return
	}

	prDescPath := filepath.Join(job.WorktreePath, ".ai", "pr-description.md")
	body, err := os.ReadFile(prDescPath)
	if err != nil {
		logger.Error("failed to read pr-description.md", slog.String("step", "ReadFile"), slog.String("err", err.Error()))
		r.markNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	ticket, err := r.linear.GetTicket(ctx, evt.TicketID)
	if err != nil {
		logger.Error("failed to get ticket for PR title", slog.String("step", "GetTicket"), slog.String("err", err.Error()))
		r.markNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	prNumber, err := r.github.CreatePR(ctx, github.CreatePRInput{
		Title: ticket.Identifier + ": " + ticket.Title,
		Body:  string(body),
		Head:  evt.Branch,
		Base:  r.cfg.BaseBranch,
	})
	if err != nil {
		logger.Error("failed to create PR", slog.String("step", "CreatePR"), slog.String("err", err.Error()))
		r.markNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	if err := r.store.SetPR(ctx, evt.TicketID, int64(prNumber)); err != nil {
		logger.Error("failed to store PR number", slog.String("step", "SetPR"), slog.String("err", err.Error()))
		r.markNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	if err := r.github.ApplyLabel(ctx, prNumber, r.cfg.TrackingLabel); err != nil {
		logger.Error("failed to apply label", slog.String("step", "ApplyLabel"), slog.String("err", err.Error()))
		r.markNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	if err := r.archiver.Archive(ctx, evt.TicketID, evt.Kind); err != nil {
		logger.Error("archival failed", slog.String("step", "Archive"), slog.String("err", err.Error()))
	}
	if err := r.auditor.Audit(ctx, evt.TicketID, evt.Kind); err != nil {
		logger.Error("audit failed", slog.String("step", "Audit"), slog.String("err", err.Error()))
	}

	// Idempotent: cleans up sandbox VM if one was used for this session.
	sandbox.Cleanup(ctx, sandbox.Name(evt.TicketID))

	logger.Info("completion processed", slog.Int("pr_number", prNumber))
}

func (r *Router) handleFailed(ctx context.Context, evt callback.Event, logger *slog.Logger) {
	job, err := r.store.GetJob(ctx, evt.TicketID)
	if err != nil {
		logger.Error("failed to get job", slog.String("step", "GetJob"), slog.String("err", err.Error()))
	}

	// Slot release must always happen, even if other steps fail.
	defer func() {
		if releaseErr := r.store.ReleaseSlot(ctx, r.cfg.ProjectID); releaseErr != nil {
			logger.Error("failed to release slot", slog.String("step", "ReleaseSlot"), slog.String("err", releaseErr.Error()))
		}
	}()

	if statusErr := r.linear.SetTicketStatus(ctx, evt.TicketID, "needs-attention"); statusErr != nil {
		logger.Error("failed to set Linear status", slog.String("step", "SetTicketStatus"), slog.String("err", statusErr.Error()))
	}

	if job != nil {
		if statusErr := r.store.SetJobStatus(ctx, evt.TicketID, state.StatusNeedsAttention); statusErr != nil {
			logger.Error("failed to set job status", slog.String("step", "SetJobStatus"), slog.String("err", statusErr.Error()))
		}
	}

	if commentErr := r.linear.PostComment(ctx, evt.TicketID, evt.Reason); commentErr != nil {
		logger.Error("failed to post failure comment", slog.String("step", "PostComment"), slog.String("err", commentErr.Error()))
	}

	if archiveErr := r.archiver.Archive(ctx, evt.TicketID, evt.Kind); archiveErr != nil {
		logger.Error("archival failed", slog.String("step", "Archive"), slog.String("err", archiveErr.Error()))
	}

	if r.dispatcher != nil {
		r.dispatcher.NotifyResult(evt.TicketID, SessionResult{
			Kind:     SessionFailed,
			TicketID: evt.TicketID,
		})
	}

	sandbox.Cleanup(ctx, sandbox.Name(evt.TicketID))

	logger.Info("failure processed", slog.String("reason", evt.Reason))
}

func (r *Router) handleCommentResolved(ctx context.Context, evt callback.Event, logger *slog.Logger) {
	job, err := r.store.GetJob(ctx, evt.TicketID)
	if err != nil {
		logger.Error("failed to get job", slog.String("step", "GetJob"), slog.String("err", err.Error()))
		return
	}
	if job == nil {
		logger.Warn("no job found for ticket, discarding event")
		return
	}

	prNumber, commentID, err := r.resolveComment(ctx, evt, job, logger)
	if err != nil {
		logger.Error("comment resolution failed", slog.String("err", err.Error()))
		r.markNeedsAttention(ctx, evt.TicketID, job, logger)
		return
	}

	if err := r.archiver.Archive(ctx, evt.TicketID, evt.Kind); err != nil {
		logger.Error("archival failed", slog.String("step", "Archive"), slog.String("err", err.Error()))
	}
	if err := r.auditor.Audit(ctx, evt.TicketID, evt.Kind); err != nil {
		logger.Error("audit failed", slog.String("step", "Audit"), slog.String("err", err.Error()))
	}

	if err := r.store.ReleaseSlot(ctx, r.cfg.ProjectID); err != nil {
		logger.Error("failed to release slot", slog.String("step", "ReleaseSlot"), slog.String("err", err.Error()))
	}

	if r.dispatcher != nil {
		r.dispatcher.NotifyResult(evt.TicketID, SessionResult{
			Kind:     SessionResolved,
			TicketID: evt.TicketID,
			PRNumber: prNumber,
		})
	}

	logger.Info("comment resolution processed", slog.Int("pr_number", prNumber), slog.Int64("comment_id", commentID))
}

// resolveComment performs the core steps of comment resolution: parsing IDs,
// updating the PR body, posting the reply, and advancing the watermark.
func (r *Router) resolveComment(ctx context.Context, evt callback.Event, job *state.Job, logger *slog.Logger) (int, int64, error) {
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

	if err := r.github.UpdatePRBody(ctx, prNumber, string(body)); err != nil {
		return 0, 0, fmt.Errorf("update PR body: %w", err)
	}

	if _, err := r.github.PostCommentReply(ctx, prNumber, github.CommentKindReview, commentID, evt.Resolution); err != nil {
		return 0, 0, fmt.Errorf("post comment reply: %w", err)
	}

	if err := r.store.SetCommentWatermark(ctx, *job.PRNumber, commentID); err != nil {
		return 0, 0, fmt.Errorf("set comment watermark: %w", err)
	}

	return prNumber, commentID, nil
}

// markNeedsAttention moves a ticket to needs-attention in both Linear and
// the local state store. Errors are logged but not returned since this is
// called from error-handling paths.
func (r *Router) markNeedsAttention(ctx context.Context, ticketID string, job *state.Job, logger *slog.Logger) {
	if err := r.linear.SetTicketStatus(ctx, ticketID, "needs-attention"); err != nil {
		logger.Error("failed to set Linear needs-attention",
			slog.String("step", "markNeedsAttention.SetTicketStatus"),
			slog.String("err", err.Error()),
		)
	}
	if job != nil {
		if err := r.store.SetJobStatus(ctx, ticketID, state.StatusNeedsAttention); err != nil {
			logger.Error("failed to set job needs-attention",
				slog.String("step", "markNeedsAttention.SetJobStatus"),
				slog.String("err", fmt.Sprintf("%v", err)),
			)
		}
	}
}
