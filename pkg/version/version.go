// Package version provides build-time version information for the go-scream bot.
// The variables are intended to be set via linker flags at build time:
//
//	go build -ldflags "-X github.com/JamesPrial/go-scream/pkg/version.Version=1.0.0 \
//	                   -X github.com/JamesPrial/go-scream/pkg/version.Commit=abc1234 \
//	                   -X github.com/JamesPrial/go-scream/pkg/version.Date=2026-01-15"
package version

import "fmt"

// Version is the semantic version string. Defaults to "dev".
var Version = "dev"

// Commit is the git commit hash at build time. Defaults to "unknown".
var Commit = "unknown"

// Date is the build timestamp. Defaults to "unknown".
var Date = "unknown"

// String returns a human-readable version string in the format:
//
//	VERSION (commit: COMMIT, built: DATE)
func String() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date)
}
