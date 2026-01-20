package shutdown

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestManager_GracefulShutdownViaSignal(t *testing.T) {
	m := New(5 * time.Second)

	serviceRan := atomic.Bool{}
	serviceExited := atomic.Bool{}
	sigCh := make(chan os.Signal, 2)

	var runErr error
	done := make(chan struct{})
	go func() {
		runErr = m.runWithSignalChan([]ServiceFunc{
			func(ctx context.Context) error {
				serviceRan.Store(true)
				<-ctx.Done()
				serviceExited.Store(true)
				return nil
			},
		}, sigCh)
		close(done)
	}()

	// Wait for service to start
	time.Sleep(10 * time.Millisecond)

	// Send shutdown signal
	sigCh <- syscall.SIGINT

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for shutdown")
	}

	if runErr != nil {
		t.Errorf("expected no error, got %v", runErr)
	}
	if !serviceRan.Load() {
		t.Error("service never ran")
	}
	if !serviceExited.Load() {
		t.Error("service did not exit cleanly")
	}
}

func TestManager_ServiceError(t *testing.T) {
	m := New(5 * time.Second)

	expectedErr := errors.New("service failed")
	serviceRan := atomic.Bool{}
	otherServiceCancelled := atomic.Bool{}
	sigCh := make(chan os.Signal, 2)

	err := m.runWithSignalChan([]ServiceFunc{
		func(ctx context.Context) error {
			serviceRan.Store(true)
			return expectedErr
		},
		func(ctx context.Context) error {
			// This should get cancelled when the other service errors
			<-ctx.Done()
			otherServiceCancelled.Store(true)
			return nil
		},
	}, sigCh)

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if !serviceRan.Load() {
		t.Error("failing service never ran")
	}
	if !otherServiceCancelled.Load() {
		t.Error("other service was not cancelled on error")
	}
}

func TestManager_ContextPropagationOnSignal(t *testing.T) {
	m := New(5 * time.Second)
	sigCh := make(chan os.Signal, 2)

	contextCancelled := atomic.Int32{}
	ready := make(chan struct{}, 3)

	// All services should receive context cancellation
	svc := func(ctx context.Context) error {
		ready <- struct{}{} // Signal ready
		<-ctx.Done()
		contextCancelled.Add(1)
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- m.runWithSignalChan([]ServiceFunc{svc, svc, svc}, sigCh)
	}()

	// Wait for all services to be ready
	for i := 0; i < 3; i++ {
		<-ready
	}

	// Send shutdown signal
	sigCh <- syscall.SIGTERM

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for shutdown")
	}

	if got := contextCancelled.Load(); got != 3 {
		t.Errorf("expected 3 services to receive cancellation, got %d", got)
	}
}

func TestManager_NoServices(t *testing.T) {
	m := New(100 * time.Millisecond)
	sigCh := make(chan os.Signal, 2)

	// Run with no services should complete immediately
	err := m.runWithSignalChan([]ServiceFunc{}, sigCh)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
