package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, server *httptest.Server, apiKey, domain string) *Client {
	t.Helper()
	c := NewClient(apiKey, domain)
	c.httpClient = server.Client()
	c.baseURL = server.URL
	return c
}

func TestSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("expected 'Bearer test-key', got %q", got)
		}
		if got := r.Header.Get("X-Mintlify-Domain"); got != "docs.example.com" {
			t.Errorf("expected domain 'docs.example.com', got %q", got)
		}

		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Query != "auth" {
			t.Errorf("expected query 'auth', got %q", req.Query)
		}
		if req.PageSize != 5 {
			t.Errorf("expected pageSize 5, got %d", req.PageSize)
		}

		resp := searchResponse{
			Results: []SearchResult{
				{Title: "Authentication", Content: "How to authenticate", Path: "/auth"},
				{Title: "OAuth", Content: "OAuth flow", Path: "/oauth", Section: "Setup"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server, "test-key", "docs.example.com")
	results, err := client.Search(context.Background(), "auth", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Title != "Authentication" {
		t.Errorf("expected 'Authentication', got %q", results[0].Title)
	}
	if results[1].Section != "Setup" {
		t.Errorf("expected section 'Setup', got %q", results[1].Section)
	}
}

func TestSearch_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := newTestClient(t, server, "bad-key", "docs.example.com")
	_, err := client.Search(context.Background(), "auth", 5)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestSearch_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(t, server, "key", "bad.domain")
	_, err := client.Search(context.Background(), "auth", 5)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSearch_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := searchResponse{Results: []SearchResult{}}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server, "key", "docs.example.com")
	results, err := client.Search(context.Background(), "nonexistent", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
