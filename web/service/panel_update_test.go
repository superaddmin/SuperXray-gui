package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCheckPanelUpdateComparesVersions(t *testing.T) {
	t.Setenv("XUI_RELEASE_REPO", "superaddmin/SuperXray-gui")
	originalClient := serviceHTTPClient
	t.Cleanup(func() { serviceHTTPClient = originalClient })

	tests := []struct {
		name      string
		current   string
		tag       string
		available bool
	}{
		{name: "newer release", current: "3.5.0", tag: "v3.6.0", available: true},
		{name: "same release", current: "v3.5.0", tag: "v3.5.0", available: false},
		{name: "older release", current: "3.5.0", tag: "v3.4.9", available: false},
		{name: "prerelease suffix ignored in comparison", current: "3.5.0", tag: "v3.5.1-rc.1", available: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			serviceHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://api.github.com/repos/superaddmin/SuperXray-gui/releases/latest" {
					t.Fatalf("unexpected request URL: %s", req.URL)
				}
				if req.Header.Get("Accept") != "application/vnd.github+json" {
					t.Fatalf("unexpected Accept header: %q", req.Header.Get("Accept"))
				}
				if !strings.HasPrefix(req.Header.Get("User-Agent"), "SuperXray-gui/") {
					t.Fatalf("missing User-Agent header: %q", req.Header.Get("User-Agent"))
				}
				return fakeReleaseResponse(t, http.StatusOK, fmt.Sprintf(
					`{"tag_name":%q,"name":"Release %s","html_url":"https://github.com/superaddmin/SuperXray-gui/releases/tag/%s","published_at":"2026-10-22T00:00:00Z","draft":false,"prerelease":false}`,
					test.tag,
					test.tag,
					test.tag,
				)), nil
			})}

			info, err := (&ServerService{}).checkPanelUpdate(test.current)
			if err != nil {
				t.Fatalf("checkPanelUpdate returned error: %v", err)
			}
			if info.UpdateAvailable != test.available {
				t.Fatalf("UpdateAvailable = %v, want %v", info.UpdateAvailable, test.available)
			}
			if info.CurrentVersion != strings.TrimPrefix(test.current, "v") {
				t.Fatalf("CurrentVersion = %q", info.CurrentVersion)
			}
			if info.LatestVersion != strings.TrimPrefix(test.tag, "v") {
				t.Fatalf("LatestVersion = %q", info.LatestVersion)
			}
		})
	}
}

func TestCheckPanelUpdateRejectsInvalidReleaseData(t *testing.T) {
	t.Setenv("XUI_RELEASE_REPO", "superaddmin/SuperXray-gui")
	originalClient := serviceHTTPClient
	t.Cleanup(func() { serviceHTTPClient = originalClient })

	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{
			name:   "invalid tag",
			status: http.StatusOK,
			body:   `{"tag_name":"nightly","draft":false,"prerelease":false}`,
			want:   "invalid release tag",
		},
		{
			name:   "prerelease marked",
			status: http.StatusOK,
			body:   `{"tag_name":"v3.6.0","draft":false,"prerelease":true}`,
			want:   "not a stable release",
		},
		{
			name:   "github rate limit",
			status: http.StatusForbidden,
			body:   `{"message":"API rate limit exceeded"}`,
			want:   "API rate limit exceeded",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			serviceHTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return fakeReleaseResponse(t, test.status, test.body), nil
			})}

			_, err := (&ServerService{}).checkPanelUpdate("3.5.0")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestPanelReleaseURLFallsBackForNonGitHubHosts(t *testing.T) {
	got := panelReleaseURL("https://evil.example/releases/v3.6.0", "superaddmin/SuperXray-gui", "v3.6.0")
	want := "https://github.com/superaddmin/SuperXray-gui/releases/tag/v3.6.0"
	if got != want {
		t.Fatalf("panelReleaseURL = %q, want %q", got, want)
	}
}

func fakeReleaseResponse(t *testing.T, status int, body string) *http.Response {
	t.Helper()
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
