package middleware

import (
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/logger"

	"github.com/gin-gonic/gin"
)

// RequestLogEntry captures a structured view of a completed request.
type RequestLogEntry struct {
	RequestID  string
	Method     string
	Path       string
	Route      string
	Status     int
	DurationMs int64
	ClientIP   string
	Error      string
}

// RequestMetricsCollector is implemented by metrics stores that track request counters.
type RequestMetricsCollector interface {
	RequestStarted()
	RequestCompleted(method string, route string, status int, duration time.Duration)
}

// RequestLogOptions configures structured request logging.
type RequestLogOptions struct {
	Now     func() time.Time
	Log     func(RequestLogEntry)
	Metrics RequestMetricsCollector
}

// StructuredRequestLoggerMiddleware records completed requests and forwards them to the logger sink.
func StructuredRequestLoggerMiddleware(opts RequestLogOptions) gin.HandlerFunc {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Log == nil {
		opts.Log = func(entry RequestLogEntry) {
			logger.Infof(
				"request completed requestId=%s method=%s path=%s route=%s status=%d durationMs=%d clientIp=%s error=%s",
				entry.RequestID, entry.Method, entry.Path, entry.Route, entry.Status, entry.DurationMs, entry.ClientIP, entry.Error,
			)
		}
	}

	return func(c *gin.Context) {
		started := opts.Now()
		if opts.Metrics != nil {
			opts.Metrics.RequestStarted()
		}

		c.Next()

		finished := opts.Now()
		duration := finished.Sub(started)
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		entry := RequestLogEntry{
			RequestID:  RequestID(c),
			Method:     c.Request.Method,
			Path:       route,
			Route:      route,
			Status:     c.Writer.Status(),
			DurationMs: durationMs(duration),
			ClientIP:   c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			entry.Error = c.Errors.String()
		}
		if opts.Metrics != nil {
			opts.Metrics.RequestCompleted(entry.Method, route, entry.Status, duration)
		}
		opts.Log(entry)
	}
}

func durationMs(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return d.Milliseconds()
}
