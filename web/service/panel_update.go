package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/superaddmin/SuperXray-gui/v2/config"
)

// PanelUpdateInfo describes the latest stable SuperXray release relative to the running panel.
type PanelUpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ReleaseName     string `json:"releaseName"`
	ReleaseURL      string `json:"releaseUrl"`
	PublishedAt     string `json:"publishedAt"`
}

type panelRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

type panelVersion [3]int

var panelVersionPattern = regexp.MustCompile(`^v?([0-9]+)\.([0-9]+)\.([0-9]+)(?:[-+][0-9A-Za-z.-]+)?$`)

// CheckPanelUpdate compares the running panel version with the latest stable GitHub release.
func (s *ServerService) CheckPanelUpdate() (*PanelUpdateInfo, error) {
	return s.checkPanelUpdate(config.GetVersion())
}

func (s *ServerService) checkPanelUpdate(currentVersion string) (*PanelUpdateInfo, error) {
	current, ok := parsePanelVersion(currentVersion)
	if !ok {
		return nil, fmt.Errorf("invalid current panel version %q", currentVersion)
	}

	repo := config.GetReleaseRepo()
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "SuperXray-gui/"+normalizePanelVersion(currentVersion))

	resp, err := serviceHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := validateContentLength(resp, maxAPIResponseBytes); err != nil {
		return nil, err
	}
	body, err := readBodyLimited(resp.Body, maxAPIResponseBytes)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var apiError struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &apiError) == nil && apiError.Message != "" {
			return nil, fmt.Errorf("GitHub API error: %s", apiError.Message)
		}
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var release panelRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("parse GitHub release: %w", err)
	}
	if release.Draft || release.Prerelease {
		return nil, fmt.Errorf("latest GitHub release %q is not a stable release", release.TagName)
	}
	latest, ok := parsePanelVersion(release.TagName)
	if !ok {
		return nil, fmt.Errorf("invalid release tag %q", release.TagName)
	}

	return &PanelUpdateInfo{
		CurrentVersion:  normalizePanelVersion(currentVersion),
		LatestVersion:   normalizePanelVersion(release.TagName),
		UpdateAvailable: comparePanelVersions(current, latest) < 0,
		ReleaseName:     firstNonEmpty(strings.TrimSpace(release.Name), strings.TrimSpace(release.TagName)),
		ReleaseURL:      panelReleaseURL(release.HTMLURL, repo, release.TagName),
		PublishedAt:     strings.TrimSpace(release.PublishedAt),
	}, nil
}

func parsePanelVersion(value string) (panelVersion, bool) {
	var version panelVersion
	matches := panelVersionPattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return version, false
	}
	for i := 0; i < 3; i++ {
		number, err := strconv.Atoi(matches[i+1])
		if err != nil {
			return version, false
		}
		version[i] = number
	}
	return version, true
}

func comparePanelVersions(left, right panelVersion) int {
	for i := 0; i < 3; i++ {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	return 0
}

func normalizePanelVersion(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}

func panelReleaseURL(raw, repo, tag string) string {
	fallback := fmt.Sprintf("https://github.com/%s/releases/tag/%s", repo, url.PathEscape(strings.TrimSpace(tag)))
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, "github.com") {
		return fallback
	}
	return parsed.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
