package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/urfave/cli/v2"
)

// newSandboxCommand returns the `brains sandbox` command tree, exposing
// the orchestrator's sandbox primitives for ad-hoc operator use.
func newSandboxCommand() *cli.Command {
	return &cli.Command{
		Name:  "sandbox",
		Usage: "Manage Docker Sandboxes (sbx) using orchestrator conventions",
		Subcommands: []*cli.Command{
			newSandboxCreateCommand(),
			newSandboxCleanupCommand(),
			newSandboxAvailableCommand(),
			newSandboxNameCommand(),
			newSandboxListCommand(),
		},
	}
}

// sandboxConfigFromFlags builds a sandbox.Config from the CLI flags,
// starting from sandbox.DefaultConfig and overriding fields where flags
// were provided.
func sandboxConfigFromFlags(c *cli.Context) sandbox.Config {
	cfg := sandbox.DefaultConfig()
	if mounts := c.StringSlice("mounts"); len(mounts) > 0 {
		cfg.Mounts = mounts
	}
	if mem := c.String("memory"); mem != "" {
		cfg.Memory = mem
	}
	if tpl := c.String("template"); tpl != "" {
		cfg.Template = tpl
	}
	return cfg
}

// sandboxConfigFlags are the flags shared by `sandbox create` and other
// commands that take a sandbox config.
func sandboxConfigFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "mounts",
			Usage: "Override default mounts (repeatable). Format: PATH or PATH:ro. Replaces defaults entirely.",
		},
		&cli.StringFlag{
			Name:  "memory",
			Usage: "VM memory limit (e.g., 8g)",
		},
		&cli.StringFlag{
			Name:  "template",
			Usage: "Custom container template (default: claude agent template)",
		},
	}
}

// newSandboxCreateCommand provisions a Docker Sandbox for a ticket's
// worktree using deterministic naming.
func newSandboxCreateCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a Docker Sandbox for a ticket's worktree",
		ArgsUsage: "<TICKET-ID> <WORKTREE-PATH>",
		Flags:     sandboxConfigFlags(),
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: brains sandbox create <TICKET-ID> <WORKTREE-PATH>")
			}
			ticketID := c.Args().Get(0)
			worktreePath := c.Args().Get(1)

			if !sandbox.Available() {
				return fmt.Errorf("sbx not found on PATH; install Docker Sandbox first")
			}

			name := sandbox.Name(ticketID)
			cfg := sandboxConfigFromFlags(c)

			if err := sandbox.Create(c.Context, name, worktreePath, cfg); err != nil {
				return err
			}

			fmt.Println(name)
			return nil
		},
	}
}

// newSandboxCleanupCommand removes a ticket's sandbox. Idempotent: silent
// success if the sandbox is missing or sbx is unavailable.
func newSandboxCleanupCommand() *cli.Command {
	return &cli.Command{
		Name:      "cleanup",
		Usage:     "Remove a sandbox (idempotent)",
		ArgsUsage: "<TICKET-ID>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("usage: brains sandbox cleanup <TICKET-ID>")
			}
			ticketID := c.Args().Get(0)
			sandbox.Cleanup(c.Context, sandbox.Name(ticketID))
			return nil
		},
	}
}

// newSandboxAvailableCommand reports whether the sbx CLI is installed.
// Exits zero when available, non-zero otherwise.
func newSandboxAvailableCommand() *cli.Command {
	return &cli.Command{
		Name:  "available",
		Usage: "Check whether the sbx CLI is installed",
		Action: func(c *cli.Context) error {
			if !sandbox.Available() {
				return fmt.Errorf("sbx not found on PATH")
			}
			fmt.Println("sbx is available")
			return nil
		},
	}
}

// newSandboxNameCommand prints the deterministic sandbox name for a ticket
// ID so external scripts can address it without re-implementing the rule.
func newSandboxNameCommand() *cli.Command {
	return &cli.Command{
		Name:      "name",
		Usage:     "Print the deterministic sandbox name for a ticket ID",
		ArgsUsage: "<TICKET-ID>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("usage: brains sandbox name <TICKET-ID>")
			}
			fmt.Println(sandbox.Name(c.Args().Get(0)))
			return nil
		},
	}
}

// newSandboxListCommand lists zombiekit-managed sandboxes by filtering
// `sbx ls --quiet` output to entries with the zk- prefix.
func newSandboxListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List zombiekit-managed sandboxes (zk-* prefix)",
		Action: func(c *cli.Context) error {
			if !sandbox.Available() {
				return fmt.Errorf("sbx not found on PATH")
			}

			cmd := exec.CommandContext(c.Context, "sbx", "ls", "--quiet")
			out, err := cmd.Output()
			if err != nil {
				return fmt.Errorf("sbx ls: %w", err)
			}

			zkSandboxes := filterZKPrefix(string(out))
			if len(zkSandboxes) == 0 {
				return nil
			}
			for _, name := range zkSandboxes {
				if _, err := fmt.Fprintln(os.Stdout, name); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// filterZKPrefix returns lines from sbx output that start with the zk-
// prefix, in their input order.
func filterZKPrefix(output string) []string {
	var result []string
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "zk-") {
			result = append(result, line)
		}
	}
	return result
}
