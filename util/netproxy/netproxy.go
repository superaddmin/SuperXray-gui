// Package netproxy builds HTTP clients for admin-configured outbound proxies.
package netproxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// NewHTTPClient returns an HTTP client whose transport honors proxyURL.
//
// Empty proxyURL keeps the existing direct-connection behavior. HTTP/HTTPS
// proxy URLs use the standard library proxy support. SOCKS5 resolves target
// names locally, while SOCKS5H sends target names to the proxy for resolution.
// Unsupported schemes return an error so callers can log and fall back safely.
func NewHTTPClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	if strings.TrimSpace(proxyURL) == "" {
		return &http.Client{Timeout: timeout}, nil
	}

	parsed, err := ParseProxyURL(proxyURL)
	if err != nil {
		return nil, err
	}

	transport := baseTransport()
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		transport.Proxy = http.ProxyURL(parsed)
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if parsed.User != nil {
			password, _ := parsed.User.Password()
			auth = &proxy.Auth{User: parsed.User.Username(), Password: password}
		}
		dialer, err := proxy.SOCKS5("tcp", parsed.Host, auth, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("create socks5 dialer: %w", err)
		}
		transport.DialContext = newSOCKSDialContext(dialer, strings.EqualFold(parsed.Scheme, "socks5h"))
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q", parsed.Scheme)
	}

	return &http.Client{Timeout: timeout, Transport: transport}, nil
}

func newSOCKSDialContext(dialer proxy.Dialer, remoteDNS bool) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		if remoteDNS {
			return dialProxyContext(ctx, dialer, network, address)
		}

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("parse SOCKS5 target address: %w", err)
		}
		if net.ParseIP(host) != nil {
			return dialProxyContext(ctx, dialer, network, address)
		}

		lookupNetwork := "ip"
		if strings.HasSuffix(network, "4") {
			lookupNetwork = "ip4"
		} else if strings.HasSuffix(network, "6") {
			lookupNetwork = "ip6"
		}
		addresses, err := net.DefaultResolver.LookupIP(ctx, lookupNetwork, host)
		if err != nil {
			return nil, fmt.Errorf("resolve SOCKS5 target locally: %w", err)
		}
		if len(addresses) == 0 {
			return nil, fmt.Errorf("resolve SOCKS5 target locally: no addresses found")
		}

		var lastErr error
		for _, ip := range addresses {
			conn, dialErr := dialProxyContext(ctx, dialer, network, net.JoinHostPort(ip.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		return nil, fmt.Errorf("dial locally resolved SOCKS5 target: %w", lastErr)
	}
}

func dialProxyContext(ctx context.Context, dialer proxy.Dialer, network string, address string) (net.Conn, error) {
	if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
		return contextDialer.DialContext(ctx, network, address)
	}
	return dialer.Dial(network, address)
}

// ParseProxyURL validates and parses an admin-configured panel proxy URL.
func ParseProxyURL(proxyURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(proxyURL))
	if err != nil {
		return nil, fmt.Errorf("proxy URL is invalid")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks5", "socks5h":
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("proxy URL host is required")
	}
	if strings.EqualFold(parsed.Scheme, "socks5") || strings.EqualFold(parsed.Scheme, "socks5h") {
		if parsed.Port() == "" {
			return nil, fmt.Errorf("proxy URL port is required for %s", strings.ToLower(parsed.Scheme))
		}
	}
	return parsed, nil
}

// RedactProxyURL removes userinfo before logging an admin-configured proxy URL.
func RedactProxyURL(proxyURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(proxyURL))
	if err != nil {
		return "<invalid proxy url>"
	}
	parsed.User = nil
	return parsed.String()
}

func baseTransport() *http.Transport {
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		return base.Clone()
	}
	return &http.Transport{}
}
