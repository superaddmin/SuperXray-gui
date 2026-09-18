package service

import (
	"context"
	"fmt"
	"strings"
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

func TestRequestMetricsStoreBoundsRouteDimensionsAndPreservesAggregates(t *testing.T) {
	const (
		requestCount      = 300
		maxRouteDimension = 256
	)
	store := NewRequestMetricsStore()

	for i := 0; i < requestCount; i++ {
		store.RequestCompleted(fmt.Sprintf("M%03d", i), fmt.Sprintf("/route/%03d", i), 200, 10*time.Millisecond)
	}

	report := store.Snapshot(context.Background())
	if got := len(report.Requests.ByRoute); got > maxRouteDimension {
		t.Fatalf("route dimensions = %d, want at most %d", got, maxRouteDimension)
	}
	if got := sumRouteTotals(report.Requests.ByRoute); got != requestCount {
		t.Fatalf("sum of route totals = %d, want %d", got, requestCount)
	}
	if report.Requests.Total != requestCount {
		t.Fatalf("total = %d, want %d", report.Requests.Total, requestCount)
	}
	if report.Requests.AverageLatencyMs != 10 {
		t.Fatalf("averageLatencyMs = %d, want 10", report.Requests.AverageLatencyMs)
	}
	overflow := report.Requests.ByRoute["<overflow>"]
	if overflow.Total != requestCount-(maxRouteDimension-1) || overflow.AverageLatencyMs != 10 {
		t.Fatalf("overflow route metric = %#v, want total %d and averageLatencyMs 10", overflow, requestCount-(maxRouteDimension-1))
	}
}

func TestRequestMetricsStoreBoundsStatusDimensionsAndPreservesCounts(t *testing.T) {
	const (
		requestCount       = 100
		maxStatusDimension = 64
	)
	store := NewRequestMetricsStore()

	for i := 0; i < requestCount; i++ {
		store.RequestCompleted("GET", "/status-test", 1000+i, time.Millisecond)
	}

	report := store.Snapshot(context.Background())
	if got := len(report.Requests.ByStatus); got > maxStatusDimension {
		t.Fatalf("status dimensions = %d, want at most %d", got, maxStatusDimension)
	}
	var total int64
	for _, count := range report.Requests.ByStatus {
		total += count
	}
	if total != requestCount {
		t.Fatalf("sum of status totals = %d, want %d", total, requestCount)
	}
	if overflow := report.Requests.ByStatus["<overflow>"]; overflow != requestCount-(maxStatusDimension-1) {
		t.Fatalf("overflow status count = %d, want %d", overflow, requestCount-(maxStatusDimension-1))
	}
}

func TestRequestMetricsStoreDoesNotRetainOversizedLabels(t *testing.T) {
	store := NewRequestMetricsStore()
	method := strings.Repeat("M", 1024)
	route := "/" + strings.Repeat("r", 4096)

	store.RequestCompleted(method, route, 200, 25*time.Millisecond)
	store.RequestCompleted(method+"X", route+"x", 200, 35*time.Millisecond)

	report := store.Snapshot(context.Background())
	if got := len(report.Requests.ByRoute); got != 1 {
		t.Fatalf("route dimensions = %d, want 1", got)
	}
	for key, metric := range report.Requests.ByRoute {
		if len(key) > 600 {
			t.Fatalf("route metric key retained oversized labels: length = %d", len(key))
		}
		if metric.Total != 2 || metric.AverageLatencyMs != 30 {
			t.Fatalf("route metric = %#v, want total 2 and averageLatencyMs 30", metric)
		}
	}
}

func sumRouteTotals(routes map[string]RouteMetricsSnapshot) int64 {
	var total int64
	for _, metric := range routes {
		total += metric.Total
	}
	return total
}
