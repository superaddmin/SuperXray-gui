package network

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

type initialReadConn struct {
	net.Conn
	data  string
	err   error
	reads int
}

func TestAutoHTTPSConcurrentReadsConsumeInitialBufferOnce(t *testing.T) {
	const initial = "abcdefghijklmnop"
	conn := NewAutoHttpsConn(&initialReadConn{data: initial, err: io.EOF})
	results := make(chan byte, len(initial))
	for range len(initial) {
		go func() {
			var buf [1]byte
			_, err := io.ReadFull(conn, buf[:])
			if err != nil {
				t.Errorf("read: %v", err)
			}
			results <- buf[0]
		}()
	}
	got := make([]byte, 0, len(initial))
	for range len(initial) {
		got = append(got, <-results)
	}
	slices.Sort(got)
	if string(got) != initial {
		t.Fatalf("read bytes = %q, want %q", got, initial)
	}
}

func TestAutoHTTPSRedirectsPlainHTTP(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	deadline := time.Now().Add(3 * time.Second)
	_ = server.SetDeadline(deadline)
	_ = client.SetDeadline(deadline)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = NewAutoHttpsConn(server).Read(make([]byte, 1))
	}()
	for _, fragment := range []string{
		"G", "ET /panel/?a=1 HTTP/1.1\r\nHost: panel.example\r\n",
		"X-Test: " + strings.Repeat("x", 4096), "\r\n\r", "\n",
	} {
		if _, err := io.WriteString(client, fragment); err != nil {
			t.Fatal(err)
		}
	}
	resp, err := http.ReadResponse(bufio.NewReader(client), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTemporaryRedirect || resp.Header.Get("Location") != "https://panel.example/panel/?a=1" {
		t.Fatalf("unexpected redirect: %s, %q", resp.Status, resp.Header.Get("Location"))
	}
	<-done
}

func TestAutoHTTPSImmediatelyPassesThroughTLS(t *testing.T) {
	raw := &initialReadConn{data: "\x16\x03\x01"}
	conn := NewAutoHttpsConn(raw)
	var first [3]byte
	if _, err := io.ReadFull(conn, first[:]); err != nil || string(first[:]) != raw.data || raw.reads != 1 {
		t.Fatalf("TLS prefix = %x, err = %v, reads = %d", first, err, raw.reads)
	}
}

func TestAutoHTTPSReplaysOversizedHeader(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	deadline := time.Now().Add(5 * time.Second)
	_ = server.SetDeadline(deadline)
	_ = client.SetDeadline(deadline)
	header := "GET / HTTP/1.1\r\nHost: panel.example\r\nX-Test: " + strings.Repeat("x", http.DefaultMaxHeaderBytes) + "\r\n\r\n"
	go func() {
		defer client.Close()
		_, _ = io.WriteString(client, header)
	}()
	got, err := io.ReadAll(NewAutoHttpsConn(server))
	if err != nil || string(got) != header {
		t.Fatalf("oversized header passthrough: got %d bytes, want %d, err=%v", len(got), len(header), err)
	}
}

func (c *initialReadConn) Read(p []byte) (int, error) {
	c.reads++
	if c.reads == 1 {
		return copy(p, c.data), c.err
	}
	return 0, io.EOF
}

func TestAutoHTTPSPreservesFirstReadError(t *testing.T) {
	for _, first := range []string{"", "abcdef"} {
		raw := &initialReadConn{data: first, err: io.ErrUnexpectedEOF}
		conn := NewAutoHttpsConn(raw)
		if n, err := conn.Read(nil); n != 0 || err != nil || raw.reads != 0 {
			t.Fatalf("empty read = %d, %v; underlying reads = %d", n, err, raw.reads)
		}
		var got string
		buf := make([]byte, 2)
		for {
			n, err := conn.Read(buf)
			got += string(buf[:n])
			if err != nil {
				if !errors.Is(err, io.ErrUnexpectedEOF) || raw.reads != 1 {
					t.Fatalf("first error = %v, reads = %d", err, raw.reads)
				}
				break
			}
			if n == 0 {
				t.Fatal("read returned (0, nil) before the initial error")
			}
		}
		if got != first {
			t.Fatalf("data = %q, want %q", got, first)
		}
	}
}
