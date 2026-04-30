package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/2bit-software/zombiekit/internal/worktree"
	"github.com/urfave/cli/v2"
)

// runGitWorktreeList shells out to `git worktree list` from repoDir and
// streams output to the CLI's stdout.
func runGitWorktreeList(c *cli.Context, repoDir string) error {
	cmd := exec.CommandContext(c.Context, "git", "worktree", "list")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// newWorktreeCommand returns the `brains worktree` command tree, exposing
// the orchestrator's worktree primitives for ad-hoc operator use.
func newWorktreeCommand() *cli.Command {
	return &cli.Command{
		Name:  "worktree",
		Usage: "Manage git worktrees using orchestrator conventions",
		Subcommands: []*cli.Command{
			newWorktreeCreateCommand(),
			newWorktreeDeleteCommand(),
			newWorktreePushCommand(),
			newWorktreeCleanBranchCommand(),
			newWorktreeListCommand(),
		},
	}
}

// loadWorktreeManager resolves the project config and constructs a
// worktree.GitManager wired with the configured root and copy-files.
func loadWorktreeManager(c *cli.Context) (*worktree.GitManager, error) {
	proj, err := loadProjectConfig(c)
	if err != nil {
		return nil, err
	}

	opts := []worktree.Option{worktree.WithWorktreesRoot(proj.WorktreesRoot)}
	if len(proj.CopyFiles) > 0 {
		opts = append(opts, worktree.WithCopyFiles(proj.CopyFiles))
	}

	mgr, err := worktree.New(proj.RepoDir, opts...)
	if err != nil {
		return nil, fmt.Errorf("init worktree manager: %w", err)
	}
	return mgr, nil
}

// newWorktreeCreateCommand creates a worktree at {root}/{ticket-id} on a
// branch derived from the title. Prints the resulting worktree path on success.
func newWorktreeCreateCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a worktree for a ticket",
		ArgsUsage: "<TICKET-ID> <TITLE>",
		Flags:     []cli.Flag{configFlag(), projectFlag()},
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: brains worktree create <TICKET-ID> <TITLE>")
			}
			ticketID := c.Args().Get(0)
			title := c.Args().Get(1)

			mgr, err := loadWorktreeManager(c)
			if err != nil {
				return err
			}

			path, err := mgr.CreateWorktree(c.Context, ticketID, title)
			if err != nil {
				return err
			}

			fmt.Println(path)
			return nil
		},
	}
}

// newWorktreeDeleteCommand removes a worktree directory and its branch.
func newWorktreeDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a worktree and its branch",
		ArgsUsage: "<PATH>",
		Flags:     []cli.Flag{configFlag(), projectFlag()},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("usage: brains worktree delete <PATH>")
			}
			path := c.Args().Get(0)

			mgr, err := loadWorktreeManager(c)
			if err != nil {
				return err
			}

			return mgr.DeleteWorktree(c.Context, path)
		},
	}
}

// newWorktreePushCommand pushes a worktree's branch to origin.
func newWorktreePushCommand() *cli.Command {
	return &cli.Command{
		Name:      "push",
		Usage:     "Push a worktree branch to origin",
		ArgsUsage: "<WORKTREE-PATH> <BRANCH>",
		Flags:     []cli.Flag{configFlag(), projectFlag()},
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: brains worktree push <WORKTREE-PATH> <BRANCH>")
			}
			path := c.Args().Get(0)
			branch := c.Args().Get(1)

			mgr, err := loadWorktreeManager(c)
			if err != nil {
				return err
			}

			return mgr.PushBranch(c.Context, path, branch)
		},
	}
}

// newWorktreeCleanBranchCommand force-deletes a local branch left behind by
// a removed worktree.
func newWorktreeCleanBranchCommand() *cli.Command {
	return &cli.Command{
		Name:      "clean-branch",
		Usage:     "Delete a local branch (e.g., orphan from a removed worktree)",
		ArgsUsage: "<BRANCH>",
		Flags:     []cli.Flag{configFlag(), projectFlag()},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("usage: brains worktree clean-branch <BRANCH>")
			}
			branch := c.Args().Get(0)

			mgr, err := loadWorktreeManager(c)
			if err != nil {
				return err
			}

			return mgr.CleanBranch(c.Context, branch)
		},
	}
}

// newWorktreeListCommand prints the worktrees the project's git repo knows
// about, in `git worktree list` format.
func newWorktreeListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List worktrees in the project repo",
		Flags: []cli.Flag{configFlag(), projectFlag()},
		Action: func(c *cli.Context) error {
			proj, err := loadProjectConfig(c)
			if err != nil {
				return err
			}
			return runGitWorktreeList(c, proj.RepoDir)
		},
	}
}
