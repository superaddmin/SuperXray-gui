package sub

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestUpstreamTLSCipherSuitesShareLinks(t *testing.T) {
	const suites = "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
	svc := &SubService{address: "vpn.example.test", remarkModel: "-ieo"}
	for _, protocol := range []model.Protocol{model.VLESS, model.VMESS, model.Trojan, model.Shadowsocks} {
		for _, fingerprint := range []string{"", "chrome", "unsafe"} {
			t.Run(string(protocol)+"/"+fingerprint, func(t *testing.T) {
				client := matrixPasswordClient("cipher@example.test")
				client.ID = "00000000-0000-4000-8000-000000000001"
				inbound := matrixInbound(protocol, 443, client)
				if protocol == model.Shadowsocks {
					inbound = matrixShadowsocksInbound(443, client)
				}
				stream := map[string]any{"network": "tcp", "security": "tls", "tlsSettings": map[string]any{
					"serverName": "tls.example.test", "cipherSuites": suites,
					"settings": map[string]any{"fingerprint": fingerprint},
				}}
				encoded, _ := json.Marshal(stream)
				inbound.StreamSettings = string(encoded)
				before := inbound.StreamSettings
				fields := upstreamShareFields(t, svc.getLink(inbound, client.Email))
				wantFingerprint := fingerprint
				if wantFingerprint == "" {
					wantFingerprint = "unsafe"
				}
				if fields["cs"] != suites || fields["fp"] != wantFingerprint {
					t.Fatalf("share fields = %#v, want cs and fp=%q", fields, wantFingerprint)
				}
				if inbound.StreamSettings != before {
					t.Fatal("generating a link mutated stored TLS settings")
				}
			})
		}
	}
}

func TestUpstreamTLSCipherSuitesExternalProxyIsolation(t *testing.T) {
	svc := &SubService{address: "vpn.example.test", remarkModel: "-ieo"}
	for _, protocol := range []model.Protocol{model.VLESS, model.VMESS} {
		t.Run(string(protocol), func(t *testing.T) {
			client := matrixClient("external-cipher@example.test")
			inbound := matrixInbound(protocol, 443, client)
			inbound.StreamSettings = `{"network":"tcp","security":"tls","tlsSettings":{"cipherSuites":"base-suite","settings":{"fingerprint":""}},"externalProxy":[{"dest":"first.example.test","port":443,"forceTls":"same","cipherSuites":"host-suite","fingerprint":"firefox"},{"dest":"second.example.test","port":443,"forceTls":"same"},{"dest":"plain.example.test","port":80,"forceTls":"none","cipherSuites":"ignored-suite"}]}`
			links := strings.Split(svc.getLink(inbound, client.Email), "\n")
			if len(links) != 3 {
				t.Fatalf("links = %d, want 3", len(links))
			}
			first, second, plain := upstreamShareFields(t, links[0]), upstreamShareFields(t, links[1]), upstreamShareFields(t, links[2])
			if first["cs"] != "host-suite" || first["fp"] != "firefox" {
				t.Fatalf("external override = %#v", first)
			}
			if second["cs"] != "base-suite" || second["fp"] != "unsafe" {
				t.Fatalf("external endpoint leaked fields = %#v", second)
			}
			if plain["cs"] != "" || plain["fp"] != "" {
				t.Fatalf("plaintext link retained TLS fields = %#v", plain)
			}
		})
	}
}

func TestUpstreamTLSWithoutCipherSuitesDoesNotForceFingerprint(t *testing.T) {
	svc := &SubService{address: "vpn.example.test", remarkModel: "-ieo"}
	client := matrixClient("no-cipher@example.test")
	inbound := matrixInbound(model.VLESS, 443, client)
	inbound.StreamSettings = `{"network":"tcp","security":"tls","tlsSettings":{"settings":{"fingerprint":""}}}`
	fields := upstreamShareFields(t, svc.getLink(inbound, client.Email))
	if fields["cs"] != "" || fields["fp"] != "" {
		t.Fatalf("empty TLS fields changed = %#v", fields)
	}
}

func TestUpstreamShadowsocksHTTPObfsCompatibility(t *testing.T) {
	client := matrixPasswordClient("obfs@example.test")
	svc := &SubClashService{SubService: &SubService{remarkModel: "-ieo"}}
	cases := []struct {
		name     string
		protocol model.Protocol
		request  map[string]any
		want     bool
	}{
		{"default_root", model.Shadowsocks, nil, true},
		{"explicit_root", model.Shadowsocks, map[string]any{"path": []any{"/"}, "headers": map[string]any{"hOsT": []any{"cdn.example.test"}}}, true},
		{"root_among_paths", model.Shadowsocks, map[string]any{"path": []any{"/custom", "/"}}, true},
		{"custom_path", model.Shadowsocks, map[string]any{"path": []any{"/custom"}}, false},
		{"malformed_path", model.Shadowsocks, map[string]any{"path": "/custom"}, false},
		{"vmess_header", model.VMESS, map[string]any{"path": []any{"/"}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inbound := matrixShadowsocksInbound(8080, client)
			inbound.Protocol = tc.protocol
			stream := map[string]any{"network": "tcp", "security": "none", "tcpSettings": map[string]any{
				"header": map[string]any{"type": "http", "request": tc.request},
			}}
			encoded, _ := json.Marshal(stream)
			inbound.StreamSettings = string(encoded)
			proxies := svc.getProxies(inbound, client, "vpn.example.test")
			if !tc.want {
				if len(proxies) != 0 {
					t.Fatalf("incompatible HTTP header exported: %#v", proxies)
				}
				return
			}
			if len(proxies) != 1 || proxies[0]["plugin"] != "obfs" {
				t.Fatalf("compatible SS header omitted: %#v", proxies)
			}
			opts, _ := proxies[0]["plugin-opts"].(map[string]any)
			if opts["mode"] != "http" {
				t.Fatalf("obfs options = %#v", opts)
			}
			if tc.name == "explicit_root" && opts["host"] != "cdn.example.test" {
				t.Fatalf("obfs host = %#v", opts)
			}
		})
	}
}

func TestUpstreamMkcplegacyShareFields(t *testing.T) {
	for _, tc := range []struct{ header, value, wantHeader, wantSeed string }{
		{"", "seed-value", "", "seed-value"},
		{"", "", "", ""},
		{"dns", "dns.example.test", "dns", ""},
		{"wechat", "", "wechat-video", ""},
		{"srtp", "", "srtp", ""},
	} {
		t.Run(tc.header+"/"+tc.value, func(t *testing.T) {
			stream := map[string]any{"kcpSettings": map[string]any{"mtu": float64(1350), "tti": float64(20)}, "finalmask": map[string]any{
				"udp": []any{map[string]any{"type": "mkcp-legacy", "settings": map[string]any{"header": tc.header, "value": tc.value, "customExtension": true}}},
			}}
			params := map[string]string{}
			applyKcpShareParams(stream, params)
			applyFinalMaskParams(stream["finalmask"].(map[string]any), params)
			if params["headerType"] != tc.wantHeader || params["seed"] != tc.wantSeed {
				t.Fatalf("legacy fields = %#v", params)
			}
			if !strings.Contains(params["fm"], `"mkcp-legacy"`) || !strings.Contains(params["fm"], `"customExtension":true`) {
				t.Fatalf("mkcp mask/extension dropped: %#v", params)
			}
			obj := map[string]any{}
			applyKcpShareObj(stream, obj)
			if seed, _ := obj["path"].(string); seed != tc.wantSeed {
				t.Fatalf("VMess seed = %#v", obj)
			}
		})
	}
}

func TestUpstreamMkcplegacyHeaderDoesNotReplaceSeed(t *testing.T) {
	stream := map[string]any{"finalmask": map[string]any{"udp": []any{
		map[string]any{"type": "mkcp-legacy", "settings": map[string]any{"value": "encryption-seed"}},
		map[string]any{"type": "mkcp-legacy", "settings": map[string]any{"header": "dns", "value": "cover.example.test"}},
	}}}
	params := map[string]string{}
	applyKcpShareParams(stream, params)
	if params["headerType"] != "dns" || params["seed"] != "encryption-seed" {
		t.Fatalf("chained mKCP masks changed encryption: %#v", params)
	}
	obj := map[string]any{}
	applyKcpShareObj(stream, obj)
	if obj["type"] != "dns" || obj["path"] != "encryption-seed" {
		t.Fatalf("VMess chained mKCP masks = %#v", obj)
	}
}

func upstreamShareFields(t *testing.T, link string) map[string]string {
	t.Helper()
	fields := map[string]string{}
	if strings.HasPrefix(link, "vmess://") {
		payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(link, "vmess://"))
		if err != nil {
			t.Fatal(err)
		}
		var object map[string]any
		if err := json.Unmarshal(payload, &object); err != nil {
			t.Fatal(err)
		}
		for key, value := range object {
			if text, ok := value.(string); ok {
				fields[key] = text
			}
		}
		return fields
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	for key, values := range parsed.Query() {
		fields[key] = values[0]
	}
	return fields
}
