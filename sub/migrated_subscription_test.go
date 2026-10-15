package sub

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/xraycompat"
)

func normalizeSubscriptionFixture(t *testing.T, inbound *model.Inbound) {
	t.Helper()
	settings, stream, _, err := xraycompat.NormalizeInbound(string(inbound.Protocol.XrayProtocol()), inbound.Settings, inbound.StreamSettings)
	if err != nil {
		t.Fatal(err)
	}
	inbound.Settings, inbound.StreamSettings = settings, stream
}

func TestMigratedXHTTPSubscriptionsRetainSessionSettings(t *testing.T) {
	base := &SubService{address: "fixture.example.test", remarkModel: "-ieo"}
	jsonService := &SubJsonService{SubService: base, configJson: map[string]any{}}
	clashService := &SubClashService{SubService: base}
	for _, protocol := range []model.Protocol{model.VLESS, model.VMESS} {
		t.Run(string(protocol), func(t *testing.T) {
			client := matrixClient("fixture@example.test")
			inbound := matrixInbound(protocol, 12345, client)
			inbound.StreamSettings = `{"network":"xhttp","security":"none","xhttpSettings":{"path":"/fixture","mode":"auto","extra":{"sessionPlacement":"header","sessionKey":"X-Fixture","sessionIDTable":"Base62","sessionIDLength":"8-16"}}}`
			normalizeSubscriptionFixture(t, inbound)
			link := base.getLink(cloneMatrixInbound(inbound), client.Email)
			var extraText string
			if protocol == model.VMESS {
				data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(link, "vmess://"))
				if err != nil {
					t.Fatal(err)
				}
				var object map[string]any
				if err := json.Unmarshal(data, &object); err != nil {
					t.Fatal(err)
				}
				extraText, _ = object["extra"].(string)
			} else {
				parsed, err := url.Parse(link)
				if err != nil {
					t.Fatal(err)
				}
				extraText = parsed.Query().Get("extra")
			}
			var extra map[string]any
			if err := json.Unmarshal([]byte(extraText), &extra); err != nil {
				t.Fatalf("decode migrated share extra: %v", err)
			}
			assertSession := func(fields map[string]any) {
				t.Helper()
				for key, want := range map[string]string{"sessionIDPlacement": "header", "sessionIDKey": "X-Fixture", "sessionIDTable": "Base62", "sessionIDLength": "8-16"} {
					if fields[key] != want {
						t.Errorf("%s = %v, want %s", key, fields[key], want)
					}
				}
			}
			assertSession(extra)
			configs := jsonService.getConfig(cloneMatrixInbound(inbound), client, base.address)
			if len(configs) != 1 {
				t.Fatalf("JSON configurations = %d, want 1", len(configs))
			}
			outbound := firstMatrixOutbound(t, configs[0])
			xhttp := outbound["streamSettings"].(map[string]any)["xhttpSettings"].(map[string]any)
			assertSession(xhttp["extra"].(map[string]any))
			if got := clashService.getProxies(cloneMatrixInbound(inbound), client, base.address); len(got) != 0 {
				t.Fatal("unsupported XHTTP transport was silently downgraded for Clash")
			}
		})
	}
}

func TestMigratedHysteriaHopSurvivesLinksJSONAndClash(t *testing.T) {
	client := matrixHysteriaClient("fixture@example.test")
	inbound := matrixHysteriaInbound(12345, client)
	normalizeSubscriptionFixture(t, inbound)
	base := &SubService{address: "fixture.example.test", remarkModel: "-ieo"}
	link, err := url.Parse(base.getLink(cloneMatrixInbound(inbound), client.Email))
	if err != nil {
		t.Fatal(err)
	}
	if link.Query().Get("mport") != "40000-45000" {
		t.Fatal("URI dropped migrated remote hopping ports")
	}
	assertHop := func(finalmask map[string]any) {
		t.Helper()
		for _, raw := range finalmask["udp"].([]any) {
			mask := raw.(map[string]any)
			if mask["type"] == "udphop" {
				settings := mask["settings"].(map[string]any)
				if settings["remotePorts"] != "40000-45000" || settings["interval"] != "5-10" || settings["mode"] != "intervalRemote" {
					t.Fatal("migrated UDP hop settings changed in subscription")
				}
				return
			}
		}
		t.Fatal("subscription dropped UDP hop mask")
	}
	var fm map[string]any
	if err := json.Unmarshal([]byte(link.Query().Get("fm")), &fm); err != nil {
		t.Fatal(err)
	}
	assertHop(fm)
	jsonService := &SubJsonService{SubService: base, configJson: map[string]any{}}
	configs := jsonService.getConfig(cloneMatrixInbound(inbound), client, base.address)
	if len(configs) != 1 {
		t.Fatalf("JSON configurations = %d, want 1", len(configs))
	}
	outbound := firstMatrixOutbound(t, configs[0])
	assertHop(outbound["streamSettings"].(map[string]any)["finalmask"].(map[string]any))
	clash := (&SubClashService{SubService: base}).getProxies(cloneMatrixInbound(inbound), client, base.address)
	if len(clash) != 1 || clash[0]["ports"] != "40000-45000" || clash[0]["obfs"] != "salamander" {
		t.Fatalf("Clash dropped migrated hopping or obfuscation: %#v", clash)
	}
}

func TestHysteriaHopPortsKeepsRemoteAndLocalModesDistinct(t *testing.T) {
	for _, tc := range []struct {
		mode  string
		ports any
		want  string
	}{
		{"intervalRemote", "20000-30000", "20000-30000"},
		{"intervalRemote", float64(23456), "23456"},
		{"intervalLocal,intervalRemote", "20000-30000", "20000-30000"},
		{"intervalLocal", "20000-30000", ""},
		{"perConnRemote", "20000-30000", ""},
	} {
		stream := map[string]any{"finalmask": map[string]any{"udp": []any{map[string]any{"type": "udphop", "settings": map[string]any{"mode": tc.mode, "remotePorts": tc.ports}}}}}
		if actual := hysteriaHopPorts(stream); actual != tc.want {
			t.Errorf("mode %s, ports %v: got %q, want %q", tc.mode, tc.ports, actual, tc.want)
		}
	}
}

func TestMigratedXMCUsesSupportedSubscriptionTargets(t *testing.T) {
	client := matrixClient("fixture@example.test")
	inbound := matrixInbound(model.VLESS, 12345, client)
	inbound.StreamSettings = `{"network":"tcp","security":"none","finalmask":{"tcp":[{"type":"xmc","settings":{"password":"fixture-password","profiles":[{"username":"Fixture_1","uuid":"00000000-0000-4000-8000-000000000001","texturesValue":"fixture-value","texturesSignature":"fixture-signature"}]}}]}}`
	normalizeSubscriptionFixture(t, inbound)
	base := &SubService{address: "fixture.example.test", remarkModel: "-ieo"}
	link, err := url.Parse(base.getLink(cloneMatrixInbound(inbound), client.Email))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(link.Query().Get("fm"), `"type":"xmc"`) {
		t.Fatal("URI dropped XMC")
	}
	configs := (&SubJsonService{SubService: base, configJson: map[string]any{}}).getConfig(cloneMatrixInbound(inbound), client, base.address)
	if len(configs) != 1 || !strings.Contains(string(configs[0]), `"xmc"`) {
		t.Fatal("Xray JSON dropped XMC")
	}
	if proxies := (&SubClashService{SubService: base}).getProxies(cloneMatrixInbound(inbound), client, base.address); len(proxies) != 0 {
		t.Fatal("unsupported XMC was silently omitted from a Clash node")
	}
}

func TestLegacyInsecureTLSUsesSupportedSubscriptionTargets(t *testing.T) {
	client := matrixHysteriaClient("fixture@example.test")
	inbound := matrixHysteriaInbound(12345, client)
	inbound.StreamSettings = strings.Replace(inbound.StreamSettings, `"allowInsecure":false`, `"allowInsecure":true`, 1)
	if !strings.Contains(inbound.StreamSettings, `"allowInsecure":true`) {
		t.Fatal("fixture must retain legacy insecure TLS")
	}
	normalizeSubscriptionFixture(t, inbound)
	base := &SubService{address: "fixture.example.test", remarkModel: "-ieo"}
	link, err := url.Parse(base.getLink(cloneMatrixInbound(inbound), client.Email))
	if err != nil {
		t.Fatal(err)
	}
	if link.Query().Get("insecure") != "1" {
		t.Fatal("Hysteria URI lost supported insecure TLS setting")
	}
	clash := (&SubClashService{SubService: base}).getProxies(cloneMatrixInbound(inbound), client, base.address)
	if len(clash) != 1 || clash[0]["skip-cert-verify"] != true {
		t.Fatal("Mihomo lost supported insecure TLS setting")
	}
	jsonService := &SubJsonService{SubService: base, configJson: map[string]any{}}
	if configs := jsonService.getConfig(cloneMatrixInbound(inbound), client, base.address); len(configs) != 0 {
		t.Fatal("Xray JSON silently changed the unsupported TLS verification behavior")
	}
}

func TestXrayJSONFiltersOnlyEffectiveInsecureTLS(t *testing.T) {
	base := &SubService{address: "fixture.example.test", remarkModel: "-ieo"}
	jsonService := &SubJsonService{SubService: base, configJson: map[string]any{}}
	for _, tc := range []struct {
		name, settings, forceTLS string
		want                     int
	}{
		{"legacy true", `{"allowInsecure":true}`, "same", 0},
		{"client true", `{"settings":{"allowInsecure":true}}`, "same", 0},
		{"client false", `{"settings":{"allowInsecure":false}}`, "same", 1},
		{"missing optional", `{}`, "same", 1},
		{"TLS explicitly disabled", `{"settings":{"allowInsecure":true}}`, "none", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := matrixClient("fixture@example.test")
			inbound := matrixInbound(model.VLESS, 12345, client)
			inbound.StreamSettings = `{"network":"tcp","security":"tls","tlsSettings":` + tc.settings + `,"externalProxy":[{"dest":"edge.example.test","port":443,"forceTls":"` + tc.forceTLS + `"}]}`
			before := inbound.StreamSettings
			if configs := jsonService.getConfig(inbound, client, base.address); len(configs) != tc.want {
				t.Fatalf("configurations = %d, want %d", len(configs), tc.want)
			}
			if inbound.StreamSettings != before {
				t.Fatal("subscription generation modified stored TLS settings")
			}
		})
	}
}

func TestSubscriptionsAcceptMissingAndMalformedOptionalStreamFields(t *testing.T) {
	base := &SubService{address: "fixture.example.test", remarkModel: "-ieo"}
	jsonService := &SubJsonService{SubService: base, configJson: map[string]any{}}
	for _, protocol := range []model.Protocol{model.VLESS, model.VMESS} {
		for _, raw := range []string{
			`{"network":"tcp","security":"reality","realitySettings":{"shortIds":[],"serverNames":[]}}`,
			`{"network":"tcp","security":"reality","realitySettings":{"shortIds":[42,"ab"],"serverNames":[false,"valid.example"]}}`,
			`{"network":"tcp","security":"reality"}`,
			`{"network":"tcp","security":"tls","tlsSettings":{"alpn":[42,"h2",false]}}`,
			`{"network":"ws","security":"none","wsSettings":{"headers":{"Host":[42,"valid.example"]}}}`,
			`{"network":"httpupgrade","security":"none"}`,
			`{"network":"grpc","security":"none","grpcSettings":{"multiMode":"wrong-type"}}`,
			`{"network":"xhttp","security":"none","xhttpSettings":{}}`,
			`{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"http","request":{"path":[],"headers":{"Host":false}}}}}`,
		} {
			t.Run(string(protocol)+raw, func(t *testing.T) {
				client := matrixClient("fixture@example.test")
				inbound := matrixInbound(protocol, 12345, client)
				inbound.StreamSettings = raw
				if link := base.getLink(cloneMatrixInbound(inbound), client.Email); link == "" {
					t.Fatal("optional fields prevented URI generation")
				}
				if configs := jsonService.getConfig(cloneMatrixInbound(inbound), client, base.address); len(configs) != 1 {
					t.Fatal("optional fields prevented JSON generation")
				}
			})
		}
	}
	for _, settings := range []string{`null`, `42`, `{}`} {
		client := matrixHysteriaClient("fixture@example.test")
		inbound := matrixHysteriaInbound(12345, client)
		inbound.StreamSettings = `{"network":"hysteria","security":"tls","hysteriaSettings":` + settings + `}`
		configs := jsonService.getConfig(inbound, client, base.address)
		if len(configs) != 1 || firstMatrixOutbound(t, configs[0])["protocol"] != "hysteria" {
			t.Fatal("missing Hysteria optional settings prevented JSON generation")
		}
	}
}

func TestClashSkipsMalformedExternalProxies(t *testing.T) {
	client := matrixClient("fixture@example.test")
	inbound := matrixInbound(model.VLESS, 12345, client)
	inbound.StreamSettings = `{"network":"tcp","security":"none","externalProxy":[null,1,{}, {"dest":42,"port":443}, {"dest":"invalid.example","port":"bad"}, {"dest":"valid.example","port":443}]}`
	service := &SubClashService{SubService: &SubService{address: "fixture.example.test", remarkModel: "-ieo"}}
	proxies := service.getProxies(inbound, client, "fixture.example.test")
	if len(proxies) != 1 || proxies[0]["server"] != "valid.example" {
		t.Fatal("invalid external proxies affected the valid endpoint")
	}
}
