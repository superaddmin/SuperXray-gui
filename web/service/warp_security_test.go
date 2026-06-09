package service

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestBuildWarpRegistrationPayloadEscapesJSON(t *testing.T) {
	body, err := buildWarpRegistrationPayload(`pub"key`, "2026-04-28T00:00:00.000Z", `host"name`)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("payload is invalid JSON: %v\n%s", err, body)
	}

	if parsed["key"] != `pub"key` {
		t.Fatalf("key = %q", parsed["key"])
	}
	if parsed["name"] != `host"name` {
		t.Fatalf("name = %q", parsed["name"])
	}
}

func TestBuildWarpLicensePayloadEscapesJSON(t *testing.T) {
	body, err := buildWarpLicensePayload(`license"key`)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("payload is invalid JSON: %v\n%s", err, body)
	}

	if parsed["license"] != `license"key` {
		t.Fatalf("license = %q", parsed["license"])
	}
}

type warpRoundTripFunc func(*http.Request) (*http.Response, error)

func (f warpRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestWarpAPIRequestsUsePanelProxyHTTPClientFactory(t *testing.T) {
	setupSettingServiceTestDB(t)
	warpSvc := &WarpService{}
	if err := warpSvc.SetWarp(`{"access_token":"token-1","device_id":"device-1","license_key":"license-1","private_key":"secret-1"}`); err != nil {
		t.Fatalf("SetWarp failed: %v", err)
	}

	oldFactory := newWarpHTTPClient
	t.Cleanup(func() { newWarpHTTPClient = oldFactory })

	var calls []string
	newWarpHTTPClient = func(_ *SettingService, timeout time.Duration) *http.Client {
		if timeout != 15*time.Second {
			t.Fatalf("warp HTTP timeout = %v, want 15s", timeout)
		}
		return &http.Client{
			Transport: warpRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls = append(calls, req.Method+" "+req.URL.Path)
				body := `{}`
				switch {
				case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/reg/device-1"):
					if req.Header.Get("Authorization") != "Bearer token-1" {
						t.Fatalf("GetWarpConfig Authorization = %q", req.Header.Get("Authorization"))
					}
					body = `{"id":"device-1"}`
				case req.Method == http.MethodPost && req.URL.Path == "/v0a2158/reg":
					body = `{"id":"device-2","token":"token-2","account":{"license":"license-2"}}`
				case req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/reg/device-2/account"):
					if req.Header.Get("Authorization") != "Bearer token-2" {
						t.Fatalf("SetWarpLicense Authorization = %q", req.Header.Get("Authorization"))
					}
					body = `{"success":true}`
				default:
					t.Fatalf("unexpected WARP request: %s %s", req.Method, req.URL.String())
				}
				return &http.Response{
					StatusCode:    http.StatusOK,
					Status:        "200 OK",
					Header:        make(http.Header),
					Body:          io.NopCloser(strings.NewReader(body)),
					ContentLength: int64(len(body)),
				}, nil
			}),
		}
	}

	if _, err := warpSvc.GetWarpConfig(); err != nil {
		t.Fatalf("GetWarpConfig failed: %v", err)
	}
	if _, err := warpSvc.RegWarp("secret-2", "public-2"); err != nil {
		t.Fatalf("RegWarp failed: %v", err)
	}
	if _, err := warpSvc.SetWarpLicense("license-3"); err != nil {
		t.Fatalf("SetWarpLicense failed: %v", err)
	}

	want := []string{
		http.MethodGet + " /v0a2158/reg/device-1",
		http.MethodPost + " /v0a2158/reg",
		http.MethodPut + " /v0a2158/reg/device-2/account",
	}
	if strings.Join(calls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("warp calls = %#v, want %#v", calls, want)
	}
}
