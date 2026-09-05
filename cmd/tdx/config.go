package main

// These variables are set at build time via ldflags from tdx.toml
// Build with: mise run build

var (
	Version     = "dev"
	Description = "dev build"
)
