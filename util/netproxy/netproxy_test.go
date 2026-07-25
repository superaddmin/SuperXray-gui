package netproxy

import (
	"context"
	"fmt"
	"io"
	"net"
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

func TestNewHTTPClientUsesSchemeSpecificSOCKSDNSResolution(t *testing.T) {
	tests := []struct {
		name          string
		scheme        string
		wantRemoteDNS bool
	}{
		{name: "SOCKS5 resolves locally", scheme: "socks5", wantRemoteDNS: false},
		{name: "SOCKS5H resolves through proxy", scheme: "socks5h", wantRemoteDNS: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxyAddress, requestedHost := startRecordingSOCKS5Server(t)
			client, err := NewHTTPClient(fmt.Sprintf("%s://%s", tt.scheme, proxyAddress), 5*time.Second)
			if err != nil {
				t.Fatalf("NewHTTPClient %s proxy returned error: %v", tt.scheme, err)
			}
			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("transport = %T, want *http.Transport", client.Transport)
			}

			conn, err := transport.DialContext(context.Background(), "tcp", "localhost:443")
			if err != nil {
				t.Fatalf("DialContext through %s proxy returned error: %v", tt.scheme, err)
			}
			_ = conn.Close()

			select {
			case result := <-requestedHost:
				if result.err != nil {
					t.Fatalf("record SOCKS5 target: %v", result.err)
				}
				if tt.wantRemoteDNS && result.host != "localhost" {
					t.Fatalf("%s proxy target host = %q, want unresolved localhost", tt.scheme, result.host)
				}
				if !tt.wantRemoteDNS && net.ParseIP(result.host) == nil {
					t.Fatalf("%s proxy target host = %q, want locally resolved IP", tt.scheme, result.host)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for SOCKS5 target")
			}
		})
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

type socksTargetResult struct {
	host string
	err  error
}

func startRecordingSOCKS5Server(t *testing.T) (string, <-chan socksTargetResult) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for SOCKS5 test server: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	result := make(chan socksTargetResult, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			result <- socksTargetResult{err: acceptErr}
			return
		}
		defer conn.Close()

		host, handleErr := handleSOCKS5Handshake(conn)
		result <- socksTargetResult{host: host, err: handleErr}
	}()

	return listener.Addr().String(), result
}

func handleSOCKS5Handshake(conn net.Conn) (string, error) {
	greeting := make([]byte, 2)
	if _, err := io.ReadFull(conn, greeting); err != nil {
		return "", err
	}
	methods := make([]byte, int(greeting[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", err
	}
	if _, err := conn.Write([]byte{5, 0}); err != nil {
		return "", err
	}

	request := make([]byte, 4)
	if _, err := io.ReadFull(conn, request); err != nil {
		return "", err
	}
	var host string
	switch request[3] {
	case 1:
		address := make([]byte, net.IPv4len)
		if _, err := io.ReadFull(conn, address); err != nil {
			return "", err
		}
		host = net.IP(address).String()
	case 3:
		length := make([]byte, 1)
		if _, err := io.ReadFull(conn, length); err != nil {
			return "", err
		}
		address := make([]byte, int(length[0]))
		if _, err := io.ReadFull(conn, address); err != nil {
			return "", err
		}
		host = string(address)
	case 4:
		address := make([]byte, net.IPv6len)
		if _, err := io.ReadFull(conn, address); err != nil {
			return "", err
		}
		host = net.IP(address).String()
	default:
		return "", fmt.Errorf("unexpected SOCKS5 address type %d", request[3])
	}
	port := make([]byte, 2)
	if _, err := io.ReadFull(conn, port); err != nil {
		return "", err
	}
	if _, err := conn.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0}); err != nil {
		return "", err
	}
	return host, nil
}
