package cache

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type entry struct {
	val       any
	expiresAt time.Time
}

// Cache 是带 TTL 的进程内缓存。过期项仍保留并可作为 stale 数据兜底返回（GetOrLoad 用）。
type Cache struct {
	mu     sync.RWMutex
	data   map[string]entry
	hits   atomic.Int64
	misses atomic.Int64
}

// Stats 统计信息
type Stats struct {
	Hits, Misses int64
	Size         int
}

// New 创建空缓存。
func New() *Cache {
	return &Cache{data: make(map[string]entry)}
}

// Set 写入 key 并设置 TTL。重复 key 直接覆盖。
func (c *Cache) Set(key string, val any, ttl time.Duration) {
	c.mu.Lock()
	c.data[key] = entry{val: val, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

// Get 返回 (val, ok, stale)。
//
//	ok=false  → 完全没此 key
//	ok=true, stale=false → 命中且未过期
//	ok=true, stale=true  → 过期但保留作 fallback
func (c *Cache) Get(key string) (any, bool, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		c.misses.Add(1)
		return nil, false, false
	}
	c.hits.Add(1)
	stale := time.Now().After(e.expiresAt)
	return e.val, true, stale
}

// Delete 删除一个 key。
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
}

// DeletePrefix 删除所有以 prefix 开头的 key，返回删除数量。
func (c *Cache) DeletePrefix(prefix string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for k := range c.data {
		if strings.HasPrefix(k, prefix) {
			delete(c.data, k)
			n++
		}
	}
	return n
}

// Clear 清空所有缓存项。
func (c *Cache) Clear() {
	c.mu.Lock()
	c.data = make(map[string]entry)
	c.mu.Unlock()
}

// Stats 返回当前命中率统计。
func (c *Cache) Stats() Stats {
	c.mu.RLock()
	size := len(c.data)
	c.mu.RUnlock()
	return Stats{Hits: c.hits.Load(), Misses: c.misses.Load(), Size: size}
}
