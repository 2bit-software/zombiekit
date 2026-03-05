package hybrid

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	profilev1 "github.com/zombiekit/brains/gen/zombiekit/brains/profile/v1"
	profiletool "github.com/zombiekit/brains/internal/mcp/tools/profile"
	"github.com/zombiekit/brains/internal/proxy/handlers"
)

type profileEntry struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description,omitempty"`
}

// NewProfileListHandler creates a hybrid handler that lists profiles from
// both local filesystem and remote server. Local profiles override remote
// profiles with the same name.
func NewProfileListHandler(conn profileConnection) handlers.Handler {
	localTool := profiletool.NewTool()

	return func(ctx context.Context, args map[string]any) (string, error) {
		localResult, localErr := localTool.HandleList(ctx, args)

		if conn == nil || !conn.IsConfigured() {
			return localResult, localErr
		}

		workDir := ""
		if wd, ok := args["working_directory"].(string); ok {
			workDir = wd
		}

		resp, err := conn.Profiles().ListProfiles(ctx,
			connect.NewRequest(&profilev1.ListProfilesRequest{
				WorkingDirectory: workDir,
			}))
		if err != nil {
			if localErr != nil {
				return "", localErr
			}
			return "[warning: server unreachable, showing local profiles only]\n" + localResult, nil
		}

		return mergeProfileLists(localResult, resp.Msg.GetProfiles()), nil
	}
}

func mergeProfileLists(localOutput string, remoteProfiles []*profilev1.Profile) string {
	// Build set of local profile names from the local output
	localNames := parseLocalNames(localOutput)

	var entries []profileEntry
	// Add remote profiles not shadowed by local
	for _, rp := range remoteProfiles {
		if _, shadowed := localNames[rp.GetName()]; !shadowed {
			entries = append(entries, profileEntry{
				Name:   rp.GetName(),
				Source: "remote",
			})
		}
	}

	if len(entries) == 0 {
		return localOutput
	}

	// Append remote-only profiles to local output
	var sb strings.Builder
	sb.WriteString(localOutput)
	sb.WriteString("\nRemote-only profiles:\n\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("- %s (remote)\n", e.Name))
	}

	return sb.String()
}

func parseLocalNames(output string) map[string]struct{} {
	names := make(map[string]struct{})
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			// Format: "- name (source): description"
			rest := strings.TrimPrefix(line, "- ")
			if idx := strings.Index(rest, " ("); idx > 0 {
				names[rest[:idx]] = struct{}{}
			}
		}
	}
	return names
}

