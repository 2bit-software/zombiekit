// Package main demonstrates robust signal handling for multi-service orchestration.
//
// This spike validates the pattern required for the unified startup command:
// - SIGINT/SIGTERM captured and propagated via context
// - Multiple services coordinate shutdown via errgroup
// - 10 second timeout before force exit
// - Double Ctrl+C forces immediate exit
//
// Run: go run main.go
// Test: Press Ctrl+C once for graceful, twice for force
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	shutdownTimeout = 10 * time.Second
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	log.Println("starting services...")
	log.Println("press Ctrl+C to initiate graceful shutdown")
	log.Println("press Ctrl+C again to force exit")
	log.Println()

	// Create base context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling with buffered channel (capacity 2 for double-signal)
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Create errgroup with context - any error cancels all goroutines
	g, ctx := errgroup.WithContext(ctx)

	// Launch simulated services
	g.Go(func() error {
		return runService(ctx, "gui", 2*time.Second) // Fast shutdown
	})

	g.Go(func() error {
		return runService(ctx, "recall", 3*time.Second) // Medium shutdown
	})

	// Optionally test slow service (uncomment to test timeout)
	// g.Go(func() error {
	// 	return runService(ctx, "slow-service", 15*time.Second) // Exceeds timeout
	// })

	// Signal handler goroutine - part of the errgroup
	g.Go(func() error {
		select {
		case sig := <-sigCh:
			log.Printf("[signal] received %v, initiating graceful shutdown", sig)
			cancel() // Cancel context for all services
			return nil
		case <-ctx.Done():
			// Context canceled by another goroutine (e.g., service error)
			return nil
		}
	})

	// Wait for errgroup with timeout and second-signal handling
	done := make(chan error, 1)
	go func() {
		done <- g.Wait()
	}()

	// Main select: wait for completion, timeout, or force signal
	select {
	case err := <-done:
		if err != nil {
			log.Printf("[main] shutdown completed with error: %v", err)
			os.Exit(1)
		}
		log.Println("[main] shutdown completed successfully")

	case <-time.After(shutdownTimeout):
		log.Printf("[main] shutdown timeout (%v) exceeded, forcing exit", shutdownTimeout)
		os.Exit(1)

	case sig := <-sigCh:
		log.Printf("[main] received second signal (%v), forcing exit", sig)
		os.Exit(1)
	}
}

// runService simulates a service that:
// 1. Runs a periodic task
// 2. Responds to context cancellation
// 3. Takes `shutdownDuration` to clean up
func runService(ctx context.Context, name string, shutdownDuration time.Duration) error {
	log.Printf("[%s] service started", name)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] received shutdown signal, cleaning up (will take %v)...", name, shutdownDuration)

			// Simulate cleanup work
			cleanupDone := make(chan struct{})
			go func() {
				time.Sleep(shutdownDuration)
				close(cleanupDone)
			}()

			// Wait for cleanup (could be interrupted by force exit at main level)
			<-cleanupDone
			log.Printf("[%s] cleanup complete", name)
			return nil

		case <-ticker.C:
			counter++
			log.Printf("[%s] heartbeat %d", name, counter)
		}
	}
}
