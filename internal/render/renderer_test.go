package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func mustJSON(t *testing.T, r Renderer, payload any) string {
	t.Helper()
	var buf bytes.Buffer
	if err := r.Render(&buf, payload); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestJSONRenderer_Search_Golden(t *testing.T) {
	r := New(FormatJSON)
	got := mustJSON(t, r, SearchPayload{
		Query: "auth",
		Results: []SearchEntry{
			{Title: "Auth", URL: "https://x.com/a", Page: "a", Content: "c1"},
			{Title: "Login", URL: "https://x.com/b", Page: "b", Content: "c2"},
		},
	})
	want := `{"query":"auth","results":[{"title":"Auth","url":"https://x.com/a","page":"a","content":"c1"},{"title":"Login","url":"https://x.com/b","page":"b","content":"c2"}]}` + "\n"
	if got != want {
		t.Fatalf("mismatch\n got:  %q\n want: %q", got, want)
	}
}

func TestJSONRenderer_IsDeterministic(t *testing.T) {
	r := New(FormatJSON)
	payload := SearchPayload{Query: "q", Results: []SearchEntry{{Title: "t", URL: "u", Page: "p", Content: "c"}}}
	a := mustJSON(t, r, payload)
	b := mustJSON(t, r, payload)
	if a != b {
		t.Fatalf("not deterministic:\n a=%q\n b=%q", a, b)
	}
}

func TestJSONRenderer_FS(t *testing.T) {
	r := New(FormatJSON)
	got := mustJSON(t, r, FSPayload{ExitCode: 0, Stdout: "hello", Stderr: "", Truncated: false, Bytes: 5})
	want := `{"exit_code":0,"stdout":"hello","stderr":"","truncated":false,"bytes":5}` + "\n"
	if got != want {
		t.Fatalf("mismatch: %q vs %q", got, want)
	}
}

func TestJSONRenderer_NoTrailingWhitespace(t *testing.T) {
	r := New(FormatJSON)
	got := mustJSON(t, r, SearchPayload{Query: "x"})
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("expected trailing newline")
	}
	if strings.Contains(got[:len(got)-1], "\n") {
		t.Fatalf("expected minified (single line): %q", got)
	}
}

func TestTextRenderer_Search(t *testing.T) {
	r := New(FormatText)
	var buf bytes.Buffer
	_ = r.Render(&buf, SearchPayload{
		Query: "auth",
		Results: []SearchEntry{
			{Title: "Auth", URL: "https://x.com/a", Page: "a", Content: "short"},
		},
	})
	out := buf.String()
	if !strings.Contains(out, "1. Auth") || !strings.Contains(out, "https://x.com/a") {
		t.Fatalf("unexpected text output: %q", out)
	}
}

func TestTextRenderer_SearchEmpty(t *testing.T) {
	r := New(FormatText)
	var buf bytes.Buffer
	_ = r.Render(&buf, SearchPayload{Query: "nothing"})
	if !strings.Contains(buf.String(), `no results for query "nothing"`) {
		t.Fatalf("got: %q", buf.String())
	}
}

func TestTextRenderer_FSEmitsStdoutOnly(t *testing.T) {
	r := New(FormatText)
	var buf bytes.Buffer
	_ = r.Render(&buf, FSPayload{Stdout: "line1\nline2", Stderr: "ignored"})
	if buf.String() != "line1\nline2\n" {
		t.Fatalf("got: %q", buf.String())
	}
}

func TestTextRenderer_Read(t *testing.T) {
	r := New(FormatText)
	var buf bytes.Buffer
	_ = r.Render(&buf, ReadPayload{Content: "markdown body"})
	if buf.String() != "markdown body\n" {
		t.Fatalf("got: %q", buf.String())
	}
}

func TestRawRenderer(t *testing.T) {
	r := New(FormatRaw)
	var buf bytes.Buffer
	raw := json.RawMessage(`{"result":{"x":1}}`)
	_ = r.Render(&buf, RawPayload{Result: raw})
	if buf.String() != string(raw)+"\n" {
		t.Fatalf("got: %q", buf.String())
	}
}

func TestRawRenderer_Nil(t *testing.T) {
	r := New(FormatRaw)
	var buf bytes.Buffer
	if err := r.Render(&buf, nil); err == nil {
		t.Fatal("expected error on nil")
	}
}

func TestDoctorText(t *testing.T) {
	r := New(FormatText)
	var buf bytes.Buffer
	_ = r.Render(&buf, DoctorPayload{
		OK: true,
		Checks: []DoctorCheck{
			{Name: "Config", Status: "pass", Detail: "loaded"},
			{Name: "MCP", Status: "pass", DurationMs: 400},
		},
	})
	s := buf.String()
	if !strings.Contains(s, "[PASS] Config") || !strings.Contains(s, "All checks passed.") {
		t.Fatalf("got: %q", s)
	}
}
