package sub

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestJSONLocalInboundsBindLoopback(t *testing.T) {
	var config map[string]any
	if err := json.Unmarshal([]byte(defaultJson), &config); err != nil {
		t.Fatalf("decode default config: %v", err)
	}
	for _, raw := range config["inbounds"].([]any) {
		inbound := raw.(map[string]any)
		if inbound["listen"] != "127.0.0.1" {
			t.Fatalf("local inbound listen = %#v", inbound["listen"])
		}
	}
}

func TestJSONVLESSFlowMatchesTransportSupport(t *testing.T) {
	service := &SubJsonService{}
	client := model.Client{ID: "00000000-0000-4000-8000-000000000001", Flow: "xtls-rprx-vision"}
	inbound := &model.Inbound{Protocol: model.VLESS, Listen: "example.test", Port: 443}
	cases := []struct {
		name   string
		stream string
		want   bool
	}{
		{"ws_tls", `{"network":"ws","security":"tls"}`, false},
		{"tcp_none", `{"network":"tcp","security":"none"}`, false},
		{"tcp_tls", `{"network":"tcp","security":"tls"}`, true},
		{"tcp_reality", `{"network":"tcp","security":"reality"}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var outbound map[string]any
			if err := json.Unmarshal(service.genVless(inbound, []byte(tc.stream), client), &outbound); err != nil {
				t.Fatalf("decode outbound: %v", err)
			}
			settings, _ := outbound["settings"].(map[string]any)
			users, _ := settings["vnext"].([]any)
			user := users[0].(map[string]any)["users"].([]any)[0].(map[string]any)
			_, got := user["flow"]
			if got != tc.want {
				t.Fatalf("flow present = %v, want %v: %s", got, tc.want, strings.TrimSpace(string(service.genVless(inbound, []byte(tc.stream), client))))
			}
		})
	}
}

func TestJSONTLSPinsUseXrayStringShape(t *testing.T) {
	service := &SubJsonService{}
	got := service.tlsData(map[string]any{
		"serverName": "example.test",
		"settings": map[string]any{
			"pinnedPeerCertSha256": []any{"sha256/a", "sha256/b"},
		},
	})
	if got["pinnedPeerCertSha256"] != "sha256/a,sha256/b" {
		t.Fatalf("pins = %#v", got["pinnedPeerCertSha256"])
	}
}
