package cache

import (
	"time"

	"golang.org/x/sync/singleflight"
)

var sfGroup singleflight.Group

// GetOrLoad 是缓存的核心入口：
//  1. 缓存命中且未过期 → 直接返回，stale=false
//  2. 缓存命中但过期 → 调用 loader 刷新；loader 失败时返回旧数据（stale=true）
//  3. 缓存未命中 → singleflight 合并并发请求，调用 loader 一次
//
// 注意：本方法依赖 *Cache 的 Get 在过期时仍返回旧值的行为。
func (c *Cache) GetOrLoad(key string, ttl time.Duration, loader func() (any, error)) (any, bool, error) {
	if v, ok, stale := c.Get(key); ok && !stale {
		return v, false, nil
	}
	result, err, _ := sfGroup.Do(key, func() (any, error) {
		val, err := loader()
		if err != nil {
			// loader 失败，但如果有旧值就返回旧值
			if old, ok, _ := c.Get(key); ok {
				return staleResult{val: old}, nil
			}
			return nil, err
		}
		c.Set(key, val, ttl)
		return val, nil
	})
	if err != nil {
		return nil, false, err
	}
	if s, ok := result.(staleResult); ok {
		return s.val, true, nil
	}
	return result, false, nil
}

type staleResult struct{ val any }
