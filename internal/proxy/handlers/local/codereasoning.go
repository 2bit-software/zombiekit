package local

import (
	"context"

	"github.com/zombiekit/brains/internal/mcp/tools/codereasoning"
	"github.com/zombiekit/brains/internal/proxy/handlers"
)

func NewCodeReasoningHandler() handlers.Handler {
	sm := codereasoning.NewSessionManager()
	tool := codereasoning.NewTool(sm)

	return func(ctx context.Context, args map[string]any) (string, error) {
		return tool.Execute(ctx, "default", args)
	}
}
