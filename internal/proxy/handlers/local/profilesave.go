package local

import (
	"context"

	profiletool "github.com/zombiekit/brains/internal/mcp/tools/profile"
	"github.com/zombiekit/brains/internal/proxy/handlers"
)

func NewProfileSaveHandler() handlers.Handler {
	tool := profiletool.NewTool()

	return func(ctx context.Context, args map[string]any) (string, error) {
		return tool.HandleSave(ctx, args)
	}
}
