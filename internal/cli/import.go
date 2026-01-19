// Package cli implements the brains command-line interface.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/zombiekit/brains/internal/memory/importer"
)

// newImportCommand creates the import subcommand for the db command.
func newImportCommand() *cli.Command {
	return &cli.Command{
		Name:  "import",
		Usage: "Import memory data from SQLite to PostgreSQL",
		Description: `Imports memory data from a SQLite database to PostgreSQL.
Supports incremental imports where only items created or updated since
the last import are transferred.

Examples:
  # Basic import
  brains db import --from ~/.brains/memories.db --to "postgres://localhost:5432/brains"

  # Preview import (dry-run)
  brains db import --from ~/.brains/memories.db --dry-run

  # Import with progress
  brains db import --from ~/.brains/memories.db --verbose

  # CI/CD usage (JSON output)
  brains db import --from backup.db --format json`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "from",
				Aliases:  []string{"f"},
				Usage:    "Path to source SQLite database file",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "to",
				Aliases: []string{"t"},
				Usage:   "PostgreSQL connection URL",
				EnvVars: []string{"BRAINS_POSTGRES_URL"},
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "Preview import without making changes",
			},
			&cli.IntFlag{
				Name:  "batch-size",
				Usage: "Items per batch transaction",
				Value: 100,
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Show detailed progress",
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "Output format: text, json",
				Value: "text",
			},
		},
		Action: dbImport,
	}
}

// importOutput represents the JSON output format.
type importOutput struct {
	Source string       `json:"source"`
	Target string       `json:"target"`
	DryRun bool         `json:"dry_run"`
	Result importResult `json:"result"`
}

type importResult struct {
	Imported     int                  `json:"imported"`
	Skipped      int                  `json:"skipped"`
	Errors       int                  `json:"errors"`
	ErrorDetails []importer.ItemError `json:"error_details,omitempty"`
	DurationMS   int64                `json:"duration_ms"`
}

func dbImport(c *cli.Context) error {
	ctx := context.Background()

	// Validate required flags
	sourcePath := c.String("from")
	targetURL := c.String("to")
	dryRun := c.Bool("dry-run")
	batchSize := c.Int("batch-size")
	verbose := c.Bool("verbose")
	format := c.String("format")

	if targetURL == "" && !dryRun {
		return cli.Exit("Error: --to flag or BRAINS_POSTGRES_URL environment variable is required", 1)
	}

	// For dry-run without target, we still need a target to check existing items
	if targetURL == "" && dryRun {
		return cli.Exit("Error: --to flag or BRAINS_POSTGRES_URL environment variable is required even for dry-run", 1)
	}

	// Set up progress callback for verbose mode
	var progressFunc importer.ProgressFunc
	if verbose && format == "text" {
		progressFunc = func(imported, total int, currentItem string) {
			fmt.Fprintf(c.App.Writer, "\rProgress: %d/%d items (%s)     ", imported, total, currentItem)
		}
	}

	// Create importer
	opts := importer.ImportOptions{
		SourcePath: sourcePath,
		TargetURL:  targetURL,
		BatchSize:  batchSize,
		DryRun:     dryRun,
		OnProgress: progressFunc,
	}

	imp, err := importer.New(ctx, opts)
	if err != nil {
		if format == "json" {
			return outputDBImportError(c, sourcePath, targetURL, dryRun, err)
		}
		return cli.Exit(fmt.Sprintf("Error: %v", err), 1)
	}
	defer imp.Close()

	// Print starting message
	if format == "text" {
		if dryRun {
			fmt.Fprintln(c.App.Writer, "Dry run - no changes will be made")
			fmt.Fprintln(c.App.Writer)
		}
		fmt.Fprintf(c.App.Writer, "Importing from %s to PostgreSQL...\n", sourcePath)
	}

	// Run import
	result, err := imp.Import(ctx)
	if err != nil {
		if format == "json" {
			return outputDBImportError(c, sourcePath, targetURL, dryRun, err)
		}
		return cli.Exit(fmt.Sprintf("Error: %v", err), 1)
	}

	// Clear progress line if verbose
	if verbose && format == "text" {
		fmt.Fprintln(c.App.Writer)
	}

	// Output results
	if format == "json" {
		return outputDBImportJSON(c, result)
	}

	return outputDBImportText(c, result, dryRun, verbose)
}

func outputDBImportJSON(c *cli.Context, result *importer.ImportResult) error {
	output := importOutput{
		Source: result.SourcePath,
		Target: result.TargetURL,
		DryRun: result.DryRun,
		Result: importResult{
			Imported:     result.Imported,
			Skipped:      result.Skipped,
			Errors:       result.ErrorCount,
			ErrorDetails: result.ErrorDetails,
			DurationMS:   result.Duration.Milliseconds(),
		},
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return cli.Exit(fmt.Sprintf("Error: failed to marshal JSON: %v", err), 1)
	}

	fmt.Fprintln(c.App.Writer, string(data))

	// Exit code 2 for partial failure
	if result.ErrorCount > 0 {
		return cli.Exit("", 2)
	}
	return nil
}

func outputDBImportText(c *cli.Context, result *importer.ImportResult, dryRun, verbose bool) error {
	if dryRun {
		fmt.Fprintf(c.App.Writer, "Would import from %s:\n", result.SourcePath)
		fmt.Fprintf(c.App.Writer, "  Total items in source: %d\n", result.TotalInSource)
		fmt.Fprintf(c.App.Writer, "  New items to import: %d\n", result.Imported)
		fmt.Fprintf(c.App.Writer, "  Already imported: %d\n", result.Skipped)

		if verbose && len(result.PendingItems) > 0 {
			fmt.Fprintln(c.App.Writer)
			fmt.Fprintln(c.App.Writer, "Items to import:")
			for _, item := range result.PendingItems {
				fmt.Fprintf(c.App.Writer, "  - %s (version %d)\n", item.Name, item.Version)
			}
		}
		return nil
	}

	fmt.Fprintf(c.App.Writer, "Import completed in %s\n", formatDuration(result.Duration))
	fmt.Fprintln(c.App.Writer, "Summary:")
	fmt.Fprintf(c.App.Writer, "  Imported: %d\n", result.Imported)
	fmt.Fprintf(c.App.Writer, "  Skipped: %d\n", result.Skipped)
	fmt.Fprintf(c.App.Writer, "  Errors: %d\n", result.ErrorCount)

	if result.ErrorCount > 0 && verbose {
		fmt.Fprintln(c.App.Writer)
		fmt.Fprintln(c.App.Writer, "Error details:")
		for _, e := range result.ErrorDetails {
			fmt.Fprintf(c.App.Writer, "  - %s v%d: %s\n", e.Name, e.Version, e.Error)
		}
	}

	// Exit code 2 for partial failure
	if result.ErrorCount > 0 {
		return cli.Exit("", 2)
	}
	return nil
}

func outputDBImportError(c *cli.Context, source, target string, dryRun bool, err error) error {
	output := importOutput{
		Source: source,
		Target: maskURL(target),
		DryRun: dryRun,
		Result: importResult{
			Errors: 1,
			ErrorDetails: []importer.ItemError{{
				Name:  "_connection",
				Error: err.Error(),
			}},
		},
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Fprintln(os.Stderr, string(data))
	return cli.Exit("", 1)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func maskURL(url string) string {
	if len(url) > 20 {
		return url[:20] + "..."
	}
	return url
}
