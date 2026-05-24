package cache

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetGet_Basic(t *testing.T) {
	c := New()
	c.Set("a", "hello", 1*time.Second)
	v, ok, stale := c.Get("a")
	require.True(t, ok)
	assert.Equal(t, "hello", v)
	assert.False(t, stale)
}

func TestGet_Miss(t *testing.T) {
	c := New()
	_, ok, _ := c.Get("missing")
	assert.False(t, ok)
}

func TestExpire_BecomesStale(t *testing.T) {
	c := New()
	c.Set("a", 123, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	v, ok, stale := c.Get("a")
	require.True(t, ok, "过期数据仍可作为 stale 返回")
	assert.Equal(t, 123, v)
	assert.True(t, stale)
}

func TestDelete(t *testing.T) {
	c := New()
	c.Set("a", "x", time.Second)
	c.Delete("a")
	_, ok, _ := c.Get("a")
	assert.False(t, ok)
}

func TestDeletePrefix(t *testing.T) {
	c := New()
	c.Set("lb:rich:p1", 1, time.Second)
	c.Set("lb:rich:p2", 2, time.Second)
	c.Set("lb:foodie:p1", 3, time.Second)
	n := c.DeletePrefix("lb:rich:")
	assert.Equal(t, 2, n)
	_, ok, _ := c.Get("lb:rich:p1")
	assert.False(t, ok)
	_, ok, _ = c.Get("lb:foodie:p1")
	assert.True(t, ok)
}

func TestClear(t *testing.T) {
	c := New()
	c.Set("a", 1, time.Second)
	c.Set("b", 2, time.Second)
	c.Clear()
	_, ok, _ := c.Get("a")
	assert.False(t, ok)
}

func TestStats(t *testing.T) {
	c := New()
	c.Set("a", 1, time.Second)
	_, _, _ = c.Get("a")    // hit
	_, _, _ = c.Get("miss") // miss
	s := c.Stats()
	assert.EqualValues(t, 1, s.Hits)
	assert.EqualValues(t, 1, s.Misses)
}

func TestConcurrentSafety(t *testing.T) {
	c := New()
	var wg sync.WaitGroup
	var done int32
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				c.Set("k", j, time.Second)
				_, _, _ = c.Get("k")
			}
			atomic.AddInt32(&done, 1)
		}(i)
	}
	wg.Wait()
	assert.EqualValues(t, 50, done)
}
