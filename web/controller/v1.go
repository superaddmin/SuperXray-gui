package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/web/entity"
	"github.com/superaddmin/SuperXray-gui/v2/web/middleware"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"

	"github.com/gin-gonic/gin"
)

type v1HealthService interface {
	Live(context.Context) service.HealthReport
	Ready(context.Context) service.HealthReport
}

type v1MetricsService interface {
	Snapshot(context.Context) service.MetricsReport
}

// V1Controller registers versioned API endpoints with the unified API contract.
type V1Controller struct {
	health    v1HealthService
	metrics   v1MetricsService
	rateLimit middleware.RateLimitOptions
}

// V1ControllerOptions configures the versioned API controller.
type V1ControllerOptions struct {
	Health    v1HealthService
	Metrics   v1MetricsService
	RateLimit *middleware.RateLimitOptions
}

func NewV1Controller(g *gin.RouterGroup, health v1HealthService) *V1Controller {
	return NewV1ControllerWithOptions(g, V1ControllerOptions{Health: health})
}

func NewV1ControllerWithOptions(g *gin.RouterGroup, options V1ControllerOptions) *V1Controller {
	health := options.Health
	if health == nil {
		health = service.NewHealthService()
	}
	metrics := options.Metrics
	if metrics == nil {
		metrics = service.NewRequestMetricsStore()
	}
	rateLimit := defaultV1RateLimitOptions()
	if options.RateLimit != nil {
		rateLimit = *options.RateLimit
	}
	rateLimit.OnLimitExceeded = func(c *gin.Context) {
		WriteAPIError(c, http.StatusTooManyRequests, entity.APIErrorCodeRateLimited, "rate limit exceeded", nil)
	}

	a := &V1Controller{
		health:    health,
		metrics:   metrics,
		rateLimit: rateLimit,
	}
	a.initRouter(g)
	return a
}

func (a *V1Controller) initRouter(g *gin.RouterGroup) {
	api := g.Group("/api/v1")
	api.Use(middleware.RateLimitMiddleware(a.rateLimit))
	health := api.Group("/health")
	health.GET("/live", a.live)
	health.GET("/ready", a.ready)
	api.GET("/metrics", a.metricsSnapshot)
}

func (a *V1Controller) live(c *gin.Context) {
	writeAPISuccess(c, http.StatusOK, a.health.Live(c.Request.Context()))
}

func (a *V1Controller) ready(c *gin.Context) {
	writeAPISuccess(c, http.StatusOK, a.health.Ready(c.Request.Context()))
}

func (a *V1Controller) metricsSnapshot(c *gin.Context) {
	writeAPISuccess(c, http.StatusOK, a.metrics.Snapshot(c.Request.Context()))
}

func defaultV1RateLimitOptions() middleware.RateLimitOptions {
	return middleware.RateLimitOptions{
		Limit:  120,
		Window: time.Minute,
	}
}
