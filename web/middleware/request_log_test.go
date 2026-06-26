package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeRequestMetricsRecorder struct {
	started   int
	completed []completedRequest
}

type completedRequest struct {
	method   string
	route    string
	status   int
	duration time.Duration
}

func (r *fakeRequestMetricsRecorder) RequestStarted() {
	r.started++
}

func (r *fakeRequestMetricsRecorder) RequestCompleted(method string, route string, status int, duration time.Duration) {
	r.completed = append(r.completed, completedRequest{
		method:   method,
		route:    route,
		status:   status,
		duration: duration,
	})
}

func TestStructuredRequestLoggerEmitsSafeRequestFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	nowCalls := 0
	times := []time.Time{
		time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 26, 10, 0, 0, 125000000, time.UTC),
	}
	now := func() time.Time {
		value := times[nowCalls]
		nowCalls++
		return value
	}

	var entries []RequestLogEntry
	metrics := &fakeRequestMetricsRecorder{}
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.Use(StructuredRequestLoggerMiddleware(RequestLogOptions{
		Now:     now,
		Metrics: metrics,
		Log: func(entry RequestLogEntry) {
			entries = append(entries, entry)
		},
	}))
	router.GET("/items/:id", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.test/items/42?token=secret", nil)
	req.Header.Set(RequestIDHeader, "req-log")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.RequestID != "req-log" {
		t.Fatalf("requestId = %q, want req-log", entry.RequestID)
	}
	if entry.Method != http.MethodGet {
		t.Fatalf("method = %q, want %q", entry.Method, http.MethodGet)
	}
	if entry.Route != "/items/:id" {
		t.Fatalf("route = %q, want /items/:id", entry.Route)
	}
	if entry.Path != "/items/:id" {
		t.Fatalf("path = %q, want route template without query or token", entry.Path)
	}
	if entry.Status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", entry.Status, http.StatusCreated)
	}
	if entry.DurationMs != 125 {
		t.Fatalf("durationMs = %v, want 125", entry.DurationMs)
	}
	if entry.ClientIP == "" {
		t.Fatal("clientIp must be recorded")
	}

	if metrics.started != 1 {
		t.Fatalf("metrics started = %d, want 1", metrics.started)
	}
	if len(metrics.completed) != 1 {
		t.Fatalf("metrics completed = %d, want 1", len(metrics.completed))
	}
	completed := metrics.completed[0]
	if completed.method != http.MethodGet || completed.route != "/items/:id" || completed.status != http.StatusCreated || completed.duration != 125*time.Millisecond {
		t.Fatalf("completed metrics = %#v, want GET /items/:id %d 125ms", completed, http.StatusCreated)
	}
}
