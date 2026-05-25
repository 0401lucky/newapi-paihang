package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func newIPLimiter(perMin int) *ipLimiter {
	burst := perMin / 4
	if burst < 1 {
		burst = 1
	}
	return &ipLimiter{
		limiters: map[string]*rate.Limiter{},
		rps:      rate.Limit(float64(perMin) / 60.0),
		burst:    burst,
	}
}

func (l *ipLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.limiters[ip]
	if !ok {
		lim = rate.NewLimiter(l.rps, l.burst)
		l.limiters[ip] = lim
	}
	return lim
}

func (l *ipLimiter) startGC() {
	go func() {
		t := time.NewTicker(10 * time.Minute)
		for range t.C {
			l.mu.Lock()
			for ip, lim := range l.limiters {
				if lim.Tokens() >= float64(l.burst-1) {
					delete(l.limiters, ip)
				}
			}
			l.mu.Unlock()
		}
	}()
}

func RateLimit(perMin int) gin.HandlerFunc {
	lim := newIPLimiter(perMin)
	lim.startGC()
	return func(c *gin.Context) {
		if !lim.get(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 429, "msg": "请求过于频繁，请稍后再试",
			})
			return
		}
		c.Next()
	}
}
