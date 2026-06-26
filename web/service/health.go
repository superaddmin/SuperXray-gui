package service

import (
	"context"
	"time"
)

// XrayRuntimeStatus exposes a read-only snapshot of the Xray runtime.
type XrayRuntimeStatus interface {
	IsXrayRunning() bool
	GetXrayErr() error
	GetXrayResult() string
	GetXrayVersion() string
}

// MetricsSnapshotter exposes a read-only metrics snapshot.
type MetricsSnapshotter interface {
	Snapshot(context.Context) MetricsReport
}

// XrayHealthReport is the health API's Xray runtime projection.
type XrayHealthReport struct {
	State    ProcessState `json:"state"`
	ErrorMsg string       `json:"errorMsg,omitempty"`
	Version  string       `json:"version"`
}

// HealthReport is the versioned API health payload.
type HealthReport struct {
	Status    string            `json:"status"`
	CheckedAt string            `json:"checkedAt"`
	Xray      *XrayHealthReport `json:"xray,omitempty"`
	Metrics   *MetricsReport    `json:"metrics,omitempty"`
}

// HealthService provides side-effect-free process health reports.
type HealthService struct {
	now     func() time.Time
	xray    XrayRuntimeStatus
	metrics MetricsSnapshotter
}

func NewHealthService() *HealthService {
	return &HealthService{now: time.Now}
}

func NewHealthServiceWithXray(xray XrayRuntimeStatus) *HealthService {
	return &HealthService{
		now:  time.Now,
		xray: xray,
	}
}

func NewHealthServiceWithXrayAndMetrics(xray XrayRuntimeStatus, metrics MetricsSnapshotter) *HealthService {
	return &HealthService{
		now:     time.Now,
		xray:    xray,
		metrics: metrics,
	}
}

func (s *HealthService) Live(context.Context) HealthReport {
	return s.report("live")
}

func (s *HealthService) Ready(context.Context) HealthReport {
	return s.report("ready")
}

func (s *HealthService) report(status string) HealthReport {
	now := time.Now
	if s != nil && s.now != nil {
		now = s.now
	}
	return HealthReport{
		Status:    status,
		CheckedAt: now().UTC().Format(time.RFC3339),
		Xray:      s.xrayReport(),
		Metrics:   s.metricsReport(),
	}
}

func (s *HealthService) xrayReport() *XrayHealthReport {
	if s == nil || s.xray == nil {
		return nil
	}

	report := &XrayHealthReport{
		State:   Stop,
		Version: s.xray.GetXrayVersion(),
	}
	if s.xray.IsXrayRunning() {
		report.State = Running
		return report
	}
	if err := s.xray.GetXrayErr(); err != nil {
		report.State = Error
		report.ErrorMsg = s.xray.GetXrayResult()
	}
	return report
}

func (s *HealthService) metricsReport() *MetricsReport {
	if s == nil || s.metrics == nil {
		return nil
	}
	report := s.metrics.Snapshot(context.Background())
	return &report
}
