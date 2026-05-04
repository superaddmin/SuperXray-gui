package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/web/session"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
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
	csp := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src-attr 'none'") {
		t.Fatalf("Content-Security-Policy script-src-attr missing: %q", csp)
	}
}

func TestSecurityHeadersMiddlewareKeepsLegacyCSPForOldUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/panel/legacy/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/legacy/", nil)
	router.ServeHTTP(recorder, req)

	csp := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(cspDirective(csp, "script-src"), "'unsafe-eval'") {
		t.Fatalf("legacy UI CSP must keep unsafe-eval until Vue 2 templates are gone: %q", csp)
	}
	if !strings.Contains(cspDirective(csp, "script-src"), "'unsafe-inline'") {
		t.Fatalf("legacy UI CSP must keep unsafe-inline until inline scripts are gone: %q", csp)
	}
	if strings.Contains(cspDirective(csp, "script-src"), "'nonce-") {
		t.Fatalf("legacy UI CSP must not mix nonce with unsafe-inline: %q", csp)
	}
	if !strings.Contains(cspDirective(csp, "style-src"), "'unsafe-inline'") {
		t.Fatalf("legacy UI CSP must keep unsafe-inline until inline styles are gone: %q", csp)
	}
}

func TestSecurityHeadersMiddlewareUsesStrictCSPForNewUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/panel/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/", nil)
	router.ServeHTTP(recorder, req)

	csp := recorder.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header is missing")
	}
	if strings.Contains(cspDirective(csp, "script-src"), "'unsafe-eval'") {
		t.Fatalf("new UI CSP must not allow unsafe-eval: %q", csp)
	}
	if strings.Contains(cspDirective(csp, "script-src"), "'unsafe-inline'") {
		t.Fatalf("new UI script-src must not allow unsafe-inline: %q", csp)
	}
	if !strings.Contains(cspDirective(csp, "script-src"), "'nonce-") {
		t.Fatalf("new UI script-src nonce missing: %q", csp)
	}
	styleSrc := cspDirective(csp, "style-src")
	if strings.Contains(styleSrc, "'unsafe-inline'") {
		t.Fatalf("new UI style-src must not allow unsafe-inline: %q", csp)
	}
	if !strings.Contains(styleSrc, "'nonce-") {
		t.Fatalf("new UI style-src nonce missing: %q", csp)
	}
	styleSrcAttr := cspDirective(csp, "style-src-attr")
	if strings.Contains(styleSrcAttr, "'unsafe-inline'") {
		t.Fatalf("new UI style-src-attr must not allow unsafe-inline: %q", csp)
	}
	if styleSrcAttr != "style-src-attr 'none'" {
		t.Fatalf("new UI style-src-attr must block inline style attributes: %q", csp)
	}
}

func TestSecurityHeadersMiddlewareUsesStrictCSPForNewUILogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/panel/login", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/login", nil)
	router.ServeHTTP(recorder, req)

	csp := recorder.Header().Get("Content-Security-Policy")
	if strings.Contains(cspDirective(csp, "script-src"), "'unsafe-eval'") {
		t.Fatalf("new UI login CSP must not allow unsafe-eval: %q", csp)
	}
	if strings.Contains(cspDirective(csp, "script-src"), "'unsafe-inline'") {
		t.Fatalf("new UI login script-src must not allow unsafe-inline: %q", csp)
	}
	if !strings.Contains(cspDirective(csp, "script-src"), "'nonce-") {
		t.Fatalf("new UI login script-src nonce missing: %q", csp)
	}
	if strings.Contains(cspDirective(csp, "style-src"), "'unsafe-inline'") {
		t.Fatalf("new UI login style-src must not allow unsafe-inline: %q", csp)
	}
}

func TestSecurityHeadersMiddlewareKeepsCompatibilityNewUIStrict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/panel/ui/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/ui/", nil)
	router.ServeHTTP(recorder, req)

	csp := recorder.Header().Get("Content-Security-Policy")
	if strings.Contains(cspDirective(csp, "script-src"), "'unsafe-eval'") {
		t.Fatalf("compatibility new UI CSP must not allow unsafe-eval: %q", csp)
	}
}

func TestSecurityHeadersMiddlewareRecognizesNewUIUnderBasePath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/random-base/panel/ui/assets/index.js", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/random-base/panel/ui/assets/index.js", nil)
	router.ServeHTTP(recorder, req)

	csp := recorder.Header().Get("Content-Security-Policy")
	if strings.Contains(cspDirective(csp, "script-src"), "'unsafe-eval'") {
		t.Fatalf("new UI assets under base path must use strict CSP: %q", csp)
	}
}

func TestCSRFMiddlewareRejectsUnsafeRequestWithoutSameOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newCSRFTestRouter()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/mutate", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddlewareRejectsAjaxHeaderWithoutSameOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newCSRFTestRouter()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/mutate", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddlewareRejectsSameOriginUnsafeRequestWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newCSRFTestRouter()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/mutate", nil)
	req.Header.Set("Origin", "http://example.test")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddlewareAllowsValidCSRFTokenUnsafeRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newCSRFTestRouter()
	token, cookies := issueCSRFToken(t, router)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/mutate", nil)
	req.Header.Set("Origin", "http://example.test")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, cookies)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestCSRFMiddlewareRejectsInvalidCSRFToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newCSRFTestRouter()
	_, cookies := issueCSRFToken(t, router)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/mutate", nil)
	req.Header.Set("Origin", "http://example.test")
	req.Header.Set("X-CSRF-Token", "invalid-token")
	addCookies(req, cookies)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddlewareRejectsCrossOriginUnsafeRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newCSRFTestRouter()
	token, cookies := issueCSRFToken(t, router)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/mutate", nil)
	req.Header.Set("Origin", "http://evil.test")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, cookies)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddlewareRejectsSchemeMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newCSRFTestRouter()
	token, cookies := issueCSRFToken(t, router)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/mutate", nil)
	req.Header.Set("Origin", "https://example.test")
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, cookies)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func newCSRFTestRouter() *gin.Engine {
	router := gin.New()
	router.Use(sessions.Sessions("SuperXray", cookie.NewStore([]byte("csrf-test-secret"))))
	router.Use(CSRFMiddleware())
	router.GET("/csrf-token", func(c *gin.Context) {
		c.String(http.StatusOK, session.EnsureCSRFToken(c))
	})
	router.POST("/mutate", func(c *gin.Context) {
		c.String(http.StatusOK, "mutated")
	})
	return router
}

func issueCSRFToken(t *testing.T, router *gin.Engine) (string, []*http.Cookie) {
	t.Helper()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/csrf-token", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("token status = %d, want %d", recorder.Code, http.StatusOK)
	}
	token := strings.TrimSpace(recorder.Body.String())
	if token == "" {
		t.Fatal("issued CSRF token is empty")
	}
	return token, recorder.Result().Cookies()
}

func addCookies(req *http.Request, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
}

func cspDirective(csp, name string) string {
	for _, directive := range strings.Split(csp, ";") {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, name+" ") {
			return directive
		}
	}
	return ""
}
