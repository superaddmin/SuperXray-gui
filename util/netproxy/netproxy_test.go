package netproxy

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPClientReturnsDirectClientForEmptyProxy(t *testing.T) {
	client, err := NewHTTPClient("", 7*time.Second)
	if err != nil {
		t.Fatalf("NewHTTPClient empty proxy returned error: %v", err)
	}
	if client.Timeout != 7*time.Second {
		t.Fatalf("client timeout = %v, want 7s", client.Timeout)
	}
	if client.Transport != nil {
		t.Fatalf("empty proxy transport = %#v, want default nil transport", client.Transport)
	}
}

func TestNewHTTPClientConfiguresHTTPProxyTransport(t *testing.T) {
	client, err := NewHTTPClient("http://127.0.0.1:18080", 5*time.Second)
	if err != nil {
		t.Fatalf("NewHTTPClient http proxy returned error: %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("HTTP proxy transport.Proxy is nil")
	}
	req, err := http.NewRequest(http.MethodGet, "https://example.test/path", nil)
	if err != nil {
		t.Fatal(err)
	}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy returned error: %v", err)
	}
	if proxyURL.String() != "http://127.0.0.1:18080" {
		t.Fatalf("proxy URL = %q", proxyURL.String())
	}
}

func TestNewHTTPClientRejectsUnsupportedProxyScheme(t *testing.T) {
	if _, err := NewHTTPClient("ftp://127.0.0.1:21", time.Second); err == nil {
		t.Fatal("NewHTTPClient accepted unsupported ftp proxy scheme")
	}
}

func TestNewHTTPClientRejectsMissingProxyHost(t *testing.T) {
	for _, proxyURL := range []string{"http://", "https://", "socks5://", "socks5h://"} {
		t.Run(proxyURL, func(t *testing.T) {
			if _, err := NewHTTPClient(proxyURL, time.Second); err == nil {
				t.Fatalf("NewHTTPClient accepted %q without host", proxyURL)
			}
		})
	}
}

func TestParseProxyURLDoesNotExposeCredentialsOnParseError(t *testing.T) {
	proxyURL := "http://user:pa ss@example.com:8080"
	_, err := ParseProxyURL(proxyURL)
	if err == nil {
		t.Fatalf("ParseProxyURL accepted invalid proxy URL %q", proxyURL)
	}
	for _, secret := range []string{"user", "pa ss", "example.com", proxyURL} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("ParseProxyURL error leaked %q: %v", secret, err)
		}
	}
}

func TestParseProxyURLRequiresSocksPort(t *testing.T) {
	for _, proxyURL := range []string{"socks5://127.0.0.1", "socks5h://proxy.example"} {
		t.Run(proxyURL, func(t *testing.T) {
			if _, err := ParseProxyURL(proxyURL); err == nil {
				t.Fatalf("ParseProxyURL accepted SOCKS proxy without port: %q", proxyURL)
			}
		})
	}
}

func TestRedactProxyURLHidesUserInfo(t *testing.T) {
	redacted := RedactProxyURL("socks5://user:password@127.0.0.1:1080")
	if redacted != "socks5://127.0.0.1:1080" {
		t.Fatalf("RedactProxyURL = %q", redacted)
	}
	if redacted == "" || redacted == "socks5://user:password@127.0.0.1:1080" {
		t.Fatalf("RedactProxyURL did not redact credentials: %q", redacted)
	}
}
