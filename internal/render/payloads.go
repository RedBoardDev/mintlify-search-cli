package render

import "encoding/json"

// Payloads are the typed inputs accepted by Renderer implementations. Each
// struct is designed so its JSON encoding is deterministic and token-lean
// (field order driven, omitempty on optional fields).

// SearchPayload is the output of `msc search`.
type SearchPayload struct {
	Query   string        `json:"query"`
	Results []SearchEntry `json:"results"`
}

// SearchEntry mirrors the fields an agent needs to act on a hit: human title,
// absolute URL, page path (consumable by `msc read`), content preview.
type SearchEntry struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Page    string `json:"page"`
	Content string `json:"content"`
}

// FSPayload is the output of `msc fs --json`. Mirrors the shape an agent can
// branch on directly: exit code, captured streams, truncation flag.
type FSPayload struct {
	ExitCode  int    `json:"exit_code"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Truncated bool   `json:"truncated"`
	Bytes     int    `json:"bytes"`
}

// ReadPayload is the output of `msc read --json`.
type ReadPayload struct {
	Page         string `json:"page"`
	ResolvedPath string `json:"resolved_path"`
	Content      string `json:"content"`
	Truncated    bool   `json:"truncated"`
	Bytes        int    `json:"bytes"`
}

// OpenPayload is the output of `msc open --json`: search metadata + full
// markdown of the top hit.
type OpenPayload struct {
	Title        string `json:"title"`
	URL          string `json:"url"`
	Page         string `json:"page"`
	ResolvedPath string `json:"resolved_path"`
	Content      string `json:"content"`
	Truncated    bool   `json:"truncated"`
}

// DoctorPayload is the output of `msc doctor`.
type DoctorPayload struct {
	OK     bool          `json:"ok"`
	Checks []DoctorCheck `json:"checks"`
}

// DoctorCheck is a single named verification step.
type DoctorCheck struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // "pass" | "fail"
	Detail     string `json:"detail,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// ConfigPayload is the output of `msc config list --json`.
type ConfigPayload struct {
	MCPURL       string      `json:"mcp_url"`
	DefaultLimit int         `json:"default_limit"`
	Cache        CachePayload `json:"cache"`
	Path         string      `json:"path"`
}

type CachePayload struct {
	Enabled         bool `json:"enabled"`
	TTLSeconds      int  `json:"ttl_seconds"`
	ToolsTTLSeconds int  `json:"tools_ttl_seconds"`
}

// RawPayload holds an unparsed MCP result for `--raw` output.
type RawPayload struct {
	Result json.RawMessage
}
