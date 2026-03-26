package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client := NewClient(server.URL)
	client.httpClient = server.Client()
	return client
}

func TestDiscover(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"server":{"name":"Docs","version":"1.0.0","transport":"http"},
			"capabilities":{"tools":{"search_docs":{"name":"search_docs","description":"Search docs","inputSchema":{"type":"object","required":["query"],"properties":{"query":{"type":"string"}}}}}}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	discovery, err := client.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover: %v", err)
	}

	if discovery.Server.Name != "Docs" {
		t.Fatalf("unexpected server name: %q", discovery.Server.Name)
	}

	tool, err := FindSearchTool(discovery)
	if err != nil {
		t.Fatalf("find search tool: %v", err)
	}
	if tool.Name != "search_docs" {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
}

func TestInitialize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json, text/event-stream" {
			t.Fatalf("unexpected accept header: %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"protocolVersion\":\"2024-11-05\",\"capabilities\":{\"tools\":{}},\"serverInfo\":{\"name\":\"Docs\",\"version\":\"1.0.0\",\"transport\":\"http\"}}}\n\n"))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	result, _, err := client.Initialize(context.Background())
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if result.ProtocolVersion != "2024-11-05" {
		t.Fatalf("unexpected protocol version: %q", result.ProtocolVersion)
	}
}

func TestListTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"tools\":[{\"name\":\"search_docs\",\"description\":\"Search docs\",\"inputSchema\":{\"type\":\"object\",\"required\":[\"query\"],\"properties\":{\"query\":{\"type\":\"string\"}}}}]}}\n\n"))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "search_docs" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestCallToolAndNormalize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"Title: About MCP\\nLink: https://docs.example.com/mcp\\nContent: About <mark><b>MCP</b></mark> servers\\n\"}]}}\n\n"))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	resp, err := client.CallTool(context.Background(), "search_docs", map[string]any{"query": "mcp"})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}

	call, err := ParseToolCallResult(resp)
	if err != nil {
		t.Fatalf("parse tool call: %v", err)
	}

	results := NormalizeSearchResults(call)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "About MCP" {
		t.Fatalf("unexpected title: %q", results[0].Title)
	}
	if results[0].URL != "https://docs.example.com/mcp" {
		t.Fatalf("unexpected url: %q", results[0].URL)
	}
	if strings.Contains(results[0].Content, "<mark>") {
		t.Fatalf("expected tags to be stripped, got %q", results[0].Content)
	}
}

func TestCallTool_RPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":-32602,\"message\":\"bad args\"}}\n\n"))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.CallTool(context.Background(), "search_docs", map[string]any{"query": "mcp"})
	if err == nil || !strings.Contains(err.Error(), "bad args") {
		t.Fatalf("expected rpc error, got %v", err)
	}
}

func TestFindSearchToolFromList(t *testing.T) {
	tools := []ToolDefinition{
		{
			Name: "search_docs",
			InputSchema: InputSchema{
				Required: []string{"query"},
				Properties: map[string]SchemaProperty{
					"query": {Type: "string"},
				},
			},
		},
	}

	tool, err := FindSearchToolFromList(tools)
	if err != nil {
		t.Fatalf("find tool: %v", err)
	}
	if tool.Name != "search_docs" {
		t.Fatalf("unexpected tool name: %q", tool.Name)
	}
}

func TestNormalizeSearchResults_SkipsBlocksWithoutURL(t *testing.T) {
	call := &ToolCallResult{
		Content: []ContentBlock{
			{Type: "text", Text: "Title: Missing link\nContent: ignored\n"},
			{Type: "text", Text: "Title: Valid\nLink: https://docs.example.com/a\nContent: ok\n"},
		},
	}

	results := NormalizeSearchResults(call)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Valid" {
		t.Fatalf("unexpected result: %+v", results[0])
	}
}
