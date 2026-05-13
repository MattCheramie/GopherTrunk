// Package version exposes build metadata injected at link time via
// `go build -ldflags "-X ...Version=..." -ldflags "-X ...Commit=..."
// -ldflags "-X ...BuildTime=..."`. Default values keep development
// builds informative without requiring an LDFLAGS dance.
package version

// Version is the human-readable build version. Set at link time by
// the Makefile + release workflow to the git tag (e.g. "v1.0.0") or
// `git describe` output for in-development builds. Defaults to "dev"
// for `go run` / `go test` invocations.
var Version = "dev"

// Commit is the short git SHA the binary was built from. Set at
// link time via -ldflags. Empty for non-LDFLAGS builds.
var Commit = ""

// BuildTime is the ISO-8601 UTC timestamp at which the binary was
// built. Set at link time via -ldflags. Empty for non-LDFLAGS builds.
var BuildTime = ""

// String returns a single-line summary suitable for logging or
// for the `gophertrunk version` subcommand: "vX.Y.Z (sha=ABC1234,
// built=2026-05-13T19:00:00Z)". Empty Commit / BuildTime are
// omitted so dev builds stay compact.
func String() string {
	out := Version
	if Commit != "" || BuildTime != "" {
		out += " ("
		if Commit != "" {
			out += "sha=" + Commit
		}
		if Commit != "" && BuildTime != "" {
			out += ", "
		}
		if BuildTime != "" {
			out += "built=" + BuildTime
		}
		out += ")"
	}
	return out
}
