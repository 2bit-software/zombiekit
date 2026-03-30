package callback

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/2bit-software/zombiekit/internal/logging"
)

type completePayload struct {
	Status   string `json:"status"`
	TicketID string `json:"ticket_id"`
	Branch   string `json:"branch"`
}

type commentResolvedPayload struct {
	Status     string `json:"status"`
	TicketID   string `json:"ticket_id"`
	CommentID  string `json:"comment_id"`
	Resolution string `json:"resolution"`
}

type failedPayload struct {
	Status    string `json:"status"`
	TicketID  string `json:"ticket_id"`
	Reason    string `json:"reason"`
	CommentID string `json:"comment_id,omitempty"`
}

func (s *CallbackServer) handleComplete(w http.ResponseWriter, r *http.Request) {
	ticketID := r.PathValue("ticketID")

	payload, err := decodeJSON[completePayload](r, maxBodyBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if payload.Status != "complete" {
		writeError(w, http.StatusBadRequest, "status field must be 'complete' for this route")
		return
	}
	if payload.TicketID == "" {
		writeError(w, http.StatusBadRequest, "missing required field: ticket_id")
		return
	}
	if payload.Branch == "" {
		writeError(w, http.StatusBadRequest, "missing required field: branch")
		return
	}

	if payload.TicketID != ticketID {
		logging.Logger().Warn("ticket ID mismatch between URL and body",
			slog.String("url_ticket_id", ticketID),
			slog.String("body_ticket_id", payload.TicketID),
		)
	}

	event := Event{
		Kind:      EventComplete,
		TicketID:  ticketID,
		Timestamp: time.Now(),
		Branch:    payload.Branch,
	}

	s.sendEvent(w, event)
}

func (s *CallbackServer) handleCommentResolved(w http.ResponseWriter, r *http.Request) {
	ticketID := r.PathValue("ticketID")

	payload, err := decodeJSON[commentResolvedPayload](r, maxBodyBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if payload.Status != "comment-resolved" {
		writeError(w, http.StatusBadRequest, "status field must be 'comment-resolved' for this route")
		return
	}
	if payload.TicketID == "" {
		writeError(w, http.StatusBadRequest, "missing required field: ticket_id")
		return
	}
	if payload.CommentID == "" {
		writeError(w, http.StatusBadRequest, "missing required field: comment_id")
		return
	}
	if payload.Resolution == "" {
		writeError(w, http.StatusBadRequest, "missing required field: resolution")
		return
	}

	if payload.TicketID != ticketID {
		logging.Logger().Warn("ticket ID mismatch between URL and body",
			slog.String("url_ticket_id", ticketID),
			slog.String("body_ticket_id", payload.TicketID),
		)
	}

	event := Event{
		Kind:       EventCommentResolved,
		TicketID:   ticketID,
		Timestamp:  time.Now(),
		CommentID:  payload.CommentID,
		Resolution: payload.Resolution,
	}

	s.sendEvent(w, event)
}

func (s *CallbackServer) handleFailed(w http.ResponseWriter, r *http.Request) {
	ticketID := r.PathValue("ticketID")

	payload, err := decodeJSON[failedPayload](r, maxBodyBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if payload.Status != "failed" {
		writeError(w, http.StatusBadRequest, "status field must be 'failed' for this route")
		return
	}
	if payload.TicketID == "" {
		writeError(w, http.StatusBadRequest, "missing required field: ticket_id")
		return
	}
	if payload.Reason == "" {
		writeError(w, http.StatusBadRequest, "missing required field: reason")
		return
	}

	if payload.TicketID != ticketID {
		logging.Logger().Warn("ticket ID mismatch between URL and body",
			slog.String("url_ticket_id", ticketID),
			slog.String("body_ticket_id", payload.TicketID),
		)
	}

	event := Event{
		Kind:      EventFailed,
		TicketID:  ticketID,
		Timestamp: time.Now(),
		Reason:    payload.Reason,
		CommentID: payload.CommentID,
	}

	s.sendEvent(w, event)
}

func (s *CallbackServer) sendEvent(w http.ResponseWriter, event Event) {
	select {
	case s.events <- event:
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		writeError(w, http.StatusServiceUnavailable, "event queue full, retry later")
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
