package main

// version, commit, and date are set at build time via -ldflags.
// They fall back to safe defaults for untagged/local builds.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)
