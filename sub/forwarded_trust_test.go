package sub

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/superaddmin/SuperXray-gui/v2/util/trustedproxy"
)

func TestSubscriptionHostOnlyUsesTrustedProxyMetadata(t *testing.T) {
	t.Setenv(trustedproxy.EnvironmentVariable, "127.0.0.0/8")
	for _, peer := range []string{"198.51.100.1:1234", "127.0.0.1:1234"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "http://panel.example:2096/sub/id", nil)
		c.Request.RemoteAddr = peer
		c.Request.Header.Set("X-Forwarded-Proto", "https")
		c.Request.Header.Set("X-Forwarded-Host", "public.example:443")
		scheme, host, hostWithPort, _ := (&SubService{}).ResolveRequest(c)
		if peer == "127.0.0.1:1234" {
			if scheme != "https" || host != "public.example" || hostWithPort != "public.example:443" {
				t.Fatal("trusted subscription origin was not used")
			}
		} else if scheme != "http" || host != "panel.example" || hostWithPort != "panel.example:2096" {
			t.Fatal("direct request injected subscription origin")
		}
	}
}
