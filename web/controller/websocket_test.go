package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/superaddmin/SuperXray-gui/v2/util/trustedproxy"
)

func TestWebSocketOriginFollowsPanelAuthority(t *testing.T) {
	t.Setenv(trustedproxy.EnvironmentVariable, "127.0.0.0/8")
	for _, tc := range []struct {
		name, requestURL, origin, forwarded string
		want                                bool
	}{
		{"without Origin", "http://panel.example/", "", "", true},
		{"native TLS", "https://panel.example/", "https://PANEL.example", "", true},
		{"default HTTPS port", "https://panel.example:443/", "https://panel.example", "", true},
		{"trusted proxy", "http://panel.example/", "https://panel.example", "https", true},
		{"different scheme", "http://panel.example/", "https://panel.example", "", false},
		{"IPv6 and port", "http://[::1]:8080/", "http://[::1]:8080", "", true},
		{"different port", "http://panel.example:8080/", "http://panel.example:8081", "", false},
		{"different IPv6", "http://[::1]/", "http://[::2]", "", false},
		{"invalid URL", "http://panel.example/", "null", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", tc.requestURL, nil)
			c.Request.RemoteAddr = "127.0.0.1:4567"
			if tc.origin != "" {
				c.Request.Header.Set("Origin", tc.origin)
			}
			c.Request.Header.Set("X-Forwarded-Proto", tc.forwarded)
			if got := webSocketOriginAllowed(c); got != tc.want {
				t.Fatalf("origin accepted = %v, want %v", got, tc.want)
			}
		})
	}
}
