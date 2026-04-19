package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/redboard/mintlify-search-cli/internal/text"
)

// fsTruncateThreshold is the byte threshold above which we assume the MCP
// server truncated its output. Empirically confirmed at ~30 046 bytes for
// the Mintlify query_docs_filesystem tool; no explicit marker is appended.
const fsTruncateThreshold = 30_000

// SearchBlock is a single normalized entry returned by the MCP search tool.
type SearchBlock struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Page    string `json:"page"`
	Content string `json:"content"`
}

// ParseSearchBlocks decodes a tools/call response for a search tool into a
// slice of SearchBlock. Blocks missing a Link: line are skipped. HTML tags
// and entities in the Content section are stripped.
func ParseSearchBlocks(resp *RPCResponse) ([]SearchBlock, error) {
	call, err := ParseToolCallResult(resp)
	if err != nil {
		return nil, err
	}
	if call.IsError {
		msg := "search tool reported an error"
		if len(call.Content) > 0 && call.Content[0].Text != "" {
			msg = strings.TrimSpace(call.Content[0].Text)
		}
		return nil, errors.New(msg)
	}

	blocks := make([]SearchBlock, 0, len(call.Content))
	for _, b := range call.Content {
		if b.Type != "text" || strings.TrimSpace(b.Text) == "" {
			continue
		}
		blk := parseSearchTextBlock(b.Text)
		if blk.URL == "" {
			continue
		}
		blocks = append(blocks, blk)
	}
	return blocks, nil
}

// parseSearchTextBlock parses the `Title:/Link:/Page:/Content:` text format
// used by Mintlify's search tool. Unknown preamble fields are ignored; any
// content after the `Content:` line is collected until end of block.
func parseSearchTextBlock(raw string) SearchBlock {
	lines := strings.Split(raw, "\n")
	var blk SearchBlock
	var contentParts []string
	inContent := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inContent {
				contentParts = append(contentParts, "")
			}
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "Title:"):
			blk.Title = text.Clean(strings.TrimSpace(strings.TrimPrefix(trimmed, "Title:")))
			inContent = false
		case strings.HasPrefix(trimmed, "Link:"):
			blk.URL = strings.TrimSpace(strings.TrimPrefix(trimmed, "Link:"))
			inContent = false
		case strings.HasPrefix(trimmed, "Page:"):
			blk.Page = strings.TrimSpace(strings.TrimPrefix(trimmed, "Page:"))
			inContent = false
		case strings.HasPrefix(trimmed, "Content:"):
			contentParts = append(contentParts, strings.TrimSpace(strings.TrimPrefix(trimmed, "Content:")))
			inContent = true
		case inContent:
			contentParts = append(contentParts, trimmed)
		}
	}

	blk.Content = text.Clean(strings.Join(contentParts, "\n"))
	if blk.Title == "" {
		blk.Title = blk.URL
	}
	return blk
}

// FSResult is a parsed response from the MCP docs filesystem tool.
type FSResult struct {
	Exit      int    `json:"exit_code"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Truncated bool   `json:"truncated"`
	Bytes     int    `json:"bytes"`
}

var exitLine = regexp.MustCompile(`^exit:\s*(-?\d+)`)

// ParseFSResult decodes a tools/call response for the docs filesystem tool
// into an FSResult. It handles both successful runs (with exit: 0) and
// error results returned as isError=true (sandbox validation failure etc).
func ParseFSResult(resp *RPCResponse) (*FSResult, error) {
	call, err := ParseToolCallResult(resp)
	if err != nil {
		return nil, err
	}

	var combined strings.Builder
	for _, b := range call.Content {
		if b.Type == "text" {
			combined.WriteString(b.Text)
		}
	}
	raw := combined.String()

	if call.IsError {
		return &FSResult{
			Exit:   1,
			Stderr: strings.TrimSpace(raw),
			Bytes:  len(raw),
		}, nil
	}

	res := &FSResult{Bytes: len(raw)}
	if res.Bytes >= fsTruncateThreshold && !strings.Contains(raw, "--- stderr ---") {
		res.Truncated = true
	}

	lines := strings.Split(raw, "\n")
	var (
		stdout strings.Builder
		stderr strings.Builder
		in     = "none" // "none" | "stdout" | "stderr"
	)
	for i, line := range lines {
		if i == 0 {
			m := exitLine.FindStringSubmatch(line)
			if m != nil {
				n, _ := strconv.Atoi(m[1])
				res.Exit = n
				continue
			}
			// No exit: header — treat the whole payload as stdout.
			stdout.WriteString(line)
			stdout.WriteByte('\n')
			in = "stdout"
			continue
		}
		if strings.TrimSpace(line) == "--- stdout ---" {
			in = "stdout"
			continue
		}
		if strings.TrimSpace(line) == "--- stderr ---" {
			in = "stderr"
			continue
		}
		switch in {
		case "stdout":
			stdout.WriteString(line)
			stdout.WriteByte('\n')
		case "stderr":
			stderr.WriteString(line)
			stderr.WriteByte('\n')
		}
	}
	res.Stdout = trimTrailingNewline(stdout.String())
	res.Stderr = trimTrailingNewline(stderr.String())
	return res, nil
}

func trimTrailingNewline(s string) string {
	return strings.TrimRight(s, "\n")
}

// ParseToolCallResult decodes a tools/call response's result envelope. It is
// the low-level building block used by Parse* helpers.
func ParseToolCallResult(resp *RPCResponse) (*ToolCallResult, error) {
	if len(resp.Result) == 0 {
		return nil, fmt.Errorf("missing result payload")
	}
	var r ToolCallResult
	if err := json.Unmarshal(resp.Result, &r); err != nil {
		return nil, fmt.Errorf("decoding tool call result: %w", err)
	}
	return &r, nil
}
