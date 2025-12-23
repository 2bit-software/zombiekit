// Package cli implements the brains command-line interface.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/urfave/cli/v2"

	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/database"
)

// newDBCommand creates the db subcommand with migration operations.
func newDBCommand() *cli.Command {
	return &cli.Command{
		Name:  "db",
		Usage: "Database management commands",
		Subcommands: []*cli.Command{
			{
				Name:   "migrate",
				Usage:  "Apply pending database migrations",
				Action: dbMigrate,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "db-type",
						Value:   "sqlite",
						Usage:   "Database backend: sqlite, postgres",
						EnvVars: []string{"BRAINS_BACKEND"},
					},
				},
			},
			{
				Name:   "status",
				Usage:  "Show migration status",
				Action: dbStatus,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "db-type",
						Value:   "sqlite",
						Usage:   "Database backend: sqlite, postgres",
						EnvVars: []string{"BRAINS_BACKEND"},
					},
					&cli.StringFlag{
						Name:  "format",
						Usage: "Output format (text, json)",
						Value: "text",
					},
				},
			},
		},
	}
}

func dbMigrate(c *cli.Context) error {
	ctx := context.Background()
	cfg := config.LoadStorageConfigFromEnv()

	// Override with CLI flag if provided
	if dbType := c.String("db-type"); dbType != "" {
		cfg.Backend = config.BackendType(dbType)
	}

	// Get status before migrations to know what's pending
	beforeStatuses, _ := database.GetMigrationStatus(ctx, cfg)
	pendingBefore := 0
	for _, s := range beforeStatuses {
		if !s.Applied {
			pendingBefore++
		}
	}

	// Run migrations
	err := database.RunMigrations(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get status after migrations
	afterStatuses, err := database.GetMigrationStatus(ctx, cfg)
	if err != nil {
		// Migrations ran but status check failed - still success
		fmt.Fprintln(c.App.Writer, "Migrations applied successfully.")
		return nil
	}

	// Count how many were applied
	pendingAfter := 0
	for _, s := range afterStatuses {
		if !s.Applied {
			pendingAfter++
		}
	}

	appliedCount := pendingBefore - pendingAfter

	if appliedCount == 0 {
		fmt.Fprintln(c.App.Writer, "No pending migrations to apply.")
	} else {
		fmt.Fprintf(c.App.Writer, "Applied %d migration(s) successfully.\n", appliedCount)
	}

	return nil
}

func dbStatus(c *cli.Context) error {
	ctx := context.Background()
	cfg := config.LoadStorageConfigFromEnv()

	// Override with CLI flag if provided
	if dbType := c.String("db-type"); dbType != "" {
		cfg.Backend = config.BackendType(dbType)
	}

	statuses, err := database.GetMigrationStatus(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	format := c.String("format")
	return outputMigrationStatus(c, statuses, format)
}

func outputMigrationStatus(c *cli.Context, statuses []database.MigrationStatus, format string) error {
	if format == "json" {
		data, _ := json.MarshalIndent(statuses, "", "  ")
		fmt.Fprintln(c.App.Writer, string(data))
		return nil
	}

	if len(statuses) == 0 {
		fmt.Fprintln(c.App.Writer, "No migrations found.")
		return nil
	}

	w := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "VERSION\tNAME\tSTATUS\tAPPLIED AT")
	for _, s := range statuses {
		status := "pending"
		appliedAt := "-"
		if s.Applied {
			status = "applied"
			appliedAt = s.AppliedAt.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
			s.Version,
			s.Name,
			status,
			appliedAt,
		)
	}
	w.Flush()

	return nil
}
