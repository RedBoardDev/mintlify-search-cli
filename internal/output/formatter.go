package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/redboard/mintlify-search-cli/internal/mcp"
)

type Format int

const (
	FormatText Format = iota
	FormatJSON
)

func Render(w io.Writer, results []mcp.SearchResult, format Format) error {
	if len(results) == 0 {
		_, err := fmt.Fprintln(w, "No results found.")
		return err
	}

	switch format {
	case FormatJSON:
		return renderJSON(w, results)
	default:
		return renderText(w, results)
	}
}

func renderJSON(w io.Writer, results []mcp.SearchResult) error {
	data, err := json.Marshal(results)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = fmt.Fprintln(w)
	return err
}

func renderText(w io.Writer, results []mcp.SearchResult) error {
	for i, r := range results {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}

		title := r.Title
		snippet := truncate(r.Content, 200)

		if _, err := fmt.Fprintf(w, "[%d] %s\n", i+1, title); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "    %s\n", r.URL); err != nil {
			return err
		}
		if snippet != "" {
			if _, err := fmt.Fprintf(w, "    %s\n", snippet); err != nil {
				return err
			}
		}
	}
	return nil
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen-3]) + "..."
}
