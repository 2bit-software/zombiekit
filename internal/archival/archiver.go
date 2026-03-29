package archival

import (
	"context"

	"github.com/zombiekit/brains/internal/callback"
)

// Archiver defines the interface for post-session conversation archival.
// Real implementations are provided in Epic 5; this package provides the
// interface and a no-op stub for wiring.
type Archiver interface {
	Archive(ctx context.Context, ticketID string, eventKind callback.EventKind) error
}

// NoopArchiver satisfies Archiver without performing any work.
type NoopArchiver struct{}

// Archive is a no-op that always returns nil.
func (NoopArchiver) Archive(context.Context, string, callback.EventKind) error { return nil }
