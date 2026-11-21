// Package network provides network utilities for the SuperXray web panel,
// including automatic HTTP to HTTPS redirection functionality.
package network

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"slices"
	"sync"
)

// AutoHttpsConn wraps a net.Conn to provide automatic HTTP to HTTPS redirection.
// It intercepts the first read to detect HTTP requests and responds with a 307 redirect
// to the HTTPS equivalent URL. Subsequent reads work normally for HTTPS connections.
type AutoHttpsConn struct {
	net.Conn

	firstBuf []byte
	firstErr error
	bufMu    sync.Mutex

	readRequestOnce sync.Once
}

// NewAutoHttpsConn creates a new AutoHttpsConn that wraps the given connection.
// It enables automatic redirection of HTTP requests to HTTPS.
func NewAutoHttpsConn(conn net.Conn) net.Conn {
	return &AutoHttpsConn{
		Conn: conn,
	}
}

func (c *AutoHttpsConn) readRequest() {
	// A TCP read can end anywhere in the request headers. Retain every byte
	// until a complete header parses, so TLS and unrecognised input can replay it.
	for {
		start := len(c.firstBuf)
		chunk := min(2048, http.DefaultMaxHeaderBytes-start)
		if chunk == 0 {
			return
		}
		c.firstBuf = slices.Grow(c.firstBuf, chunk)
		n, err := c.Conn.Read(c.firstBuf[start : start+chunk])
		c.firstBuf = c.firstBuf[:start+n]
		c.firstErr = err
		if n == 0 || c.firstBuf[0] < '!' || c.firstBuf[0] > '~' {
			return
		}
		// Include the last two bytes of the previous read for split delimiters.
		tail := c.firstBuf[max(0, start-2):]
		if bytes.Contains(tail, []byte("\n\r\n")) || bytes.Contains(tail, []byte("\n\n")) {
			break
		}
		if err != nil {
			return
		}
	}
	reader := bytes.NewReader(c.firstBuf)
	bufReader := bufio.NewReader(reader)
	request, err := http.ReadRequest(bufReader)
	if err != nil {
		return
	}
	resp := http.Response{
		Header: http.Header{},
	}
	resp.StatusCode = http.StatusTemporaryRedirect
	location := fmt.Sprintf("https://%v%v", request.Host, request.RequestURI)
	resp.Header.Set("Location", location)
	if err := resp.Write(c.Conn); err != nil {
		return
	}
	if err := c.Close(); err != nil {
		return
	}
	c.firstBuf = nil
}

// Read implements the net.Conn Read method with automatic HTTPS redirection.
// On the first read, it checks if the request is HTTP and redirects to HTTPS if so.
// Subsequent reads work normally.
func (c *AutoHttpsConn) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	c.readRequestOnce.Do(c.readRequest)
	c.bufMu.Lock()
	if len(c.firstBuf) > 0 {
		n := copy(buf, c.firstBuf)
		c.firstBuf = c.firstBuf[n:]
		c.bufMu.Unlock()
		return n, nil
	}
	c.firstBuf = nil
	err := c.firstErr
	c.firstErr = nil
	c.bufMu.Unlock()
	if err != nil {
		return 0, err
	}

	return c.Conn.Read(buf)
}
