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
// proxy URLs use the standard library proxy support; SOCKS5/SOCKS5H URLs use
// golang.org/x/net/proxy. Unsupported schemes return an error so callers can
// log and fall back safely.
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
		if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = contextDialer.DialContext
		} else {
			transport.DialContext = func(_ context.Context, network string, address string) (net.Conn, error) {
				return dialer.Dial(network, address)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q", parsed.Scheme)
	}

	return &http.Client{Timeout: timeout, Transport: transport}, nil
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
