package text

import "testing"

func TestClean(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain", "hello world", "hello world"},
		{"html mark", "rate <mark><b>limit</b></mark> exceeded", "rate limit exceeded"},
		{"nested tags", "<p>hello <em><strong>there</strong></em></p>", "hello there"},
		{"multiline collapsed", "line one\n\nline two\n   line three", "line one line two line three"},
		{"tabs and nbsp", "foo\tbar\tbaz", "foo bar baz"},
		{"unicode preserved", "café — 日本語", "café — 日本語"},
		{"self-closing tag", "break<br/>line", "breakline"},
		{"attrs stripped", `<a href="x" class="y">link</a>`, "link"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Clean(tc.in); got != tc.want {
				t.Fatalf("Clean(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"no tags", "no tags"},
		{"<b>bold</b>", "bold"},
		{"<a href='x'>a</a>", "a"},
		{"a<br>b", "ab"},
	}
	for _, tc := range cases {
		if got := StripHTML(tc.in); got != tc.want {
			t.Fatalf("StripHTML(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCollapseWhitespace(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"   ", ""},
		{"  a  b  ", "a b"},
		{"a\n\nb\tc", "a b c"},
	}
	for _, tc := range cases {
		if got := CollapseWhitespace(tc.in); got != tc.want {
			t.Fatalf("CollapseWhitespace(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
