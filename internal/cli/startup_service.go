// Package startup provides service wrappers for the brains start command.
package cli

import (
	"context"
	"log/slog"

	"github.com/2bit-software/zombiekit/internal/logging"
)

// Service represents a long-running service that can be started and stopped.
type Service interface {
	// Name returns the service identifier used for logging.
	Name() string

	// Run starts the service and blocks until the context is cancelled
	// or an error occurs. It should return nil on graceful shutdown.
	Run(ctx context.Context) error
}

// ServiceLogger returns a logger prefixed with the service name.
func ServiceLogger(serviceName string) *slog.Logger {
	return logging.Logger().WithGroup(serviceName)
}
