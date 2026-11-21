package ldaputil

import (
	"net"
	"testing"
	"time"
)

func TestUnresponsiveDirectoryReleasesBind(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	conn, err := dialDirectory(Config{Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port}, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	peer, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	done := make(chan error, 1)
	go func() { done <- conn.Bind("cn=test", "fixture") }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("unresponsive directory authenticated a user")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("bind did not release its login reservation within the deadline")
	}
}
