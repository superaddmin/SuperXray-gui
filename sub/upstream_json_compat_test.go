package sub

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/json_util"
)

func TestUpstreamJSONDoesNotShareSourceIPFilters(t *testing.T) {
	s := &SubJsonService{}
	for _, raw := range []string{
		`{"finalmask":{"tcp":[{"type":"filter","settings":{"ips":["192.0.2.0/24"]}}],"udp":[{"type":"filter","settings":{"ips":["198.51.100.0/24"]}},{"type":"noise","settings":{"script":"noise.lua"}}],"vendor":true}}`,
		`{"finalmask":{"udp":[{"type":"filter","settings":{"ips":["192.0.2.0/24"]}}]}}`,
	} {
		got, _ := s.streamData(raw)
		wire, _ := json.Marshal(got)
		if strings.Contains(string(wire), `"filter"`) || strings.Contains(string(wire), "192.0.2.") {
			t.Fatal(string(wire))
		}
		if strings.Contains(raw, "noise.lua") && (!strings.Contains(string(wire), "noise.lua") || !strings.Contains(string(wire), `"vendor":true`)) {
			t.Fatal(string(wire))
		}
	}
}

func TestUpstreamJSONTLSCipherSuitesSelectUnsafeOnlyWithoutFingerprint(t *testing.T) {
	s := &SubJsonService{}
	for _, fp := range []string{"", "chrome"} {
		got := s.tlsData(map[string]any{"cipherSuites": "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "settings": map[string]any{"fingerprint": fp}})
		want := fp
		if want == "" {
			want = "unsafe"
		}
		if got["cipherSuites"] != "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256" || got["fingerprint"] != want {
			t.Fatal(got)
		}
	}
	if got := s.tlsData(map[string]any{}); got["fingerprint"] == "unsafe" {
		t.Fatal(got)
	}
}

func TestUpstreamXDriveJSONRequiresMux(t *testing.T) {
	client := model.Client{ID: "00000000-0000-4000-8000-000000000001"}
	inbound := &model.Inbound{Protocol: model.VLESS, Settings: `{"encryption":"fixture-encryption"}`}
	stream := json_util.RawMessage(`{"network":"xdrive"}`)
	for _, mux := range []string{"", `{"enabled":false}`, `{"enabled":true,"concurrency":-1}`, `{"enabled":true,"xudpProxyUDP443":"skip"}`, `{"enabled":true,"concurrency":16,"vendor":true}`} {
		s := &SubJsonService{mux: mux}
		var outbound map[string]any
		if err := json.Unmarshal(s.genVless(inbound, stream, client), &outbound); err != nil {
			t.Fatal(err)
		}
		got, ok := outbound["mux"].(map[string]any)
		if !ok || got["enabled"] != true || got["concurrency"] == float64(-1) || got["xudpProxyUDP443"] == "skip" {
			t.Fatal(outbound)
		}
		if strings.Contains(mux, "vendor") && got["vendor"] != true {
			t.Fatal(outbound)
		}
	}
}

func TestUpstreamJSONExternalTLSOverridesStayIsolated(t *testing.T) {
	client := matrixClient("cipher@example.test")
	inbound := matrixInbound(model.VLESS, 443, client)
	inbound.StreamSettings = `{"network":"tcp","security":"tls","tlsSettings":{"cipherSuites":"base-suite","settings":{"fingerprint":"chrome"}},"externalProxy":[{"dest":"first.example.test","port":443,"cipherSuites":"host-suite","fingerprint":""},{"dest":"second.example.test","port":443},{"dest":"plain.example.test","port":80,"forceTls":"none"}]}`
	s := &SubJsonService{SubService: &SubService{remarkModel: "-ieo"}}
	configs := s.getConfig(inbound, client, "vpn.example.test")
	if len(configs) != 3 {
		t.Fatalf("configs = %d", len(configs))
	}
	for i, raw := range configs {
		var cfg map[string]any
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatal(err)
		}
		stream := cfg["outbounds"].([]any)[0].(map[string]any)["streamSettings"].(map[string]any)
		tls, _ := stream["tlsSettings"].(map[string]any)
		switch i {
		case 0:
			if tls["cipherSuites"] != "host-suite" || tls["fingerprint"] != "unsafe" {
				t.Fatal(tls)
			}
		case 1:
			if tls["cipherSuites"] != "base-suite" || tls["fingerprint"] != "chrome" {
				t.Fatal(tls)
			}
		case 2:
			if tls != nil || stream["security"] != "none" {
				t.Fatal(stream)
			}
		}
	}
}

func TestUpstreamXDriveJSONSkipsMissingClientEncryption(t *testing.T) {
	client := matrixClient("xdrive@example.test")
	inbound := matrixInbound(model.VLESS, 443, client)
	inbound.StreamSettings = `{"network":"xdrive"}`
	inbound.Settings = `{"decryption":"fixture-server-decryption"}`
	s := &SubJsonService{SubService: &SubService{remarkModel: "-ieo"}}
	if got := s.getConfig(inbound, client, "vpn.example.test"); len(got) != 0 {
		t.Fatalf("invalid XDRIVE client exported: %s", got)
	}
}
