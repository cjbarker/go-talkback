package main

// version is set at build time via:
//
//	go build -ldflags "-X main.version=<version>"
//
// When built without that flag (e.g. `go run .`) it falls back to "dev".
var version = "dev"
