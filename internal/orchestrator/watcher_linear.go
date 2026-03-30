package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/zombiekit/brains/internal/linear"
	"github.com/zombiekit/brains/internal/logging"
	"github.com/zombiekit/brains/internal/shutdown"
)

const (
	labelAIReady        = "ai-ready"
	labelNeedsAttention = "needs-attention"
	statusInProgress    = "In Progress"
)

// NewLinearPoller returns a ServiceFunc that polls Linear for ai-ready tickets
// and processes them through the pickup pipeline.
func (o *Orchestrator) NewLinearPoller() shutdown.ServiceFunc {
	return func(ctx context.Context) error {
		logger := logging.Logger()
		logger.Info("linear poller started", "pollInterval", o.cfg.PollInterval)

		ticker := time.NewTicker(o.cfg.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("linear poller stopping")
				return nil
			case <-ticker.C:
				o.pollAndProcess(ctx)
			}
		}
	}
}

// pollAndProcess runs one poll cycle: fetch tickets, process each sequentially.
func (o *Orchestrator) pollAndProcess(ctx context.Context) {
	logger := logging.Logger()

	tickets, err := o.linear.PollReadyTickets(ctx, labelAIReady, o.cfg.ProjectID)
	if err != nil {
		logger.Error("failed to poll ready tickets", "error", err)
		return
	}

	for _, ticket := range tickets {
		if ctx.Err() != nil {
			logger.Info("context cancelled, stopping ticket processing")
			return
		}
		if err := o.processTicket(ctx, ticket); err != nil {
			logger.Error("failed to process ticket", "ticket", ticket.Identifier, "error", err)
		}
	}
}

// processTicket runs the pickup pipeline for a single ticket.
func (o *Orchestrator) processTicket(ctx context.Context, ticket linear.Ticket) error {
	logger := logging.Logger()

	// FR-012: skip tickets that already have a job
	existing, err := o.store.GetJob(ctx, ticket.Identifier)
	if err != nil {
		return fmt.Errorf("check existing job: %w", err)
	}
	if existing != nil {
		logger.Debug("skipping ticket with existing job", "ticket", ticket.Identifier)
		return nil
	}

	// FR-002: acquire concurrency slot
	acquired, err := o.store.TryAcquireSlot(ctx, o.cfg.ProjectID, o.cfg.ConcurrencyLimit)
	if err != nil {
		return fmt.Errorf("acquire slot: %w", err)
	}
	if !acquired {
		logger.Debug("concurrency limit reached, deferring ticket", "ticket", ticket.Identifier)
		return nil
	}

	// Track what we've created for rollback
	var worktreePath string
	var sessionRef string
	succeeded := false

	defer func() {
		if succeeded {
			return
		}
		// Rollback in reverse order
		if sessionRef != "" {
			if killErr := o.sessions.KillSession(ctx, ticket.Identifier); killErr != nil {
				logger.Error("rollback: failed to kill session", "ticket", ticket.Identifier, "error", killErr)
			}
		}
		if worktreePath != "" {
			if delErr := o.worktrees.DeleteWorktree(ctx, worktreePath); delErr != nil {
				logger.Error("rollback: failed to delete worktree", "ticket", ticket.Identifier, "error", delErr)
			}
		}
		if releaseErr := o.store.ReleaseSlot(ctx, o.cfg.ProjectID); releaseErr != nil {
			logger.Error("rollback: failed to release slot", "ticket", ticket.Identifier, "error", releaseErr)
		}

		// FR-013: mark ticket as needs-attention
		o.markNeedsAttention(ctx, ticket)
	}()

	// FR-003: create worktree
	worktreePath, err = o.worktrees.CreateWorktree(ctx, ticket.Identifier, shortTitle(ticket.Title))
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}

	// FR-004: write ticket content to .ai/ticket.md
	aiDir := filepath.Join(worktreePath, ".ai")
	if err := os.MkdirAll(aiDir, 0o755); err != nil {
		return fmt.Errorf("create .ai directory: %w", err)
	}
	ticketFile := filepath.Join(aiDir, "ticket.md")
	if err := os.WriteFile(ticketFile, []byte(ticket.Description), 0o644); err != nil {
		return fmt.Errorf("write ticket file: %w", err)
	}

	// FR-005 + FR-006: spawn session with callback URL
	env := map[string]string{
		"WORK_CALLBACK_URL": fmt.Sprintf("http://localhost:%d/%s", o.cfg.CallbackPort, ticket.Identifier),
	}
	sessionRef, err = o.sessions.SpawnSession(ctx, ticket.Identifier, ticket.Title, worktreePath, env)
	if err != nil {
		return fmt.Errorf("spawn session: %w", err)
	}

	// FR-007: record job in state
	if err := o.store.CreateJob(ctx, ticket.Identifier, worktreePath, sessionRef, o.cfg.ProjectID); err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	// Pipeline complete — mark as succeeded before Linear updates
	succeeded = true

	// FR-008 + FR-014: update ticket status (log on failure, don't roll back)
	if err := o.linear.SetTicketStatus(ctx, ticket.ID, statusInProgress); err != nil {
		logger.Error("failed to set ticket status (job is running)", "ticket", ticket.Identifier, "error", err)
	}

	// FR-009 + FR-014: remove ai-ready label (log on failure, don't roll back)
	if err := o.linear.RemoveLabel(ctx, ticket.ID, labelAIReady); err != nil {
		logger.Error("failed to remove ai-ready label (job is running)", "ticket", ticket.Identifier, "error", err)
	}

	logger.Info("ticket picked up successfully", "ticket", ticket.Identifier, "worktree", worktreePath, "session", sessionRef)
	return nil
}

// markNeedsAttention applies the needs-attention label and removes ai-ready.
// Errors are logged but not returned — this is best-effort after a failure.
func (o *Orchestrator) markNeedsAttention(ctx context.Context, ticket linear.Ticket) {
	logger := logging.Logger()

	if err := o.linear.ApplyLabel(ctx, ticket.ID, labelNeedsAttention); err != nil {
		logger.Error("failed to apply needs-attention label", "ticket", ticket.Identifier, "error", err)
	}
	if err := o.linear.RemoveLabel(ctx, ticket.ID, labelAIReady); err != nil {
		logger.Error("failed to remove ai-ready label after failure", "ticket", ticket.Identifier, "error", err)
	}
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// shortTitle derives a filesystem-safe short name from a ticket title.
// e.g., "Watcher 1 — ready ticket pickup" → "ready-ticket-pickup"
func shortTitle(title string) string {
	lower := strings.ToLower(title)
	safe := nonAlphanumeric.ReplaceAllString(lower, "-")
	safe = strings.Trim(safe, "-")

	// Truncate to a reasonable length
	if len(safe) > 50 {
		safe = safe[:50]
		safe = strings.TrimRight(safe, "-")
	}
	return safe
}
