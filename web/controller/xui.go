package controller

import (
	"net/http"

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

	// Phase 10 keeps the old HTML UI as an explicit rollback and compatibility boundary.
	g.GET("/panel/legacy", a.checkLogin, func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path")+"panel/legacy/")
	})

	legacy := g.Group("/panel/legacy")
	legacy.Use(a.checkLogin)

	legacy.GET("/", a.index)
	legacy.GET("/inbounds", a.inbounds)
	legacy.GET("/settings", a.settings)
	legacy.GET("/xray", a.xraySettings)
}

// index renders the main panel index page.
func (a *XUIController) index(c *gin.Context) {
	html(c, "index.html", "pages.index.title", nil)
}

// inbounds renders the inbounds management page.
func (a *XUIController) inbounds(c *gin.Context) {
	html(c, "inbounds.html", "pages.inbounds.title", nil)
}

// settings renders the settings management page.
func (a *XUIController) settings(c *gin.Context) {
	html(c, "settings.html", "pages.settings.title", nil)
}

// xraySettings renders the Xray settings page.
func (a *XUIController) xraySettings(c *gin.Context) {
	html(c, "xray.html", "pages.xray.title", nil)
}
