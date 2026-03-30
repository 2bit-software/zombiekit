package callback

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/2bit-software/zombiekit/internal/logging"
)

const (
	defaultBufferSize  = 64
	shutdownTimeout    = 5 * time.Second
	readHeaderTimeout  = 5 * time.Second
	writeTimeout       = 10 * time.Second
	idleTimeout        = 30 * time.Second
)

// CallbackServer receives HTTP POST callbacks from agent sessions and delivers
// parsed events to a consumer via a buffered channel.
type CallbackServer struct {
	port       int
	events     chan Event
	httpServer *http.Server
	mux        *http.ServeMux
}

// New creates a CallbackServer that will listen on the given port.
// The event channel is buffered at 64 entries. If the buffer fills,
// incoming requests receive 503 Service Unavailable.
func New(port int) *CallbackServer {
	s := &CallbackServer{
		port:   port,
		events: make(chan Event, defaultBufferSize),
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// Events returns a read-only channel of parsed callback events.
// The channel is closed when Run returns.
func (s *CallbackServer) Events() <-chan Event {
	return s.events
}

// Run starts the HTTP server and blocks until ctx is cancelled.
// On cancellation, it drains in-flight requests (5s timeout) and closes
// the events channel before returning.
func (s *CallbackServer) Run(ctx context.Context) error {
	s.httpServer = &http.Server{
		Handler:           s.mux,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		close(s.events)
		return fmt.Errorf("callback server listen: %w", err)
	}

	logging.Logger().Info("callback server started",
		"addr", ln.Addr().String(),
	)

	errCh := make(chan error, 1)
	go func() {
		if serveErr := s.httpServer.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		err := s.httpServer.Shutdown(shutdownCtx)
		close(s.events)
		return err
	case err := <-errCh:
		close(s.events)
		return err
	}
}

func (s *CallbackServer) registerRoutes() {
	s.mux.HandleFunc("POST /{ticketID}/complete", s.handleComplete)
	s.mux.HandleFunc("POST /{ticketID}/comment-resolved", s.handleCommentResolved)
	s.mux.HandleFunc("POST /{ticketID}/failed", s.handleFailed)
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
}

func (s *CallbackServer) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}
