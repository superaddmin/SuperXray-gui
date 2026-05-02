package config

import (
	"strings"
	"testing"
)

func TestGetVersionReturnsSemanticVersion(t *testing.T) {
	want := strings.TrimSpace(version)
	if got := GetVersion(); got != want {
		t.Fatalf("GetVersion() = %q, want %q", got, want)
	}
}

func TestGetAssetVersionIncludesBuildHash(t *testing.T) {
	oldBuildHash := buildHash
	buildHash = "abc123"
	t.Cleanup(func() {
		buildHash = oldBuildHash
	})

	want := GetVersion() + ".abc123"
	if got := GetAssetVersion(); got != want {
		t.Fatalf("GetAssetVersion() = %q, want %q", got, want)
	}
}
