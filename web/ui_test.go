package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/web/middleware"
	"github.com/superaddmin/SuperXray-gui/v2/web/session"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestInjectNewUIRuntimeConfig(t *testing.T) {
	html := []byte("<!doctype html><html><head><title>SuperXray</title></head><body></body></html>")
	runtime := newUIRuntimeConfig{
		APIBasePath: "/base/",
		BasePath:    "/base/",
		CSPNonce:    "nonce-123",
		CSRFToken:   "csrf-123",
		UIBasePath:  "/base/panel/ui/",
		Version:     "test-version",
	}

	got := string(injectNewUIRuntimeConfig(html, runtime))

	if !strings.Contains(got, `<base href="/base/panel/ui/">`) {
		t.Fatalf("expected base href injection, got %s", got)
	}
	if !strings.Contains(got, `<script nonce="nonce-123">`) {
		t.Fatalf("expected CSP nonce injection, got %s", got)
	}
	if !strings.Contains(got, `"apiBasePath":"/base/"`) {
		t.Fatalf("expected runtime config injection, got %s", got)
	}
	if !strings.Contains(got, `"csrfToken":"csrf-123"`) {
		t.Fatalf("expected CSRF token injection, got %s", got)
	}
	if !strings.Contains(got, `document.__superxrayStyleNoncePatched=true`) {
		t.Fatalf("expected dynamic style nonce bootstrap, got %s", got)
	}
	if !strings.Contains(got, `el.setAttribute("nonce",n)`) {
		t.Fatalf("expected style nonce assignment, got %s", got)
	}
}

func TestCleanNewUIPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
		ok   bool
	}{
		{name: "empty", path: "", want: "", ok: true},
		{name: "asset", path: "/assets/app.js", want: "assets/app.js", ok: true},
		{name: "nested route", path: "/dashboard/overview", want: "dashboard/overview", ok: true},
		{name: "traversal", path: "/../config.json", ok: false},
		{name: "windows separator", path: "/assets\\app.js", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := cleanNewUIPath(tt.path)
			if ok != tt.ok {
				t.Fatalf("cleanNewUIPath() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("cleanNewUIPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewUIRoutesRedirectAnonymousUserToNewLogin(t *testing.T) {
	router := newTestNewUIRouter(false)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTemporaryRedirect)
	}
	if got := recorder.Header().Get("Location"); got != "/panel/login" {
		t.Fatalf("Location = %q, want /panel/login", got)
	}
}

func TestNewUIRoutesServeLoginForAnonymousUser(t *testing.T) {
	router := newTestNewUIRouter(false)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/login", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `window.__SUPERXRAY_UI_CONFIG__=`) {
		t.Fatalf("expected runtime config injection, got %s", body)
	}
	if !strings.Contains(body, `"uiBasePath":"/panel/"`) {
		t.Fatalf("expected login page to use default UI base path, got %s", body)
	}
}

func TestNewUIRoutesServeInjectedIndexForLoggedInUser(t *testing.T) {
	router := newTestNewUIRouter(true)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `window.__SUPERXRAY_UI_CONFIG__=`) {
		t.Fatalf("expected runtime config injection, got %s", body)
	}
	if !strings.Contains(body, `"uiBasePath":"/panel/"`) {
		t.Fatalf("expected UI base path in runtime config, got %s", body)
	}
	if !strings.Contains(body, `"csrfToken":"`) {
		t.Fatalf("expected CSRF token in runtime config, got %s", body)
	}
}

func TestNewUIRoutesKeepCompatibilityEntryForLoggedInUser(t *testing.T) {
	router := newTestNewUIRouter(true)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/ui/", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"uiBasePath":"/panel/ui/"`) {
		t.Fatalf("expected compatibility UI base path in runtime config, got %s", body)
	}
}

func TestNewUIRoutesServeImmutableAssetsForLoggedInUser(t *testing.T) {
	assetPath := firstNewUIAsset(t)
	router := newTestNewUIRouter(true)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/"+assetPath, nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want immutable asset cache", got)
	}
	if strings.Contains(recorder.Body.String(), `window.__SUPERXRAY_UI_CONFIG__=`) {
		t.Fatal("asset response unexpectedly returned injected index HTML")
	}
}

func TestNewUIRoutesServeImmutableAssetsForAnonymousUser(t *testing.T) {
	assetPath := firstNewUIAsset(t)
	router := newTestNewUIRouter(false)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/"+assetPath, nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want immutable asset cache", got)
	}
}

func newTestNewUIRouter(authenticated bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(sessions.Sessions("SuperXray", cookie.NewStore([]byte("secret"))))
	router.Use(func(c *gin.Context) {
		c.Set("base_path", "/")
		if authenticated {
			session.SetLoginUser(c, &model.User{Id: 1, Username: "admin"})
		}
		c.Next()
	})

	(&Server{}).registerNewUIRoutes(router.Group("/"), "/")
	return router
}

func firstNewUIAsset(t *testing.T) string {
	t.Helper()

	entries, err := fs.ReadDir(newUIFS, "ui/assets")
	if err != nil {
		t.Fatalf("read embedded UI assets: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return "assets/" + entry.Name()
		}
	}
	t.Fatal("embedded UI assets are missing")
	return ""
}
