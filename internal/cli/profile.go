package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
	"github.com/2bit-software/zombiekit/internal/profile"
)

func newProfileCommand() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "Manage AI assistant profiles",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "source",
				Aliases: []string{"s"},
				Value:   "brains",
				Usage:   "Profile source: brains (default) or claude",
			},
		},
		Subcommands: []*cli.Command{
			newProfileComposeCommand(),
			newProfileListCommand(),
			newProfileShowCommand(),
			newProfileCreateCommand(),
			newProfileValidateCommand(),
			newProfileImportCommand(),
		},
	}
}

// getSourceType extracts the source type from the CLI context.
// It checks both the current context and parent context for the --source flag.
func getSourceType(c *cli.Context) (profile.SourceType, error) {
	// Check parent context for the --source flag (defined on profile command group)
	sourceStr := c.String("source")
	if sourceStr == "" {
		// Try parent context
		if parent := c.Lineage(); len(parent) > 1 {
			sourceStr = parent[1].String("source")
		}
	}
	return profile.ParseSourceType(sourceStr)
}

func newProfileComposeCommand() *cli.Command {
	return &cli.Command{
		Name:      "compose",
		Usage:     "Compose one or more profiles into merged content",
		ArgsUsage: "<profiles...>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "format",
				Value: "text",
				Usage: "Output format: text or json",
			},
		},
		Action: func(c *cli.Context) error {
			args := c.Args().Slice()
			if len(args) == 0 {
				return fmt.Errorf("at least one profile name is required")
			}

			// Parse profile names (support comma-separated and space-separated)
			profileNames := parseProfileNames(args)
			if len(profileNames) == 0 {
				return fmt.Errorf("at least one profile name is required")
			}

			sourceType, err := getSourceType(c)
			if err != nil {
				return err
			}

			svc, err := profile.NewServiceWithSource(sourceType, "")
			if err != nil {
				return fmt.Errorf("initializing profile service: %w", err)
			}

			result, err := svc.Compose(profileNames)
			if err != nil {
				return handleProfileError(c, err)
			}

			if c.String("format") == "json" {
				return outputComposeJSON(result)
			}

			// Text output: just the content
			fmt.Println(result.Content)
			return nil
		},
	}
}

func newProfileListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all available profiles",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "format",
				Value: "text",
				Usage: "Output format: text or json",
			},
		},
		Action: func(c *cli.Context) error {
			sourceType, err := getSourceType(c)
			if err != nil {
				return err
			}

			svc, err := profile.NewServiceWithSource(sourceType, "")
			if err != nil {
				return fmt.Errorf("initializing profile service: %w", err)
			}

			entries, err := svc.List()
			if err != nil {
				return fmt.Errorf("listing profiles: %w", err)
			}

			if c.String("format") == "json" {
				return outputListJSON(entries)
			}

			return outputListText(entries)
		},
	}
}

func newProfileShowCommand() *cli.Command {
	return &cli.Command{
		Name:      "show",
		Usage:     "Show a specific profile's content",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "raw",
				Usage: "Show raw file content without inheritance resolution",
			},
			&cli.StringFlag{
				Name:  "format",
				Value: "text",
				Usage: "Output format: text or json",
			},
		},
		Action: func(c *cli.Context) error {
			name := c.Args().First()
			if name == "" {
				return fmt.Errorf("profile name is required")
			}

			sourceType, err := getSourceType(c)
			if err != nil {
				return err
			}

			svc, err := profile.NewServiceWithSource(sourceType, "")
			if err != nil {
				return fmt.Errorf("initializing profile service: %w", err)
			}

			result, err := svc.Show(name, c.Bool("raw"))
			if err != nil {
				return handleProfileError(c, err)
			}

			if c.String("format") == "json" {
				return outputShowJSON(result)
			}

			// Text output: just the content
			fmt.Println(result.Content)
			return nil
		},
	}
}

func newProfileCreateCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new profile",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "global",
				Usage: "Create in global directory instead of local",
			},
		},
		Action: func(c *cli.Context) error {
			name := c.Args().First()
			if name == "" {
				return fmt.Errorf("profile name is required")
			}

			sourceType, err := getSourceType(c)
			if err != nil {
				return err
			}

			svc, err := profile.NewServiceWithSource(sourceType, "")
			if err != nil {
				return fmt.Errorf("initializing profile service: %w", err)
			}

			path, err := svc.Create(name, c.Bool("global"))
			if err != nil {
				return handleProfileError(c, err)
			}

			fmt.Printf("Created profile: %s\n", path)
			return nil
		},
	}
}

func newProfileValidateCommand() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate all profiles for errors",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "format",
				Value: "text",
				Usage: "Output format: text or json",
			},
		},
		Action: func(c *cli.Context) error {
			sourceType, err := getSourceType(c)
			if err != nil {
				return err
			}

			svc, err := profile.NewServiceWithSource(sourceType, "")
			if err != nil {
				return fmt.Errorf("initializing profile service: %w", err)
			}

			result, err := svc.Validate()
			if err != nil {
				return fmt.Errorf("validating profiles: %w", err)
			}

			if c.String("format") == "json" {
				return outputValidateJSON(result)
			}

			return outputValidateText(result)
		},
	}
}

// parseProfileNames parses profile names from args, handling comma-separated values.
func parseProfileNames(args []string) []string {
	var names []string
	seen := make(map[string]bool)

	for _, arg := range args {
		// Split by comma
		parts := strings.Split(arg, ",")
		for _, part := range parts {
			name := strings.TrimSpace(part)
			if name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}

	return names
}

// handleProfileError handles profile-specific errors with appropriate exit codes.
func handleProfileError(c *cli.Context, err error) error {
	if c.String("format") == "json" {
		return outputErrorJSON(err)
	}

	switch e := err.(type) {
	case *profile.ProfileNotFoundError:
		msg := fmt.Sprintf("Profile %q not found", e.Name)
		if len(e.Suggestions) > 0 {
			msg += fmt.Sprintf(" (did you mean: %s?)", strings.Join(e.Suggestions, ", "))
		}
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	case *profile.CycleError:
		fmt.Fprintf(os.Stderr, "Circular dependency detected: %s\n", strings.Join(e.Cycle, " -> "))
		os.Exit(2)
	case *profile.NotInitializedError:
		fmt.Fprintln(os.Stderr, e.Error())
		os.Exit(2)
	case *profile.ProfileExistsError:
		fmt.Fprintln(os.Stderr, e.Error())
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

// JSON output helpers

func outputComposeJSON(result *profile.CompositionResult) error {
	output := map[string]interface{}{
		"content":          result.Content,
		"profiles_used":    result.ProfilesUsed,
		"character_count":  result.CharacterCount,
		"estimated_tokens": result.EstimatedTokens,
		"warnings":         result.Warnings,
		"resolution":       formatResolutionLog(result.ResolutionLog),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func formatResolutionLog(log []profile.ResolutionEntry) []map[string]interface{} {
	var result []map[string]interface{}
	for _, entry := range log {
		result = append(result, map[string]interface{}{
			"name":   entry.Name,
			"source": entry.Source.String(),
			"path":   entry.Path,
		})
	}
	return result
}

func outputListJSON(entries []profile.ListEntry) error {
	output := map[string]interface{}{
		"profiles": entries,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputListText(entries []profile.ListEntry) error {
	if len(entries) == 0 {
		fmt.Println("No profiles found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROFILE\tSOURCE\tDESCRIPTION")

	for _, entry := range entries {
		if !entry.Shadowed {
			desc := entry.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", entry.Name, entry.SourceStr, desc)
		}
	}

	return w.Flush()
}

func outputShowJSON(result *profile.ShowResult) error {
	output := map[string]interface{}{
		"name":        result.Name,
		"source":      result.SourceStr,
		"path":        result.Path,
		"description": result.Description,
		"includes":    result.Includes,
		"inherits":    result.Inherits,
		"content":     result.Content,
		"raw_content": result.RawContent,
	}

	// Add optional fields only if they have values
	if result.Model != "" {
		output["model"] = result.Model
	}
	if result.Color != "" {
		output["color"] = result.Color
	}
	if result.Type != "" {
		output["type"] = result.Type
	}

	if len(result.InheritedFrom) > 0 {
		var inherited []map[string]string
		for _, inf := range result.InheritedFrom {
			inherited = append(inherited, map[string]string{
				"source": inf.Source.String(),
				"path":   inf.Path,
			})
		}
		output["inherited_from"] = inherited
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputValidateJSON(result *profile.ValidationResult) error {
	output := map[string]interface{}{
		"valid":            result.Valid,
		"profiles_checked": result.ProfilesChecked,
		"errors":           formatValidationErrors(result.Errors),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func formatValidationErrors(errors []profile.ValidationError) []map[string]interface{} {
	var result []map[string]interface{}
	for _, err := range errors {
		item := map[string]interface{}{
			"profile": err.Profile,
			"code":    err.Code,
			"message": err.Message,
		}
		if len(err.Suggestions) > 0 {
			item["suggestions"] = err.Suggestions
		}
		if len(err.Cycle) > 0 {
			item["cycle"] = err.Cycle
		}
		result = append(result, item)
	}
	return result
}

func outputValidateText(result *profile.ValidationResult) error {
	if result.Valid {
		fmt.Printf("✓ All %d profiles validated successfully\n", result.ProfilesChecked)
		return nil
	}

	fmt.Printf("✗ Validation failed with %d errors:\n\n", len(result.Errors))

	for _, err := range result.Errors {
		switch err.Code {
		case "MISSING_INCLUDE":
			msg := fmt.Sprintf("  %s: %s", err.Profile, err.Message)
			if len(err.Suggestions) > 0 {
				msg += fmt.Sprintf(" (did you mean %q?)", err.Suggestions[0])
			}
			fmt.Println(msg)
		case "CIRCULAR_DEPENDENCY":
			fmt.Printf("  %s: %s\n", strings.Join(err.Cycle, " -> "), err.Message)
		default:
			fmt.Printf("  %s: %s\n", err.Profile, err.Message)
		}
	}

	os.Exit(1)
	return nil
}

func outputErrorJSON(err error) error {
	output := map[string]interface{}{
		"error": map[string]interface{}{
			"message": err.Error(),
		},
	}

	switch e := err.(type) {
	case *profile.ProfileNotFoundError:
		output["error"].(map[string]interface{})["code"] = "PROFILE_NOT_FOUND"
		if len(e.Suggestions) > 0 {
			output["error"].(map[string]interface{})["suggestions"] = e.Suggestions
		}
	case *profile.CycleError:
		output["error"].(map[string]interface{})["code"] = "CIRCULAR_DEPENDENCY"
		output["error"].(map[string]interface{})["cycle"] = e.Cycle
	case *profile.NotInitializedError:
		output["error"].(map[string]interface{})["code"] = "NOT_INITIALIZED"
	case *profile.ProfileExistsError:
		output["error"].(map[string]interface{})["code"] = "PROFILE_EXISTS"
	default:
		output["error"].(map[string]interface{})["code"] = "UNKNOWN_ERROR"
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return err
	}

	os.Exit(1)
	return nil
}

func newProfileImportCommand() *cli.Command {
	return &cli.Command{
		Name:      "import",
		Usage:     "Import profiles from an external source",
		ArgsUsage: "<source>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show what would be imported without making changes",
			},
			&cli.StringFlag{
				Name:  "format",
				Value: "text",
				Usage: "Output format: text or json",
			},
		},
		Action: func(c *cli.Context) error {
			sourceArg := c.Args().First()
			if sourceArg == "" {
				return fmt.Errorf("source is required (e.g., 'claude')")
			}

			// Validate source type
			if sourceArg != "claude" {
				return fmt.Errorf("unsupported source %q (supported: claude)", sourceArg)
			}

			// Create importer and run import
			importer, err := profile.NewImporter("")
			if err != nil {
				return fmt.Errorf("initializing importer: %w", err)
			}

			result, err := importer.Import(c.Bool("dry-run"))
			if err != nil {
				return fmt.Errorf("importing profiles: %w", err)
			}

			if c.String("format") == "json" {
				return outputImportJSON(result)
			}

			return outputImportText(result)
		},
	}
}

func outputImportText(result *profile.ImportResult) error {
	total := result.Created + result.Overwritten

	if total == 0 && result.Failed == 0 {
		if result.DryRun {
			fmt.Println("Dry run: No profiles to import")
		} else {
			fmt.Println("No profiles to import")
		}
		return nil
	}

	// Header
	if result.DryRun {
		fmt.Printf("Dry run: Would import %d profiles from claude source:\n", total)
	} else {
		fmt.Printf("Imported %d profiles from claude source:\n", total)
	}

	// Summary counts
	fmt.Printf("  Created:     %d profiles\n", result.Created)
	fmt.Printf("  Overwritten: %d profiles\n", result.Overwritten)
	if result.Failed > 0 {
		fmt.Printf("  Failed:      %d agents\n", result.Failed)
	}

	// Created paths
	if len(result.CreatedPaths) > 0 {
		fmt.Println("\nCreated:")
		for _, path := range result.CreatedPaths {
			fmt.Printf("  %s\n", path)
		}
	}

	// Overwritten paths
	if len(result.OverwrittenPaths) > 0 {
		fmt.Println("\nOverwritten:")
		for _, path := range result.OverwrittenPaths {
			fmt.Printf("  %s\n", path)
		}
	}

	// Failed agents
	if len(result.FailedAgents) > 0 {
		fmt.Println("\nFailed:")
		for _, fail := range result.FailedAgents {
			fmt.Printf("  %s: %s\n", fail.AgentName, fail.Error)
		}
	}

	return nil
}

func outputImportJSON(result *profile.ImportResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
