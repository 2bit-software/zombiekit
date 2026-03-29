package orchestrator

import (
	"context"
	"log/slog"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDispatcher() *CommentDispatcher {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return NewCommentDispatcher(logger)
}

func TestCommentDispatcher_RegisterAndNotify(t *testing.T) {
	d := newTestDispatcher()

	ch := d.RegisterSession("TICKET-1", 42)

	want := SessionResult{
		Kind:     SessionResolved,
		TicketID: "TICKET-1",
		PRNumber: 42,
	}
	d.NotifyResult("TICKET-1", want)

	got := <-ch
	assert.Equal(t, want, got)
}

func TestCommentDispatcher_NotifyWithoutRegistration(t *testing.T) {
	d := newTestDispatcher()

	// Should not panic when notifying an unregistered ticket.
	require.NotPanics(t, func() {
		d.NotifyResult("GHOST-99", SessionResult{
			Kind:     SessionFailed,
			TicketID: "GHOST-99",
			PRNumber: 7,
		})
	})
}

func TestCommentDispatcher_CreateAndRemoveQueue(t *testing.T) {
	d := newTestDispatcher()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := d.CreateQueue(10, cancel)
	require.NotNil(t, q)

	got := d.GetQueue(10)
	assert.Equal(t, q, got, "GetQueue should return the queue we just created")

	d.RemoveQueue(10)

	assert.Nil(t, d.GetQueue(10), "GetQueue should return nil after removal")

	// Verify the context was cancelled by RemoveQueue.
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Fatal("expected context to be cancelled after RemoveQueue")
	}
}

func TestCommentDispatcher_ActivePRs(t *testing.T) {
	d := newTestDispatcher()

	_, cancel1 := context.WithCancel(context.Background())
	_, cancel2 := context.WithCancel(context.Background())
	_, cancel3 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()
	defer cancel3()

	d.CreateQueue(1, cancel1)
	d.CreateQueue(5, cancel2)
	d.CreateQueue(42, cancel3)

	prs := d.ActivePRs()
	sort.Ints(prs)

	assert.Equal(t, []int{1, 5, 42}, prs)
}
