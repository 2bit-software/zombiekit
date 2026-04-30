package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/2bit-software/zombiekit/internal/linear"
	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/workspace"
)

const (
	labelAIReady        = "ai-ready"
	labelNeedsAttention = "needs-attention"
	statusInProgress    = "In Progress"
)

// pollAndProcess runs one poll cycle: fetch tickets, process each sequentially.
func (p *ProjectRunner) pollAndProcess(ctx context.Context) {
	tickets, err := p.linear.PollReadyTickets(ctx, labelAIReady, p.id)
	if err != nil {
		p.logger.Error("failed to poll ready tickets", "error", err)
		return
	}

	for _, ticket := range tickets {
		if ctx.Err() != nil {
			p.logger.Info("context cancelled, stopping ticket processing")
			return
		}
		if err := p.processTicket(ctx, ticket); err != nil {
			p.logger.Error("failed to process ticket", "ticket", ticket.Identifier, "error", err)
		}
	}
}

// processTicket runs the pickup pipeline for a single ticket.
func (p *ProjectRunner) processTicket(ctx context.Context, ticket linear.Ticket) error {
	existing, err := p.store.GetJob(ctx, p.id, ticket.Identifier)
	if err != nil {
		return fmt.Errorf("check existing job: %w", err)
	}
	if existing != nil {
		p.logger.Debug("skipping ticket with existing job", "ticket", ticket.Identifier)
		return nil
	}

	acquired, err := p.store.TryAcquireSlot(ctx, p.id, p.cfg.ConcurrencyLimit)
	if err != nil {
		return fmt.Errorf("acquire slot: %w", err)
	}
	if !acquired {
		p.logger.Debug("concurrency limit reached, deferring ticket", "ticket", ticket.Identifier)
		return nil
	}

	sessionRef, worktreePath, err := p.runTicketPipeline(ctx, ticket)
	if err != nil {
		p.rollbackTicket(ctx, ticket, sessionRef, worktreePath, p.logger)
		return err
	}

	p.updateLinearAfterPickup(ctx, ticket, p.logger)
	p.logger.Info("ticket picked up successfully", "ticket", ticket.Identifier, "worktree", worktreePath, "session", sessionRef)
	return nil
}

// runTicketPipeline performs the core pickup sequence by delegating the
// worktree+sandbox+session steps to workspace.Prep, then records the job
// in the state store. Returns the session ref and worktree path so the
// caller can roll back on failure.
func (p *ProjectRunner) runTicketPipeline(ctx context.Context, ticket linear.Ticket) (sessionRef, worktreePath string, err error) {
	env := map[string]string{
		"WORK_CALLBACK_URL": fmt.Sprintf("http://localhost:%d/project/%s/%s", p.cfg.CallbackPort, p.id, ticket.Identifier),
	}
	if p.sandboxAvailable {
		env[sandbox.EnvSandboxName] = sandbox.Name(ticket.Identifier)
		for k, v := range p.sandboxConfig.HostEnv() {
			env[k] = v
		}
	}

	prompt := "Read .ai/ticket.md — this is your assigned ticket. Use /brains.new to begin."
	if hasLabel(ticket.Labels, "automode") {
		prompt = "Read .ai/ticket.md — this is your assigned ticket. Use /brains.new automode to begin."
	}

	result, err := p.workspace.Prep(ctx, workspace.PrepInput{
		TicketID:    ticket.Identifier,
		Title:       ticket.Title,
		Description: ticket.Description,
		Sandbox:     p.sandboxAvailable,
		Spawn:       &workspace.SpawnInput{Prompt: prompt, Env: env, SessionTitle: ticket.Title},
	})
	if err != nil {
		// workspace.Prep handled its own internal rollback already. Return
		// empty refs so rollbackTicket skips redundant teardown.
		return "", "", err
	}

	if err := p.store.CreateJob(ctx, ticket.Identifier, result.WorktreePath, result.SessionRef, p.id); err != nil {
		return result.SessionRef, result.WorktreePath, fmt.Errorf("create job: %w", err)
	}

	return result.SessionRef, result.WorktreePath, nil
}

// rollbackTicket reverses a partially-completed pipeline by tearing down
// the workspace (only when Prep succeeded but a later step failed),
// releasing the concurrency slot, and marking the ticket needs-attention.
// When Prep itself fails, it has already rolled back internally; we only
// release the slot and update Linear here.
func (p *ProjectRunner) rollbackTicket(ctx context.Context, ticket linear.Ticket, _ string, worktreePath string, logger *slog.Logger) {
	if worktreePath != "" {
		if err := p.workspace.Teardown(ctx, ticket.Identifier, worktreePath); err != nil {
			logger.Error("rollback: workspace teardown", "ticket", ticket.Identifier, "error", err)
		}
	}
	if releaseErr := p.store.ReleaseSlot(ctx, p.id); releaseErr != nil {
		logger.Error("rollback: failed to release slot", "ticket", ticket.Identifier, "error", releaseErr)
	}
	p.markTicketNeedsAttention(ctx, ticket)
}

// updateLinearAfterPickup sets the ticket status and removes the ai-ready
// label. These are best-effort -- the job is already running, so failures are
// logged but not propagated.
func (p *ProjectRunner) updateLinearAfterPickup(ctx context.Context, ticket linear.Ticket, logger *slog.Logger) {
	if err := p.linear.SetTicketStatus(ctx, ticket.ID, statusInProgress); err != nil {
		logger.Error("failed to set ticket status (job is running)", "ticket", ticket.Identifier, "error", err)
	}
	if err := p.linear.RemoveLabel(ctx, ticket.ID, labelAIReady); err != nil {
		logger.Error("failed to remove ai-ready label (job is running)", "ticket", ticket.Identifier, "error", err)
	}
}

// markTicketNeedsAttention applies the needs-attention label and removes ai-ready.
// Errors are logged but not returned -- this is best-effort after a failure.
func (p *ProjectRunner) markTicketNeedsAttention(ctx context.Context, ticket linear.Ticket) {
	if err := p.linear.ApplyLabel(ctx, ticket.ID, labelNeedsAttention); err != nil {
		p.logger.Error("failed to apply needs-attention label", "ticket", ticket.Identifier, "error", err)
	}
	if err := p.linear.RemoveLabel(ctx, ticket.ID, labelAIReady); err != nil {
		p.logger.Error("failed to remove ai-ready label after failure", "ticket", ticket.Identifier, "error", err)
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
