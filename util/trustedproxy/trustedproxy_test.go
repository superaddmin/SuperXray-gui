package trustedproxy

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTrustedProxyDefaultIsLoopbackOnly(t *testing.T) {
	t.Setenv(EnvironmentVariable, "")
	if err := os.Unsetenv(EnvironmentVariable); err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "http://panel.example/", nil)
	for _, test := range []struct {
		peer string
		want bool
	}{
		{"127.0.0.1:1234", true}, {"[::1]:1234", true}, {"10.0.0.1:1234", false}, {"198.51.100.1:1234", false},
	} {
		c.Request.RemoteAddr = test.peer
		if IsTrusted(c) != test.want {
			t.Fatalf("unexpected default trust for peer %s", test.peer)
		}
	}
}

func TestTrustedProxyMetadataUsesTheSameBoundary(t *testing.T) {
	for _, test := range []struct {
		name, policy, peer, forwarded, wantIP string
		trusted                               bool
	}{
		{"remote direct", "127.0.0.0/8,::1/128", "198.51.100.10:1234", "203.0.113.1", "198.51.100.10", false},
		{"loopback", "127.0.0.0/8,::1/128", "127.0.0.1:1234", "203.0.113.1", "203.0.113.1", true},
		{"IPv6 loopback", "127.0.0.0/8,::1/128", "[::1]:1234", "203.0.113.1", "203.0.113.1", true},
		{"explicit proxy", "10.20.0.0/16", "10.20.1.2:1234", "203.0.113.1, 10.20.1.3", "203.0.113.1", true},
		{"untrusted chain hop", "10.20.0.0/16", "10.20.1.2:1234", "203.0.113.1, 192.0.2.1", "192.0.2.1", true},
		{"disabled", "", "127.0.0.1:1234", "203.0.113.1", "127.0.0.1", false},
		{"malformed chain", "127.0.0.0/8", "127.0.0.1:1234", "bad, 203.0.113.1", "127.0.0.1", true},
		{"mapped peer", "127.0.0.0/8", "[::ffff:127.0.0.1]:1234", "203.0.113.1", "203.0.113.1", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(EnvironmentVariable, test.policy)
			router := gin.New()
			if err := Configure(router); err != nil {
				t.Fatal(err)
			}
			router.GET("/", func(c *gin.Context) {
				if got := ClientIP(c); got != test.wantIP {
					t.Errorf("ClientIP = %q, want %q", got, test.wantIP)
				}
				if IsTrusted(c) != test.trusted {
					t.Error("unexpected immediate-peer trust")
				}
				if (ForwardedHost(c) == "public.example") != test.trusted {
					t.Error("forwarded host crossed trust boundary")
				}
				if (Scheme(c) == "https") != test.trusted {
					t.Error("forwarded protocol crossed trust boundary")
				}
			})
			request := httptest.NewRequest(http.MethodGet, "http://panel.example/", nil)
			request.RemoteAddr = test.peer
			request.Header.Set("X-Forwarded-For", test.forwarded)
			request.Header.Set("X-Real-IP", "203.0.113.99")
			request.Header.Set("X-Forwarded-Proto", "https")
			request.Header.Set("X-Forwarded-Host", "public.example")
			router.ServeHTTP(httptest.NewRecorder(), request)
		})
	}
}

func TestTrustedProxyInvalidConfigurationFailsClosed(t *testing.T) {
	for _, value := range []string{"all", "192.0.2.1", "127.0.0.1/32,"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv(EnvironmentVariable, value)
			if err := Configure(gin.New()); err == nil {
				t.Fatal("invalid trusted-proxy CIDR accepted")
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "http://panel.example/", nil)
			c.Request.RemoteAddr = "127.0.0.1:1234"
			c.Request.Header.Set("X-Forwarded-Proto", "https")
			if IsTrusted(c) || Scheme(c) != "http" {
				t.Fatal("malformed fallback policy trusted proxy headers")
			}
		})
	}
}

func TestTrustedProxyRejectsAmbiguousMetadataAndKeepsNativeTLS(t *testing.T) {
	t.Setenv(EnvironmentVariable, "127.0.0.0/8")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "https://panel.example/", nil)
	c.Request.RemoteAddr = "127.0.0.1:1234"
	c.Request.TLS = &tls.ConnectionState{}
	c.Request.Header.Set("X-Forwarded-Proto", "http, https")
	c.Request.Header.Set("X-Forwarded-Host", "public.example, attacker.example")
	c.Request.Header.Set("X-Real-IP", "203.0.113.44")
	if Scheme(c) != "https" || ForwardedHost(c) != "" || ClientIP(c) != "203.0.113.44" {
		t.Fatal("ambiguous headers or native TLS handled incorrectly")
	}
}
