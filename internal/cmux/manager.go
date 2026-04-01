package cmux

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// New creates a CmuxManager after validating cmux is available and running.
func New(opts ...Option) (*CmuxManager, error) {
	cmuxBin, err := exec.LookPath("cmux")
	if err != nil {
		return nil, newError(ErrBinaryNotFound, "cmux not found on PATH", err)
	}

	m := &CmuxManager{
		cmuxBin:  cmuxBin,
		command:  "claude --permission-mode auto",
		sessions: make(map[string]sessionEntry),
	}

	for _, opt := range opts {
		opt(m)
	}

	if _, err := m.run(context.Background(), "ping"); err != nil {
		return nil, newError(ErrCmuxUnavailable, "cmux is not running or unreachable", err)
	}

	return m, nil
}

func (_m *CmuxManager) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, _m.cmuxBin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr == "" {
			stderrStr = strings.TrimSpace(stdout.String())
		}
		kind := classifyError(stderrStr)
		return "", newErrorf(kind, err, "cmux %s: %s", args[0], stderrStr)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// SpawnSession creates a cmux workspace for the given ticket.
//
// It checks both internal tracking and live cmux state before creating.
// The workspace is named "{ticketID}: {title}" for human identification.
func (_m *CmuxManager) SpawnSession(ctx context.Context, ticketID, title, worktreePath string, env map[string]string, prompt string) (string, error) {
	_m.mu.Lock()
	defer _m.mu.Unlock()

	if _, exists := _m.sessions[ticketID]; exists {
		return "", newErrorf(ErrSessionExists, nil, "session already tracked for %s", ticketID)
	}

	listOut, err := _m.run(ctx, "list-workspaces")
	if err != nil {
		return "", err
	}
	entries, err := parseListWorkspaces(listOut)
	if err != nil {
		return "", newError(ErrCommandFailed, err.Error(), err)
	}
	if found := findByTicketID(entries, ticketID); found != nil {
		return "", newErrorf(ErrSessionExists, nil,
			"cmux workspace already exists for %s: %s", ticketID, found.ref)
	}

	cmdStr, err := buildCommand(env, _m.command, prompt)
	if err != nil {
		return "", err
	}

	createOut, err := _m.run(ctx, "new-workspace", "--cwd", worktreePath, "--command", cmdStr)
	if err != nil {
		return "", err
	}
	ref, err := parseNewWorkspace(createOut)
	if err != nil {
		return "", newError(ErrCommandFailed, err.Error(), err)
	}

	name := ticketID + ": " + title
	if _, err := _m.run(ctx, "rename-workspace", "--workspace", ref, name); err != nil {
		_, _ = _m.run(ctx, "close-workspace", "--workspace", ref)
		return "", fmt.Errorf("rename-workspace failed (workspace closed to avoid orphan): %w", err)
	}

	_m.sessions[ticketID] = sessionEntry{ref: ref, name: name}
	return ref, nil
}

// KillSession closes the cmux workspace for the given ticket.
func (_m *CmuxManager) KillSession(ctx context.Context, ticketID string) error {
	_m.mu.Lock()
	defer _m.mu.Unlock()

	entry, exists := _m.sessions[ticketID]
	if !exists {
		return newErrorf(ErrSessionNotFound, nil, "no tracked session for %s", ticketID)
	}

	if _, err := _m.run(ctx, "close-workspace", "--workspace", entry.ref); err != nil {
		return err
	}

	delete(_m.sessions, ticketID)
	return nil
}

// SessionExists checks live cmux state for a workspace matching the ticket ID.
//
// If the internal tracker has a stale entry (workspace no longer in cmux), it
// is cleaned up automatically.
func (_m *CmuxManager) SessionExists(ctx context.Context, ticketID string) (bool, error) {
	listOut, err := _m.run(ctx, "list-workspaces")
	if err != nil {
		return false, err
	}

	entries, err := parseListWorkspaces(listOut)
	if err != nil {
		return false, err
	}

	if findByTicketID(entries, ticketID) != nil {
		return true, nil
	}

	_m.mu.Lock()
	delete(_m.sessions, ticketID)
	_m.mu.Unlock()

	return false, nil
}
