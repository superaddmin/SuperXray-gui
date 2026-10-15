package xraycompat

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeConfigPreservesUnknownAndIsIdempotent(t *testing.T) {
	raw := `{"vendor":{"large":9007199254740993},"outbounds":[{"protocol":"wireguard","settings":{"domainStrategy":"ForceIPv4","remoteDNS":["local"],"vendor":true},"streamSettings":{"method":"xhttp","xhttpSettings":{"extra":{"sessionPlacement":"header","sessionKey":"old","vendor":9}}},"proxySettings":{"tag":"proxy"}}]}`
	got, changed, err := NormalizeConfig(raw)
	if err != nil || !changed {
		t.Fatalf("normalize: %v %v", changed, err)
	}
	if !strings.Contains(got, "9007199254740993") || !strings.Contains(got, `"sessionIDPlacement":"header"`) || !strings.Contains(got, `"dialerProxy":"proxy"`) || strings.Contains(got, `"remoteDNS"`) {
		t.Fatalf("lost semantics: %s", got)
	}
	second, changed, err := NormalizeConfig(got)
	if err != nil || changed || second != got {
		t.Fatalf("not idempotent: %v %v %s", changed, err, second)
	}
}

func TestNormalizeStreamLegacyMasks(t *testing.T) {
	raw := `{"network":"hysteria","finalmask":{"quicParams":{"udpHop":{"ports":"1000-2000","interval":30},"vendor":true},"udp":[{"type":"xdns","settings":{"domains":["example.test:a"],"resolvers":["example.test:a+udp://127.0.0.1:53"],"vendor":42}}]}}`
	got, changed, err := NormalizeStream(raw)
	if err != nil || !changed {
		t.Fatal(changed, err)
	}
	for _, want := range []string{`"names":["example.test"]`, `"types":[1]`, `"edns0":1232`, `"addrs":["udp://127.0.0.1:53"]`, `"mode":"intervalRemote"`, `"remotePorts":"1000-2000"`, `"vendor":42`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
	if _, changed, err := NormalizeStream(got); err != nil || changed {
		t.Fatalf("second pass: %v %v", changed, err)
	}
}

func TestNormalizeTLSCertificatePinsPreservesTrustAndUnknownFields(t *testing.T) {
	raw := `{"security":"tls","tlsSettings":{"pinnedPeerCertSha256":["` + strings.Repeat("01", 32) + `","` + strings.Repeat("02", 32) + `"],"vendor":9007199254740993}}`
	got, changed, err := NormalizeStream(raw)
	if err != nil || !changed || !strings.Contains(got, strings.Repeat("01", 32)+","+strings.Repeat("02", 32)) || !strings.Contains(got, "9007199254740993") {
		t.Fatalf("TLS trust migration: %s, %v, %v", got, changed, err)
	}
	if second, changed, err := NormalizeStream(got); err != nil || changed || second != got {
		t.Fatalf("TLS migration is not idempotent: %v", err)
	}
	for _, raw := range []string{`{"tlsSettings":{"pinnedPeerCertSha256":["good",1]}}`, `{"tlsSettings":{"allowInsecure":true}}`} {
		if got, changed, err := NormalizeStream(raw); err == nil || changed || got != raw {
			t.Fatalf("unsafe TLS migration did not fail atomically: %s %v", got, err)
		}
	}
}

func TestNormalizeRejectsLossyOrInvalidConfigurationAtomically(t *testing.T) {
	cases := []string{
		`{"network":"tcp","security":"reality","finalmask":{"tcp":[{"type":"fragment"}]}}`,
		`{"finalmask":{"tcp":[{"type":"xmc","settings":{}}]}}`,
		`{"finalmask":{"udp":[{"type":"salamander","settings":{"password":"x"}}]}}`,
		`{"network":"hysteria","hysteriaSettings":{"version":1}}`,
		`{"method":"xhttp","xhttpSettings":{"sessionIDLength":"32-0"}}`,
		`{"finalmask":{"udp":[{"type":"xdns","settings":{"domains":["broken:badtype"]}}]}}`,
		`null`, `[]`, `{} {}`,
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			got, changed, err := NormalizeStream(raw)
			if err == nil || changed || got != raw {
				t.Fatalf("not rejected atomically: %q %v %v", got, changed, err)
			}
		})
	}
}

func TestNormalizePartialTemplateDoesNotInventMissingFields(t *testing.T) {
	for _, raw := range []string{
		`{"inbounds":[{"protocol":"vmess","settings":{"clients":[{"security":"none"}]}}]}`,
		`{"inbounds":[{"protocol":"vless","streamSettings":{"method":"tcp"}}]}`,
		`{"inbounds":[{"protocol":"vmess","settings":"{\"clients\":[{\"security\":\"none\"}]}"}]}`,
	} {
		got, changed, err := NormalizeConfig(raw)
		if err != nil || !changed {
			t.Fatalf("normalize %s: %v %v", raw, changed, err)
		}
		var before, after map[string]any
		_ = json.Unmarshal([]byte(raw), &before)
		_ = json.Unmarshal([]byte(got), &after)
		original := before["inbounds"].([]any)[0].(map[string]any)
		migrated := after["inbounds"].([]any)[0].(map[string]any)
		for _, key := range []string{"settings", "streamSettings"} {
			if _, exists := original[key]; !exists {
				if _, exists := migrated[key]; exists {
					t.Fatalf("invented %s in %s", key, got)
				}
			}
			if _, text := original[key].(string); text {
				if _, text := migrated[key].(string); !text {
					t.Fatalf("changed JSON text representation: %s", got)
				}
			}
		}
		if second, changed, err := NormalizeConfig(got); err != nil || changed || second != got {
			t.Fatal(second, changed, err)
		}
	}
}

func TestSessionLengthAcceptsOnlyPositiveInt32Values(t *testing.T) {
	for _, value := range []string{`16`, `"16"`, `"32-16"`, `2147483647`, `"1-2147483647"`} {
		raw := `{"xhttpSettings":{"sessionIDLength":` + value + `}}`
		if _, _, err := NormalizeStream(raw); err != nil {
			t.Errorf("valid %s: %v", value, err)
		}
	}
	for _, value := range []string{`0`, `-1`, `1.5`, `2147483648`, `"2147483648"`, `"1-2147483648"`, `"1-0"`, `true`, `null`, `[]`, `{}`} {
		raw := `{"method":"xhttp","xhttpSettings":{"extra":{"sessionIDLength":` + value + `}}}`
		if got, changed, err := NormalizeStream(raw); err == nil || changed || got != raw {
			t.Errorf("invalid %s: %s %v %v", value, got, changed, err)
		}
	}
}

func TestNormalizeWireguardAndNullInboundSettings(t *testing.T) {
	settings, _, changed, err := NormalizeInbound("wireguard", `{"workers":4,"vendor":true}`, "")
	if err != nil || !changed || settings != `{"vendor":true}` {
		t.Fatal(settings, changed, err)
	}
	got, changed, err := NormalizeConfig(`{"outbounds":[{"protocol":"wireguard","settings":{"workers":4,"vendor":true}}]}`)
	if err != nil || !changed || strings.Contains(got, "workers") || !strings.Contains(got, `"vendor":true`) {
		t.Fatal(got, changed, err)
	}
	settings, stream, changed, err := NormalizeInbound("hysteria", `null`, `{}`)
	if err == nil || changed || settings != `null` || stream != `{}` {
		t.Fatal(settings, stream, changed, err)
	}
}

func TestUDPHopMigrationPreservesExtensionsAndRejectsConflicts(t *testing.T) {
	raw := `{"quicParams":{"udpHop":{"ports":"1000-2000","interval":30,"vendor":{"n":9007199254740993}}},"udp":[{"type":"custom","settings":{"keep":true}}]}`
	got, changed, err := NormalizeFinalMask(raw)
	if err != nil || !changed || !strings.Contains(got, `"vendor":{"n":9007199254740993}`) || !strings.Contains(got, `"interval":30`) {
		t.Fatal(got, changed, err)
	}
	for _, raw := range []string{
		`{"quicParams":{"udpHop":true}}`,
		`{"quicParams":{"udpHop":{"interval":30}}}`,
		`{"quicParams":{"udpHop":{"ports":"1000","vendor":1}},"udp":[{"type":"udphop","settings":{"mode":"intervalRemote","remotePorts":"1000","vendor":2}}]}`,
		`{"udp":[{"type":"udphop","settings":{"mode":"intervalRemote","remotePorts":"1000"}},{"type":"udphop","settings":{"mode":"intervalRemote","remotePorts":"2000"}}]}`,
	} {
		if got, changed, err := NormalizeFinalMask(raw); err == nil || changed || got != raw {
			t.Errorf("conflict lost: %s %v %v", got, changed, err)
		}
	}
	raw = `{"udp":[{"type":"udphop","settings":{"mode":"intervalRemote","remotePorts":"1000","vendor":1}},{"type":"custom"},{"settings":{"vendor":1,"remotePorts":"1000","mode":"intervalRemote"},"type":"udphop"}]}`
	got, changed, err = NormalizeFinalMask(raw)
	if err != nil || !changed || strings.Count(got, `"type":"udphop"`) != 1 {
		t.Fatal(got, changed, err)
	}
	if second, changed, err := NormalizeFinalMask(got); err != nil || changed || second != got {
		t.Fatal(second, changed, err)
	}
}

func TestXdnsResolverKeepsDifferentRecordTypes(t *testing.T) {
	raw := `{"udp":[{"type":"xdns","settings":{"domains":["example.test:a"],"resolvers":["example.test:txt+udp://127.0.0.1:53"]}}]}`
	got, changed, err := NormalizeFinalMask(raw)
	if err != nil || !changed || !strings.Contains(got, `"types":[1]`) || !strings.Contains(got, `"types":[16]`) {
		t.Fatal(got, changed, err)
	}
}

func TestXdnsMixedDomainFormsKeepNamesAndDoNotOverwriteInvalidValues(t *testing.T) {
	raw := `{"udp":[{"type":"xdns","settings":{"domains":[{"name":"legacy.test","names":["new.test"],"vendor":true},{"names":["same.test"],"types":[1],"edns0":1232}],"resolvers":["same.test:a+udp://127.0.0.1:53"]}}]}`
	got, _, err := NormalizeFinalMask(raw)
	if err != nil || !strings.Contains(got, `"names":["new.test","legacy.test"]`) || strings.Count(got, `"names":["same.test"]`) != 1 {
		t.Fatal(got, err)
	}
	raw = `{"udp":[{"type":"xdns","settings":{"domains":{"vendor":true},"resolvers":["same.test:a+udp://127.0.0.1:53"]}}]}`
	if got, changed, err := NormalizeFinalMask(raw); err == nil || changed || got != raw {
		t.Fatal(got, changed, err)
	}
}

func TestNormalizeInboundKeepsClientExtensions(t *testing.T) {
	raw := `{"clients":[{"id":"test","security":"none","vendor":{"n":9007199254740993}}]}`
	settings, stream, changed, err := NormalizeInbound("vmess", raw, `{}`)
	if err != nil || !changed || stream != `{}` || !strings.Contains(settings, `"security":"auto"`) || !strings.Contains(settings, "9007199254740993") {
		t.Fatal(settings, stream, changed, err)
	}
	bad := `{"method":"none","clients":[]}`
	settings, _, changed, err = NormalizeInbound("shadowsocks", bad, `{}`)
	if err == nil || changed || settings != bad {
		t.Fatal("removed cipher must require client coordination")
	}
}

func TestDNSLegacyRulesKeepQTypeZero(t *testing.T) {
	got, _, err := NormalizeConfig(`{"outbounds":[{"protocol":"dns","settings":{"nonIPQuery":"drop","blockTypes":[0,16]}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(got), &cfg); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"qType":"0,16"`) || !strings.Contains(got, `"action":"hijack"`) {
		t.Fatal(got)
	}
}
