package orchestrator

import (
	"context"
	"log/slog"
	"time"

	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/2bit-software/zombiekit/internal/shutdown"
)

const (
	WatcherLinearPoller   = "linear-poller"
	WatcherPRWatcher      = "pr-watcher"
	WatcherCommentWatcher = "comment-watcher"
)

// NewWatcherStub returns a ServiceFunc that logs start/stop and blocks until
// context cancellation. The pollInterval is captured for future use but unused
// by the stub.
func NewWatcherStub(name string, pollInterval time.Duration) shutdown.ServiceFunc {
	return func(ctx context.Context) error {
		logger := logging.Logger().With(slog.String("watcher", name))
		logger.Info("watcher started", slog.Duration("poll_interval", pollInterval))
		<-ctx.Done()
		logger.Info("watcher stopped")
		return nil
	}
}
