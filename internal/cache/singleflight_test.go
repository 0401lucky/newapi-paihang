package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrLoad_FirstCallFetches(t *testing.T) {
	c := New()
	calls := atomic.Int32{}
	v, stale, err := c.GetOrLoad("k", time.Second, func() (any, error) {
		calls.Add(1)
		return 42, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 42, v)
	assert.False(t, stale)
	assert.EqualValues(t, 1, calls.Load())
}

func TestGetOrLoad_CacheHitSkipsLoader(t *testing.T) {
	c := New()
	c.Set("k", "cached", time.Second)
	calls := atomic.Int32{}
	v, _, err := c.GetOrLoad("k", time.Second, func() (any, error) {
		calls.Add(1)
		return "loaded", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "cached", v)
	assert.EqualValues(t, 0, calls.Load())
}

func TestGetOrLoad_Singleflight(t *testing.T) {
	c := New()
	calls := atomic.Int32{}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = c.GetOrLoad("k", time.Second, func() (any, error) {
				calls.Add(1)
				time.Sleep(50 * time.Millisecond)
				return "ok", nil
			})
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, calls.Load(), "20 个并发只触发 1 次 loader")
}

func TestGetOrLoad_StaleFallback(t *testing.T) {
	c := New()
	c.Set("k", "old", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	v, stale, err := c.GetOrLoad("k", time.Second, func() (any, error) {
		return nil, errors.New("db down")
	})
	require.NoError(t, err, "loader 失败但有 stale 数据时不报错")
	assert.Equal(t, "old", v)
	assert.True(t, stale)
}

func TestGetOrLoad_LoaderErrorNoStale(t *testing.T) {
	c := New()
	_, _, err := c.GetOrLoad("k", time.Second, func() (any, error) {
		return nil, errors.New("boom")
	})
	require.Error(t, err)
}
