package proxy

import (
	"context"
	"fmt"

	"github.com/zombiekit/brains/internal/proxy/handlers"
)

type Router struct {
	handlers map[string]handlers.Handler
}

func NewRouter() *Router {
	return &Router{handlers: make(map[string]handlers.Handler)}
}

func (r *Router) Register(toolName string, handler handlers.Handler) {
	if _, exists := r.handlers[toolName]; exists {
		panic(fmt.Sprintf("duplicate handler registration for tool %q", toolName))
	}
	r.handlers[toolName] = handler
}

func (r *Router) Dispatch(ctx context.Context, toolName string, args map[string]any) (string, error) {
	handler, ok := r.handlers[toolName]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
	return handler(ctx, args)
}
