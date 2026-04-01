package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/2bit-software/zombiekit/internal/linear"
	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/shutdown"
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

	existing, err := o.store.GetJob(ctx, ticket.Identifier)
	if err != nil {
		return fmt.Errorf("check existing job: %w", err)
	}
	if existing != nil {
		logger.Debug("skipping ticket with existing job", "ticket", ticket.Identifier)
		return nil
	}

	acquired, err := o.store.TryAcquireSlot(ctx, o.cfg.ProjectID, o.cfg.ConcurrencyLimit)
	if err != nil {
		return fmt.Errorf("acquire slot: %w", err)
	}
	if !acquired {
		logger.Debug("concurrency limit reached, deferring ticket", "ticket", ticket.Identifier)
		return nil
	}

	sessionRef, worktreePath, err := o.runTicketPipeline(ctx, ticket)
	if err != nil {
		o.rollbackTicket(ctx, ticket, sessionRef, worktreePath, logger)
		return err
	}

	o.updateLinearAfterPickup(ctx, ticket, logger)
	logger.Info("ticket picked up successfully", "ticket", ticket.Identifier, "worktree", worktreePath, "session", sessionRef)
	return nil
}

// runTicketPipeline performs the core pickup sequence: worktree creation,
// session spawn, and job recording. Returns the session ref and worktree path
// so the caller can roll back on failure.
func (o *Orchestrator) runTicketPipeline(ctx context.Context, ticket linear.Ticket) (sessionRef, worktreePath string, err error) {
	worktreePath, err = o.setupWorktree(ctx, ticket)
	if err != nil {
		return "", worktreePath, err
	}

	if o.cfg.SandboxAvailable {
		sbxName := sandbox.Name(ticket.Identifier)
		if err := sandbox.Create(ctx, sbxName, worktreePath, o.cfg.SandboxConfig); err != nil {
			return "", worktreePath, fmt.Errorf("create sandbox: %w", err)
		}
	}

	env := map[string]string{
		"WORK_CALLBACK_URL": fmt.Sprintf("http://localhost:%d/%s", o.cfg.CallbackPort, ticket.Identifier),
	}
	if o.cfg.SandboxAvailable {
		env[sandbox.EnvSandboxName] = sandbox.Name(ticket.Identifier)
		for k, v := range o.cfg.SandboxConfig.HostEnv() {
			env[k] = v
		}
	}
	prompt := "Read .ai/ticket.md — this is your assigned ticket. Use /brains.new to begin."
	if hasLabel(ticket.Labels, "automode") {
		prompt = "Read .ai/ticket.md — this is your assigned ticket. Use /brains.new automode to begin."
	}
	sessionRef, err = o.sessions.SpawnSession(ctx, ticket.Identifier, ticket.Title, worktreePath, env, prompt)
	if err != nil {
		return "", worktreePath, fmt.Errorf("spawn session: %w", err)
	}

	if err := o.store.CreateJob(ctx, ticket.Identifier, worktreePath, sessionRef, o.cfg.ProjectID); err != nil {
		return sessionRef, worktreePath, fmt.Errorf("create job: %w", err)
	}

	return sessionRef, worktreePath, nil
}

// rollbackTicket reverses a partially-completed pipeline in reverse order,
// then marks the ticket as needs-attention.
func (o *Orchestrator) rollbackTicket(ctx context.Context, ticket linear.Ticket, sessionRef, worktreePath string, logger *slog.Logger) {
	if sessionRef != "" {
		if killErr := o.sessions.KillSession(ctx, ticket.Identifier); killErr != nil {
			logger.Error("rollback: failed to kill session", "ticket", ticket.Identifier, "error", killErr)
		}
	}

	if o.cfg.SandboxAvailable {
		sandbox.Cleanup(ctx, sandbox.Name(ticket.Identifier))
	}

	if worktreePath != "" {
		if delErr := o.worktrees.DeleteWorktree(ctx, worktreePath); delErr != nil {
			logger.Error("rollback: failed to delete worktree", "ticket", ticket.Identifier, "error", delErr)
		}
	}
	if releaseErr := o.store.ReleaseSlot(ctx, o.cfg.ProjectID); releaseErr != nil {
		logger.Error("rollback: failed to release slot", "ticket", ticket.Identifier, "error", releaseErr)
	}
	o.markNeedsAttention(ctx, ticket)
}

// updateLinearAfterPickup sets the ticket status and removes the ai-ready
// label. These are best-effort -- the job is already running, so failures are
// logged but not propagated.
func (o *Orchestrator) updateLinearAfterPickup(ctx context.Context, ticket linear.Ticket, logger *slog.Logger) {
	if err := o.linear.SetTicketStatus(ctx, ticket.ID, statusInProgress); err != nil {
		logger.Error("failed to set ticket status (job is running)", "ticket", ticket.Identifier, "error", err)
	}
	if err := o.linear.RemoveLabel(ctx, ticket.ID, labelAIReady); err != nil {
		logger.Error("failed to remove ai-ready label (job is running)", "ticket", ticket.Identifier, "error", err)
	}
}

// setupWorktree creates a git worktree for the ticket and writes the ticket
// description to .ai/ticket.md inside it.
func (o *Orchestrator) setupWorktree(ctx context.Context, ticket linear.Ticket) (string, error) {
	worktreePath, err := o.worktrees.CreateWorktree(ctx, ticket.Identifier, shortTitle(ticket.Title))
	if err != nil {
		return "", fmt.Errorf("create worktree: %w", err)
	}

	aiDir := filepath.Join(worktreePath, ".ai")
	if err := os.MkdirAll(aiDir, 0o755); err != nil {
		return worktreePath, fmt.Errorf("create .ai directory: %w", err)
	}

	ticketFile := filepath.Join(aiDir, "ticket.md")
	if err := os.WriteFile(ticketFile, []byte(ticket.Description), 0o644); err != nil {
		return worktreePath, fmt.Errorf("write ticket file: %w", err)
	}

	return worktreePath, nil
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

// hasLabel reports whether labels contains the given name (case-insensitive).
func hasLabel(labels []string, name string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, name) {
			return true
		}
	}
	return false
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
