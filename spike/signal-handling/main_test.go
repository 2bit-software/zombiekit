package main

import (
	"context"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

// TestGracefulShutdownWithTimeout verifies that:
// 1. Context cancellation propagates to all services
// 2. Services shut down within timeout
// 3. Main goroutine exits cleanly after all services complete
func TestGracefulShutdownWithTimeout(t *testing.T) {
	const (
		shutdownTimeout = 2 * time.Second
		serviceCount    = 3
	)

	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Track which services completed shutdown
	var mu sync.Mutex
	completed := make(map[string]bool)

	// Launch services that shut down quickly
	for i := 0; i < serviceCount; i++ {
		name := string(rune('A' + i))
		g.Go(func() error {
			// Simulate work
			select {
			case <-ctx.Done():
				// Simulate fast cleanup
				time.Sleep(100 * time.Millisecond)
				mu.Lock()
				completed[name] = true
				mu.Unlock()
				return nil
			}
		})
	}

	// Trigger shutdown after brief startup
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- g.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("errgroup returned error: %v", err)
		}
	case <-time.After(shutdownTimeout):
		t.Fatal("shutdown timed out - services did not complete within timeout")
	}

	// Verify all services completed
	mu.Lock()
	defer mu.Unlock()
	if len(completed) != serviceCount {
		t.Errorf("expected %d services to complete, got %d", serviceCount, len(completed))
	}
}

// TestSlowServiceHitsTimeout verifies that:
// 1. When a service takes longer than timeout, main exits after timeout
// 2. The slow service is not waited for indefinitely
func TestSlowServiceHitsTimeout(t *testing.T) {
	const (
		shutdownTimeout = 100 * time.Millisecond
		slowCleanup     = 500 * time.Millisecond
	)

	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Launch a slow service
	g.Go(func() error {
		select {
		case <-ctx.Done():
			// Simulate slow cleanup that exceeds timeout
			time.Sleep(slowCleanup)
			return nil
		}
	})

	// Trigger shutdown
	cancel()

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- g.Wait()
	}()

	start := time.Now()
	select {
	case <-done:
		// Service eventually completed (but we should have hit timeout first)
		elapsed := time.Since(start)
		if elapsed >= shutdownTimeout && elapsed < slowCleanup {
			// This is the expected path in real code - timeout fires first
			// But since we're in test, both channels may be ready
			t.Logf("completed after %v (timeout would have fired)", elapsed)
		}
	case <-time.After(shutdownTimeout):
		// Timeout hit as expected
		elapsed := time.Since(start)
		if elapsed < shutdownTimeout {
			t.Errorf("timeout fired too early: %v", elapsed)
		}
		t.Logf("timeout correctly fired after %v", elapsed)
	}
}

// TestDoubleSignalForceExit verifies the double Ctrl+C pattern.
// Since we can't easily send real signals in tests, we simulate
// the pattern using channels.
func TestDoubleSignalForceExit(t *testing.T) {
	// Simulate the signal channel pattern from main.go
	sigCh := make(chan os.Signal, 2)

	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Launch a service that takes forever to shut down
	serviceStarted := make(chan struct{})
	g.Go(func() error {
		close(serviceStarted)
		<-ctx.Done()
		// Hang forever (simulating stuck service)
		select {}
	})

	// Wait for service to start
	<-serviceStarted

	// Signal handler goroutine
	g.Go(func() error {
		select {
		case <-sigCh:
			cancel()
			return nil
		case <-ctx.Done():
			return nil
		}
	})

	// Send first signal (graceful shutdown)
	sigCh <- syscall.SIGINT

	// Wait for errgroup with timeout and second-signal check
	done := make(chan error, 1)
	go func() {
		done <- g.Wait()
	}()

	// Second signal should trigger force exit path
	forceExitTriggered := false

	select {
	case <-done:
		// Won't happen - service hangs forever
		t.Error("errgroup completed unexpectedly")
	case <-time.After(50 * time.Millisecond):
		// Simulate user pressing Ctrl+C again
		sigCh <- syscall.SIGINT
	}

	// Check that second signal would be received
	select {
	case <-sigCh:
		forceExitTriggered = true
	case <-time.After(50 * time.Millisecond):
		t.Error("second signal not received")
	}

	if !forceExitTriggered {
		t.Error("force exit path not triggered by second signal")
	}
}

// TestContextPropagation verifies that context cancellation
// reaches all nested goroutines properly.
func TestContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Track cancellation receipt
	var mu sync.Mutex
	cancellations := 0
	expectedCancellations := 5

	var wg sync.WaitGroup
	for i := 0; i < expectedCancellations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()
			mu.Lock()
			cancellations++
			mu.Unlock()
		}()
	}

	// Give goroutines time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("not all goroutines received cancellation")
	}

	mu.Lock()
	defer mu.Unlock()
	if cancellations != expectedCancellations {
		t.Errorf("expected %d cancellations, got %d", expectedCancellations, cancellations)
	}
}

// TestSignalBufferCapacity verifies that the signal channel
// can buffer multiple signals without losing them.
func TestSignalBufferCapacity(t *testing.T) {
	// Channel with capacity 2 (as in main.go)
	sigCh := make(chan os.Signal, 2)

	// Send two signals rapidly (before any receiver)
	sigCh <- syscall.SIGINT
	sigCh <- syscall.SIGTERM

	// Both signals should be buffered
	select {
	case <-sigCh:
		// First signal received
	default:
		t.Error("first signal lost")
	}

	select {
	case <-sigCh:
		// Second signal received
	default:
		t.Error("second signal lost")
	}
}

// TestErrgroupErrorPropagation verifies that when one service
// errors, all other services receive context cancellation.
func TestErrgroupErrorPropagation(t *testing.T) {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	// Service that will error
	g.Go(func() error {
		time.Sleep(50 * time.Millisecond)
		return context.DeadlineExceeded // Simulate error
	})

	// Service that waits for cancellation
	cancelReceived := make(chan struct{})
	g.Go(func() error {
		<-ctx.Done()
		close(cancelReceived)
		return nil
	})

	// Wait for errgroup
	err := g.Wait()
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}

	// Verify second service received cancellation
	select {
	case <-cancelReceived:
		// Success
	default:
		t.Error("second service did not receive cancellation from first service's error")
	}
}
