package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeXrayRuntimeStatus struct {
	running bool
	err     error
	result  string
	version string
}

func (s fakeXrayRuntimeStatus) IsXrayRunning() bool {
	return s.running
}

func (s fakeXrayRuntimeStatus) GetXrayErr() error {
	return s.err
}

func (s fakeXrayRuntimeStatus) GetXrayResult() string {
	return s.result
}

func (s fakeXrayRuntimeStatus) GetXrayVersion() string {
	return s.version
}

func TestHealthServiceReadyIncludesXrayRuntimeSnapshot(t *testing.T) {
	svc := NewHealthServiceWithXray(fakeXrayRuntimeStatus{
		running: true,
		version: "v26.3.27",
	})

	report := svc.Ready(context.Background())

	if report.Status != "ready" {
		t.Fatalf("status = %q, want ready", report.Status)
	}
	if report.Xray == nil {
		t.Fatal("xray snapshot must be present")
	}
	if report.Xray.State != Running {
		t.Fatalf("xray.state = %q, want %q", report.Xray.State, Running)
	}
	if report.Xray.Version != "v26.3.27" {
		t.Fatalf("xray.version = %q, want v26.3.27", report.Xray.Version)
	}
}

func TestHealthServiceReadyReportsXrayErrorSnapshot(t *testing.T) {
	svc := NewHealthServiceWithXray(fakeXrayRuntimeStatus{
		err:     errors.New("xray crashed"),
		result:  "panic",
		version: "v26.3.27",
	})

	report := svc.Ready(context.Background())

	if report.Xray == nil {
		t.Fatal("xray snapshot must be present")
	}
	if report.Xray.State != Error {
		t.Fatalf("xray.state = %q, want %q", report.Xray.State, Error)
	}
	if report.Xray.ErrorMsg != "panic" {
		t.Fatalf("xray.errorMsg = %q, want panic", report.Xray.ErrorMsg)
	}
}

func TestHealthServiceReadyIncludesMetricsSnapshot(t *testing.T) {
	metrics := NewRequestMetricsStore()
	metrics.RequestStarted()
	metrics.RequestCompleted("GET", "/api/v1/health/live", 200, 20*time.Millisecond)

	svc := NewHealthServiceWithXrayAndMetrics(nil, metrics)
	report := svc.Ready(context.Background())

	if report.Metrics == nil {
		t.Fatal("metrics snapshot must be present")
	}
	if report.Metrics.Requests.Total != 1 {
		t.Fatalf("metrics.requests.total = %d, want 1", report.Metrics.Requests.Total)
	}
}
