package handlers

import "context"

// Handler is the common signature for all tool handlers.
type Handler func(ctx context.Context, args map[string]any) (string, error)
