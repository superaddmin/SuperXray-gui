package xraycompat

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/xtls/xray-core/infra/conf"
)

func TestRuntimeFreedomFinalRulesAcrossCoreVersions(t *testing.T) {
	input := `[{"protocol":"direct","settings":{"finalRules":[
		{"action":"Allow","blockDelay":9,"vendor":9007199254740993},
		{"action":"Block","delay":2,"blockDelay":"3-5","logLevel":"Warning"},
		{"action":"block","delay":0,"blockDelay":null}]}}]`
	wire, err := RuntimeOutbounds([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err := decode(string(wire), &got); err != nil {
		t.Fatal(err)
	}
	if err := decode(`[{"protocol":"direct","settings":{"finalRules":[
		{"action":"Allow","vendor":9007199254740993},{"action":"Block","blockDelay":"3-5","logLevel":"Warning"},
		{"action":"block","blockDelay":0}]}}]`, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected runtime rules: %s", wire)
	}
	if again, err := RuntimeOutbounds(wire); err != nil || string(again) != string(wire) {
		t.Fatalf("second projection changed rules: %s %v", again, err)
	}
	for _, rule := range []string{
		`{"action":"allow","delay":0}`, `{"action":"allow","logLevel":"none"}`,
		`{"action":"block","logLevel":[]}`, `{"action":"block","logLevel":"invalid"}`,
	} {
		_, err := RuntimeOutbounds([]byte(`[{"protocol":"freedom","settings":{"finalRules":[` + rule + `]}}]`))
		if err == nil || !strings.Contains(err.Error(), "finalRules[0]") {
			t.Fatalf("invalid rule was not located: %s %v", rule, err)
		}
	}
}

func TestRuntimeFreedomUsesCoreFieldMatching(t *testing.T) {
	raw := `[{"Protocol":"freedom","Settings":{"FinalRules":[
		{"Action":"Allow","BlockDelay":3,"note":9007199254740993},
		{"Action":"Block","BLOCKDELAY":1,"blockdelay":2,"Delay":9},
		{"Action":"Block","Delay":4,"LOGLEVEL":"debug"}]}}]`
	wire, err := RuntimeOutbounds([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err := decode(string(wire), &got); err != nil {
		t.Fatal(err)
	}
	if err := decode(`[{"Protocol":"freedom","settings":{"finalRules":[
		{"Action":"Allow","note":9007199254740993},
		{"Action":"Block","blockDelay":2},
		{"Action":"Block","blockDelay":4,"LOGLEVEL":"debug"}]}}]`, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("historical key spellings were not projected: %s", wire)
	}
}

func TestUpstreamLuaConflictsFailAtomically(t *testing.T) {
	for _, raw := range []string{
		`{"routing":{"script":"route.lua","rules":[{"outboundTag":"direct"}]}}`,
		`{"routing":{"script":"route.lua","domainStrategy":"AsIs"}}`,
		`{"dns":{"script":"dns.lua","disableFallback":true}}`,
		`{"dns":{"script":"dns.lua","disableFallbackIfMatch":true}}`,
		`{"dns":{"script":"dns.lua","enableParallelQuery":true}}`,
		`{"dns":{"script":"dns.lua","servers":[{"address":"localhost","domains":["example.test"]}]}}`,
		`{"dns":{"script":"dns.lua","servers":[{"address":"localhost","skipFallback":true}]}}`,
		`{"dns":{"script":"dns.lua","servers":[{"address":"localhost","finalQuery":true}]}}`,
	} {
		got, changed, err := NormalizeConfig(raw)
		if err == nil || !strings.Contains(err.Error(), "script") || changed || got != raw {
			t.Errorf("conflict must preserve original: %s => %s, %v, %v", raw, got, changed, err)
		}
	}
	valid := `{"routing":{"script":"route.lua","vendor":1},"dns":{"script":"dns.lua","servers":[{"id":"local","address":"localhost","domains":[],"skipFallback":false}]}}`
	if got, changed, err := NormalizeConfig(valid); err != nil || changed || got != valid {
		t.Fatal(got, changed, err)
	}
}

func TestUpstreamMigratedKCPMasksBuildInPinnedCore(t *testing.T) {
	for _, mask := range []string{
		`{"type":"mkcp-original"}`,
		`{"type":"mkcp-aes128gcm","settings":{"password":"fixture-seed"}}`,
		`{"type":"header-dns","settings":{"domain":"fixture.test"}}`,
		`{"type":"header-dtls"}`, `{"type":"header-srtp"}`, `{"type":"header-utp"}`, `{"type":"header-wechat"}`, `{"type":"header-wireguard"}`,
	} {
		raw := `{"network":"kcp","finalmask":{"udp":[` + mask + `]}}`
		got, changed, err := NormalizeStream(raw)
		if err != nil || !changed {
			t.Fatal(got, changed, err)
		}
		var stream conf.StreamConfig
		if err := json.Unmarshal([]byte(got), &stream); err != nil {
			t.Fatal(err)
		}
		if _, err := stream.Build(); err != nil {
			t.Fatalf("target core rejected %s: %v", got, err)
		}
	}
}

func TestUpstreamXDriveInboundRulesAndServiceAlias(t *testing.T) {
	stream := `{"network":"xdrive","xdriveSettings":{"service":"Google Drive","vendor":42}}`
	for _, tc := range []struct{ protocol, settings string }{
		{"vmess", `{}`},
		{"vless", `{"decryption":"none"}`},
		{"vless", `{"decryption":"","clients":[]}`},
		{"vless", `{"decryption":"fixture-encryption","clients":[{"flow":"xtls-rprx-vision"}]}`},
	} {
		settings, got, changed, err := NormalizeInbound(tc.protocol, tc.settings, stream)
		if err == nil || changed || settings != tc.settings || got != stream {
			t.Errorf("invalid XDRIVE accepted or mutated: %s, %s, %v", tc.protocol, tc.settings, err)
		}
	}
	_, got, changed, err := NormalizeInbound("vless", `{"decryption":"fixture-encryption","clients":[{"flow":""}]}`, stream)
	if err != nil || !changed || !strings.Contains(got, `"service":"google-drive"`) || !strings.Contains(got, `"vendor":42`) {
		t.Fatal(got, changed, err)
	}
}

func TestUpstreamXDriveOutboundChecksEveryUserAndMux(t *testing.T) {
	for _, settings := range []string{
		`{"encryption":"none"}`,
		`{"encryption":"fixture-encryption","flow":"xtls-rprx-vision"}`,
		`{"encryption":"fixture-encryption","testpre":1}`,
		`{"vnext":[{"users":[{"encryption":"fixture-encryption"},{"encryption":"none"}]}]}`,
	} {
		raw := `{"outbounds":[{"protocol":"vless","settings":` + settings + `,"streamSettings":{"network":"xdrive"},"mux":{"enabled":true}}]}`
		if got, changed, err := NormalizeConfig(raw); err == nil || changed || got != raw {
			t.Errorf("invalid account accepted: %s => %s, %v", raw, got, err)
		}
	}
	for _, mux := range []string{`null`, `{}`, `{"enabled":false}`, `{"enabled":true,"concurrency":-1}`, `{"enabled":true,"xudpConcurrency":-1}`, `{"enabled":true,"xudpProxyUDP443":"skip"}`} {
		raw := `{"outbounds":[{"protocol":"vless","settings":{"encryption":"fixture-encryption"},"streamSettings":{"network":"xdrive"},"mux":` + mux + `}]}`
		if _, _, err := NormalizeConfig(raw); err == nil {
			t.Errorf("invalid mux accepted: %s", mux)
		}
	}
	for _, suffix := range []string{`,"mux":{"enabled":true,"concurrency":0}`, ``} {
		settings := `{"encryption":"fixture-encryption"}`
		if suffix == "" {
			settings = `{"encryption":"fixture-encryption","reverse":{"tag":"reverse"}}`
		}
		raw := `{"outbounds":[{"protocol":"vless","settings":` + settings + `,"streamSettings":{"network":"xdrive"}` + suffix + `}]}`
		if _, _, err := NormalizeConfig(raw); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUpstreamLegacyKCPMasksPreserveExtensions(t *testing.T) {
	raw := `{"udp":[{"type":"mkcp-aes128gcm","settings":{"password":"fixture-seed","vendor":9}},{"type":"header-dns","settings":{"domain":"fixture.test"}},{"type":"custom","settings":{"keep":true}}]}`
	got, changed, err := NormalizeFinalMask(raw)
	if err != nil || !changed {
		t.Fatal(got, changed, err)
	}
	var fm map[string]any
	if err := json.Unmarshal([]byte(got), &fm); err != nil {
		t.Fatal(err)
	}
	masks := fm["udp"].([]any)
	first := masks[0].(map[string]any)
	if first["type"] != "mkcp-legacy" || first["settings"].(map[string]any)["value"] != "fixture-seed" || first["settings"].(map[string]any)["vendor"] != float64(9) {
		t.Fatal(got)
	}
	second := masks[1].(map[string]any)["settings"].(map[string]any)
	if second["header"] != "dns" || second["value"] != "fixture.test" || masks[2].(map[string]any)["type"] != "custom" {
		t.Fatal(got)
	}
	if again, changed, err := NormalizeFinalMask(got); err != nil || changed || again != got {
		t.Fatal(again, changed, err)
	}
	conflict := `{"udp":[{"type":"mkcp-aes128gcm","settings":{"password":"old","value":"new"}}]}`
	if got, changed, err := NormalizeFinalMask(conflict); err == nil || changed || got != conflict {
		t.Fatal(got, changed, err)
	}
}

func TestUpstreamLegacyEmptyAESPasswordNeverBecomesPlaintext(t *testing.T) {
	for _, settings := range []string{`{}`, `{"password":""}`, `{"value":"new"}`} {
		raw := `{"udp":[{"type":"mkcp-aes128gcm","settings":` + settings + `}]}`
		if got, changed, err := NormalizeFinalMask(raw); err == nil || changed || got != raw {
			t.Fatal(got, changed, err)
		}
	}
}

func TestUpstreamXDriveMuxMatchesCoreFieldBounds(t *testing.T) {
	for _, raw := range []string{`{"enabled":true,"concurrency":32768}`, `{"enabled":true,"xudpConcurrency":32768}`, `{"enabled":true,"xudpProxyUDP443":"invalid"}`} {
		if XDriveMuxValid(raw) {
			t.Fatalf("invalid core mux accepted: %s", raw)
		}
	}
	for _, raw := range []string{`{"enabled":true,"concurrency":32767}`, `{"enabled":true,"xudpConcurrency":32767,"xudpProxyUDP443":"allow"}`} {
		if !XDriveMuxValid(raw) {
			t.Fatalf("valid core mux rejected: %s", raw)
		}
	}
}
