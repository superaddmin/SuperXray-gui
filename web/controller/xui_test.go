package controller

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestXUIControllerDoesNotRegisterLegacyPanelRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	NewXUIController(router.Group("/"))

	for _, route := range router.Routes() {
		if strings.Contains(route.Path, "/panel/legacy") {
			t.Fatalf("legacy panel route must be retired, found %s %s", route.Method, route.Path)
		}
	}
}
