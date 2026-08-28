package version

import "fmt"

// Version is the CLI release version. Override at build time via -ldflags.
var Version = "1.0.6"

// Commit is the git commit hash injected at build time. Defaults to "dev".
var Commit = "dev"

// BuildDate is the build timestamp injected at build time.
var BuildDate = "unknown"

// String returns the semantic version label (e.g. "1.0.0").
func String() string {
	return Version
}

// Display returns a user-facing tool identifier with version (e.g. "schedulegate v1.0.0").
func Display() string {
	return fmt.Sprintf("ScheduleGate v%s", Version)
}

// Long returns the full build identity for --version output.
func Long() string {
	return fmt.Sprintf("%s (commit %s, built %s)", Display(), Commit, BuildDate)
}
