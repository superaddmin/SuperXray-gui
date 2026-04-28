package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersMiddlewareSetsBaselineHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := recorder.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", got)
	}
	if got := recorder.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("Content-Security-Policy header is missing")
	}
}

func TestCSRFMiddlewareRejectsUnsafeRequestWithoutAjaxHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFMiddleware())
	router.POST("/mutate", func(c *gin.Context) {
		c.String(http.StatusOK, "mutated")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/mutate", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddlewareAllowsAjaxUnsafeRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFMiddleware())
	router.POST("/mutate", func(c *gin.Context) {
		c.String(http.StatusOK, "mutated")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/mutate", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}
