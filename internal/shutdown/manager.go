// Package shutdown provides graceful shutdown coordination for multi-service applications.
package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

// Manager coordinates graceful shutdown of multiple services.
type Manager struct {
	timeout time.Duration
}

// New creates a Manager with the specified shutdown timeout.
func New(timeout time.Duration) *Manager {
	return &Manager{timeout: timeout}
}

// ServiceFunc is a function that runs a service until its context is cancelled.
type ServiceFunc func(ctx context.Context) error

// Run starts all services and waits for completion or shutdown signal.
// On SIGINT/SIGTERM, it cancels the context and waits for services to exit.
// If any service returns (success or error), all other services are cancelled.
// A second signal or timeout expiration results in os.Exit(1).
func (m *Manager) Run(services ...ServiceFunc) error {
	return m.runWithSignalChan(services, nil)
}

// runWithSignalChan is the internal implementation that accepts an optional
// signal channel for testing. If sigCh is nil, it creates one with signal.Notify.
func (m *Manager) runWithSignalChan(services []ServiceFunc, sigCh chan os.Signal) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	if sigCh == nil {
		sigCh = make(chan os.Signal, 2)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigCh)
	}

	g, gctx := errgroup.WithContext(ctx)

	// Launch all services
	for _, svc := range services {
		svc := svc // capture for goroutine
		g.Go(func() error { return svc(gctx) })
	}

	// Wait for all services to complete
	done := make(chan error, 1)
	go func() { done <- g.Wait() }()

	// First phase: wait for shutdown trigger (signal or all services complete)
	var shutdownErr error
	select {
	case err := <-done:
		// All services completed on their own
		return err
	case <-sigCh:
		// First signal - trigger graceful shutdown
		cancel()
	}

	// Second phase: wait for services to stop (with timeout and force-exit handling)
	select {
	case shutdownErr = <-done:
		// Services stopped gracefully
	case <-time.After(m.timeout):
		os.Exit(1) // Timeout - force exit
	case <-sigCh:
		os.Exit(1) // Second signal - force exit
	}

	return shutdownErr
}
