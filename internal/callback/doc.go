// Package callback implements the agent callback HTTP server for the orchestrator.
//
// The callback server receives HTTP POST notifications from Claude Code agent
// sessions when they complete work, resolve PR comments, or encounter failures.
// It parses JSON payloads, validates them, and delivers typed [Event] values to
// a consumer via a buffered channel.
//
// # Routes
//
// The server exposes three callback routes and a health check:
//
//	POST /{ticketID}/complete          — agent finished, branch pushed
//	POST /{ticketID}/comment-resolved  — agent addressed a PR review comment
//	POST /{ticketID}/failed            — agent hit an unrecoverable error
//	GET  /healthz                      — liveness check (returns 200 "ok")
//
// # Payload Schemas
//
// All payloads are JSON objects with a status field that must match the route.
//
// Complete:
//
//	{"status": "complete", "ticket_id": "DEV-123", "branch": "DEV-123/feature"}
//
// Comment-resolved:
//
//	{"status": "comment-resolved", "ticket_id": "DEV-123", "comment_id": "IC_abc", "resolution": "Fixed"}
//
// Failed:
//
//	{"status": "failed", "ticket_id": "DEV-123", "reason": "tests failing", "comment_id": "IC_abc"}
//
// The comment_id field is optional on the failed route.
//
// # Event Consumption
//
// Create a server and consume events from the channel:
//
//	srv := callback.New(8666)
//	go func() {
//	    for event := range srv.Events() {
//	        switch event.Kind {
//	        case callback.EventComplete:
//	            fmt.Println("completed:", event.TicketID, event.Branch)
//	        case callback.EventCommentResolved:
//	            fmt.Println("resolved:", event.TicketID, event.CommentID)
//	        case callback.EventFailed:
//	            fmt.Println("failed:", event.TicketID, event.Reason)
//	        }
//	    }
//	}()
//	srv.Run(ctx) // blocks until ctx is cancelled
//
// The Events channel is closed when Run returns.
//
// # Backpressure
//
// The event channel is buffered (64 entries). If the buffer fills because the
// consumer is not draining fast enough, incoming requests receive 503 Service
// Unavailable. The agent can retry.
package callback
