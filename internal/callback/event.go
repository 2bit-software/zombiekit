package callback

import "time"

// EventKind identifies the type of callback event.
type EventKind string

const (
	// EventComplete indicates the agent finished work and pushed a branch.
	EventComplete EventKind = "complete"

	// EventCommentResolved indicates the agent addressed a PR review comment.
	EventCommentResolved EventKind = "comment-resolved"

	// EventFailed indicates the agent hit an unrecoverable error.
	EventFailed EventKind = "failed"
)

// Event represents a parsed callback from an agent session.
//
// The Kind field determines which route-specific fields are populated:
//   - EventComplete: Branch is set
//   - EventCommentResolved: CommentID and Resolution are set
//   - EventFailed: Reason is set, CommentID may be set
//
// Unused fields for a given Kind are zero-valued. Event is a value type and
// safe to pass across goroutine boundaries without copying.
type Event struct {
	Kind      EventKind
	TicketID  string
	Timestamp time.Time

	// Branch is the git branch name the agent pushed to (EventComplete only).
	Branch string

	// CommentID is the GitHub comment ID. Set for EventCommentResolved,
	// optionally set for EventFailed when the failure relates to a specific comment.
	CommentID string

	// Resolution describes how a PR comment was addressed (EventCommentResolved only).
	Resolution string

	// Reason describes the failure (EventFailed only).
	Reason string
}
