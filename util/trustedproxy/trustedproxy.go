// Package trustedproxy gives every HTTP surface the same forwarded-header
// boundary. Only a configured immediate peer may supply proxy metadata.
package trustedproxy

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const EnvironmentVariable = "SUPERXRAY_TRUSTED_PROXY_CIDRS"

const policyContextKey = "superxray.trustedProxyPolicy"

type policy struct {
	prefixes []netip.Prefix
	cidrs    []string
}

func environmentPolicy() (*policy, error) {
	raw, configured := os.LookupEnv(EnvironmentVariable)
	if !configured {
		raw = "127.0.0.0/8,::1/128"
	}
	p := &policy{}
	if strings.TrimSpace(raw) == "" {
		return p, nil
	}
	for _, value := range strings.Split(raw, ",") {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("%s must contain comma-separated IP CIDRs", EnvironmentVariable)
		}
		prefix = prefix.Masked()
		p.prefixes = append(p.prefixes, prefix)
		p.cidrs = append(p.cidrs, prefix.String())
	}
	return p, nil
}

// Configure installs a stable policy at router startup and also configures
// Gin's ClientIP for callers outside this package. Invalid settings fail closed.
func Configure(engine *gin.Engine) error {
	p, err := environmentPolicy()
	if err != nil {
		return err
	}
	if err := engine.SetTrustedProxies(p.cidrs); err != nil {
		return err
	}
	engine.Use(func(c *gin.Context) {
		c.Set(policyContextKey, p)
		c.Next()
	})
	return nil
}

func requestPolicy(c *gin.Context) *policy {
	if value, ok := c.Get(policyContextKey); ok {
		if p, ok := value.(*policy); ok {
			return p
		}
	}
	// Standalone handlers and tests use the same defaults. A malformed policy
	// never makes headers trustworthy even outside a configured server.
	p, err := environmentPolicy()
	if err != nil {
		return &policy{}
	}
	return p
}

func peerIP(request *http.Request) netip.Addr {
	if request == nil {
		return netip.Addr{}
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	ip, _ := netip.ParseAddr(host)
	return ip.Unmap()
}

func (p *policy) contains(ip netip.Addr) bool {
	for _, prefix := range p.prefixes {
		if prefix.Contains(ip.Unmap()) {
			return true
		}
	}
	return false
}

// IsTrusted reports whether the immediate TCP peer can supply proxy headers.
func IsTrusted(c *gin.Context) bool {
	return requestPolicy(c).contains(peerIP(c.Request))
}

// Scheme derives the external scheme only from a trusted peer or native TLS.
func Scheme(c *gin.Context) string {
	if IsTrusted(c) {
		proto := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")))
		if proto == "http" || proto == "https" {
			return proto
		}
	}
	if c.Request != nil && c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

// ForwardedHost returns a trusted, single proxy value. The caller still applies
// its normal host syntax validation before using it in a generated URL.
func ForwardedHost(c *gin.Context) string {
	if !IsTrusted(c) {
		return ""
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if strings.Contains(host, ",") {
		return ""
	}
	return host
}

// ClientIP walks a trusted proxy chain from right to left. A direct caller
// cannot choose its rate-limit bucket or audit identity using X-Forwarded-For.
func ClientIP(c *gin.Context) string {
	p := requestPolicy(c)
	peer := peerIP(c.Request)
	if !peer.IsValid() {
		return ""
	}
	if p.contains(peer) {
		forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-For"))
		if forwarded != "" {
			parts := strings.Split(forwarded, ",")
			addresses := make([]netip.Addr, len(parts))
			for i, part := range parts {
				address, err := netip.ParseAddr(strings.TrimSpace(part))
				if err != nil || address.Zone() != "" {
					return peer.String()
				}
				addresses[i] = address.Unmap()
			}
			for i := len(addresses) - 1; i >= 0; i-- {
				if i == 0 || !p.contains(addresses[i]) {
					return addresses[i].String()
				}
			}
		}
		if realIP, err := netip.ParseAddr(strings.TrimSpace(c.GetHeader("X-Real-IP"))); err == nil && realIP.Zone() == "" {
			return realIP.Unmap().String()
		}
	}
	return peer.String()
}
