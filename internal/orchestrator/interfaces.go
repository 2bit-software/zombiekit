package orchestrator

import (
	"context"

	"github.com/2bit-software/zombiekit/internal/callback"
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

// Auditor defines the interface for post-session friction auditing.
// Real implementations are provided in Epic 5; this package provides the
// interface and a no-op stub for wiring.
type Auditor interface {
	Audit(ctx context.Context, ticketID string, eventKind callback.EventKind) error
}

// NoopAuditor satisfies Auditor without performing any work.
type NoopAuditor struct{}

// Audit is a no-op that always returns nil.
func (NoopAuditor) Audit(context.Context, string, callback.EventKind) error { return nil }
