package callback

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestEventDemuxer_RoutesToCorrectProject(t *testing.T) {
	d := NewEventDemuxer(testLogger())
	chA := d.Register("alpha")
	chB := d.Register("beta")

	src := make(chan Event, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = d.Run(ctx, src) }()

	src <- Event{Kind: EventComplete, ProjectID: "alpha", TicketID: "T-1"}
	src <- Event{Kind: EventFailed, ProjectID: "beta", TicketID: "T-2"}

	select {
	case ev := <-chA:
		assert.Equal(t, "alpha", ev.ProjectID)
		assert.Equal(t, "T-1", ev.TicketID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for alpha event")
	}

	select {
	case ev := <-chB:
		assert.Equal(t, "beta", ev.ProjectID)
		assert.Equal(t, "T-2", ev.TicketID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for beta event")
	}
}

func TestEventDemuxer_UnknownProjectDropsEvent(t *testing.T) {
	d := NewEventDemuxer(testLogger())
	chA := d.Register("alpha")

	src := make(chan Event, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = d.Run(ctx, src) }()

	src <- Event{Kind: EventComplete, ProjectID: "unknown", TicketID: "T-1"}

	// Give demuxer time to process, then verify alpha got nothing.
	time.Sleep(50 * time.Millisecond)

	select {
	case <-chA:
		t.Fatal("alpha should not receive events for unknown project")
	default:
	}
}

func TestEventDemuxer_FullChannelDropsEvent(t *testing.T) {
	d := NewEventDemuxer(testLogger())
	ch := d.Register("proj")

	// Fill the channel.
	for i := 0; i < demuxerBufferSize; i++ {
		d.route(Event{Kind: EventComplete, ProjectID: "proj", TicketID: "fill"})
	}

	// Next event should be dropped (not block).
	d.route(Event{Kind: EventComplete, ProjectID: "proj", TicketID: "overflow"})

	// Drain and verify we got exactly demuxerBufferSize events.
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			assert.Equal(t, demuxerBufferSize, count)
			return
		}
	}
}

func TestEventDemuxer_ShutdownClosesChannels(t *testing.T) {
	d := NewEventDemuxer(testLogger())
	chA := d.Register("alpha")
	chB := d.Register("beta")

	src := make(chan Event)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- d.Run(ctx, src) }()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancel")
	}

	// Channels should be closed.
	_, okA := <-chA
	_, okB := <-chB
	assert.False(t, okA, "alpha channel should be closed")
	assert.False(t, okB, "beta channel should be closed")
}

func TestEventDemuxer_SourceCloseStopsRun(t *testing.T) {
	d := NewEventDemuxer(testLogger())
	d.Register("alpha")

	src := make(chan Event)
	close(src)

	err := d.Run(context.Background(), src)
	require.NoError(t, err)
}

func TestEventDemuxer_Deregister(t *testing.T) {
	d := NewEventDemuxer(testLogger())
	ch := d.Register("proj")
	d.Deregister("proj")

	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after deregister")

	// Routing to deregistered project should not panic.
	d.route(Event{Kind: EventComplete, ProjectID: "proj", TicketID: "T-1"})
}
