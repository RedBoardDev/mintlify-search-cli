package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func makeRPC(t *testing.T, result any) *RPCResponse {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	return &RPCResponse{JSONRPC: "2.0", ID: json.RawMessage("1"), Result: raw}
}

func TestParseSearchBlocks(t *testing.T) {
	resp := makeRPC(t, ToolCallResult{Content: []ContentBlock{
		{Type: "text", Text: "Title: About MCP\nLink: https://docs.example.com/mcp\nPage: mcp/overview\nContent: About <mark><b>MCP</b></mark> servers\n"},
		{Type: "text", Text: "Title: Missing link\nContent: ignored\n"},
		{Type: "text", Text: "Title: Multi line\nLink: https://docs.example.com/x\nPage: x/y\nContent: line one\nline two\n"},
	}})

	blocks, err := ParseSearchBlocks(resp)
	if err != nil {
		t.Fatalf("ParseSearchBlocks: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[0].Title != "About MCP" || blocks[0].URL != "https://docs.example.com/mcp" || blocks[0].Page != "mcp/overview" {
		t.Fatalf("block 0 incorrect: %+v", blocks[0])
	}
	if strings.Contains(blocks[0].Content, "<mark>") {
		t.Fatalf("html not stripped: %q", blocks[0].Content)
	}
	if blocks[1].Content != "line one line two" {
		t.Fatalf("multi-line content not joined: %q", blocks[1].Content)
	}
}

func TestParseSearchBlocks_IsError(t *testing.T) {
	resp := makeRPC(t, ToolCallResult{
		IsError: true,
		Content: []ContentBlock{{Type: "text", Text: "Search failed: query too short"}},
	})
	_, err := ParseSearchBlocks(resp)
	if err == nil || !strings.Contains(err.Error(), "query too short") {
		t.Fatalf("expected error with message, got %v", err)
	}
}

func TestParseFSResult(t *testing.T) {
	cases := []struct {
		name       string
		text       string
		wantExit   int
		wantStdout string
		wantStderr string
		wantTrunc  bool
	}{
		{
			name:       "success with stdout only",
			text:       "exit: 0\n--- stdout ---\nhello\nworld\n",
			wantExit:   0,
			wantStdout: "hello\nworld",
		},
		{
			name:       "success with both",
			text:       "exit: 0\n--- stdout ---\nout\n--- stderr ---\nerr\n",
			wantExit:   0,
			wantStdout: "out",
			wantStderr: "err",
		},
		{
			name:       "nonzero exit",
			text:       "exit: 127\n--- stderr ---\nbash: curl: command not found\n",
			wantExit:   127,
			wantStderr: "bash: curl: command not found",
		},
		{
			name:       "no exit header treated as stdout",
			text:       "bare text\n",
			wantExit:   0,
			wantStdout: "bare text",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := makeRPC(t, ToolCallResult{Content: []ContentBlock{{Type: "text", Text: tc.text}}})
			res, err := ParseFSResult(resp)
			if err != nil {
				t.Fatalf("ParseFSResult: %v", err)
			}
			if res.Exit != tc.wantExit {
				t.Errorf("exit = %d, want %d", res.Exit, tc.wantExit)
			}
			if res.Stdout != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", res.Stdout, tc.wantStdout)
			}
			if res.Stderr != tc.wantStderr {
				t.Errorf("stderr = %q, want %q", res.Stderr, tc.wantStderr)
			}
			if res.Truncated != tc.wantTrunc {
				t.Errorf("truncated = %v, want %v", res.Truncated, tc.wantTrunc)
			}
		})
	}
}

func TestParseFSResult_Truncation(t *testing.T) {
	big := strings.Repeat("x", fsTruncateThreshold+10)
	text := "exit: 0\n--- stdout ---\n" + big
	resp := makeRPC(t, ToolCallResult{Content: []ContentBlock{{Type: "text", Text: text}}})
	res, err := ParseFSResult(resp)
	if err != nil {
		t.Fatalf("ParseFSResult: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("expected truncated=true when no --- stderr --- marker")
	}
}

func TestParseFSResult_IsError(t *testing.T) {
	resp := makeRPC(t, ToolCallResult{
		IsError: true,
		Content: []ContentBlock{{Type: "text", Text: "Docs filesystem query failed: Invalid request body"}},
	})
	res, err := ParseFSResult(resp)
	if err != nil {
		t.Fatalf("ParseFSResult: %v", err)
	}
	if res.Exit != 1 || !strings.Contains(res.Stderr, "Invalid request body") {
		t.Fatalf("unexpected: %+v", res)
	}
}
