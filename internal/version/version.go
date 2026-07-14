// Package version exposes the build-time application version.
package version

// Version is set via -ldflags at build time (see scripts/version.sh and the
// Makefile/Dockerfile). It stays "dev" for `go run`/`go test` and other
// unlinked builds.
var Version = "dev"
