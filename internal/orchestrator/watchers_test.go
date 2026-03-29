package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWatcherStub_ReturnsNilOnCancel(t *testing.T) {
	setupLogger(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	stub := NewWatcherStub("test-watcher", 30*time.Second)
	err := stub(ctx)
	assert.NoError(t, err)
}

func TestWatcherStub_BlocksUntilCancel(t *testing.T) {
	setupLogger(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stub := NewWatcherStub("test-watcher", 30*time.Second)

	done := make(chan error, 1)
	go func() { done <- stub(ctx) }()

	select {
	case <-done:
		t.Fatal("stub returned before context cancelled")
	case <-time.After(50 * time.Millisecond):
		// still blocking — correct
	}

	cancel()
	assert.NoError(t, <-done)
}
