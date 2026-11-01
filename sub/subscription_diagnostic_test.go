package sub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestDiagnoseSubscriptionInboundsReportsOutputAndSkippedNodes(t *testing.T) {
	client := matrixClient("diagnose-vless@example")
	inbounds := []*model.Inbound{
		matrixInbound(model.VLESS, 13001, client),
		matrixProxyAccountInbound(model.HTTP, 13002, map[string]any{
			"user":  "diagnose-http",
			"pass":  "diagnose-pass",
			"subId": client.SubID,
		}),
		matrixInbound(model.Tunnel, 13003, client),
	}

	diagnostic := diagnoseSubscriptionInbounds(inbounds, client.SubID, subscriptionFormatURI)

	if diagnostic.TotalInbounds != 3 {
		t.Fatalf("total inbounds = %d, want 3", diagnostic.TotalInbounds)
	}
	if diagnostic.OutputNodes != 2 {
		t.Fatalf("output nodes = %d, want 2", diagnostic.OutputNodes)
	}
	if diagnostic.UDPCapableNodes != 1 || diagnostic.UDPUnsupportedNodes != 1 {
		t.Fatalf("udp capable/unsupported = %d/%d, want 1/1", diagnostic.UDPCapableNodes, diagnostic.UDPUnsupportedNodes)
	}
	if diagnostic.SkippedNodes != 1 {
		t.Fatalf("skipped nodes = %d, want 1", diagnostic.SkippedNodes)
	}
	if len(diagnostic.SkipReasons) != 1 || diagnostic.SkipReasons[0].Protocol != string(model.Tunnel) {
		t.Fatalf("skip reasons = %#v, want one tunnel skip", diagnostic.SkipReasons)
	}
	if len(diagnostic.Warnings) == 0 || !strings.Contains(strings.Join(diagnostic.Warnings, " "), "UDP") {
		t.Fatalf("warnings should explain skipped and UDP-incapable nodes: %#v", diagnostic.Warnings)
	}
}

func TestSubscriptionNodeUDPCapability(t *testing.T) {
	mixedUDPDisabled := matrixProxyAccountInbound(model.Mixed, 13004, map[string]any{
		"user":  "mixed-udp-off",
		"pass":  "mixed-pass",
		"subId": "matrix-sub",
	})
	var mixedSettings map[string]any
	if err := json.Unmarshal([]byte(mixedUDPDisabled.Settings), &mixedSettings); err != nil {
		t.Fatalf("mixed settings invalid: %v", err)
	}
	mixedSettings["udp"] = false
	encoded, err := json.Marshal(mixedSettings)
	if err != nil {
		t.Fatalf("encode mixed settings: %v", err)
	}
	mixedUDPDisabled.Settings = string(encoded)

	cases := []struct {
		name    string
		inbound *model.Inbound
		want    bool
	}{
		{name: "vless", inbound: matrixInbound(model.VLESS, 13005, matrixClient("udp-vless@example")), want: true},
		{name: "wireguard", inbound: matrixInboundWithSettings(model.WireGuard, 13006, map[string]any{}, map[string]any{}), want: true},
		{name: "http account", inbound: matrixProxyAccountInbound(model.HTTP, 13007, map[string]any{"user": "http", "pass": "pass"}), want: false},
		{name: "mixed udp enabled", inbound: matrixProxyAccountInbound(model.Mixed, 13008, map[string]any{"user": "mixed", "pass": "pass"}), want: true},
		{name: "mixed udp disabled", inbound: mixedUDPDisabled, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := subscriptionNodeUDPCapable(tc.inbound); got != tc.want {
				t.Fatalf("subscriptionNodeUDPCapable(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestDiagnoseSubscriptionInboundsReportsEmptySubscription(t *testing.T) {
	diagnostic := diagnoseSubscriptionInbounds(nil, "missing-sub", subscriptionFormatURI)

	if diagnostic.TotalInbounds != 0 || diagnostic.OutputNodes != 0 || diagnostic.SkippedNodes != 0 {
		t.Fatalf("empty diagnostic counts = %#v, want all zero", diagnostic)
	}
	if len(diagnostic.Warnings) != 1 {
		t.Fatalf("empty diagnostic warnings = %#v, want one readable warning", diagnostic.Warnings)
	}
}

func TestDiagnoseSubscriptionInboundsReportsSupportedFormatsAndProtocols(t *testing.T) {
	diagnostic := diagnoseSubscriptionInbounds(nil, "matrix-sub", subscriptionFormatClash)

	wantFormats := []string{"uri", "json", "clash", "wireguard"}
	if strings.Join(diagnostic.SupportedFormats, ",") != strings.Join(wantFormats, ",") {
		t.Fatalf("supported formats = %#v, want %#v", diagnostic.SupportedFormats, wantFormats)
	}

	wantProtocols := []string{"vmess", "vless", "http", "trojan", "shadowsocks", "mixed", "hysteria", "hysteria2", "wireguard"}
	if strings.Join(diagnostic.SupportedProtocols, ",") != strings.Join(wantProtocols, ",") {
		t.Fatalf("supported protocols = %#v, want %#v", diagnostic.SupportedProtocols, wantProtocols)
	}
}

func TestSubscriptionDiagnoseEndpointReturnsJSON(t *testing.T) {
	setupSubscriptionDiagnosticDB(t)
	client := matrixClient("diagnose-endpoint@example")
	inbound := matrixInbound(model.VLESS, 13003, client)
	inbound.Enable = true
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewSUBController(router.Group("/"), "/sub/", "/json/", "/clash/", true, true, false, true, "-ieo", "12", "", "", "", "", "", "", "", "", true, "", false)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/sub/matrix-sub/diagnose?target=xray", nil)
	request.Host = "vpn.example"
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var diagnostic SubscriptionDiagnostic
	if err := json.Unmarshal(recorder.Body.Bytes(), &diagnostic); err != nil {
		t.Fatalf("diagnostic JSON invalid: %v", err)
	}
	if diagnostic.Format != string(subscriptionFormatJSON) || diagnostic.OutputNodes != 1 {
		t.Fatalf("diagnostic = %#v, want json format with one output", diagnostic)
	}
	if diagnostic.UDPCapableNodes != 1 || diagnostic.UDPUnsupportedNodes != 0 {
		t.Fatalf("diagnostic udp = %d/%d, want 1/0", diagnostic.UDPCapableNodes, diagnostic.UDPUnsupportedNodes)
	}
}

func setupSubscriptionDiagnosticDB(t *testing.T) {
	t.Helper()
	dbDir := t.TempDir()
	t.Setenv("XUI_DB_FOLDER", dbDir)
	if err := database.InitDB(filepath.Join(dbDir, "SuperXray.db")); err != nil {
		t.Fatalf("database.InitDB failed: %v", err)
	}
	t.Cleanup(func() {
		if err := database.CloseDB(); err != nil {
			t.Logf("database.CloseDB warning: %v", err)
		}
	})
}
