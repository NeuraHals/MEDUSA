package cache

import (
	"sync"
	"time"
)

// MemoryCache is an in-process LRU-style cache for short-lived approval tokens.
// Used as a secondary idempotency layer when Redis is temporarily unavailable.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

func New() *MemoryCache {
	c := &MemoryCache{entries: make(map[string]cacheEntry)}
	go c.evictLoop()
	return c
}

func (c *MemoryCache) Set(key, value string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{value: value, expiresAt: time.Now().Add(ttl)}
}

func (c *MemoryCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.value, true
}

func (c *MemoryCache) Exists(key string) bool {
	_, ok := c.Get(key)
	return ok
}

// evictLoop removes expired entries every 30 seconds.
func (c *MemoryCache) evictLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}
