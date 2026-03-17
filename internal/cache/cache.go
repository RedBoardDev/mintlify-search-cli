package cache

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const defaultTTL = 5 * time.Minute

type Cache struct {
	dir string
	ttl time.Duration
}

func New(dir string) *Cache {
	return &Cache{
		dir: dir,
		ttl: defaultTTL,
	}
}

func Key(parts ...string) string {
	h := sha256.New()
	for i, p := range parts {
		if i > 0 {
			h.Write([]byte(":"))
		}
		h.Write([]byte(p))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Get returns nil (no error) on cache miss or expiry.
func (c *Cache) Get(key string) ([]byte, error) {
	path := c.path(key)

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if time.Since(info.ModTime()) > c.ttl {
		_ = os.Remove(path) // Best-effort cleanup of expired entry.
		return nil, nil
	}

	return os.ReadFile(path)
}

func (c *Cache) Set(key string, data []byte) error {
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(c.path(key), data, 0o600)
}

func (c *Cache) path(key string) string {
	return filepath.Join(c.dir, key)
}
