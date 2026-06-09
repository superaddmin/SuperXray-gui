package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/web/global"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
	"github.com/robfig/cron/v3"
)

type openAPIDocument struct {
	OpenAPI    string                    `yaml:"openapi"`
	Paths      map[string]map[string]any `yaml:"paths"`
	Components openAPIComponents         `yaml:"components"`
	Security   []map[string][]string     `yaml:"security"`
}

type openAPIComponents struct {
	SecuritySchemes map[string]map[string]any `yaml:"securitySchemes"`
}

func TestPanelOpenAPIRoutesStayInSyncWithGoRoutes(t *testing.T) {
	doc := loadPanelOpenAPI(t)
	goRoutes := collectPanelAPIRoutes(t)

	for routeKey := range goRoutes {
		method, path := splitRouteKey(routeKey)
		operations, ok := doc.Paths[path]
		if !ok {
			t.Errorf("OpenAPI spec is missing path %s for Go route %s", path, routeKey)
			continue
		}
		if _, ok := operations[strings.ToLower(method)]; !ok {
			t.Errorf("OpenAPI spec path %s is missing method %s", path, method)
		}
	}
	for path, operations := range doc.Paths {
		for method := range operations {
			if !isOpenAPIMethod(method) {
				continue
			}
			routeKey := fmt.Sprintf("%s %s", strings.ToUpper(method), path)
			if _, ok := goRoutes[routeKey]; !ok {
				t.Errorf("OpenAPI spec documents stale route %s", routeKey)
			}
		}
	}
}

func TestPanelOpenAPISecurityContractMatchesMiddleware(t *testing.T) {
	doc := loadPanelOpenAPI(t)

	if doc.OpenAPI == "" {
		t.Fatal("OpenAPI spec must declare the openapi version")
	}
	if doc.Components.SecuritySchemes["cookieAuth"] == nil {
		t.Fatal("OpenAPI spec must declare cookieAuth for the SuperXray session cookie")
	}
	if doc.Components.SecuritySchemes["csrfToken"] == nil {
		t.Fatal("OpenAPI spec must declare csrfToken for X-CSRF-Token protected mutations")
	}

	for path, operations := range doc.Paths {
		if !strings.HasPrefix(path, "/panel/api/") {
			t.Fatalf("panel OpenAPI spec must only document /panel/api routes, got %s", path)
		}
		for method, operation := range operations {
			operationMap, ok := operation.(map[string]any)
			if !ok {
				t.Fatalf("operation %s %s must be an object", strings.ToUpper(method), path)
			}
			if !responsesInclude(operationMap, "404") {
				t.Errorf("operation %s %s must document the unauthenticated 404 contract", strings.ToUpper(method), path)
			}
			if strings.EqualFold(method, "post") && !operationSecurityIncludes(operationMap, "csrfToken") {
				t.Errorf("state-changing operation %s %s must require csrfToken", strings.ToUpper(method), path)
			}
		}
	}
}

func loadPanelOpenAPI(t *testing.T) openAPIDocument {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	specPath := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "docs", "openapi", "panel-api.yaml"))
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read panel OpenAPI spec at %s: %v", specPath, err)
	}

	var doc openAPIDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("failed to parse panel OpenAPI spec: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("panel OpenAPI spec must define paths")
	}
	return doc
}

func collectPanelAPIRoutes(t *testing.T) map[string]struct{} {
	t.Helper()

	routes := map[string]struct{}{
		// Registered from web.registerNewUIRoutes; keep it in this contract because
		// it is served under the same authenticated /panel/api surface.
		"GET /panel/api/openapi.json": {},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	scheduler := cron.New()
	previousWebServer := global.GetWebServer()
	global.SetWebServer(openAPITestWebServer{cron: scheduler})
	t.Cleanup(func() {
		scheduler.Stop()
		global.SetWebServer(previousWebServer)
	})

	NewAPIController(router.Group("/"), service.NewCustomGeoService())
	for _, route := range router.Routes() {
		if !strings.HasPrefix(route.Path, "/panel/api/") {
			continue
		}
		routes[fmt.Sprintf("%s %s", route.Method, ginPathToOpenAPI(route.Path))] = struct{}{}
	}
	return routes
}

type openAPITestWebServer struct {
	cron *cron.Cron
}

func (s openAPITestWebServer) GetCron() *cron.Cron {
	return s.cron
}

func (s openAPITestWebServer) GetCtx() context.Context {
	return context.Background()
}

func (s openAPITestWebServer) GetWSHub() any {
	return nil
}

func ginPathToOpenAPI(routePath string) string {
	routePath = strings.TrimSpace(routePath)
	if routePath == "/" {
		return "/"
	}
	segments := strings.Split(routePath, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			segments[i] = "{" + strings.TrimPrefix(segment, ":") + "}"
		}
	}
	return strings.Join(segments, "/")
}

func splitRouteKey(routeKey string) (method string, path string) {
	parts := strings.SplitN(routeKey, " ", 2)
	if len(parts) != 2 {
		return "", routeKey
	}
	return parts[0], parts[1]
}

func isOpenAPIMethod(method string) bool {
	switch strings.ToLower(method) {
	case "get", "post", "put", "delete", "patch", "head", "options", "trace":
		return true
	default:
		return false
	}
}

func responsesInclude(operation map[string]any, statusCode string) bool {
	responses, ok := operation["responses"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = responses[statusCode]
	return ok
}

func operationSecurityIncludes(operation map[string]any, scheme string) bool {
	security, ok := operation["security"].([]any)
	if !ok {
		return false
	}
	for _, item := range security {
		requirement, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if _, ok := requirement[scheme]; ok {
			return true
		}
	}
	return false
}
