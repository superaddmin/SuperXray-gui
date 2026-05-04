package web

import (
	"bytes"
	"embed"
	"encoding/json"
	"html"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/superaddmin/SuperXray-gui/v2/config"
	"github.com/superaddmin/SuperXray-gui/v2/web/middleware"
	"github.com/superaddmin/SuperXray-gui/v2/web/session"

	"github.com/gin-gonic/gin"
)

//go:embed ui
var newUIFS embed.FS

type newUIRuntimeConfig struct {
	APIBasePath string `json:"apiBasePath"`
	BasePath    string `json:"basePath"`
	CSPNonce    string `json:"cspNonce"`
	CSRFToken   string `json:"csrfToken"`
	UIBasePath  string `json:"uiBasePath"`
	Version     string `json:"version"`
}

func (s *Server) registerNewUIRoutes(g *gin.RouterGroup, basePath string) {
	auth := newUICheckLogin()

	g.GET("/panel/login", s.serveNewUIIndex(basePath, basePath+"panel/"))
	g.GET("/panel/assets/*path", s.serveNewUIAsset())

	g.GET("/panel", auth, func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, basePath+"panel/")
	})
	g.GET("/panel/", auth, s.serveNewUIIndex(basePath, basePath+"panel/"))
	for _, route := range []string{
		"/panel/dashboard",
		"/panel/logs",
		"/panel/xray",
		"/panel/inbounds",
		"/panel/settings",
	} {
		g.GET(route, auth, s.serveNewUIIndex(basePath, basePath+"panel/"))
	}

	g.GET("/panel/ui", auth, func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, basePath+"panel/ui/")
	})

	ui := g.Group("/panel/ui")
	ui.Use(auth)
	ui.GET("/*path", s.serveNewUI(basePath, basePath+"panel/ui/"))
}

func newUICheckLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if session.IsLogin(c) {
			c.Next()
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path")+"panel/login")
		c.Abort()
	}
}

func (s *Server) serveNewUI(basePath string, uiBasePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestedPath, ok := cleanNewUIPath(c.Param("path"))
		if !ok {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if requestedPath != "" && serveNewUIStaticFile(c, requestedPath) {
			return
		}

		s.serveNewUIIndex(basePath, uiBasePath)(c)
	}
}

func (s *Server) serveNewUIIndex(basePath string, uiBasePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		indexHTML, err := newUIFS.ReadFile("ui/index.html")
		if err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		runtime := newUIRuntimeConfig{
			APIBasePath: basePath,
			BasePath:    basePath,
			CSPNonce:    middleware.CSPNonce(c),
			CSRFToken:   session.EnsureCSRFToken(c),
			UIBasePath:  uiBasePath,
			Version:     config.GetAssetVersion(),
		}

		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "text/html; charset=utf-8", injectNewUIRuntimeConfig(indexHTML, runtime))
	}
}

func (s *Server) serveNewUIAsset() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestedPath, ok := cleanNewUIPath(c.Param("path"))
		if !ok || requestedPath == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if serveNewUIStaticFile(c, "assets/"+requestedPath) {
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
	}
}

func serveNewUIStaticFile(c *gin.Context, requestedPath string) bool {
	file, err := newUIFS.Open("ui/" + requestedPath)
	if err != nil {
		return false
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return false
	}

	data, err := fs.ReadFile(newUIFS, "ui/"+requestedPath)
	if err != nil {
		return false
	}

	if strings.HasPrefix(requestedPath, "assets/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "no-cache")
	}

	c.Data(http.StatusOK, detectNewUIContentType(requestedPath, data), data)
	return true
}

func injectNewUIRuntimeConfig(indexHTML []byte, runtime newUIRuntimeConfig) []byte {
	runtimeJSON, err := json.Marshal(runtime)
	if err != nil {
		runtimeJSON = []byte("{}")
	}

	baseHref := html.EscapeString(runtime.UIBasePath)
	nonce := html.EscapeString(runtime.CSPNonce)
	nonceJSON, err := json.Marshal(runtime.CSPNonce)
	if err != nil {
		nonceJSON = []byte(`""`)
	}
	bootstrap := `window.__SUPERXRAY_UI_CONFIG__=` + string(runtimeJSON) + `;` +
		`(function(n){if(!n||typeof document==="undefined"||document.__superxrayStyleNoncePatched)return;document.__superxrayStyleNoncePatched=true;var createElement=document.createElement.bind(document);document.createElement=function(tagName,options){var el=createElement(tagName,options);if(String(tagName).toLowerCase()==="style"){el.setAttribute("nonce",n)}return el};})(` + string(nonceJSON) + `);`
	injection := []byte(`<base href="` + baseHref + `">` +
		`<script nonce="` + nonce + `">` + bootstrap + `</script>`)

	if bytes.Contains(indexHTML, []byte("<head>")) {
		return bytes.Replace(indexHTML, []byte("<head>"), append([]byte("<head>"), injection...), 1)
	}

	return append(injection, indexHTML...)
}

func cleanNewUIPath(rawPath string) (string, bool) {
	trimmed := strings.TrimPrefix(rawPath, "/")
	if trimmed == "" {
		return "", true
	}
	if strings.Contains(trimmed, "\\") {
		return "", false
	}

	cleaned := path.Clean(trimmed)
	if cleaned == "." {
		return "", true
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}

	return cleaned, true
}

func detectNewUIContentType(filePath string, data []byte) string {
	contentType := mime.TypeByExtension(path.Ext(filePath))
	if contentType != "" {
		return contentType
	}
	return http.DetectContentType(data)
}
