package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitOptions configures an in-memory fixed-window rate limiter.
type RateLimitOptions struct {
	Limit           int
	Window          time.Duration
	KeyFunc         func(*gin.Context) string
	OnLimitExceeded func(*gin.Context)
}

type fixedWindowRateLimiter struct {
	limit   int
	window  time.Duration
	mu      sync.Mutex
	clients map[string]fixedWindowCounter
}

type fixedWindowCounter struct {
	windowStart time.Time
	count       int
}

func newFixedWindowRateLimiter(limit int, window time.Duration) *fixedWindowRateLimiter {
	return &fixedWindowRateLimiter{
		limit:   limit,
		window:  window,
		clients: map[string]fixedWindowCounter{},
	}
}

func (l *fixedWindowRateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	counter := l.clients[key]
	if counter.windowStart.IsZero() || !now.Before(counter.windowStart.Add(l.window)) {
		l.clients[key] = fixedWindowCounter{
			windowStart: now,
			count:       1,
		}
		return true, 0
	}

	if counter.count >= l.limit {
		return false, counter.windowStart.Add(l.window).Sub(now)
	}

	counter.count++
	l.clients[key] = counter
	return true, 0
}

// RateLimitMiddleware limits requests by client key in a fixed time window.
func RateLimitMiddleware(opts RateLimitOptions) gin.HandlerFunc {
	if opts.Limit <= 0 || opts.Window <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	limiter := newFixedWindowRateLimiter(opts.Limit, opts.Window)
	keyFunc := opts.KeyFunc
	if keyFunc == nil {
		keyFunc = defaultRateLimitKey
	}

	return func(c *gin.Context) {
		allowed, retryAfter := limiter.allow(keyFunc(c), time.Now())
		if allowed {
			c.Next()
			return
		}

		c.Header("Retry-After", strconv.Itoa(retryAfterSeconds(retryAfter)))
		if opts.OnLimitExceeded != nil {
			opts.OnLimitExceeded(c)
			return
		}
		c.AbortWithStatus(http.StatusTooManyRequests)
	}
}

func defaultRateLimitKey(c *gin.Context) string {
	if ip := c.ClientIP(); ip != "" {
		return ip
	}
	if c.Request != nil {
		if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil && host != "" {
			return host
		}
		if c.Request.RemoteAddr != "" {
			return c.Request.RemoteAddr
		}
	}
	return "global"
}

func retryAfterSeconds(duration time.Duration) int {
	if duration <= 0 {
		return 1
	}
	seconds := int(math.Ceil(duration.Seconds()))
	if seconds < 1 {
		return 1
	}
	return seconds
}
