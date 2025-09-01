package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version of the application
	// This can be overridden at build time with -ldflags
	Version = "dev"

	// GitCommit is the git commit hash
	// This can be overridden at build time with -ldflags
	GitCommit = "unknown"

	// BuildDate is the date the binary was built
	// This can be overridden at build time with -ldflags
	BuildDate = "unknown"

	// GoVersion is the version of Go used to build the binary
	GoVersion = runtime.Version()

	// Platform is the OS/Arch the binary was built for
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetVersion returns the version information
func GetVersion() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  Platform,
	}
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	info := GetVersion()
	if info.GitCommit != "unknown" && len(info.GitCommit) > 7 {
		return fmt.Sprintf("%s (%s)", info.Version, info.GitCommit[:7])
	}
	return info.Version
}

// GetFullVersionString returns a detailed version string
func GetFullVersionString() string {
	info := GetVersion()
	return fmt.Sprintf(`blockbench version %s
Git commit: %s
Build date: %s
Go version: %s
Platform: %s`,
		info.Version,
		info.GitCommit,
		info.BuildDate,
		info.GoVersion,
		info.Platform)
}
