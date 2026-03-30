package orchestrator

import (
	"context"
	"log/slog"
	"sync"

	"github.com/2bit-software/zombiekit/internal/github"
)

// SessionResultKind indicates how a comment-resolution session ended.
type SessionResultKind string

const (
	SessionResolved SessionResultKind = "resolved"
	SessionFailed   SessionResultKind = "failed"
)

// SessionResult is the signal sent from the callback router to a per-PR
// goroutine after a comment-resolution session finishes.
type SessionResult struct {
	Kind     SessionResultKind
	TicketID string
	PRNumber int
}

// prQueue holds the per-PR goroutine state: a buffered channel of comments
// and a cancel function to tear down the goroutine.
type prQueue struct {
	comments chan github.PRComment
	cancel   context.CancelFunc
}

// CommentDispatcher coordinates between the comment watcher's polling loop,
// per-PR processing goroutines, and the callback router. It owns the session
// completion signaling contract.
type CommentDispatcher struct {
	mu       sync.Mutex
	queues   map[int]*prQueue
	sessions map[string]chan SessionResult
	logger   *slog.Logger
}

func NewCommentDispatcher(logger *slog.Logger) *CommentDispatcher {
	return &CommentDispatcher{
		queues:   make(map[int]*prQueue),
		sessions: make(map[string]chan SessionResult),
		logger:   logger,
	}
}

// RegisterSession creates a completion channel for a session. The per-PR
// goroutine calls this before SpawnSession and blocks on the returned channel.
// Channel is buffered(1) so NotifyResult never blocks.
func (d *CommentDispatcher) RegisterSession(ticketID string, prNumber int) <-chan SessionResult {
	d.mu.Lock()
	defer d.mu.Unlock()
	ch := make(chan SessionResult, 1)
	d.sessions[ticketID] = ch
	return ch
}

// NotifyResult signals that a session completed. Called by the Router after
// handling CommentResolvedEvent or FailureEvent. Safe to call for sessions
// not registered (e.g. Watcher 1 failures) — logs at debug level, no-op.
func (d *CommentDispatcher) NotifyResult(ticketID string, result SessionResult) {
	d.mu.Lock()
	ch, ok := d.sessions[ticketID]
	if ok {
		delete(d.sessions, ticketID)
	}
	d.mu.Unlock()

	if !ok {
		d.logger.Debug("notify for unregistered session", slog.String("ticket_id", ticketID))
		return
	}
	ch <- result
}

// CreateQueue creates a per-PR goroutine queue with a buffered comment channel.
func (d *CommentDispatcher) CreateQueue(prNumber int, cancel context.CancelFunc) *prQueue {
	d.mu.Lock()
	defer d.mu.Unlock()
	q := &prQueue{
		comments: make(chan github.PRComment, 100),
		cancel:   cancel,
	}
	d.queues[prNumber] = q
	return q
}

// GetQueue returns the queue for a PR, or nil if none exists.
func (d *CommentDispatcher) GetQueue(prNumber int) *prQueue {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.queues[prNumber]
}

// RemoveQueue cancels the per-PR context and removes the queue entry.
func (d *CommentDispatcher) RemoveQueue(prNumber int) {
	d.mu.Lock()
	q, ok := d.queues[prNumber]
	if ok {
		delete(d.queues, prNumber)
	}
	d.mu.Unlock()

	if ok && q.cancel != nil {
		q.cancel()
	}
}

// ActivePRs returns the PR numbers with active queues.
func (d *CommentDispatcher) ActivePRs() []int {
	d.mu.Lock()
	defer d.mu.Unlock()
	prs := make([]int, 0, len(d.queues))
	for pr := range d.queues {
		prs = append(prs, pr)
	}
	return prs
}
