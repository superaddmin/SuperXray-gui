package service

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// RequestMetricsSnapshot captures request counters for read-only reporting.
type RequestMetricsSnapshot struct {
	Total            int64                           `json:"total"`
	InFlight         int64                           `json:"inFlight"`
	ByStatus         map[string]int64                `json:"byStatus"`
	ByRoute          map[string]RouteMetricsSnapshot `json:"byRoute"`
	AverageLatencyMs int64                           `json:"averageLatencyMs"`
}

// RouteMetricsSnapshot captures request counters for a specific route.
type RouteMetricsSnapshot struct {
	Total            int64 `json:"total"`
	AverageLatencyMs int64 `json:"averageLatencyMs"`
}

// MetricsReport is the versioned API metrics payload.
type MetricsReport struct {
	Requests   RequestMetricsSnapshot `json:"requests"`
	ObservedAt string                 `json:"observedAt"`
}

// RequestMetricsStore collects in-memory request metrics.
type RequestMetricsStore struct {
	mu           sync.Mutex
	total        int64
	inFlight     int64
	statusCount  map[string]int64
	routeCount   map[string]int64
	routeLatency map[string]time.Duration
	latency      time.Duration
}

func NewRequestMetricsStore() *RequestMetricsStore {
	return &RequestMetricsStore{
		statusCount:  map[string]int64{},
		routeCount:   map[string]int64{},
		routeLatency: map[string]time.Duration{},
	}
}

func (s *RequestMetricsStore) RequestStarted() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inFlight++
}

func (s *RequestMetricsStore) RequestCompleted(method string, route string, status int, duration time.Duration) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inFlight > 0 {
		s.inFlight--
	}
	s.total++
	s.statusCount[strconv.Itoa(status)]++
	key := method + " " + route
	s.routeCount[key]++
	s.routeLatency[key] += duration
	s.latency += duration
}

func (s *RequestMetricsStore) Snapshot(context.Context) MetricsReport {
	if s == nil {
		return MetricsReport{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	requests := RequestMetricsSnapshot{
		Total:    s.total,
		InFlight: s.inFlight,
		ByStatus: copyInt64Map(s.statusCount),
		ByRoute:  map[string]RouteMetricsSnapshot{},
	}
	if s.total > 0 {
		requests.AverageLatencyMs = int64(s.latency / time.Duration(s.total) / time.Millisecond)
	}
	for route, count := range s.routeCount {
		avg := int64(0)
		if count > 0 {
			avg = int64(s.routeLatency[route] / time.Duration(count) / time.Millisecond)
		}
		requests.ByRoute[route] = RouteMetricsSnapshot{
			Total:            count,
			AverageLatencyMs: avg,
		}
	}
	return MetricsReport{
		Requests:   requests,
		ObservedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func copyInt64Map(src map[string]int64) map[string]int64 {
	dst := make(map[string]int64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
