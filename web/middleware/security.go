package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"github.com/superaddmin/SuperXray-gui/v2/web/session"

	"github.com/gin-gonic/gin"
)

const CSPNonceContextKey = "csp_nonce"

// SecurityHeadersMiddleware sets baseline browser hardening headers.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce, err := generateCSPNonce()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"msg":     "Unable to generate CSP nonce",
			})
			return
		}

		c.Set(CSPNonceContextKey, nonce)
		header := c.Writer.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "same-origin")
		header.Set("Content-Security-Policy", buildSecurityHeaderCSP(nonce, c.Request.URL.Path))
		c.Next()
	}
}

// CSRFMiddleware rejects cross-site state-changing requests for cookie-auth APIs.
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		if hasValidCSRFToken(c) && (!hasOriginHeader(c) || hasSameOrigin(c)) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"msg":     "CSRF validation failed",
		})
	}
}

func hasValidCSRFToken(c *gin.Context) bool {
	token := c.GetHeader("X-CSRF-Token")
	if token == "" {
		token = c.GetHeader("X-XSRF-Token")
	}
	return session.VerifyCSRFToken(c, token)
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func CSPNonce(c *gin.Context) string {
	nonce, _ := c.Get(CSPNonceContextKey)
	value, _ := nonce.(string)
	return value
}

func buildSecurityHeaderCSP(nonce string, requestPath string) string {
	if isNewUIPanelPath(requestPath) {
		return buildNewUICSP(nonce)
	}
	return buildLegacyUICSP()
}

func buildLegacyUICSP() string {
	// Existing Vue 2 in-DOM templates still require unsafe-eval until they are precompiled.
	return "default-src 'self'; " +
		"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
		"script-src-attr 'none'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: blob:; " +
		"font-src 'self' data:; " +
		"connect-src 'self' ws: wss:; " +
		"object-src 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"frame-ancestors 'none'"
}

func buildNewUICSP(nonce string) string {
	return "default-src 'self'; " +
		"script-src 'self' 'nonce-" + nonce + "'; " +
		"script-src-attr 'none'; " +
		"style-src 'self' 'nonce-" + nonce + "'; " +
		"style-src-attr 'none'; " +
		"img-src 'self' data: blob:; " +
		"font-src 'self' data:; " +
		"connect-src 'self' ws: wss:; " +
		"object-src 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"frame-ancestors 'none'"
}

func isNewUIPanelPath(path string) bool {
	normalized := "/" + strings.Trim(path, "/")
	if normalized == "/panel" ||
		strings.HasSuffix(normalized, "/panel") ||
		normalized == "/panel/login" ||
		strings.HasSuffix(normalized, "/panel/login") ||
		strings.Contains(normalized, "/panel/assets/") ||
		strings.Contains(normalized, "/panel/ui/") ||
		normalized == "/panel/ui" ||
		strings.HasSuffix(normalized, "/panel/ui") {
		return !strings.Contains(normalized, "/panel/legacy")
	}

	for _, route := range []string{
		"/panel/dashboard",
		"/panel/logs",
		"/panel/xray",
		"/panel/inbounds",
		"/panel/settings",
	} {
		if normalized == route || strings.HasSuffix(normalized, route) {
			return true
		}
	}

	return false
}

func generateCSPNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(bytes), nil
}

func hasSameOrigin(c *gin.Context) bool {
	origin := c.GetHeader("Origin")
	if origin == "" {
		origin = c.GetHeader("Referer")
	}
	if origin == "" {
		return false
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}

	return strings.EqualFold(parsed.Scheme, requestScheme(c)) &&
		strings.EqualFold(parsed.Host, c.Request.Host)
}

func hasOriginHeader(c *gin.Context) bool {
	return c.GetHeader("Origin") != "" || c.GetHeader("Referer") != ""
}

func requestScheme(c *gin.Context) string {
	forwardedProto := c.GetHeader("X-Forwarded-Proto")
	if forwardedProto != "" {
		return strings.ToLower(strings.TrimSpace(strings.Split(forwardedProto, ",")[0]))
	}
	if c.Request.TLS != nil {
		return "https"
	}
	return "http"
}
