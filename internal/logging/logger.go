// Package logging provides structured logging setup using slog.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Factory creates configured loggers.
type Factory interface {
	CreateLogger(level string, jsonOutput bool) *slog.Logger
}

// logLevel holds the current log level, allowing runtime changes.
var logLevel = new(slog.LevelVar)

// singleton holds the global logger instance, set by InitLogger.
var singleton *slog.Logger

// SetupLogger creates a new structured logger with the specified configuration.
//
// Parameters:
//   - level: Log level (debug, info, warn, error). Defaults to info.
//   - jsonOutput: If true, outputs JSON format. Otherwise, outputs text format.
//   - w: Writer for log output. If nil, defaults to os.Stderr.
func SetupLogger(level string, jsonOutput bool, w io.Writer) *slog.Logger {
	// Parse and set level
	switch strings.ToLower(level) {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "info":
		logLevel.Set(slog.LevelInfo)
	case "warn", "warning":
		logLevel.Set(slog.LevelWarn)
	case "error":
		logLevel.Set(slog.LevelError)
	default:
		logLevel.Set(slog.LevelInfo)
	}

	if w == nil {
		w = os.Stderr
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if jsonOutput {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}

// InitLogger initializes the singleton logger with the specified configuration.
// Panics if called more than once (prevents accidental re-initialization).
// Returns the logger for backward compatibility during migration.
func InitLogger(level string, jsonOutput bool, w io.Writer) *slog.Logger {
	if singleton != nil {
		panic("logging: InitLogger called more than once")
	}
	singleton = SetupLogger(level, jsonOutput, w)
	return singleton
}

// Logger returns the singleton logger.
// Panics if InitLogger was not called (fail-fast for configuration errors).
func Logger() *slog.Logger {
	if singleton == nil {
		panic("logging: Logger() called before InitLogger()")
	}
	return singleton
}

// ResetLogger clears the singleton for testing.
// Should only be called from tests.
func ResetLogger() {
	singleton = nil
}
