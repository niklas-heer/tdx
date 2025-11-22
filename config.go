package main

// These variables are set at build time via ldflags from tdx.toml
// Build with: just build-go

var (
	Version     = "dev"
	Description = "dev build"
)
