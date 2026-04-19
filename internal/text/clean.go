// Package text provides text normalization helpers used across the CLI.
package text

import (
	"regexp"
	"strings"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

// Clean strips HTML tags and collapses all whitespace runs into a single space.
// Used to normalize search result previews returned by the MCP server, which
// may embed <mark><b>...</b></mark> around matching terms.
func Clean(value string) string {
	value = StripHTML(value)
	value = strings.ReplaceAll(value, "\n", " ")
	return CollapseWhitespace(value)
}

// StripHTML removes all HTML tags from the string. It does not decode
// entities; the MCP server has not been observed to emit entity-encoded text.
func StripHTML(value string) string {
	return htmlTagPattern.ReplaceAllString(value, "")
}

// CollapseWhitespace replaces any run of whitespace (including tabs and
// newlines) with a single space and trims leading/trailing whitespace.
func CollapseWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
