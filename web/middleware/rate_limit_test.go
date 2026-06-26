package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimitMiddlewareRejectsAfterLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware(RateLimitOptions{
		Limit:  1,
		Window: time.Minute,
	}))
	router.GET("/limited", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "http://example.test/limited", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusNoContent)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "http://example.test/limited", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header must be set on rate-limited responses")
	}
}

func TestFixedWindowRateLimiterResetsAfterWindow(t *testing.T) {
	limiter := newFixedWindowRateLimiter(1, time.Second)
	now := time.Unix(100, 0)

	allowed, retryAfter := limiter.allow("client-a", now)
	if !allowed {
		t.Fatalf("first request allowed = false, retryAfter = %s", retryAfter)
	}

	allowed, retryAfter = limiter.allow("client-a", now.Add(500*time.Millisecond))
	if allowed {
		t.Fatal("second request allowed = true, want false")
	}
	if retryAfter <= 0 {
		t.Fatalf("retryAfter = %s, want positive duration", retryAfter)
	}

	allowed, retryAfter = limiter.allow("client-a", now.Add(time.Second))
	if !allowed {
		t.Fatalf("request after window allowed = false, retryAfter = %s", retryAfter)
	}
}
