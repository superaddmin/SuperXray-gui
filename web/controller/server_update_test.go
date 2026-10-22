package controller

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestServerRoutesExposePanelUpdateCheckAsGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller := &ServerController{}
	controller.initRouter(router.Group("/panel/api/server"))

	for _, route := range router.Routes() {
		if route.Method == "GET" && route.Path == "/panel/api/server/checkUpdate" {
			return
		}
	}
	t.Fatal("panel update check GET route is missing")
}
