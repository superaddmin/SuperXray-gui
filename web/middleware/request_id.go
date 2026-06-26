package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	RequestIDHeader     = "X-Request-ID"
	RequestIDContextKey = "request_id"
)

// RequestIDMiddleware attaches a stable request id to the context and response.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := sanitizeRequestID(c.GetHeader(RequestIDHeader))
		if requestID == "" {
			requestID = newRequestID()
		}

		c.Set(RequestIDContextKey, requestID)
		c.Writer.Header().Set(RequestIDHeader, requestID)
		c.Next()
	}
}

// RequestID returns the request id stored in Gin context by RequestIDMiddleware.
func RequestID(c *gin.Context) string {
	value, _ := c.Get(RequestIDContextKey)
	requestID, _ := value.(string)
	return requestID
}

func sanitizeRequestID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || len(requestID) > 128 {
		return ""
	}
	for _, r := range requestID {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return requestID
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err == nil {
		return "req_" + hex.EncodeToString(bytes)
	}
	return "req_" + strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
}
