package version

import (
	"fmt"
	"runtime"
)

// These variables are set at build time using -ldflags
var (
	// Version is the semantic version
	Version = "0.1.0"

	// GitCommit is the git commit hash
	GitCommit = "dev"

	// BuildDate is the build timestamp
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()
)

// GetVersion returns the full version string
func GetVersion() string {
	if GitCommit != "dev" && len(GitCommit) > 7 {
		return fmt.Sprintf("%s-%s", Version, GitCommit[:7])
	}
	return Version
}

// GetFullVersion returns detailed version information
func GetFullVersion() string {
	return fmt.Sprintf("HelixCode %s (commit: %s, built: %s, go: %s)",
		Version, GitCommit, BuildDate, GoVersion)
}

// GetBuildInfo returns structured build information
func GetBuildInfo() map[string]string {
	return map[string]string{
		"version":    Version,
		"commit":     GitCommit,
		"build_date": BuildDate,
		"go_version": GoVersion,
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	}
}
