package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSidebarComponentExposesThemeSwitcher(t *testing.T) {
	sourcePath := filepath.Join("html", "component", "aSidebar.html")
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read sidebar component: %v", err)
	}

	source := string(sourceBytes)
	if !strings.Contains(source, "themeSwitcher,") {
		t.Fatalf("a-sidebar must expose themeSwitcher in data before using it in template and computed properties")
	}
}

func TestSidebarDoesNotReferenceMissingLogoIcon(t *testing.T) {
	sourcePath := filepath.Join("html", "component", "aSidebar.html")
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read sidebar component: %v", err)
	}

	if strings.Contains(string(sourceBytes), "assets/img/logo-icon.svg") {
		if _, err := os.Stat(filepath.Join("assets", "img", "logo-icon.svg")); err != nil {
			t.Fatalf("a-sidebar references missing logo-icon.svg: %v", err)
		}
	}
}

func TestSidebarUsesConfiguredBasePath(t *testing.T) {
	sourcePath := filepath.Join("html", "component", "aSidebar.html")
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read sidebar component: %v", err)
	}

	if strings.Contains(string(sourceBytes), "window.basePath") {
		t.Fatalf("a-sidebar must use the configured basePath variable so assets and logo navigation work under non-root paths")
	}
}
