// Package profile provides MCP tools for profile composition and management.
package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/2bit-software/zombiekit/internal/profile"
)

// Tool implements MCP profile tools.
type Tool struct{}

// NewTool creates a new profile Tool.
func NewTool() *Tool {
	return &Tool{}
}

// HandleCompose handles the profile-compose tool call.
func (t *Tool) HandleCompose(ctx context.Context, args map[string]interface{}) (string, error) {
	profilesArg, ok := args["profiles"]
	if !ok {
		return "", fmt.Errorf("profiles array is required")
	}

	profilesArray, ok := profilesArg.([]interface{})
	if !ok || len(profilesArray) == 0 {
		return "", fmt.Errorf("profiles must be a non-empty array of strings")
	}

	var profileNames []string
	for _, p := range profilesArray {
		name, ok := p.(string)
		if !ok {
			return "", fmt.Errorf("profile name must be a string")
		}
		profileNames = append(profileNames, name)
	}

	workingDir := getWorkingDir(args)
	svc, err := profile.NewService(workingDir)
	if err != nil {
		return "", fmt.Errorf("initializing profile service: %w", err)
	}

	result, err := svc.Compose(profileNames)
	if err != nil {
		return "", formatError(err)
	}

	return result.Content, nil
}

// HandleList handles the profile-list tool call.
func (t *Tool) HandleList(ctx context.Context, args map[string]interface{}) (string, error) {
	workingDir := getWorkingDir(args)
	svc, err := profile.NewService(workingDir)
	if err != nil {
		return "", fmt.Errorf("initializing profile service: %w", err)
	}

	entries, err := svc.List()
	if err != nil {
		return "", fmt.Errorf("listing profiles: %w", err)
	}

	if len(entries) == 0 {
		return "No profiles found.", nil
	}

	var sb strings.Builder
	sb.WriteString("Available profiles:\n\n")

	for _, entry := range entries {
		if !entry.Shadowed {
			desc := entry.Description
			if desc == "" {
				desc = "(no description)"
			}
			sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", entry.Name, entry.SourceStr, desc))
		}
	}

	return sb.String(), nil
}

// HandleShow handles the profile-show tool call.
func (t *Tool) HandleShow(ctx context.Context, args map[string]interface{}) (string, error) {
	nameArg, ok := args["name"]
	if !ok {
		return "", fmt.Errorf("name is required")
	}

	name, ok := nameArg.(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name must be a non-empty string")
	}

	raw := false
	if rawArg, ok := args["raw"].(bool); ok {
		raw = rawArg
	}

	workingDir := getWorkingDir(args)
	svc, err := profile.NewService(workingDir)
	if err != nil {
		return "", fmt.Errorf("initializing profile service: %w", err)
	}

	result, err := svc.Show(name, raw)
	if err != nil {
		return "", formatError(err)
	}

	return result.Content, nil
}

// HandleValidate handles the profile-validate tool call.
func (t *Tool) HandleValidate(ctx context.Context, args map[string]interface{}) (string, error) {
	workingDir := getWorkingDir(args)
	svc, err := profile.NewService(workingDir)
	if err != nil {
		return "", fmt.Errorf("initializing profile service: %w", err)
	}

	result, err := svc.Validate()
	if err != nil {
		return "", fmt.Errorf("validating profiles: %w", err)
	}

	if result.Valid {
		return fmt.Sprintf("All %d profiles validated successfully.", result.ProfilesChecked), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Validation failed with %d errors:\n\n", len(result.Errors)))

	for _, verr := range result.Errors {
		switch verr.Code {
		case "MISSING_INCLUDE":
			msg := fmt.Sprintf("- %s: %s", verr.Profile, verr.Message)
			if len(verr.Suggestions) > 0 {
				msg += fmt.Sprintf(" (did you mean %q?)", verr.Suggestions[0])
			}
			sb.WriteString(msg + "\n")
		case "CIRCULAR_DEPENDENCY":
			sb.WriteString(fmt.Sprintf("- %s: %s\n", strings.Join(verr.Cycle, " -> "), verr.Message))
		default:
			sb.WriteString(fmt.Sprintf("- %s: %s\n", verr.Profile, verr.Message))
		}
	}

	return sb.String(), nil
}

// HandleSave handles the profile-save tool call (renamed from profile-write).
func (t *Tool) HandleSave(ctx context.Context, args map[string]interface{}) (string, error) {
	// Extract required parameters
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name is required")
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return "", fmt.Errorf("content is required")
	}

	location, ok := args["location"].(string)
	if !ok || location == "" {
		return "", fmt.Errorf("location is required")
	}

	if location != "local" && location != "global" {
		return "", fmt.Errorf("location must be 'local' or 'global'")
	}

	// Extract optional parameters
	overwrite := false
	if ow, ok := args["overwrite"].(bool); ok {
		overwrite = ow
	}

	workingDir := getWorkingDir(args)
	svc, err := profile.NewService(workingDir)
	if err != nil {
		return "", fmt.Errorf("initializing profile service: %w", err)
	}

	path, err := svc.Write(name, content, location, overwrite)
	if err != nil {
		// Handle ProfileExistsError specially
		if existsErr, ok := err.(*profile.ProfileExistsError); ok {
			resp := SaveResponse{
				Success: false,
				Error:   "PROFILE_EXISTS",
				Message: fmt.Sprintf("Profile '%s' already exists at %s", existsErr.Name, existsErr.Path),
				Hint:    "Use overwrite: true to replace, or choose a different name",
			}
			return marshalResponse(resp)
		}
		return "", fmt.Errorf("writing profile: %w", err)
	}

	resp := SaveResponse{
		Success: true,
		Path:    path,
	}
	return marshalResponse(resp)
}

// marshalResponse marshals a response struct to JSON.
func marshalResponse(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling response: %w", err)
	}
	return string(b), nil
}

// getWorkingDir extracts the working_directory parameter from args.
func getWorkingDir(args map[string]interface{}) string {
	if wd, ok := args["working_directory"].(string); ok && wd != "" {
		return wd
	}
	return ""
}

// formatError formats profile errors with suggestions.
func formatError(err error) error {
	switch e := err.(type) {
	case *profile.ProfileNotFoundError:
		msg := fmt.Sprintf("Profile %q not found", e.Name)
		if len(e.Suggestions) > 0 {
			msg += fmt.Sprintf(". Did you mean %q?", e.Suggestions[0])
		}
		return fmt.Errorf("%s", msg)
	case *profile.CycleError:
		return fmt.Errorf("Circular dependency detected: %s", strings.Join(e.Cycle, " -> "))
	case *profile.NotInitializedError:
		return fmt.Errorf("%s", e.Error())
	default:
		return err
	}
}
