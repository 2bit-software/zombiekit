package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/2bit-software/zombiekit/internal/admin"
	"github.com/2bit-software/zombiekit/internal/cmux"
	"github.com/2bit-software/zombiekit/internal/state"
	"github.com/2bit-software/zombiekit/internal/worktree"
	"github.com/urfave/cli/v2"
)

func openStore(c *cli.Context, mustExist bool) (*state.SQLiteStore, error) {
	dbPath := c.String("db-path")
	if dbPath == "" {
		return nil, fmt.Errorf("--db-path is required (or set ORCH_DB_PATH)")
	}
	if mustExist {
		if _, err := os.Stat(dbPath); err != nil {
			return nil, fmt.Errorf("database not found at %s", dbPath)
		}
	}
	return state.NewSQLiteStore(c.Context, dbPath)
}

func newAdminService(c *cli.Context) (*admin.Service, *state.SQLiteStore, error) {
	store, err := openStore(c, true)
	if err != nil {
		return nil, nil, err
	}
	return admin.New(store), store, nil
}

func formatTimestamp(t time.Time) string {
	return t.Local().Format("2006-01-02T15:04:05-07:00")
}

func truncateProjectID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func formatPR(pr *int64) string {
	if pr == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *pr)
}

// --- jobs subcommands ---

var projectFlag = &cli.StringFlag{
	Name:    "project",
	Aliases: []string{"p"},
	Usage:   "Filter by project ID",
	EnvVars: []string{"ORCH_PROJECT_ID"},
}

func jobsCommand() *cli.Command {
	return &cli.Command{
		Name:  "jobs",
		Usage: "Manage orchestrator jobs",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List all jobs",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "status",
						Usage: "Filter by status (repeatable)",
					},
					projectFlag,
				},
				Action: jobsList,
			},
			{
				Name:      "get",
				Usage:     "Show details for a single job",
				ArgsUsage: "<ticket-id>",
				Flags:     []cli.Flag{projectFlag},
				Action:    jobsGet,
			},
			{
				Name:      "delete",
				Usage:     "Delete a job and release its concurrency slot",
				ArgsUsage: "<ticket-id>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "session",
						Aliases: []string{"s"},
						Usage:   "Kill the cmux workspace for this job",
					},
					&cli.BoolFlag{
						Name:    "worktree",
						Aliases: []string{"w"},
						Usage:   "Delete the git worktree and branch for this job",
					},
					&cli.StringFlag{
						Name:    "repo-dir",
						Usage:   "Git repository root (required with --worktree)",
						EnvVars: []string{"ORCH_REPO_DIR"},
					},
					&cli.StringFlag{
						Name:    "worktrees-root",
						Usage:   "Worktrees root directory (optional, used with --worktree)",
						EnvVars: []string{"ORCH_WORKTREES_ROOT"},
					},
					projectFlag,
				},
				Action: jobsDelete,
			},
			{
				Name:      "set-status",
				Usage:     "Update a job's status",
				ArgsUsage: "<ticket-id> <status>",
				Flags:     []cli.Flag{projectFlag},
				Action:    jobsSetStatus,
			},
		},
	}
}

func jobsList(c *cli.Context) error {
	svc, store, err := newAdminService(c)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	defer func() { _ = store.Close() }()

	projectID := c.String("project")
	filter := admin.JobFilter{Statuses: c.StringSlice("status")}
	jobs, err := svc.ListJobs(c.Context, projectID, filter)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "TICKET\tSTATUS\tPROJECT\tPR\tUPDATED")
	for _, j := range jobs {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			j.TicketID, j.Status, truncateProjectID(j.ProjectID),
			formatPR(j.PRNumber), formatTimestamp(j.UpdatedAt),
		)
	}
	return w.Flush()
}

func jobsGet(c *cli.Context) error {
	ticketID := c.Args().First()
	if ticketID == "" {
		return cli.Exit("usage: orchestrator jobs get <ticket-id>", 1)
	}

	svc, store, err := newAdminService(c)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	defer func() { _ = store.Close() }()

	projectID := c.String("project")
	job, err := svc.GetJob(c.Context, projectID, ticketID)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	fmt.Printf("Ticket:     %s\n", job.TicketID)
	fmt.Printf("Status:     %s\n", job.Status)
	fmt.Printf("Project:    %s\n", job.ProjectID)
	fmt.Printf("Worktree:   %s\n", job.WorktreePath)
	fmt.Printf("Session:    %s\n", job.CmuxSession)
	fmt.Printf("PR:         %s\n", formatPR(job.PRNumber))
	fmt.Printf("Created:    %s\n", formatTimestamp(job.CreatedAt))
	fmt.Printf("Updated:    %s\n", formatTimestamp(job.UpdatedAt))
	return nil
}

func jobsDelete(c *cli.Context) error {
	ticketID := c.Args().First()
	if ticketID == "" {
		return cli.Exit("usage: orchestrator jobs delete <ticket-id>", 1)
	}

	svc, store, err := newAdminService(c)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	defer func() { _ = store.Close() }()

	var opts []admin.DeleteOption

	if c.Bool("session") {
		sessionMgr, err := cmux.New()
		if err != nil {
			return cli.Exit(fmt.Sprintf("--session: %s", err), 1)
		}
		opts = append(opts, admin.WithSessionCleanup(sessionMgr))
	}

	if c.Bool("worktree") {
		repoDir := c.String("repo-dir")
		if repoDir == "" {
			return cli.Exit("--worktree requires --repo-dir or ORCH_REPO_DIR", 1)
		}
		var wtOpts []worktree.Option
		if root := c.String("worktrees-root"); root != "" {
			wtOpts = append(wtOpts, worktree.WithWorktreesRoot(root))
		}
		wtMgr, err := worktree.New(repoDir, wtOpts...)
		if err != nil {
			return cli.Exit(fmt.Sprintf("--worktree: %s", err), 1)
		}
		opts = append(opts, admin.WithWorktreeCleanup(wtMgr))
	}

	projectID := c.String("project")
	result, err := svc.DeleteJob(c.Context, projectID, ticketID, opts...)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	msg := fmt.Sprintf("Deleted job %s (was: %s, project: %s)",
		ticketID, result.Job.Status, truncateProjectID(result.Job.ProjectID))
	if result.SlotReleased {
		msg += ", slot released"
	}
	fmt.Println(msg)

	if result.SessionKilled {
		fmt.Printf("  session killed: %s\n", result.Job.CmuxSession)
	}
	if result.SessionErr != nil {
		fmt.Printf("  session kill failed: %s\n", result.SessionErr)
	}
	if result.WorktreeDeleted {
		fmt.Printf("  worktree deleted: %s\n", result.Job.WorktreePath)
	}
	if result.WorktreeErr != nil {
		fmt.Printf("  worktree delete failed: %s\n", result.WorktreeErr)
	}

	return nil
}

func jobsSetStatus(c *cli.Context) error {
	args := c.Args()
	ticketID := args.Get(0)
	newStatus := args.Get(1)
	if ticketID == "" || newStatus == "" {
		return cli.Exit("usage: orchestrator jobs set-status <ticket-id> <status>", 1)
	}

	svc, store, err := newAdminService(c)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	defer func() { _ = store.Close() }()

	projectID := c.String("project")

	// Get current status for the confirmation message
	job, err := svc.GetJob(c.Context, projectID, ticketID)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	oldStatus := job.Status

	if err := svc.SetJobStatus(c.Context, projectID, ticketID, newStatus); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	fmt.Printf("Updated %s status: %s -> %s\n", ticketID, oldStatus, newStatus)
	return nil
}

// --- slots subcommands ---

func slotsCommand() *cli.Command {
	return &cli.Command{
		Name:  "slots",
		Usage: "Manage concurrency slots",
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List concurrency slot state",
				Action: slotsList,
			},
			{
				Name:   "reset",
				Usage:  "Reset all concurrency slots to zero",
				Action: slotsReset,
			},
		},
	}
}

func slotsList(c *cli.Context) error {
	svc, store, err := newAdminService(c)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	defer func() { _ = store.Close() }()

	slots, err := svc.ListSlots(c.Context)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "PROJECT\tACTIVE\tLIMIT")
	for _, s := range slots {
		_, _ = fmt.Fprintf(w, "%s\t%d\t%d\n", s.ProjectID, s.ActiveCount, s.SlotLimit)
	}
	return w.Flush()
}

func slotsReset(c *cli.Context) error {
	svc, store, err := newAdminService(c)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	defer func() { _ = store.Close() }()

	n, err := svc.ResetSlots(context.Background())
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	if n == 0 {
		fmt.Println("No slots to reset")
	} else {
		fmt.Printf("Reset %d project slot(s) to 0\n", n)
	}
	return nil
}
