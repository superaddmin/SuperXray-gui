package controller

import (
	"context"

	panelcore "github.com/superaddmin/SuperXray-gui/v2/core"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"

	"github.com/gin-gonic/gin"
)

type coreAPIService interface {
	List(ctx context.Context) ([]panelcore.Instance, error)
	Get(ctx context.Context, id string) (panelcore.Instance, error)
	Status(ctx context.Context, id string) (panelcore.Status, error)
	Validate(ctx context.Context, id string) (panelcore.LifecycleResult, error)
	Start(ctx context.Context, id string) (panelcore.LifecycleResult, error)
	Stop(ctx context.Context, id string) (panelcore.LifecycleResult, error)
	Restart(ctx context.Context, id string) (panelcore.LifecycleResult, error)
}

type CoreController struct {
	BaseController
	coreService coreAPIService
}

func NewCoreController(g *gin.RouterGroup, services ...coreAPIService) *CoreController {
	a := &CoreController{coreService: service.NewCoreService()}
	if len(services) > 0 && services[0] != nil {
		a.coreService = services[0]
	}
	a.initRouter(g)
	return a
}

func (a *CoreController) initRouter(g *gin.RouterGroup) {
	g.GET("/instances", a.listInstances)
	g.GET("/instances/:id", a.getInstance)
	g.GET("/instances/:id/status", a.getStatus)
	g.POST("/instances/:id/validate", a.validate)
	g.POST("/instances/:id/start", a.start)
	g.POST("/instances/:id/stop", a.stop)
	g.POST("/instances/:id/restart", a.restart)
}

func (a *CoreController) listInstances(c *gin.Context) {
	instances, err := a.coreService.List(c.Request.Context())
	jsonMsgObj(c, "list core instances", instances, err)
}

func (a *CoreController) getInstance(c *gin.Context) {
	instance, err := a.coreService.Get(c.Request.Context(), c.Param("id"))
	jsonMsgObj(c, "get core instance", instance, err)
}

func (a *CoreController) getStatus(c *gin.Context) {
	status, err := a.coreService.Status(c.Request.Context(), c.Param("id"))
	jsonMsgObj(c, "get core status", status, err)
}

func (a *CoreController) validate(c *gin.Context) {
	result, err := a.coreService.Validate(c.Request.Context(), c.Param("id"))
	jsonMsgObj(c, "validate core instance", result, err)
}

func (a *CoreController) start(c *gin.Context) {
	result, err := a.coreService.Start(c.Request.Context(), c.Param("id"))
	jsonMsgObj(c, "start core instance", result, err)
}

func (a *CoreController) stop(c *gin.Context) {
	result, err := a.coreService.Stop(c.Request.Context(), c.Param("id"))
	jsonMsgObj(c, "stop core instance", result, err)
}

func (a *CoreController) restart(c *gin.Context) {
	result, err := a.coreService.Restart(c.Request.Context(), c.Param("id"))
	jsonMsgObj(c, "restart core instance", result, err)
}
