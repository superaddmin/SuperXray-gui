package controller

import (
	"github.com/superaddmin/SuperXray-gui/v2/web/session"

	"github.com/gin-gonic/gin"
)

// XUIController is the main controller for the X-UI panel, managing sub-controllers.
type XUIController struct {
	BaseController

	settingController     *SettingController
	xraySettingController *XraySettingController
}

// NewXUIController creates a new XUIController and initializes its routes.
func NewXUIController(g *gin.RouterGroup) *XUIController {
	a := &XUIController{}
	a.initRouter(g)
	return a
}

// initRouter sets up the main panel routes and initializes sub-controllers.
func (a *XUIController) initRouter(g *gin.RouterGroup) {
	panel := g.Group("/panel")
	panel.Use(a.checkLogin)

	a.settingController = NewSettingController(panel)
	a.xraySettingController = NewXraySettingController(panel)

	// SPA pages can fetch the current session CSRF token when they do not have a
	// server-rendered token available. New Vue UI normally receives it through
	// runtime config; this endpoint keeps parity with upstream SPA behavior.
	panel.GET("/csrf-token", a.csrfToken)
}

// csrfToken returns the session CSRF token to authenticated SPA clients.
func (a *XUIController) csrfToken(c *gin.Context) {
	jsonObj(c, session.EnsureCSRFToken(c), nil)
}
