package trunking

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Cache persists the last-known control-channel frequency per system to a
// JSON file. The hunter consults this cache on startup so it can re-tune to
// a known-good CC before scanning the full frequency list.
type Cache struct {
	path string
	mu   sync.RWMutex
	data cacheFile
}

type cacheFile struct {
	Version int                     `json:"version"`
	Systems map[string]CachedSystem `json:"systems"`
}

// CachedSystem is the on-disk record for one system.
type CachedSystem struct {
	LastFrequencyHz uint32    `json:"last_frequency_hz"`
	LastLockAt      time.Time `json:"last_lock_at,omitempty"`
	NAC             uint16    `json:"nac,omitempty"`
}

// OpenCache loads (or creates) a cache file at path. A non-existent file is
// treated as an empty cache.
func OpenCache(path string) (*Cache, error) {
	c := &Cache{path: path, data: cacheFile{Version: 1, Systems: map[string]CachedSystem{}}}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, fmt.Errorf("trunking/cache: read %s: %w", path, err)
	}
	if err := json.Unmarshal(b, &c.data); err != nil {
		return nil, fmt.Errorf("trunking/cache: parse %s: %w", path, err)
	}
	if c.data.Systems == nil {
		c.data.Systems = map[string]CachedSystem{}
	}
	return c, nil
}

// Get returns the cached entry for the named system, if any.
func (c *Cache) Get(name string) (CachedSystem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data.Systems[name]
	return v, ok
}

// Set updates the cached entry for name and persists the file. The write is
// atomic via rename.
func (c *Cache) Set(name string, entry CachedSystem) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data.Systems[name] = entry
	return c.flush()
}

// Names returns the system names currently in the cache, sorted for
// deterministic iteration.
func (c *Cache) Names() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, 0, len(c.data.Systems))
	for k := range c.data.Systems {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (c *Cache) flush() error {
	if c.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("trunking/cache: mkdir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(c.path), filepath.Base(c.path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("trunking/cache: create tmp: %w", err)
	}
	defer os.Remove(tmp.Name())
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c.data); err != nil {
		tmp.Close()
		return fmt.Errorf("trunking/cache: encode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("trunking/cache: close tmp: %w", err)
	}
	if err := os.Rename(tmp.Name(), c.path); err != nil {
		return fmt.Errorf("trunking/cache: rename: %w", err)
	}
	return nil
}
