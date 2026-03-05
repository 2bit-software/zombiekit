package local

import (
	"context"

	workflowtool "github.com/zombiekit/brains/internal/mcp/tools/workflow"
	"github.com/zombiekit/brains/internal/proxy/handlers"
)

func NewWorkflowHandler() handlers.Handler {
	tool := workflowtool.NewTool()

	return func(ctx context.Context, args map[string]any) (string, error) {
		return tool.HandleCompose(ctx, args)
	}
}
