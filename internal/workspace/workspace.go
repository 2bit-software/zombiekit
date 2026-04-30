package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/worktree"
)

// Sandbox is the subset of internal/sandbox the workspace Manager depends
// on. Tests substitute fakes so they don't require Docker.
type Sandbox interface {
	Available() bool
	Name(ticketID string) string
	Create(ctx context.Context, name, worktreePath string, cfg sandbox.Config) error
	Cleanup(ctx context.Context, name string)
}

// Spawner is the subset of cmux.SessionManager the workspace Manager depends
// on. Pass nil to disable session spawning entirely.
type Spawner interface {
	SpawnSession(ctx context.Context, ticketID, title, worktreePath string, env map[string]string, prompt string) (string, error)
	KillSession(ctx context.Context, ticketID string) error
}

// Manager coordinates worktree, sandbox, and (optional) session spawn
// steps for a single project.
type Manager struct {
	wt        worktree.Manager
	sbx       Sandbox
	sbxCfg    sandbox.Config
	spawner   Spawner
	logger    *slog.Logger
	rootGuess string
}

// Option configures a Manager.
type Option func(*Manager)

// WithSpawner attaches a session spawner so Prep can launch a cmux session
// when PrepInput.Spawn is set. Without one, --spawn is a no-op.
func WithSpawner(s Spawner) Option {
	return func(m *Manager) { m.spawner = s }
}

// WithSandbox replaces the default sandbox passthrough with a custom
// implementation, e.g. an in-memory fake for tests.
func WithSandbox(s Sandbox) Option {
	return func(m *Manager) { m.sbx = s }
}

// WithLogger overrides the default slog logger.
func WithLogger(l *slog.Logger) Option {
	return func(m *Manager) { m.logger = l }
}

// WithWorktreesRoot tells Teardown where to look up worktrees by ticket ID
// when the caller does not pass an explicit path. Defaults to empty (in
// which case the caller MUST pass a worktree path to Teardown).
func WithWorktreesRoot(root string) Option {
	return func(m *Manager) { m.rootGuess = root }
}

// NewManager builds a workspace Manager. wt and sbxCfg are required; the
// rest are optional.
func NewManager(wt worktree.Manager, sbxCfg sandbox.Config, opts ...Option) *Manager {
	m := &Manager{
		wt:     wt,
		sbx:    defaultSandbox{},
		sbxCfg: sbxCfg,
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// PrepInput is everything Prep needs to set up a workspace.
type PrepInput struct {
	TicketID    string
	Title       string
	Description string // written to .ai/ticket.md

	// Sandbox controls whether to create a Docker sandbox after the worktree.
	// Even when true, Prep skips the sandbox step if the underlying sbx CLI
	// is unavailable.
	Sandbox bool

	// Spawn, when non-nil, asks Prep to launch a cmux session after the
	// other steps. Requires that the Manager was built with WithSpawner.
	Spawn *SpawnInput
}

// SpawnInput captures the cmux session arguments. The orchestrator builds
// these (WORK_CALLBACK_URL, sandbox env, automode prompt) and the CLI
// builds simpler defaults — workspace stays neutral on the contents.
type SpawnInput struct {
	Prompt       string
	Env          map[string]string
	SessionTitle string
}

// PrepResult is the durable state Prep produced.
type PrepResult struct {
	WorktreePath string
	Branch       string
	SandboxName  string
	SessionRef   string
}

// Prep performs the orchestrator pickup sequence end-to-end:
// create worktree, write .ai/ticket.md, write .ai/workspace.json,
// (optional) create sandbox, (optional) spawn session. Failures
// roll back any preceding successful steps so the caller never
// inherits partial state.
func (m *Manager) Prep(ctx context.Context, in PrepInput) (PrepResult, error) {
	if in.TicketID == "" {
		return PrepResult{}, errors.New("workspace.Prep: TicketID is required")
	}

	branch := branchName(in.TicketID, in.Title)

	worktreePath, err := m.wt.CreateWorktree(ctx, in.TicketID, ShortTitle(in.Title))
	if err != nil {
		return PrepResult{}, fmt.Errorf("create worktree: %w", err)
	}

	result := PrepResult{WorktreePath: worktreePath, Branch: branch}

	rollback := func(reason error) (PrepResult, error) {
		if delErr := m.wt.DeleteWorktree(ctx, worktreePath); delErr != nil {
			m.logger.Error("workspace.Prep rollback: delete worktree", "ticket", in.TicketID, "error", delErr)
		}
		return PrepResult{}, reason
	}

	aiDir := filepath.Join(worktreePath, ".ai")
	if err := os.MkdirAll(aiDir, 0o755); err != nil {
		return rollback(fmt.Errorf("create .ai directory: %w", err))
	}

	ticketFile := filepath.Join(aiDir, "ticket.md")
	if err := os.WriteFile(ticketFile, []byte(in.Description), 0o644); err != nil {
		return rollback(fmt.Errorf("write ticket file: %w", err))
	}

	if in.Sandbox && m.sbx.Available() {
		name := m.sbx.Name(in.TicketID)
		if err := m.sbx.Create(ctx, name, worktreePath, m.sbxCfg); err != nil {
			return rollback(fmt.Errorf("create sandbox: %w", err))
		}
		result.SandboxName = name
	}

	if in.Spawn != nil && m.spawner != nil {
		title := in.Spawn.SessionTitle
		if title == "" {
			title = in.Title
		}
		ref, err := m.spawner.SpawnSession(ctx, in.TicketID, title, worktreePath, in.Spawn.Env, in.Spawn.Prompt)
		if err != nil {
			if result.SandboxName != "" {
				m.sbx.Cleanup(ctx, result.SandboxName)
			}
			return rollback(fmt.Errorf("spawn session: %w", err))
		}
		result.SessionRef = ref
	}

	marker := Marker{
		TicketID:     in.TicketID,
		Title:        in.Title,
		Branch:       branch,
		WorktreePath: worktreePath,
		SandboxName:  result.SandboxName,
		Spawned:      result.SessionRef != "",
		CreatedAt:    time.Now().UTC(),
	}
	if in.Spawn != nil {
		marker.Prompt = in.Spawn.Prompt
	}
	if err := writeMarker(worktreePath, marker); err != nil {
		// Marker write is the last step; Prep already succeeded otherwise.
		// Log and continue rather than rolling back — the workspace is
		// usable, just lacks the marker for Teardown's convenience.
		m.logger.Warn("workspace.Prep: write marker", "ticket", in.TicketID, "error", err)
	}

	return result, nil
}

// Teardown reverses a workspace setup. If worktreePath is empty, Teardown
// derives it from the manager's worktreesRoot guess. Each step is
// best-effort: failures are aggregated into a multierror but later steps
// still run.
func (m *Manager) Teardown(ctx context.Context, ticketID, worktreePath string) error {
	if ticketID == "" {
		return errors.New("workspace.Teardown: ticketID is required")
	}

	if worktreePath == "" && m.rootGuess != "" {
		worktreePath = filepath.Join(m.rootGuess, ticketID)
	}

	var marker Marker
	if worktreePath != "" {
		if mk, err := ReadMarker(worktreePath); err == nil {
			marker = mk
		}
	}

	sbxName := marker.SandboxName
	if sbxName == "" {
		sbxName = m.sbx.Name(ticketID)
	}

	var errs []error

	if m.spawner != nil {
		if err := m.spawner.KillSession(ctx, ticketID); err != nil {
			// Most "session not found" errors are expected; log at debug.
			m.logger.Debug("workspace.Teardown: kill session (may not exist)",
				"ticket", ticketID, "error", err)
		}
	}

	m.sbx.Cleanup(ctx, sbxName)

	if worktreePath != "" {
		if err := m.wt.DeleteWorktree(ctx, worktreePath); err != nil {
			errs = append(errs, fmt.Errorf("delete worktree: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// ShortTitle derives a filesystem-safe short name from a ticket title,
// matching the orchestrator's historic shortTitle behavior.
//
//	ShortTitle("Watcher 1 — ready ticket pickup") -> "watcher-1-ready-ticket-pickup"
func ShortTitle(title string) string {
	lower := strings.ToLower(title)
	safe := nonAlphanumeric.ReplaceAllString(lower, "-")
	safe = strings.Trim(safe, "-")

	if len(safe) > 50 {
		safe = safe[:50]
		safe = strings.TrimRight(safe, "-")
	}
	return safe
}

// branchName mirrors the worktree.Manager naming rule: {ticketID}/{short-title}.
// The exact branch is determined inside worktree.CreateWorktree, but we mirror
// it here for marker bookkeeping. If sanitization rules drift, the marker
// tells the truth (subject to mismatch with what git actually created).
func branchName(ticketID, title string) string {
	short := ShortTitle(title)
	if short == "" {
		return ticketID
	}
	return ticketID + "/" + short
}

// defaultSandbox is the package-level passthrough wrapping internal/sandbox.
// It is the value used when no WithSandbox option is supplied.
type defaultSandbox struct{}

func (defaultSandbox) Available() bool       { return sandbox.Available() }
func (defaultSandbox) Name(id string) string { return sandbox.Name(id) }
func (defaultSandbox) Create(ctx context.Context, name, wt string, cfg sandbox.Config) error {
	return sandbox.Create(ctx, name, wt, cfg)
}
func (defaultSandbox) Cleanup(ctx context.Context, name string) { sandbox.Cleanup(ctx, name) }
