package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/2bit-software/zombiekit/internal/cmux"
	"github.com/2bit-software/zombiekit/internal/orchestrator"
	"github.com/2bit-software/zombiekit/internal/sandbox"
	"github.com/2bit-software/zombiekit/internal/workspace"
	"github.com/2bit-software/zombiekit/internal/worktree"
	"github.com/urfave/cli/v2"
)

const defaultWorkspacePrompt = "Read .ai/ticket.md — this is your assigned ticket. Use /brains.new to begin."

// newWorkspaceCommand returns the `brains workspace` command tree, exposing
// the orchestrator's per-ticket pickup composition for ad-hoc operator use.
func newWorkspaceCommand() *cli.Command {
	return &cli.Command{
		Name:  "workspace",
		Usage: "Compose worktree + sandbox + (optional) cmux session into a ready workspace",
		Subcommands: []*cli.Command{
			newWorkspacePrepCommand(),
			newWorkspaceTeardownCommand(),
			newWorkspaceGCCommand(),
		},
	}
}

// buildWorktreeManagerFor instantiates a worktree.GitManager from a
// project's config (RepoDir, WorktreesRoot, CopyFiles).
func buildWorktreeManagerFor(proj *orchestrator.ProjectConfig) (*worktree.GitManager, error) {
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

// newWorkspacePrepCommand wires the orchestrator's pickup sequence behind
// a CLI: create worktree, write .ai/ticket.md, write workspace marker,
// optionally create a sandbox, optionally spawn a cmux session.
func newWorkspacePrepCommand() *cli.Command {
	return &cli.Command{
		Name:      "prep",
		Usage:     "Prep a workspace for a ticket (worktree + ticket.md + optional sandbox/spawn)",
		ArgsUsage: "<TICKET-ID>",
		Flags: []cli.Flag{
			configFlag(),
			projectFlag(),
			&cli.StringFlag{Name: "title", Usage: "Short title used to derive the branch name", Required: true},
			&cli.StringFlag{Name: "description", Usage: "Ticket description (written to .ai/ticket.md)"},
			&cli.StringFlag{Name: "description-file", Usage: "Path to a file whose contents become .ai/ticket.md"},
			&cli.BoolFlag{Name: "no-sandbox", Usage: "Skip sandbox creation even if sbx is available"},
			&cli.BoolFlag{Name: "spawn", Usage: "Launch a cmux session after prep (requires cmux on PATH)"},
			&cli.StringFlag{Name: "prompt", Usage: "Initial prompt for the spawned session", Value: defaultWorkspacePrompt},
			&cli.StringFlag{Name: "callback-url", Usage: "Override WORK_CALLBACK_URL passed to the spawned session"},
			&cli.StringFlag{Name: "format", Value: "text", Usage: "Output format: text or json"},
		},
		Action: runWorkspacePrep,
	}
}

// runWorkspacePrep is the prep subcommand action, factored out so the
// command struct stays small.
func runWorkspacePrep(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("usage: brains workspace prep <TICKET-ID>")
	}
	ws, input, err := assemblePrep(c)
	if err != nil {
		return err
	}
	result, err := ws.Prep(c.Context, input)
	if err != nil {
		return err
	}
	return printPrepResult(c, result)
}

// assemblePrep loads project config, wires the workspace manager and
// builds the PrepInput. Splits the I/O-heavy setup from Prep execution
// so each function stays simple.
func assemblePrep(c *cli.Context) (*workspace.Manager, workspace.PrepInput, error) {
	ticketID := c.Args().Get(0)
	description, err := resolveDescription(c)
	if err != nil {
		return nil, workspace.PrepInput{}, err
	}
	proj, err := loadProjectConfig(c)
	if err != nil {
		return nil, workspace.PrepInput{}, err
	}
	wtMgr, err := buildWorktreeManagerFor(proj)
	if err != nil {
		return nil, workspace.PrepInput{}, err
	}

	sbxCfg := sandbox.DefaultConfig()
	useSandbox := !c.Bool("no-sandbox") && sandbox.Available()
	useSpawn := c.Bool("spawn")

	ws, err := buildWorkspaceManager(wtMgr, proj.WorktreesRoot, sbxCfg, useSandbox, useSpawn)
	if err != nil {
		return nil, workspace.PrepInput{}, err
	}

	input := workspace.PrepInput{
		TicketID:    ticketID,
		Title:       c.String("title"),
		Description: description,
		Sandbox:     useSandbox,
	}
	if useSpawn {
		input.Spawn = buildSpawnInput(c, sbxCfg, ticketID, useSandbox)
	}
	return ws, input, nil
}

// buildWorkspaceManager assembles a workspace.Manager wired with optional
// cmux spawner (when --spawn is set) and the configured worktrees root.
func buildWorkspaceManager(wt *worktree.GitManager, worktreesRoot string, sbxCfg sandbox.Config, useSandbox, useSpawn bool) (*workspace.Manager, error) {
	opts := []workspace.Option{workspace.WithWorktreesRoot(worktreesRoot)}
	if useSpawn {
		cmuxOpts := []cmux.Option{}
		if useSandbox {
			cmuxOpts = append(cmuxOpts, cmux.WithCommandBuilder(sandbox.NewCommandBuilder(sbxCfg)))
		}
		cmuxMgr, err := cmux.New(cmuxOpts...)
		if err != nil {
			return nil, fmt.Errorf("init cmux (required for --spawn): %w", err)
		}
		opts = append(opts, workspace.WithSpawner(cmuxMgr))
	}
	return workspace.NewManager(wt, sbxCfg, opts...), nil
}

// buildSpawnInput assembles the env+prompt the spawned session receives.
func buildSpawnInput(c *cli.Context, sbxCfg sandbox.Config, ticketID string, useSandbox bool) *workspace.SpawnInput {
	env := map[string]string{}
	if cb := c.String("callback-url"); cb != "" {
		env["WORK_CALLBACK_URL"] = cb
	}
	if useSandbox {
		env[sandbox.EnvSandboxName] = sandbox.Name(ticketID)
		for k, v := range sbxCfg.HostEnv() {
			env[k] = v
		}
	}
	return &workspace.SpawnInput{
		Prompt:       c.String("prompt"),
		Env:          env,
		SessionTitle: c.String("title"),
	}
}

// resolveDescription reads --description (literal) or --description-file (file
// contents). Empty description is allowed and yields an empty .ai/ticket.md.
func resolveDescription(c *cli.Context) (string, error) {
	if d := c.String("description"); d != "" {
		return d, nil
	}
	if path := c.String("description-file"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read description-file: %w", err)
		}
		return string(data), nil
	}
	return "", nil
}

// printPrepResult renders the PrepResult as text or JSON depending on
// --format.
func printPrepResult(c *cli.Context, r workspace.PrepResult) error {
	if c.String("format") == "json" {
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("worktree:    %s\n", r.WorktreePath)
	fmt.Printf("branch:      %s\n", r.Branch)
	if r.SandboxName != "" {
		fmt.Printf("sandbox:     %s\n", r.SandboxName)
	}
	if r.SessionRef != "" {
		fmt.Printf("session:     %s\n", r.SessionRef)
	}
	return nil
}

// newWorkspaceTeardownCommand reverses a Prep: kill session (if any),
// cleanup sandbox, delete worktree+branch. Idempotent.
func newWorkspaceTeardownCommand() *cli.Command {
	return &cli.Command{
		Name:      "teardown",
		Usage:     "Tear down a workspace previously created by prep",
		ArgsUsage: "<TICKET-ID>",
		Flags: []cli.Flag{
			configFlag(),
			projectFlag(),
			&cli.BoolFlag{Name: "force", Usage: "Continue even if no marker is found"},
		},
		Action: runWorkspaceTeardown,
	}
}

func runWorkspaceTeardown(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("usage: brains workspace teardown <TICKET-ID>")
	}
	ticketID := c.Args().Get(0)

	proj, err := loadProjectConfig(c)
	if err != nil {
		return err
	}
	wtMgr, err := buildWorktreeManagerFor(proj)
	if err != nil {
		return err
	}

	worktreePath := filepath.Join(proj.WorktreesRoot, ticketID)
	if _, statErr := os.Stat(worktreePath); errors.Is(statErr, fs.ErrNotExist) && !c.Bool("force") {
		return fmt.Errorf("no worktree at %s; pass --force to teardown the sandbox anyway", worktreePath)
	}

	opts := []workspace.Option{workspace.WithWorktreesRoot(proj.WorktreesRoot)}
	if cmuxMgr, err := cmux.New(); err == nil {
		opts = append(opts, workspace.WithSpawner(cmuxMgr))
	}

	ws := workspace.NewManager(wtMgr, sandbox.DefaultConfig(), opts...)
	return ws.Teardown(c.Context, ticketID, worktreePath)
}

// newWorkspaceGCCommand finds orphan worktrees (without a marker) and
// stale zk-* sandboxes, then either reports or removes them.
func newWorkspaceGCCommand() *cli.Command {
	return &cli.Command{
		Name:  "gc",
		Usage: "Find and (optionally) remove orphan worktrees and stale sandboxes",
		Flags: []cli.Flag{
			configFlag(),
			projectFlag(),
			&cli.BoolFlag{Name: "dry-run", Value: true, Usage: "Report only; pass --dry-run=false to actually remove"},
		},
		Action: runWorkspaceGC,
	}
}

func runWorkspaceGC(c *cli.Context) error {
	proj, err := loadProjectConfig(c)
	if err != nil {
		return err
	}
	dryRun := c.Bool("dry-run")

	if err := gcOrphanWorktrees(proj.WorktreesRoot, dryRun); err != nil {
		return err
	}
	if err := gcStaleSandboxes(c.Context, dryRun); err != nil {
		return err
	}
	if dryRun {
		fmt.Println("(dry-run; pass --dry-run=false to remove)")
	}
	return nil
}

// gcOrphanWorktrees removes worktree directories with no .ai/workspace.json
// marker. In dry-run mode it only reports the candidates.
func gcOrphanWorktrees(root string, dryRun bool) error {
	orphans, err := findOrphanWorktrees(root)
	if err != nil {
		return err
	}
	for _, p := range orphans {
		fmt.Printf("orphan worktree: %s\n", p)
		if dryRun {
			continue
		}
		if rmErr := os.RemoveAll(p); rmErr != nil {
			fmt.Fprintf(os.Stderr, "  remove: %v\n", rmErr)
		}
	}
	return nil
}

// gcStaleSandboxes removes zk-* sandboxes still on the host. No-op when
// sbx is unavailable.
func gcStaleSandboxes(ctx context.Context, dryRun bool) error {
	if !sandbox.Available() {
		return nil
	}
	stale, err := listZKSandboxes(ctx)
	if err != nil {
		return err
	}
	for _, name := range stale {
		fmt.Printf("zk sandbox: %s\n", name)
		if !dryRun {
			sandbox.Cleanup(ctx, name)
		}
	}
	return nil
}

// findOrphanWorktrees lists worktree directories under root that lack a
// .ai/workspace.json marker. Returns absolute paths.
func findOrphanWorktrees(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read worktrees root: %w", err)
	}
	var orphans []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(root, e.Name())
		if _, err := workspace.ReadMarker(path); errors.Is(err, workspace.ErrNoMarker) {
			orphans = append(orphans, path)
		}
	}
	return orphans, nil
}

// listZKSandboxes returns sbx ls entries with the zk- prefix.
func listZKSandboxes(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "sbx", "ls", "--quiet")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("sbx ls: %w", err)
	}
	var result []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "zk-") {
			result = append(result, line)
		}
	}
	return result, nil
}
