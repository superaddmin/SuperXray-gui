package service

import (
	"context"
	"testing"
	"time"
)

func TestRequestMetricsStoreSnapshotsRequestCounters(t *testing.T) {
	store := NewRequestMetricsStore()

	store.RequestStarted()
	store.RequestCompleted("GET", "/api/v1/health/live", 200, 150*time.Millisecond)
	store.RequestStarted()

	report := store.Snapshot(context.Background())

	if report.Requests.Total != 1 {
		t.Fatalf("total = %d, want 1", report.Requests.Total)
	}
	if report.Requests.InFlight != 1 {
		t.Fatalf("inFlight = %d, want 1", report.Requests.InFlight)
	}
	if report.Requests.ByStatus["200"] != 1 {
		t.Fatalf("byStatus[200] = %d, want 1", report.Requests.ByStatus["200"])
	}
	route := report.Requests.ByRoute["GET /api/v1/health/live"]
	if route.Total != 1 {
		t.Fatalf("route total = %d, want 1", route.Total)
	}
	if route.AverageLatencyMs != 150 {
		t.Fatalf("route averageLatencyMs = %v, want 150", route.AverageLatencyMs)
	}
}
