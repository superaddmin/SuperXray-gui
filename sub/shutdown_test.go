package sub

import (
	"net"
	"testing"
)

func TestStopIgnoresAlreadyClosedListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer()
	server.listener = listener
	server.httpServer = newHTTPServer(nil)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if err := server.Stop(); err != nil {
		t.Fatalf("clean Stop: %v", err)
	}
	if err := server.Stop(); err != nil {
		t.Fatalf("repeated Stop: %v", err)
	}
}
