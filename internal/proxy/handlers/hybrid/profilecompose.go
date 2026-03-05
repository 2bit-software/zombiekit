package hybrid

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	profilev1 "github.com/zombiekit/brains/gen/zombiekit/brains/profile/v1"
	"github.com/zombiekit/brains/gen/zombiekit/brains/profile/v1/profilev1connect"
	profiletool "github.com/zombiekit/brains/internal/mcp/tools/profile"
	"github.com/zombiekit/brains/internal/proxy/handlers"
)

type profileConnection interface {
	IsConfigured() bool
	Profiles() profilev1connect.ProfileServiceClient
}

// NewProfileComposeHandler creates a hybrid handler that composes profiles
// from local filesystem, with fallback to server for missing profiles.
// In local-only mode, behaves identically to the monolithic server.
func NewProfileComposeHandler(conn profileConnection) handlers.Handler {
	localTool := profiletool.NewTool()

	return func(ctx context.Context, args map[string]any) (string, error) {
		result, err := localTool.HandleCompose(ctx, args)
		if err == nil {
			return result, nil
		}

		if conn == nil || !conn.IsConfigured() {
			return "", err
		}

		// Local composition failed -- try server as fallback
		serverResult, serverErr := composeFromServer(ctx, conn, args)
		if serverErr != nil {
			return "", fmt.Errorf("[warning: server fallback also failed: %v]\nlocal error: %w", serverErr, err)
		}

		return serverResult, nil
	}
}

func composeFromServer(ctx context.Context, conn profileConnection, args map[string]any) (string, error) {
	profilesArg, ok := args["profiles"]
	if !ok {
		return "", fmt.Errorf("profiles array is required")
	}

	profilesArray, ok := profilesArg.([]any)
	if !ok {
		return "", fmt.Errorf("profiles must be an array")
	}

	names := make([]string, 0, len(profilesArray))
	for _, p := range profilesArray {
		name, ok := p.(string)
		if !ok {
			continue
		}
		names = append(names, name)
	}

	workDir := ""
	if wd, ok := args["working_directory"].(string); ok {
		workDir = wd
	}

	resp, err := conn.Profiles().ComposeProfile(ctx,
		connect.NewRequest(&profilev1.ComposeProfileRequest{
			ProfileNames:     names,
			WorkingDirectory: workDir,
		}))
	if err != nil {
		return "", fmt.Errorf("server unreachable: %w", err)
	}

	return resp.Msg.GetComposedContent(), nil
}
