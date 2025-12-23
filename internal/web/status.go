package web

import (
	"context"
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/version"
)

// StatusInfo aggregates all system status information for display.
type StatusInfo struct {
	Version  VersionInfo
	Database DatabaseStatus
	Runtime  RuntimeInfo
	Plugins  []PluginStatus
	Config   ConfigInfo
}

// VersionInfo contains application build information.
type VersionInfo struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
}

// DatabaseStatus contains database backend status.
type DatabaseStatus struct {
	Backend   string // "sqlite" or "postgres"
	Location  string // Sanitized path or host/db
	Connected bool
	Error     string // Empty if connected
}

// RuntimeInfo contains runtime environment information.
type RuntimeInfo struct {
	OS            string
	Arch          string
	Platform      string
	Uptime        time.Duration
	UptimeHuman   string
	NumCPU        int
	NumGoroutines int
}

// PluginStatus contains plugin registration status.
type PluginStatus struct {
	Name    string
	Path    string
	Healthy bool
}

// ConfigInfo contains key configuration values.
type ConfigInfo struct {
	Port         int
	LogLevel     string
	ProfilePaths []string
}

// StatusConfig holds dependencies for status gathering.
type StatusConfig struct {
	ServerPort    int
	LogLevel      string
	StorageConfig config.StorageConfig
	StartTime     time.Time
}

// GatherStatus collects all status information.
func GatherStatus(ctx context.Context, cfg StatusConfig, registry *PluginRegistry) StatusInfo {
	return StatusInfo{
		Version:  gatherVersionInfo(),
		Database: gatherDatabaseStatus(cfg.StorageConfig),
		Runtime:  gatherRuntimeInfo(cfg.StartTime),
		Plugins:  gatherPluginStatus(registry),
		Config:   gatherConfigInfo(cfg),
	}
}

// gatherVersionInfo retrieves version information from the version package.
func gatherVersionInfo() VersionInfo {
	info := version.Get()
	return VersionInfo{
		Version:   info.Version,
		Commit:    info.Commit,
		BuildDate: info.BuildDate,
		GoVersion: info.GoVersion,
	}
}

// gatherDatabaseStatus extracts database backend type and sanitized location.
func gatherDatabaseStatus(cfg config.StorageConfig) DatabaseStatus {
	status := DatabaseStatus{
		Backend:   string(cfg.Backend),
		Connected: true, // Placeholder - health check deferred
	}

	switch cfg.Backend {
	case config.BackendSQLite:
		status.Location = cfg.SQLitePath
	case config.BackendPostgres:
		status.Location = sanitizePostgresURL(cfg.PostgresURL)
	default:
		status.Location = "Unknown"
	}

	return status
}

// sanitizePostgresURL removes credentials from a PostgreSQL connection URL.
// Returns only host/database for safe display.
func sanitizePostgresURL(connURL string) string {
	if connURL == "" {
		return "(not configured)"
	}

	u, err := url.Parse(connURL)
	if err != nil {
		return "(invalid connection string)"
	}

	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		dbName = "(default)"
	}

	return fmt.Sprintf("%s/%s", u.Host, dbName)
}

// formatUptime formats a duration into a human-readable string.
func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

// gatherRuntimeInfo collects runtime environment information.
func gatherRuntimeInfo(startTime time.Time) RuntimeInfo {
	uptime := time.Since(startTime)
	return RuntimeInfo{
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		Platform:      fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Uptime:        uptime,
		UptimeHuman:   formatUptime(uptime),
		NumCPU:        runtime.NumCPU(),
		NumGoroutines: runtime.NumGoroutine(),
	}
}

// gatherPluginStatus iterates the registry and builds plugin status list.
func gatherPluginStatus(registry *PluginRegistry) []PluginStatus {
	if registry == nil {
		return nil
	}

	registered := registry.All()
	plugins := make([]PluginStatus, 0, len(registered))

	for _, rp := range registered {
		plugins = append(plugins, PluginStatus{
			Name:    rp.Name(),
			Path:    "/" + rp.Name(),
			Healthy: true, // V1 simplification: all registered plugins are healthy
		})
	}

	return plugins
}

// gatherConfigInfo extracts configuration values for display.
func gatherConfigInfo(cfg StatusConfig) ConfigInfo {
	return ConfigInfo{
		Port:         cfg.ServerPort,
		LogLevel:     cfg.LogLevel,
		ProfilePaths: []string{}, // Placeholder for V1
	}
}
