package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetSet_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := New(dir)

	key := Key("domain", "query", "5")
	data := []byte(`[{"title":"Test"}]`)

	if err := c.Set(key, data); err != nil {
		t.Fatalf("set: %v", err)
	}

	got, err := c.Get(key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("got %q, want %q", got, data)
	}
}

func TestGet_Miss(t *testing.T) {
	dir := t.TempDir()
	c := New(dir)

	got, err := c.Get("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil on miss, got %q", got)
	}
}

func TestGet_Expired(t *testing.T) {
	dir := t.TempDir()
	c := New(dir)
	c.ttl = 1 * time.Millisecond

	key := Key("domain", "query", "5")
	if err := c.Set(key, []byte("data")); err != nil {
		t.Fatalf("set: %v", err)
	}

	// Wait for expiry.
	time.Sleep(5 * time.Millisecond)

	got, err := c.Get(key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after expiry, got %q", got)
	}
}

func TestKey_Deterministic(t *testing.T) {
	a := Key("docs.example.com", "auth", "5")
	b := Key("docs.example.com", "auth", "5")
	if a != b {
		t.Errorf("keys differ: %q vs %q", a, b)
	}

	c := Key("docs.example.com", "auth", "10")
	if a == c {
		t.Error("different inputs produced same key")
	}
}

func TestSet_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "deep")
	c := New(dir)

	if err := c.Set("test-key", []byte("data")); err != nil {
		t.Fatalf("set: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test-key")); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}
