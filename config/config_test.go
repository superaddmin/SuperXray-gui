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

func TestGetReleaseRepoUsesValidEnvironmentOverride(t *testing.T) {
	t.Setenv("XUI_RELEASE_REPO", "example/panel")
	if got := GetReleaseRepo(); got != "example/panel" {
		t.Fatalf("GetReleaseRepo() = %q, want example/panel", got)
	}
}

func TestGetReleaseRepoFallsBackForInvalidValues(t *testing.T) {
	for _, value := range []string{"", "owner", "../repo", "https://github.com/owner/repo"} {
		t.Setenv("XUI_RELEASE_REPO", value)
		if got := GetReleaseRepo(); got != defaultReleaseRepo {
			t.Fatalf("GetReleaseRepo() with %q = %q, want %q", value, got, defaultReleaseRepo)
		}
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
