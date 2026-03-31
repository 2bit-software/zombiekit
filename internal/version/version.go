// Package version provides build version information.
package version

import (
	"fmt"
	"runtime"
)

// Package-level variables set via ldflags at build time.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// Provider returns build information.
type Provider interface {
	GetBuildInfo() *BuildInfo
}

// BuildInfo contains version and build information.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
}

// Get returns the build information populated from ldflags.
func Get() *BuildInfo {
	return &BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
		GoVersion: runtime.Version(),
	}
}

// Short returns a short version string suitable for --version flag.
func (b *BuildInfo) Short() string {
	if b.Version == "dev" {
		return fmt.Sprintf("dev (%s)", b.Commit)
	}
	return b.Version
}

// PrettyPrint returns a formatted multi-line version string.
func (b *BuildInfo) PrettyPrint() string {
	return fmt.Sprintf(
		"brains version %s\ncommit: %s\nbuild date: %s\ngo version: %s",
		b.Version, b.Commit, b.BuildDate, b.GoVersion,
	)
}
