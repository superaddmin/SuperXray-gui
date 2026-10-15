package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/superaddmin/SuperXray-gui/v2/util/trustedproxy"
)

func TestRateLimitCannotBeBypassedWithUntrustedForwardedIP(t *testing.T) {
	t.Setenv(trustedproxy.EnvironmentVariable, "127.0.0.0/8")
	router := gin.New()
	if err := trustedproxy.Configure(router); err != nil {
		t.Fatal(err)
	}
	router.Use(RateLimitMiddleware(RateLimitOptions{Limit: 1, Window: time.Minute}))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	for index, forwarded := range []string{"203.0.113.1", "203.0.113.2"} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.RemoteAddr = "198.51.100.1:1234"
		request.Header.Set("X-Forwarded-For", forwarded)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		want := http.StatusOK
		if index > 0 {
			want = http.StatusTooManyRequests
		}
		if recorder.Code != want {
			t.Fatalf("request %d status = %d, want %d", index, recorder.Code, want)
		}
	}
}

func TestSameOriginDoesNotTrustDirectForwardedProtocol(t *testing.T) {
	t.Setenv(trustedproxy.EnvironmentVariable, "127.0.0.0/8")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "http://panel.example/", nil)
	c.Request.RemoteAddr = "198.51.100.1:1234"
	c.Request.Header.Set("Origin", "https://panel.example")
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	if hasSameOrigin(c) {
		t.Fatal("untrusted proxy header changed CSRF origin comparison")
	}
	c.Request.RemoteAddr = "127.0.0.1:1234"
	if !hasSameOrigin(c) {
		t.Fatal("trusted TLS-terminating proxy origin rejected")
	}
}
