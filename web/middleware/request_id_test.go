package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddlewarePropagatesIncomingRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/", func(c *gin.Context) {
		if got := RequestID(c); got != "req-test-123" {
			t.Fatalf("RequestID() = %q, want req-test-123", got)
		}
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set(RequestIDHeader, "req-test-123")
	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get(RequestIDHeader); got != "req-test-123" {
		t.Fatalf("%s = %q, want req-test-123", RequestIDHeader, got)
	}
}

func TestRequestIDMiddlewareGeneratesMissingRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/", func(c *gin.Context) {
		if got := RequestID(c); got == "" {
			t.Fatal("RequestID() must not be empty")
		}
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get(RequestIDHeader); got == "" {
		t.Fatalf("%s header must not be empty", RequestIDHeader)
	}
}
