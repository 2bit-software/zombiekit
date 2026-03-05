package local

import (
	"context"

	initiativetool "github.com/zombiekit/brains/internal/mcp/tools/initiative"
	"github.com/zombiekit/brains/internal/proxy/handlers"
)

func NewInitiativeHandler() handlers.Handler {
	tool := initiativetool.NewTool()

	return func(ctx context.Context, args map[string]any) (string, error) {
		return tool.Execute(ctx, args)
	}
}
