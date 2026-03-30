// Package cli implements the brains command-line interface.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/urfave/cli/v2"

	"github.com/2bit-software/zombiekit/internal/config"
	"github.com/2bit-software/zombiekit/internal/memory"
	"github.com/2bit-software/zombiekit/internal/memory/sqlite"
)

// newMemoryCommand creates the memory subcommand with all memory operations.
func newMemoryCommand() *cli.Command {
	return &cli.Command{
		Name:  "memory",
		Usage: "Manage sticky memories",
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List all memories",
				Action: memoryList,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "format",
						Usage: "Output format (text, json)",
						Value: "text",
					},
					&cli.IntFlag{
						Name:  "limit",
						Usage: "Maximum number of items to return",
						Value: 100,
					},
				},
			},
			{
				Name:      "get",
				Usage:     "Get a memory by name",
				ArgsUsage: "<name>",
				Action:    memoryGet,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "format",
						Usage: "Output format (text, json)",
						Value: "text",
					},
				},
			},
			{
				Name:      "set",
				Usage:     "Set a memory value",
				ArgsUsage: "<name> <content>",
				Action:    memorySet,
			},
			{
				Name:      "delete",
				Usage:     "Delete a memory",
				ArgsUsage: "<name>",
				Action:    memoryDelete,
			},
			{
				Name:      "search",
				Usage:     "Search memories by name or content",
				ArgsUsage: "<query>",
				Action:    memorySearch,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "format",
						Usage: "Output format (text, json)",
						Value: "text",
					},
					&cli.IntFlag{
						Name:  "limit",
						Usage: "Maximum number of items to return",
						Value: 100,
					},
				},
			},
			{
				Name:   "clear",
				Usage:  "Clear all memories",
				Action: memoryClear,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "force",
						Usage: "Skip confirmation prompt",
						Value: false,
					},
				},
			},
		},
	}
}

func getStorage(c *cli.Context) (memory.Storage, error) {
	ctx := context.Background()
	cfg := config.LoadStorageConfigFromEnv()

	// Override with CLI flag if provided
	if dbPath := c.String("db-path"); dbPath != "" {
		cfg.SQLitePath = dbPath
	}

	return sqlite.NewSQLiteStorage(ctx, cfg.SQLitePath)
}

func memoryList(c *cli.Context) error {
	storage, err := getStorage(c)
	if err != nil {
		return fmt.Errorf("failed to connect to storage: %w", err)
	}
	defer storage.Close()

	items, err := storage.List(context.Background(), "")
	if err != nil {
		return fmt.Errorf("failed to list memories: %w", err)
	}

	limit := c.Int("limit")
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	format := c.String("format")
	return outputMemoryList(c, items, format)
}

func memoryGet(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("name is required")
	}
	name := c.Args().Get(0)

	storage, err := getStorage(c)
	if err != nil {
		return fmt.Errorf("failed to connect to storage: %w", err)
	}
	defer storage.Close()

	result, err := storage.Get(context.Background(), name)
	if err != nil {
		return fmt.Errorf("failed to get memory: %w", err)
	}

	if !result.HasValue() {
		return fmt.Errorf("memory not found: %s", name)
	}

	item := result.Value()
	format := c.String("format")

	if format == "json" {
		data, _ := json.MarshalIndent(item, "", "  ")
		fmt.Fprintln(c.App.Writer, string(data))
	} else {
		fmt.Fprintln(c.App.Writer, item.Content)
	}

	return nil
}

func memorySet(c *cli.Context) error {
	if c.NArg() < 2 {
		return fmt.Errorf("name and content are required")
	}
	name := c.Args().Get(0)
	content := c.Args().Get(1)

	if len(content) > memory.MaxContentSize {
		return fmt.Errorf("content too large: maximum size is 1MB")
	}

	storage, err := getStorage(c)
	if err != nil {
		return fmt.Errorf("failed to connect to storage: %w", err)
	}
	defer storage.Close()

	if err := storage.Set(context.Background(), name, content); err != nil {
		return fmt.Errorf("failed to set memory: %w", err)
	}

	fmt.Fprintf(c.App.Writer, "Memory '%s' saved.\n", memory.SanitizeName(name))
	return nil
}

func memoryDelete(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("name is required")
	}
	name := c.Args().Get(0)

	storage, err := getStorage(c)
	if err != nil {
		return fmt.Errorf("failed to connect to storage: %w", err)
	}
	defer storage.Close()

	if err := storage.Delete(context.Background(), name); err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	fmt.Fprintf(c.App.Writer, "Memory '%s' deleted.\n", memory.SanitizeName(name))
	return nil
}

func memorySearch(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("query is required")
	}
	query := c.Args().Get(0)

	storage, err := getStorage(c)
	if err != nil {
		return fmt.Errorf("failed to connect to storage: %w", err)
	}
	defer storage.Close()

	items, err := storage.List(context.Background(), query)
	if err != nil {
		return fmt.Errorf("failed to search memories: %w", err)
	}

	limit := c.Int("limit")
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	format := c.String("format")
	return outputMemoryList(c, items, format)
}

func memoryClear(c *cli.Context) error {
	if !c.Bool("force") {
		fmt.Fprint(c.App.Writer, "This will delete all memories. Use --force to confirm.\n")
		return nil
	}

	storage, err := getStorage(c)
	if err != nil {
		return fmt.Errorf("failed to connect to storage: %w", err)
	}
	defer storage.Close()

	count, err := storage.Clear(context.Background())
	if err != nil {
		return fmt.Errorf("failed to clear memories: %w", err)
	}

	fmt.Fprintf(c.App.Writer, "Cleared %d memories.\n", count)
	return nil
}

func outputMemoryList(c *cli.Context, items []memory.MemoryMetadata, format string) error {
	if format == "json" {
		data, _ := json.MarshalIndent(items, "", "  ")
		fmt.Fprintln(c.App.Writer, string(data))
		return nil
	}

	if len(items) == 0 {
		fmt.Fprintln(c.App.Writer, "No memories found.")
		return nil
	}

	w := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSIZE\tVERSION\tUPDATED")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%d\t%d\t%s\n",
			item.Name,
			item.Size,
			item.Version,
			item.UpdatedAt.Format("2006-01-02 15:04:05"),
		)
	}
	w.Flush()

	return nil
}
