package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/web/entity"
	"github.com/superaddmin/SuperXray-gui/v2/web/middleware"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"

	"github.com/gin-gonic/gin"
)

type fakeV1HealthService struct {
	live  service.HealthReport
	ready service.HealthReport
}

func (s fakeV1HealthService) Live(context.Context) service.HealthReport {
	return s.live
}

func (s fakeV1HealthService) Ready(context.Context) service.HealthReport {
	return s.ready
}

type fakeV1MetricsService struct {
	report service.MetricsReport
}

func (s fakeV1MetricsService) Snapshot(context.Context) service.MetricsReport {
	return s.report
}

func TestV1HealthLiveUsesUnifiedAPIResponse(t *testing.T) {
	router := newV1TestRouter(fakeV1HealthService{
		live: service.HealthReport{Status: "live", CheckedAt: "2026-06-26T00:00:00Z"},
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/api/v1/health/live", nil)
	req.Header.Set(middleware.RequestIDHeader, "req-live")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get(middleware.RequestIDHeader); got != "req-live" {
		t.Fatalf("%s = %q, want req-live", middleware.RequestIDHeader, got)
	}

	var body struct {
		Success   bool                 `json:"success"`
		RequestID string               `json:"requestId"`
		Data      service.HealthReport `json:"data"`
		Error     *entity.APIError     `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response must be JSON: %v", err)
	}
	if !body.Success {
		t.Fatal("success = false, want true")
	}
	if body.RequestID != "req-live" {
		t.Fatalf("requestId = %q, want req-live", body.RequestID)
	}
	if body.Error != nil {
		t.Fatalf("error = %#v, want nil", body.Error)
	}
	if body.Data.Status != "live" {
		t.Fatalf("data.status = %q, want live", body.Data.Status)
	}
}

func TestV1NotFoundUsesUnifiedAPIError(t *testing.T) {
	router := newV1TestRouter(fakeV1HealthService{})
	router.NoRoute(func(c *gin.Context) {
		WriteAPIError(c, http.StatusNotFound, entity.APIErrorCodeNotFound, "route not found", nil)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/api/v1/missing", nil)
	req.Header.Set(middleware.RequestIDHeader, "req-missing")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}

	var body entity.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response must be JSON: %v", err)
	}
	if body.Success {
		t.Fatal("success = true, want false")
	}
	if body.RequestID != "req-missing" {
		t.Fatalf("requestId = %q, want req-missing", body.RequestID)
	}
	if body.Error == nil {
		t.Fatal("error must be present")
	}
	if body.Error.Code != entity.APIErrorCodeNotFound {
		t.Fatalf("error.code = %q, want %q", body.Error.Code, entity.APIErrorCodeNotFound)
	}
}

func TestV1RateLimitUsesUnifiedAPIError(t *testing.T) {
	router := newV1TestRouterWithOptions(fakeV1HealthService{
		live: service.HealthReport{Status: "live", CheckedAt: "2026-06-26T00:00:00Z"},
	}, V1ControllerOptions{
		RateLimit: &middleware.RateLimitOptions{
			Limit:  1,
			Window: time.Minute,
		},
	})

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, "http://example.test/api/v1/health/live", nil)
	firstRequest.Header.Set(middleware.RequestIDHeader, "req-rate-1")
	router.ServeHTTP(first, firstRequest)
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusOK)
	}

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodGet, "http://example.test/api/v1/health/live", nil)
	secondRequest.Header.Set(middleware.RequestIDHeader, "req-rate-2")
	router.ServeHTTP(second, secondRequest)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header must be set")
	}

	var body entity.APIResponse
	if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil {
		t.Fatalf("response must be JSON: %v", err)
	}
	if body.Success {
		t.Fatal("success = true, want false")
	}
	if body.RequestID != "req-rate-2" {
		t.Fatalf("requestId = %q, want req-rate-2", body.RequestID)
	}
	if body.Error == nil {
		t.Fatal("error must be present")
	}
	if body.Error.Code != entity.APIErrorCodeRateLimited {
		t.Fatalf("error.code = %q, want %q", body.Error.Code, entity.APIErrorCodeRateLimited)
	}
}

func TestV1MetricsUsesUnifiedAPIResponse(t *testing.T) {
	router := newV1TestRouterWithOptions(fakeV1HealthService{}, V1ControllerOptions{
		Metrics: fakeV1MetricsService{
			report: service.MetricsReport{
				Requests: service.RequestMetricsSnapshot{
					Total:    3,
					InFlight: 1,
					ByStatus: map[string]int64{"200": 2, "404": 1},
				},
			},
		},
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/api/v1/metrics", nil)
	req.Header.Set(middleware.RequestIDHeader, "req-metrics")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body struct {
		Success   bool                  `json:"success"`
		RequestID string                `json:"requestId"`
		Data      service.MetricsReport `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response must be JSON: %v", err)
	}
	if !body.Success {
		t.Fatal("success = false, want true")
	}
	if body.RequestID != "req-metrics" {
		t.Fatalf("requestId = %q, want req-metrics", body.RequestID)
	}
	if body.Data.Requests.Total != 3 {
		t.Fatalf("requests.total = %d, want 3", body.Data.Requests.Total)
	}
}

func newV1TestRouter(health v1HealthService) *gin.Engine {
	return newV1TestRouterWithOptions(health, V1ControllerOptions{})
}

func newV1TestRouterWithOptions(health v1HealthService, options V1ControllerOptions) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestIDMiddleware())
	options.Health = health
	NewV1ControllerWithOptions(router.Group("/"), options)
	return router
}
