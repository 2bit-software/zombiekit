package callback

import (
	"context"
	"log/slog"
	"sync"
)

const demuxerBufferSize = 64

// EventDemuxer routes incoming callback events to per-project channels
// based on the ProjectID field. Projects register before the demuxer starts;
// events for unknown projects are dropped with a warning.
type EventDemuxer struct {
	mu       sync.Mutex
	channels map[string]chan Event
	logger   *slog.Logger
}

func NewEventDemuxer(logger *slog.Logger) *EventDemuxer {
	return &EventDemuxer{
		channels: make(map[string]chan Event),
		logger:   logger,
	}
}

// Register creates a buffered event channel for a project.
// Must be called before Run. Returns the read side of the channel.
func (d *EventDemuxer) Register(projectID string) <-chan Event {
	d.mu.Lock()
	defer d.mu.Unlock()
	ch := make(chan Event, demuxerBufferSize)
	d.channels[projectID] = ch
	return ch
}

// Deregister removes a project's channel and closes it.
func (d *EventDemuxer) Deregister(projectID string) {
	d.mu.Lock()
	ch, ok := d.channels[projectID]
	if ok {
		delete(d.channels, projectID)
	}
	d.mu.Unlock()

	if ok {
		close(ch)
	}
}

// Run reads events from the source channel and routes each to the
// registered project channel. Blocks until ctx is cancelled or the
// source channel is closed. On shutdown, closes all project channels.
func (d *EventDemuxer) Run(ctx context.Context, events <-chan Event) error {
	defer d.closeAll()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			d.route(ev)
		}
	}
}

func (d *EventDemuxer) route(ev Event) {
	d.mu.Lock()
	ch, ok := d.channels[ev.ProjectID]
	d.mu.Unlock()

	if !ok {
		d.logger.Warn("event for unknown project, dropping",
			slog.String("project_id", ev.ProjectID),
			slog.String("ticket_id", ev.TicketID),
			slog.String("kind", string(ev.Kind)),
		)
		return
	}

	select {
	case ch <- ev:
	default:
		d.logger.Warn("project event channel full, dropping",
			slog.String("project_id", ev.ProjectID),
			slog.String("ticket_id", ev.TicketID),
			slog.String("kind", string(ev.Kind)),
		)
	}
}

func (d *EventDemuxer) closeAll() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for id, ch := range d.channels {
		close(ch)
		delete(d.channels, id)
	}
}
