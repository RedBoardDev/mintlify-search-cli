package render

import (
	"fmt"
	"io"
	"strings"
)

type textRenderer struct{}

// Render produces a human-readable text form of the payload. Content
// payloads (fs/read/open) emit raw bytes without wrapping; structured
// payloads (search, doctor, config) emit a formatted listing.
func (r *textRenderer) Render(w io.Writer, payload any) error {
	switch p := payload.(type) {
	case SearchPayload:
		return renderSearchText(w, p)
	case *SearchPayload:
		return renderSearchText(w, *p)
	case FSPayload:
		// For fs, stdout is the raw content — stderr goes separately via the
		// handler; we only emit stdout here.
		_, err := io.WriteString(w, p.Stdout)
		if err == nil && p.Stdout != "" && !strings.HasSuffix(p.Stdout, "\n") {
			_, err = io.WriteString(w, "\n")
		}
		return err
	case ReadPayload:
		_, err := io.WriteString(w, p.Content)
		if err == nil && p.Content != "" && !strings.HasSuffix(p.Content, "\n") {
			_, err = io.WriteString(w, "\n")
		}
		return err
	case OpenPayload:
		_, err := io.WriteString(w, p.Content)
		if err == nil && p.Content != "" && !strings.HasSuffix(p.Content, "\n") {
			_, err = io.WriteString(w, "\n")
		}
		return err
	case DoctorPayload:
		return renderDoctorText(w, p)
	case ConfigPayload:
		return renderConfigText(w, p)
	default:
		return unsupportedPayload("text", payload)
	}
}

func renderSearchText(w io.Writer, p SearchPayload) error {
	if len(p.Results) == 0 {
		_, err := fmt.Fprintf(w, "no results for query %q\n", p.Query)
		return err
	}
	for i, r := range p.Results {
		if i > 0 {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "%d. %s\n   %s\n", i+1, r.Title, r.URL); err != nil {
			return err
		}
		if r.Page != "" {
			if _, err := fmt.Fprintf(w, "   page: %s\n", r.Page); err != nil {
				return err
			}
		}
		if r.Content != "" {
			if _, err := fmt.Fprintf(w, "   %s\n", truncate(r.Content, 200)); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderDoctorText(w io.Writer, p DoctorPayload) error {
	for _, c := range p.Checks {
		status := "PASS"
		if c.Status != "pass" {
			status = "FAIL"
		}
		extra := ""
		if c.Detail != "" {
			extra = " — " + c.Detail
		}
		if c.DurationMs > 0 {
			extra += fmt.Sprintf(" (%dms)", c.DurationMs)
		}
		if _, err := fmt.Fprintf(w, "  [%s] %s%s\n", status, c.Name, extra); err != nil {
			return err
		}
	}
	summary := "All checks passed."
	if !p.OK {
		summary = "Some checks failed."
	}
	_, err := fmt.Fprintln(w, "\n"+summary)
	return err
}

func renderConfigText(w io.Writer, p ConfigPayload) error {
	_, err := fmt.Fprintf(w,
		"config file: %s\nmcp_url: %s\ndefault_limit: %d\ncache.enabled: %t\ncache.ttl_seconds: %d\ncache.tools_ttl_seconds: %d\n",
		p.Path, p.MCPURL, p.DefaultLimit, p.Cache.Enabled, p.Cache.TTLSeconds, p.Cache.ToolsTTLSeconds,
	)
	return err
}

func truncate(s string, max int) string {
	if max <= 3 {
		return s
	}
	if len(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
