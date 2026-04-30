package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/2bit-software/zombiekit/internal/cmux"
	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "sandbox-test",
		Usage: "Test Docker Sandbox + cmux integration end-to-end",
		Commands: []*cli.Command{
			runCmd(),
			verifyCmd(),
			cleanupCmd(),
			statusCmd(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("sandbox-test failed", "error", err)
		os.Exit(1)
	}
}

func ticketFlag() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "ticket",
			Usage: "Ticket ID for the test session",
			Value: "TEST-SBX-1",
		},
	}
}

func runCmd() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Create sandbox + cmux session (full pipeline test)",
		Flags: append(ticketFlag(), []cli.Flag{
			&cli.StringFlag{
				Name:  "prompt",
				Usage: "Prompt to pass to claude inside the sandbox",
				Value: "Say 'Hello from sandbox!' and then exit with /exit",
			},
			&cli.StringFlag{
				Name:  "worktree",
				Usage: "Path to use as worktree (created in /tmp if empty)",
			},
			&cli.BoolFlag{
				Name:  "no-cmux",
				Usage: "Skip cmux session creation (just create sandbox)",
			},
		}...),
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			ticketID := c.String("ticket")
			sbxName := sandbox.Name(ticketID)

			// Preflight checks.
			if !sandbox.Available() {
				return fmt.Errorf("sbx not found on PATH")
			}
			if !c.Bool("no-cmux") {
				if _, err := exec.LookPath("cmux"); err != nil {
					return fmt.Errorf("cmux not found on PATH (use --no-cmux to skip cmux integration)")
				}
			}

			// Set up worktree.
			worktreePath := c.String("worktree")
			if worktreePath == "" {
				worktreePath = filepath.Join(os.TempDir(), "sbx-test-"+strings.ToLower(ticketID))
			}
			if err := setupTestWorktree(worktreePath, ticketID); err != nil {
				return fmt.Errorf("setup worktree: %w", err)
			}
			fmt.Printf("worktree: %s\n", worktreePath)

			// Create sandbox.
			cfg := sandbox.DefaultConfig()
			fmt.Printf("creating sandbox: %s\n", sbxName)
			fmt.Printf("  mounts: %v\n", cfg.Mounts)
			if err := sandbox.Create(ctx, sbxName, worktreePath, cfg); err != nil {
				return fmt.Errorf("create sandbox: %w", err)
			}
			fmt.Println("sandbox created")

			if c.Bool("no-cmux") {
				fmt.Printf("\nsandbox ready. Interact manually:\n")
				fmt.Printf("  sbx exec -it %s bash\n", sbxName)
				fmt.Printf("  sbx exec -it %s claude --dangerously-skip-permissions\n", sbxName)
				fmt.Printf("\ncleanup: sandbox-test cleanup --ticket %s\n", ticketID)
				return nil
			}

			// Spawn cmux session with the sandbox CommandBuilder wired in.
			builder := sandbox.NewCommandBuilder(cfg)
			env := map[string]string{
				"WORK_CALLBACK_URL":    "http://localhost:8666/" + ticketID,
				sandbox.EnvSandboxName: sbxName,
			}
			for k, v := range cfg.HostEnv() {
				env[k] = v
			}
			prompt := c.String("prompt")

			// Preview the command that will be generated.
			cmdStr, _, err := builder(worktreePath, env, "claude --dangerously-skip-permissions", prompt)
			if err != nil {
				return fmt.Errorf("build command: %w", err)
			}
			fmt.Printf("\ncmux command: %s\n", cmdStr)

			mgr, err := cmux.New(cmux.WithCommandBuilder(builder))
			if err != nil {
				return fmt.Errorf("cmux init: %w", err)
			}

			ref, err := mgr.SpawnSession(ctx, ticketID, "sandbox-test", worktreePath, env, prompt)
			if err != nil {
				return fmt.Errorf("spawn session: %w", err)
			}
			fmt.Printf("\ncmux session spawned: %s\n", ref)
			fmt.Printf("ticket: %s\n", ticketID)
			fmt.Printf("sandbox: %s\n", sbxName)
			fmt.Printf("\nThe session is now running in cmux. Switch to it with:\n")
			fmt.Printf("  cmux\n")
			fmt.Printf("\ncleanup when done:\n")
			fmt.Printf("  sandbox-test cleanup --ticket %s\n", ticketID)

			return nil
		},
	}
}

func verifyCmd() *cli.Command {
	return &cli.Command{
		Name:  "verify",
		Usage: "Verify mounts, networking, and config inside an existing sandbox",
		Flags: ticketFlag(),
		Action: func(c *cli.Context) error {
			ticketID := c.String("ticket")
			sbxName := sandbox.Name(ticketID)

			checks := []struct {
				name string
				args []string
			}{
				{"worktree mounted", []string{"exec", sbxName, "ls", "/tmp/sbx-test-" + strings.ToLower(ticketID) + "/.ai/ticket.md"}},
				{"~/.claude visible", []string{"exec", sbxName, "test", "-d", os.Getenv("HOME") + "/.claude"}},
				{"~/.brains visible", []string{"exec", sbxName, "test", "-d", os.Getenv("HOME") + "/.brains"}},
				{"claude binary exists", []string{"exec", sbxName, "which", "claude"}},
				{"host.docker.internal resolves", []string{"exec", sbxName, "getent", "hosts", "host.docker.internal"}},
				{"HTTP to host", []string{"exec", sbxName, "curl", "-sf", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "3", "http://host.docker.internal:8666/healthz"}},
			}

			pass, fail := 0, 0
			for _, check := range checks {
				cmd := exec.Command("sbx", check.args...)
				out, err := cmd.CombinedOutput()
				status := "PASS"
				if err != nil {
					status = "FAIL"
					fail++
				} else {
					pass++
				}
				detail := strings.TrimSpace(string(out))
				if len(detail) > 80 {
					detail = detail[:80] + "..."
				}
				if detail != "" {
					fmt.Printf("  [%s] %s: %s\n", status, check.name, detail)
				} else {
					fmt.Printf("  [%s] %s\n", status, check.name)
				}
			}

			fmt.Printf("\n%d passed, %d failed\n", pass, fail)
			if fail > 0 {
				return fmt.Errorf("%d checks failed", fail)
			}
			return nil
		},
	}
}

func cleanupCmd() *cli.Command {
	return &cli.Command{
		Name:  "cleanup",
		Usage: "Clean up sandbox + cmux session for a ticket",
		Flags: append(ticketFlag(), &cli.BoolFlag{
			Name:  "all",
			Usage: "Clean up all zk-* sandboxes",
		}),
		Action: func(c *cli.Context) error {
			ctx := context.Background()

			if c.Bool("all") {
				return cleanupAll(ctx)
			}

			ticketID := c.String("ticket")
			sbxName := sandbox.Name(ticketID)
			fmt.Printf("cleaning up ticket %s (sandbox: %s)\n", ticketID, sbxName)

			// Kill cmux session (best-effort).
			if mgr, err := cmux.New(); err == nil {
				if err := mgr.KillSession(ctx, ticketID); err != nil {
					fmt.Printf("  cmux kill: %s (may not exist)\n", err)
				} else {
					fmt.Println("  cmux session killed")
				}
			}

			// Cleanup sandbox (idempotent).
			sandbox.Cleanup(ctx, sbxName)
			fmt.Println("  sandbox cleaned up")

			// Remove temp worktree.
			tmpPath := filepath.Join(os.TempDir(), "sbx-test-"+strings.ToLower(ticketID))
			if err := os.RemoveAll(tmpPath); err == nil {
				fmt.Printf("  removed %s\n", tmpPath)
			}

			return nil
		},
	}
}

func statusCmd() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show current sandboxes and cmux sessions",
		Action: func(c *cli.Context) error {
			fmt.Println("=== Sandboxes ===")
			sbxOut, err := exec.Command("sbx", "ls").CombinedOutput()
			if err != nil {
				fmt.Printf("  sbx ls failed: %s\n", err)
			} else {
				out := strings.TrimSpace(string(sbxOut))
				if out == "" {
					fmt.Println("  (none)")
				} else {
					fmt.Println(out)
				}
			}

			fmt.Println("\n=== cmux sessions ===")
			cmuxOut, err := exec.Command("cmux", "list-workspaces").CombinedOutput()
			if err != nil {
				fmt.Printf("  cmux list failed: %s\n", err)
			} else {
				out := strings.TrimSpace(string(cmuxOut))
				if out == "" {
					fmt.Println("  (none)")
				} else {
					fmt.Println(out)
				}
			}

			return nil
		},
	}
}

func cleanupAll(ctx context.Context) error {
	out, err := exec.Command("sbx", "ls", "--quiet").CombinedOutput()
	if err != nil {
		return fmt.Errorf("sbx ls: %w", err)
	}

	cleaned := 0
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, "zk-") {
			fmt.Printf("  cleaning up %s\n", name)
			sandbox.Cleanup(ctx, name)
			cleaned++
		}
	}

	if cleaned == 0 {
		fmt.Println("no zk-* sandboxes found")
	} else {
		fmt.Printf("%d sandboxes cleaned up\n", cleaned)
	}
	return nil
}

func setupTestWorktree(path, ticketID string) error {
	aiDir := filepath.Join(path, ".ai")
	if err := os.MkdirAll(aiDir, 0o755); err != nil {
		return err
	}

	content := fmt.Sprintf(`# %s: Sandbox Integration Test

This is a test ticket for verifying Docker Sandbox integration.

## Acceptance Criteria
- [ ] Claude Code starts inside the sandbox
- [ ] Can read this ticket file
- [ ] Can access ~/.claude config
- [ ] Callback URL is rewritten to host.docker.internal
`, ticketID)

	return os.WriteFile(filepath.Join(aiDir, "ticket.md"), []byte(content), 0o644)
}
