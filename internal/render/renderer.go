// Package render produces CLI output in multiple formats (JSON, text, raw).
package render

import (
	"fmt"
	"io"
)

// FormatKind selects the output strategy.
type FormatKind int

const (
	// FormatJSON is the default for structured commands (search). Minified,
	// newline-terminated, deterministic field order (struct declaration).
	FormatJSON FormatKind = iota
	// FormatText is human-oriented. For content commands (fs/read/open) it
	// prints the raw bytes; for structured commands it prints a list.
	FormatText
	// FormatRaw is the unparsed MCP result envelope, JSON-minified.
	FormatRaw
)

// Renderer writes a payload to w. Each payload type has its own rendering
// rules per FormatKind — see json.go and text.go.
type Renderer interface {
	Render(w io.Writer, payload any) error
}

// New returns a Renderer for the given format.
func New(kind FormatKind) Renderer {
	switch kind {
	case FormatJSON:
		return &jsonRenderer{}
	case FormatText:
		return &textRenderer{}
	case FormatRaw:
		return &rawRenderer{}
	default:
		return &jsonRenderer{}
	}
}

// unsupportedPayload returns a stable error for payloads a renderer cannot
// handle — surfaces a clear bug if a new payload is added without updating
// every renderer.
func unsupportedPayload(format string, payload any) error {
	return fmt.Errorf("render: unsupported %s payload %T", format, payload)
}
