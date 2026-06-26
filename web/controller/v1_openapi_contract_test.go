package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

type v1OpenAPIDocument struct {
	OpenAPI    string                    `yaml:"openapi"`
	Paths      map[string]map[string]any `yaml:"paths"`
	Components openAPIComponents         `yaml:"components"`
}

func TestV1OpenAPIRoutesStayInSyncWithGoRoutes(t *testing.T) {
	doc := loadV1OpenAPI(t)
	goRoutes := collectV1APIRoutes()

	for routeKey := range goRoutes {
		method, path := splitRouteKey(routeKey)
		operations, ok := doc.Paths[path]
		if !ok {
			t.Errorf("V1 OpenAPI spec is missing path %s for Go route %s", path, routeKey)
			continue
		}
		if _, ok := operations[strings.ToLower(method)]; !ok {
			t.Errorf("V1 OpenAPI spec path %s is missing method %s", path, method)
		}
	}
	for path, operations := range doc.Paths {
		for method := range operations {
			if !isOpenAPIMethod(method) {
				continue
			}
			routeKey := fmt.Sprintf("%s %s", strings.ToUpper(method), path)
			if _, ok := goRoutes[routeKey]; !ok {
				t.Errorf("V1 OpenAPI spec documents stale route %s", routeKey)
			}
		}
	}
}

func TestV1OpenAPIResponseContract(t *testing.T) {
	doc := loadV1OpenAPI(t)

	if doc.OpenAPI == "" {
		t.Fatal("V1 OpenAPI spec must declare the openapi version")
	}
	if doc.Components.SecuritySchemes != nil {
		t.Fatal("public V1 health endpoints must not declare legacy cookie security schemes")
	}

	for path, operations := range doc.Paths {
		if !strings.HasPrefix(path, "/api/v1/") {
			t.Fatalf("V1 OpenAPI spec must only document /api/v1 routes, got %s", path)
		}
		for method, operation := range operations {
			operationMap, ok := operation.(map[string]any)
			if !ok {
				t.Fatalf("operation %s %s must be an object", strings.ToUpper(method), path)
			}
			if !responsesInclude(operationMap, "200") {
				t.Errorf("operation %s %s must document 200 success envelope", strings.ToUpper(method), path)
			}
			if !responsesInclude(operationMap, "404") {
				t.Errorf("operation %s %s must document unified 404 error envelope", strings.ToUpper(method), path)
			}
			if !responsesInclude(operationMap, "429") {
				t.Errorf("operation %s %s must document unified 429 rate-limit envelope", strings.ToUpper(method), path)
			}
		}
	}
}

func TestV1OpenAPIIncludesMetricsEndpoint(t *testing.T) {
	doc := loadV1OpenAPI(t)
	operations, ok := doc.Paths["/api/v1/metrics"]
	if !ok {
		t.Fatal("V1 OpenAPI spec must document /api/v1/metrics")
	}
	if _, ok := operations["get"]; !ok {
		t.Fatal("V1 OpenAPI spec must document GET /api/v1/metrics")
	}
}

func loadV1OpenAPI(t *testing.T) v1OpenAPIDocument {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	specPath := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "docs", "openapi", "api-v1.yaml"))
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read V1 OpenAPI spec at %s: %v", specPath, err)
	}

	var doc v1OpenAPIDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("failed to parse V1 OpenAPI spec: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("V1 OpenAPI spec must define paths")
	}
	return doc
}

func collectV1APIRoutes() map[string]struct{} {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewV1Controller(router.Group("/"), fakeV1HealthService{})

	routes := map[string]struct{}{}
	for _, route := range router.Routes() {
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			continue
		}
		routes[fmt.Sprintf("%s %s", route.Method, ginPathToOpenAPI(route.Path))] = struct{}{}
	}
	return routes
}
