//go:build tools

// Package tools pins build-time tools (not compiled into the app) so they are
// versioned in go.mod and run via `go run` — no global installs.
package tools

import (
	_ "github.com/swaggo/swag/cmd/swag"
)
