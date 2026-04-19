package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c := NewClient(server.URL)
	c.httpClient = server.Client()
	return c
}

func sseMessage(payload string) string {
	return "event: message\ndata: " + payload + "\n\n"
}

func TestClient_InitializeJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json, text/event-stream" {
			t.Fatalf("unexpected Accept: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","serverInfo":{"name":"X","version":"1"}}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	res, dur, err := c.Initialize(context.Background())
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if res.ProtocolVersion != "2025-06-18" {
		t.Fatalf("got protocolVersion %q", res.ProtocolVersion)
	}
	if dur <= 0 {
		t.Fatalf("expected positive duration")
	}
}

func TestClient_InitializeSSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sseMessage(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","serverInfo":{"name":"X","version":"1"}}}`)))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, _, err := c.Initialize(context.Background())
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
}

func TestClient_CallTool_RPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"bad args"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.CallTool(context.Background(), "search_x", map[string]any{"query": "x"})
	if err == nil || !strings.Contains(err.Error(), "bad args") {
		t.Fatalf("expected rpc error, got %v", err)
	}
}

func TestClient_Retries5xx(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("bad gateway"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestClient_NoRetryOn4xx(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected exactly 1 attempt on 4xx, got %d", got)
	}
}

func TestClient_ResourcesListAndRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(string(readAll(t, r)), "resources/list"):
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"resources":[{"uri":"x://a","name":"a","description":"d","mimeType":"text/markdown"}]}}`))
		default:
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"contents":[{"uri":"x://a","mimeType":"text/markdown","text":"hello"}]}}`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server)
	resources, err := c.ListResources(context.Background())
	if err != nil || len(resources) != 1 || resources[0].URI != "x://a" {
		t.Fatalf("ListResources: %+v err=%v", resources, err)
	}
	contents, err := c.ReadResource(context.Background(), "x://a")
	if err != nil || len(contents) != 1 || contents[0].Text != "hello" {
		t.Fatalf("ReadResource: %+v err=%v", contents, err)
	}
}

func TestClient_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := c.ListTools(ctx); err == nil {
		t.Fatal("expected context timeout error")
	}
}

func readAll(t *testing.T, r *http.Request) []byte {
	t.Helper()
	defer func() { _ = r.Body.Close() }()
	buf := make([]byte, 4096)
	n, _ := r.Body.Read(buf)
	return buf[:n]
}
