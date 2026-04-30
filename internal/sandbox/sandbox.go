// Package sandbox provides utilities for running Claude Code sessions inside
// Docker Sandboxes (sbx CLI). It is a stateless utility package -- all functions
// are safe for concurrent use.
//
// Sandbox names are derived deterministically from ticket IDs, so cleanup can
// be attempted without tracking whether a sandbox was created. If sbx is not
// installed or the sandbox does not exist, cleanup is a no-op.
//
// Prerequisites (one-time):
//
//	sbx login
//	sbx policy allow network host.docker.internal
package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Config holds sandbox settings that flow through the orchestrator pipeline.
type Config struct {
	// Mounts are extra paths to mount read-only inside the sandbox.
	// Paths ending in ":ro" are mounted read-only; otherwise ":ro" is appended.
	// Default: ["~/.claude:ro", "~/.brains:ro"]
	Mounts []string

	// CallbackHost replaces "localhost" and "127.0.0.1" in callback URLs so the
	// sandbox can reach the host. Default: "host.docker.internal"
	CallbackHost string

	// Memory is the VM memory limit (e.g., "8g"). Empty means sbx default.
	Memory string

	// Template is a custom container image for the sandbox. Empty means the
	// default claude agent template.
	Template string

	// PassthroughEnv lists environment variable names to read from the host
	// and inject into sandbox sessions. Used for API keys that the agent
	// needs but shouldn't be stored in mounted config files.
	// Default: ["ANTHROPIC_API_KEY"]
	PassthroughEnv []string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Mounts: []string{
			filepath.Join(home, ".claude"), // rw: Claude Code needs to refresh OAuth tokens
			filepath.Join(home, ".brains") + ":ro",
		},
		CallbackHost:   "host.docker.internal",
		PassthroughEnv: []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN"},
	}
}

// HostEnv reads PassthroughEnv variables from the current process environment.
// Returns only keys that are actually set. These should be merged into the
// session env map before calling SpawnSession.
func (c Config) HostEnv() map[string]string {
	env := make(map[string]string)
	for _, key := range c.PassthroughEnv {
		if val := os.Getenv(key); val != "" {
			env[key] = val
		}
	}
	return env
}

const namePrefix = "zk-"

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// Name returns a deterministic, DNS-safe sandbox name for a ticket ID.
// The name is prefixed with "zk-" to identify zombiekit-managed sandboxes.
//
//	Name("DEV-123") => "zk-dev-123"
func Name(ticketID string) string {
	lower := strings.ToLower(ticketID)
	safe := nonAlphanumeric.ReplaceAllString(lower, "-")
	safe = strings.Trim(safe, "-")

	name := namePrefix + safe
	if len(name) > 63 {
		name = name[:63]
		name = strings.TrimRight(name, "-")
	}
	return name
}

// Available reports whether the sbx CLI is on PATH.
func Available() bool {
	_, err := exec.LookPath("sbx")
	return err == nil
}

// Create creates a Docker Sandbox for the given worktree. The sandbox is named
// deterministically from the ticket ID via Name(). If a sandbox with the same
// name already exists (e.g., from a crash), it is removed first.
func Create(ctx context.Context, name, worktreePath string, cfg Config) error {
	sbxBin, err := exec.LookPath("sbx")
	if err != nil {
		return fmt.Errorf("sbx not found on PATH: %w", err)
	}

	// Remove stale sandbox with same name (crash recovery).
	cleanupSandbox(ctx, sbxBin, name)

	args := []string{"create", "claude", worktreePath}

	for _, mount := range resolveMounts(cfg.Mounts) {
		args = append(args, mount)
	}

	args = append(args, "--name", name, "-q")

	if cfg.Memory != "" {
		args = append(args, "-m", cfg.Memory)
	}
	if cfg.Template != "" {
		args = append(args, "-t", cfg.Template)
	}

	cmd := exec.CommandContext(ctx, sbxBin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sbx create %s: %s: %w", name, strings.TrimSpace(stderr.String()), err)
	}

	return nil
}

// Cleanup stops and removes a sandbox. It is idempotent: if sbx is not
// installed, the sandbox does not exist, or it is already removed, this
// function logs at debug level and returns without error.
func Cleanup(ctx context.Context, name string) {
	sbxBin, err := exec.LookPath("sbx")
	if err != nil {
		return
	}
	cleanupSandbox(ctx, sbxBin, name)
}

func cleanupSandbox(ctx context.Context, sbxBin, name string) {
	logger := slog.Default()

	stop := exec.CommandContext(ctx, sbxBin, "stop", name)
	if out, err := stop.CombinedOutput(); err != nil {
		logger.Debug("sandbox stop (may not exist)", slog.String("name", name), slog.String("output", strings.TrimSpace(string(out))))
	}

	rm := exec.CommandContext(ctx, sbxBin, "rm", name)
	if out, err := rm.CombinedOutput(); err != nil {
		logger.Debug("sandbox rm (may not exist)", slog.String("name", name), slog.String("output", strings.TrimSpace(string(out))))
	}
}

// resolveMounts expands ~ to the home directory. Mounts with a ":ro" suffix
// are read-only; bare paths are read-write. Mounts whose base path does not
// exist are skipped with a warning.
func resolveMounts(mounts []string) []string {
	home, _ := os.UserHomeDir()
	var resolved []string

	for _, m := range mounts {
		path, suffix, hasSuffix := strings.Cut(m, ":")
		if strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, path[2:])
		} else if path == "~" {
			path = home
		}

		if _, err := os.Stat(path); err != nil {
			slog.Default().Warn("sandbox mount path does not exist, skipping", slog.String("path", path))
			continue
		}

		if hasSuffix {
			resolved = append(resolved, path+":"+suffix)
		} else {
			resolved = append(resolved, path)
		}
	}

	return resolved
}
