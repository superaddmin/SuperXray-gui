package web

import (
	"io/fs"
	"net/http"
	"os"
	"strings"
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

func TestLegacyTemplatesDoNotUseVHTML(t *testing.T) {
	err := fs.WalkDir(os.DirFS("html"), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		content, err := os.ReadFile("html/" + path)
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "v-html") {
			t.Fatalf("legacy template %s still contains v-html", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIndexLogRenderingDoesNotUseHTMLSink(t *testing.T) {
	content, err := os.ReadFile("html/index.html")
	if err != nil {
		t.Fatal(err)
	}

	source := string(content)
	for _, forbidden := range []string{
		`v-html="logModal.formattedLogs"`,
		`v-html="xraylogModal.formattedLogs"`,
		"formattedLogs += `<",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("index log rendering still contains unsafe HTML sink %q", forbidden)
		}
	}
}
