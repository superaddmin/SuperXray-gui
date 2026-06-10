package web

import (
	"net/http"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewHTTPServerSetsTimeoutsAndHeaderLimit(t *testing.T) {
	server := newHTTPServer(gin.New())

	if server.ReadHeaderTimeout <= 0 {
		t.Fatal("ReadHeaderTimeout must be configured")
	}
	if server.ReadTimeout <= 0 {
		t.Fatal("ReadTimeout must be configured")
	}
	if server.WriteTimeout <= 0 {
		t.Fatal("WriteTimeout must be configured")
	}
	if server.IdleTimeout <= 0 {
		t.Fatal("IdleTimeout must be configured")
	}
	if server.MaxHeaderBytes <= 0 || server.MaxHeaderBytes == http.DefaultMaxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d, want explicit non-default limit", server.MaxHeaderBytes)
	}
}

func TestLegacyHTMLAndAssetsDirectoriesAreRetired(t *testing.T) {
	for _, path := range []string{"html", "assets"} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("legacy UI directory %s must be removed", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat legacy UI directory %s: %v", path, err)
		}
	}
}
