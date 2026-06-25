package sub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/web/middleware"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"
)

func TestSubscriptionOutputMatrixCoversLinksJSONAndClash(t *testing.T) {
	host := "vpn.example.com"
	subService := &SubService{
		address:        host,
		remarkModel:    "-ieo",
		inboundService: service.InboundService{},
	}
	jsonService := &SubJsonService{
		configJson:     map[string]any{},
		inboundService: service.InboundService{},
		SubService:     subService,
	}
	clashService := &SubClashService{
		inboundService: service.InboundService{},
		SubService:     subService,
	}

	cases := []struct {
		name         string
		inbound      *model.Inbound
		client       model.Client
		linkContains []string
		jsonProtocol string
		clashType    string
	}{
		{
			name:         "vmess",
			inbound:      matrixInbound(model.VMESS, 12001, matrixClient("matrix-vmess@example")),
			client:       matrixClient("matrix-vmess@example"),
			linkContains: []string{"vmess://"},
			jsonProtocol: "vmess",
			clashType:    "vmess",
		},
		{
			name:         "vless",
			inbound:      matrixInbound(model.VLESS, 12002, matrixClient("matrix-vless@example")),
			client:       matrixClient("matrix-vless@example"),
			linkContains: []string{"vless://", "type=tcp"},
			jsonProtocol: "vless",
			clashType:    "vless",
		},
		{
			name:         "trojan",
			inbound:      matrixInbound(model.Trojan, 12003, matrixPasswordClient("matrix-trojan@example")),
			client:       matrixPasswordClient("matrix-trojan@example"),
			linkContains: []string{"trojan://", "type=tcp"},
			jsonProtocol: "trojan",
			clashType:    "trojan",
		},
		{
			name:         "shadowsocks",
			inbound:      matrixShadowsocksInbound(12004, matrixPasswordClient("matrix-ss@example")),
			client:       matrixPasswordClient("matrix-ss@example"),
			linkContains: []string{"ss://", "type=tcp"},
			jsonProtocol: "shadowsocks",
			clashType:    "ss",
		},
		{
			name:         "hysteria2",
			inbound:      matrixHysteriaInbound(12005, matrixHysteriaClient("matrix-hy2@example")),
			client:       matrixHysteriaClient("matrix-hy2@example"),
			linkContains: []string{"hysteria2://matrix-hy2-auth@", "obfs=salamander", "obfs-password=matrix-obfs-pass", "mport=40000-45000"},
			jsonProtocol: "hysteria",
			clashType:    "hysteria2",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			link := subService.getLink(cloneMatrixInbound(tc.inbound), tc.client.Email)
			for _, want := range tc.linkContains {
				if !strings.Contains(link, want) {
					t.Fatalf("subscription link for %s missing %q:\n%s", tc.name, want, link)
				}
			}

			configs := jsonService.getConfig(cloneMatrixInbound(tc.inbound), tc.client, host)
			if len(configs) == 0 {
				t.Fatalf("json subscription for %s returned no configs", tc.name)
			}
			outbound := firstMatrixOutbound(t, configs[0])
			if outbound["protocol"] != tc.jsonProtocol {
				t.Fatalf("json protocol for %s = %v, want %s", tc.name, outbound["protocol"], tc.jsonProtocol)
			}
			if tc.name == "hysteria2" {
				streamSettings, ok := outbound["streamSettings"].(map[string]any)
				if !ok || streamSettings["finalmask"] == nil {
					t.Fatalf("hysteria2 json streamSettings missing finalmask obfs block: %#v", outbound["streamSettings"])
				}
				tlsSettings, ok := streamSettings["tlsSettings"].(map[string]any)
				if !ok {
					t.Fatalf("hysteria2 json streamSettings missing tlsSettings: %#v", streamSettings)
				}
				if tlsSettings["fingerprint"] == "chrome" {
					t.Fatalf("hysteria2 json streamSettings should not fall back to chrome fingerprint: %#v", tlsSettings)
				}
				if fp, ok := tlsSettings["fingerprint"].(string); ok && strings.TrimSpace(fp) != "" {
					t.Fatalf("hysteria2 json streamSettings exported non-empty fingerprint %q", fp)
				}
				if strings.Contains(link, "fp=") {
					t.Fatalf("hysteria2 subscription link exported empty uTLS fingerprint: %s", link)
				}
			}

			proxies := clashService.getProxies(cloneMatrixInbound(tc.inbound), tc.client, host)
			if len(proxies) == 0 {
				t.Fatalf("clash subscription for %s returned no proxies", tc.name)
			}
			if proxies[0]["type"] != tc.clashType {
				t.Fatalf("clash type for %s = %v, want %s", tc.name, proxies[0]["type"], tc.clashType)
			}
			if tc.name == "hysteria2" && proxies[0]["ports"] != "40000-45000" {
				t.Fatalf("clash hysteria2 ports = %v, want 40000-45000", proxies[0]["ports"])
			}
		})
	}
}

func TestProxyAccountSubscriptionsIncludeHTTPAndSOCKS5Outputs(t *testing.T) {
	setupSubscriptionDiagnosticDB(t)
	const subID = "matrix-proxy-sub"
	host := "vpn.example.com"
	inbounds := []*model.Inbound{
		matrixProxyAccountInbound(model.HTTP, 13101, map[string]any{
			"user":  "http-user",
			"pass":  "http/pass with space",
			"subId": subID,
		}),
		matrixProxyAccountInbound(model.Mixed, 13102, map[string]any{
			"user":  "socks-user",
			"pass":  "socks/pass with space",
			"subId": subID,
		}),
	}
	for _, inbound := range inbounds {
		if err := database.GetDB().Create(inbound).Error; err != nil {
			t.Fatalf("create %s inbound failed: %v", inbound.Protocol, err)
		}
	}

	subService := &SubService{
		remarkModel:    "-ieo",
		inboundService: service.InboundService{},
	}
	links, _, _, err := subService.GetSubs(subID, host)
	if err != nil {
		t.Fatalf("GetSubs returned error: %v", err)
	}
	joinedLinks := strings.Join(links, "\n")
	for _, want := range []string{
		"http://http-user:http%2Fpass%20with%20space@vpn.example.com:13101#matrix-http",
		"socks5://socks-user:socks%2Fpass%20with%20space@vpn.example.com:13102#matrix-mixed",
	} {
		if !strings.Contains(joinedLinks, want) {
			t.Fatalf("proxy subscription links missing %q:\n%s", want, joinedLinks)
		}
	}

	jsonService := &SubJsonService{
		configJson: map[string]any{},
		SubService: subService,
	}
	jsonText, _, err := jsonService.GetJson(subID, host)
	if err != nil {
		t.Fatalf("GetJson returned error: %v", err)
	}
	outbounds := proxyMatrixOutboundsByProtocol(t, jsonText)
	assertProxyMatrixServerUser(t, outbounds["http"], "http-user", "http/pass with space")
	assertProxyMatrixServerUser(t, outbounds["socks"], "socks-user", "socks/pass with space")

	clashService := &SubClashService{SubService: subService}
	clashText, _, err := clashService.GetClash(subID, host)
	if err != nil {
		t.Fatalf("GetClash returned error: %v", err)
	}
	for _, want := range []string{
		"type: http",
		"type: socks5",
		"username: http-user",
		"password: http/pass with space",
		"username: socks-user",
		"password: socks/pass with space",
	} {
		if !strings.Contains(clashText, want) {
			t.Fatalf("clash proxy output missing %q:\n%s", want, clashText)
		}
	}
}

func TestProxyAccountSubscriptionMatchesTopLevelSubID(t *testing.T) {
	setupSubscriptionDiagnosticDB(t)
	const subID = "matrix-top-level-sub"
	inbound := matrixProxyAccountInbound(model.HTTP, 13103, map[string]any{
		"user": "top-user",
		"pass": "top-pass",
	})
	var settings map[string]any
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		t.Fatalf("proxy settings invalid: %v", err)
	}
	settings["subId"] = subID
	settingsJSON, _ := json.Marshal(settings)
	inbound.Settings = string(settingsJSON)
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create top-level subId inbound failed: %v", err)
	}

	subService := &SubService{remarkModel: "-ieo"}
	links, _, _, err := subService.GetSubs(subID, "vpn.example")
	if err != nil {
		t.Fatalf("GetSubs returned error: %v", err)
	}
	if len(links) != 1 || !strings.Contains(links[0], "http://top-user:top-pass@vpn.example:13103") {
		t.Fatalf("top-level subId proxy links = %#v, want one HTTP proxy link", links)
	}
}

func TestProxyAccountSubscriptionSupportsUnauthenticatedTopLevelSubID(t *testing.T) {
	setupSubscriptionDiagnosticDB(t)
	const subID = "matrix-no-auth-sub"
	inbound := matrixInboundWithSettings(model.Mixed, 13104, map[string]any{
		"auth":  "noauth",
		"subId": subID,
		"udp":   false,
	}, map[string]any{})
	inbound.Enable = true
	inbound.Tag = "matrix-proxy-noauth"
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create no-auth proxy inbound failed: %v", err)
	}

	subService := &SubService{remarkModel: "-ieo"}
	links, _, _, err := subService.GetSubs(subID, "vpn.example")
	if err != nil {
		t.Fatalf("GetSubs returned error: %v", err)
	}
	if len(links) != 1 || !strings.Contains(links[0], "socks5://vpn.example:13104") {
		t.Fatalf("no-auth proxy links = %#v, want one SOCKS5 link without userinfo", links)
	}
	if strings.Contains(links[0], "@vpn.example") {
		t.Fatalf("no-auth proxy link should not include userinfo: %s", links[0])
	}
}

func TestSubClashVlessEncryptionNoneDoesNotEmitPacketEncoding(t *testing.T) {
	client := matrixClient("matrix-vless-packet@example")
	inbound := matrixInbound(model.VLESS, 12009, client)
	service := &SubClashService{SubService: &SubService{remarkModel: "-ieo"}}

	proxy := service.buildProxy(inbound, client, matrixTCPStream(), "")
	if proxy == nil {
		t.Fatal("buildProxy returned nil")
	}
	if value, ok := proxy["packet-encoding"]; ok {
		t.Fatalf("packet-encoding = %v, want field omitted for VLESS encryption none", value)
	}
}

func TestSubClashVlessPacketEncodingsEmitPacketEncoding(t *testing.T) {
	for _, encryption := range []string{"packetaddr", "xudp"} {
		t.Run(encryption, func(t *testing.T) {
			client := matrixClient("matrix-vless-" + encryption + "@example")
			inbound := matrixInboundWithSettings(model.VLESS, 12010, map[string]any{
				"decryption": "none",
				"encryption": encryption,
				"clients":    []model.Client{client},
			}, matrixTCPStream())
			service := &SubClashService{SubService: &SubService{remarkModel: "-ieo"}}

			proxy := service.buildProxy(inbound, client, matrixTCPStream(), "")
			if proxy == nil {
				t.Fatal("buildProxy returned nil")
			}
			if proxy["packet-encoding"] != encryption {
				t.Fatalf("packet-encoding = %v, want %s", proxy["packet-encoding"], encryption)
			}
		})
	}
}

func TestPublicInboundDoesNotUseFallbackMaster(t *testing.T) {
	setupSubscriptionDiagnosticDB(t)
	client := matrixClient("matrix-public@example")
	master := matrixInboundWithSettings(model.VLESS, 443, matrixSettingsWithFallback(client, "@fallback"), map[string]any{
		"network":  "tcp",
		"security": "tls",
		"tlsSettings": map[string]any{
			"serverName": "master.example",
		},
	})
	master.Listen = "master.example"
	if err := database.GetDB().Create(master).Error; err != nil {
		t.Fatalf("create fallback master failed: %v", err)
	}

	publicInbound := matrixInboundWithSettings(model.VLESS, 12010, matrixSettingsWithFallback(client, "@fallback"), map[string]any{
		"network":  "tcp",
		"security": "none",
	})
	publicInbound.Listen = "203.0.113.10"
	service := &SubService{address: "panel.example", remarkModel: "-ieo"}

	link := service.genVlessLink(publicInbound, client.Email)

	if !strings.Contains(link, "@203.0.113.10:12010") {
		t.Fatalf("public inbound link should use its own listen/port, got %q", link)
	}
	if strings.Contains(link, "master.example") || strings.Contains(link, ":443") || strings.Contains(link, "security=tls") {
		t.Fatalf("public inbound link leaked fallback master settings: %q", link)
	}
}

func TestSubClashBuildRulesUsesConfiguredRoutingRules(t *testing.T) {
	service := NewSubClashService(&SubService{}).WithRoutingRules(`
# comments and empty lines are ignored
DOMAIN-SUFFIX,example.com,DIRECT
IP-CIDR,10.0.0.0/8,DIRECT
invalid-without-comma
`)

	rules := service.buildClashRules()

	want := []string{
		"DOMAIN-SUFFIX,example.com,DIRECT",
		"IP-CIDR,10.0.0.0/8,DIRECT",
		"MATCH,PROXY",
	}
	if strings.Join(rules, "\n") != strings.Join(want, "\n") {
		t.Fatalf("rules = %#v, want %#v", rules, want)
	}
}

func TestSubClashBuildRulesIgnoresConfiguredRoutingRulesWhenDisabled(t *testing.T) {
	service := NewSubClashService(&SubService{}).WithRoutingSettings(false, `
DOMAIN-SUFFIX,example.com,DIRECT
IP-CIDR,10.0.0.0/8,DIRECT
`)

	rules := service.buildClashRules()

	want := []string{"MATCH,PROXY"}
	if strings.Join(rules, "\n") != strings.Join(want, "\n") {
		t.Fatalf("rules = %#v, want %#v", rules, want)
	}
}

func TestSubClashBuildRulesAlwaysKeepsMatchProxyFallback(t *testing.T) {
	service := NewSubClashService(&SubService{}).WithRoutingRules("MATCH,DIRECT\r\nmatch,proxy")

	rules := service.buildClashRules()

	want := []string{
		"MATCH,DIRECT",
		"match,proxy",
		"MATCH,PROXY",
	}
	if strings.Join(rules, "\n") != strings.Join(want, "\n") {
		t.Fatalf("rules = %#v, want %#v", rules, want)
	}
}

func TestSubClashVlessRealityEmitsClientFingerprint(t *testing.T) {
	client := matrixClient("matrix-vless-reality@example")
	client.Flow = "xtls-rprx-vision"
	inbound := matrixInboundWithSettings(model.VLESS, 12011, map[string]any{
		"decryption": "none",
		"encryption": "none",
		"clients":    []model.Client{client},
	}, map[string]any{
		"network":  "tcp",
		"security": "reality",
		"realitySettings": map[string]any{
			"serverNames": []any{"www.cloudflare.com"},
			"shortIds":    []any{"0123456789abcdef"},
			"settings": map[string]any{
				"publicKey":   "matrix-public-key",
				"fingerprint": "chrome",
			},
		},
	})
	service := &SubClashService{SubService: &SubService{remarkModel: "-ieo"}}

	proxies := service.getProxies(inbound, client, "vpn.example")
	if len(proxies) != 1 {
		t.Fatalf("getProxies returned %d proxies, want 1", len(proxies))
	}
	proxy := proxies[0]
	if proxy["client-fingerprint"] != "chrome" {
		t.Fatalf("client-fingerprint = %v, want chrome", proxy["client-fingerprint"])
	}
	if proxy["flow"] != "xtls-rprx-vision" {
		t.Fatalf("flow = %v, want xtls-rprx-vision", proxy["flow"])
	}
	if _, ok := proxy["packet-encoding"]; ok {
		t.Fatalf("packet-encoding should be omitted for VLESS encryption none: %#v", proxy)
	}
}

func TestSubClashApplySecurityAcceptsRawRealitySettings(t *testing.T) {
	service := &SubClashService{SubService: &SubService{remarkModel: "-ieo"}}
	proxy := map[string]any{"type": "vless"}
	stream := map[string]any{
		"security": "reality",
		"realitySettings": map[string]any{
			"serverNames": []any{"www.cloudflare.com"},
			"shortIds":    []any{"0123456789abcdef"},
			"settings": map[string]any{
				"publicKey":   "matrix-public-key",
				"fingerprint": "chrome",
			},
		},
	}

	if !service.applySecurity(proxy, "reality", stream) {
		t.Fatal("applySecurity returned false")
	}
	if proxy["client-fingerprint"] != "chrome" {
		t.Fatalf("client-fingerprint = %v, want chrome", proxy["client-fingerprint"])
	}
	if proxy["servername"] != "www.cloudflare.com" {
		t.Fatalf("servername = %v, want www.cloudflare.com", proxy["servername"])
	}
	realityOpts, ok := proxy["reality-opts"].(map[string]any)
	if !ok {
		t.Fatalf("reality-opts missing or invalid: %#v", proxy["reality-opts"])
	}
	if realityOpts["public-key"] != "matrix-public-key" {
		t.Fatalf("public-key = %v, want matrix-public-key", realityOpts["public-key"])
	}
	if realityOpts["short-id"] != "0123456789abcdef" {
		t.Fatalf("short-id = %v, want 0123456789abcdef", realityOpts["short-id"])
	}
}

func TestSubscriptionOutputMatrixCoversWireGuardPeerOutputs(t *testing.T) {
	host := "vpn.example"
	inbound := wireguardInboundForTest(`[{
		"email": "matrix-wg@example",
		"enable": true,
		"subId": "matrix-sub",
		"privateKey": "` + wireguardTestPeerPrivate + `",
		"publicKey": "` + wireguardTestPeerPublic + `",
		"preSharedKey": "` + wireguardTestPSK + `",
		"allowedIPs": ["10.0.0.2/32"],
		"keepAlive": 25
	}]`)
	peers, err := wireguardPeersBySubID(inbound, "matrix-sub")
	if err != nil {
		t.Fatalf("wireguardPeersBySubID returned error: %v", err)
	}
	if len(peers) != 1 {
		t.Fatalf("wireguard peer matrix returned %d peers, want 1", len(peers))
	}

	subService := &SubService{address: host, remarkModel: "-ieo"}
	jsonService := &SubJsonService{}
	clashService := &SubClashService{SubService: subService}

	link := subService.genWireguardConfig(inbound, peers[0])
	for _, want := range []string{"[Interface]", "[Peer]", "Endpoint = vpn.example:51820"} {
		if !strings.Contains(link, want) {
			t.Fatalf("wireguard subscription config missing %q:\n%s", want, link)
		}
	}

	outbound := parseMatrixOutbound(t, jsonService.genWireguard(inbound, peers[0], host))
	if outbound["protocol"] != "wireguard" {
		t.Fatalf("wireguard json protocol = %v, want wireguard", outbound["protocol"])
	}

	proxy := clashService.buildWireguardProxy(inbound, peers[0], host, "")
	if proxy["type"] != "wireguard" {
		t.Fatalf("wireguard clash type = %v, want wireguard", proxy["type"])
	}
}

func matrixClient(email string) model.Client {
	return model.Client{
		ID:       "11111111-1111-4111-8111-111111111111",
		Security: "auto",
		Email:    email,
		Enable:   true,
		SubID:    "matrix-sub",
	}
}

func matrixPasswordClient(email string) model.Client {
	client := matrixClient(email)
	client.Password = "matrix-password"
	return client
}

func matrixHysteriaClient(email string) model.Client {
	client := matrixClient(email)
	client.Auth = "matrix-hy2-auth"
	return client
}

func matrixInbound(protocol model.Protocol, port int, client model.Client) *model.Inbound {
	settings := matrixSettingsWithFallback(client, "")
	if protocol == model.VLESS {
		settings["decryption"] = "none"
		settings["encryption"] = "none"
	}
	return matrixInboundWithSettings(protocol, port, settings, matrixTCPStream())
}

func matrixSettingsWithFallback(client model.Client, fallbackDest string) map[string]any {
	settings := map[string]any{
		"clients": []model.Client{client},
	}
	if fallbackDest != "" {
		settings["fallbacks"] = []map[string]any{{"dest": fallbackDest}}
	}
	return settings
}

func matrixShadowsocksInbound(port int, client model.Client) *model.Inbound {
	return matrixInboundWithSettings(model.Shadowsocks, port, map[string]any{
		"method":  "chacha20-ietf-poly1305",
		"clients": []model.Client{client},
	}, matrixTCPStream())
}

func matrixHysteriaInbound(port int, client model.Client) *model.Inbound {
	return matrixInboundWithSettings(model.Hysteria, port, map[string]any{
		"version": 2,
		"clients": []model.Client{client},
	}, map[string]any{
		"network":  "hysteria",
		"security": "tls",
		"hysteriaSettings": map[string]any{
			"udpIdleTimeout": 30,
		},
		"tlsSettings": map[string]any{
			"serverName": "hy2.example",
			"alpn":       []string{"h3"},
			"settings": map[string]any{
				"allowInsecure": true,
				"fingerprint":   "",
			},
		},
		"finalmask": map[string]any{
			"udp": []map[string]any{{
				"type": "salamander",
				"settings": map[string]any{
					"password": "matrix-obfs-pass",
				},
			}},
			"quicParams": map[string]any{
				"udpHop": map[string]any{
					"ports":    "40000-45000",
					"interval": "5-10",
				},
			},
		},
	})
}

func matrixProxyAccountInbound(protocol model.Protocol, port int, account map[string]any) *model.Inbound {
	settings := map[string]any{
		"accounts": []map[string]any{account},
	}
	if protocol == model.HTTP {
		settings["allowTransparent"] = false
	}
	if protocol == model.Mixed {
		settings["auth"] = "password"
		settings["udp"] = true
		settings["ip"] = "127.0.0.1"
	}
	inbound := matrixInboundWithSettings(protocol, port, settings, map[string]any{})
	inbound.Enable = true
	inbound.Tag = "matrix-proxy-" + string(protocol)
	return inbound
}

func matrixInboundWithSettings(protocol model.Protocol, port int, settings map[string]any, stream map[string]any) *model.Inbound {
	settingsJSON, _ := json.Marshal(settings)
	streamJSON, _ := json.Marshal(stream)
	return &model.Inbound{
		Protocol:       protocol,
		Port:           port,
		Remark:         "matrix-" + string(protocol),
		Settings:       string(settingsJSON),
		StreamSettings: string(streamJSON),
	}
}

func matrixTCPStream() map[string]any {
	return map[string]any{
		"network":  "tcp",
		"security": "none",
		"tcpSettings": map[string]any{
			"acceptProxyProtocol": false,
			"header": map[string]any{
				"type": "none",
			},
		},
	}
}

func cloneMatrixInbound(inbound *model.Inbound) *model.Inbound {
	cloned := *inbound
	return &cloned
}

func firstMatrixOutbound(t *testing.T, config []byte) map[string]any {
	t.Helper()
	var parsed map[string]any
	if err := json.Unmarshal(config, &parsed); err != nil {
		t.Fatalf("subscription config JSON invalid: %v", err)
	}
	outbounds, ok := parsed["outbounds"].([]any)
	if !ok || len(outbounds) == 0 {
		t.Fatalf("subscription config missing outbounds: %#v", parsed)
	}
	outbound, ok := outbounds[0].(map[string]any)
	if !ok {
		t.Fatalf("first outbound has unexpected shape: %#v", outbounds[0])
	}
	return outbound
}

func TestBuildLinkWithParamsFiltersEmptyValuesAndOrdersQuery(t *testing.T) {
	link := buildLinkWithParams("vless://uuid@example.com:443", map[string]string{
		"type":       "tcp",
		"security":   "none",
		"empty":      "",
		"serverName": "a b.example",
	}, "remark with space")

	if strings.Contains(link, "empty=") {
		t.Fatalf("link should filter empty params: %s", link)
	}
	if !strings.Contains(link, "?security=none&serverName=a+b.example&type=tcp") {
		t.Fatalf("link query is not stable or encoded as expected: %s", link)
	}
	if !strings.HasSuffix(link, "#remark%20with%20space") {
		t.Fatalf("link fragment not encoded as expected: %s", link)
	}
}

func TestHysteriaLinkEncodesAuthUserinfo(t *testing.T) {
	client := matrixHysteriaClient("matrix-special-hy2@example")
	client.Auth = "hy2/auth=with padding"
	inbound := matrixHysteriaInbound(12014, client)
	service := &SubService{address: "vpn.example", remarkModel: "-ieo"}

	link := service.genHysteriaLink(inbound, client.Email)

	if !strings.Contains(link, "hysteria2://hy2%2Fauth%3Dwith%20padding@vpn.example:12014") {
		t.Fatalf("hysteria auth userinfo is not URI-encoded safely: %s", link)
	}
	if !strings.Contains(link, "obfs-password=matrix-obfs-pass") {
		t.Fatalf("hysteria obfs password should stay in query, got: %s", link)
	}
}

func TestHysteria2LiteralProtocolJSONNormalizesToXrayHysteria(t *testing.T) {
	client := matrixHysteriaClient("matrix-literal-hy2@example")
	inbound := matrixHysteriaInbound(12017, client)
	inbound.Protocol = model.Hysteria2
	service := &SubJsonService{
		configJson:     map[string]any{},
		inboundService: service.InboundService{},
		SubService:     &SubService{address: "vpn.example", remarkModel: "-ieo"},
	}

	configs := service.getConfig(inbound, client, "vpn.example")
	if len(configs) == 0 {
		t.Fatal("literal hysteria2 JSON subscription returned no configs")
	}
	outbound := firstMatrixOutbound(t, configs[0])

	if outbound["protocol"] != string(model.Hysteria) {
		t.Fatalf("literal hysteria2 JSON outbound protocol = %v, want %s", outbound["protocol"], model.Hysteria)
	}
}

func TestTrojanLinkEncodesPasswordUserinfo(t *testing.T) {
	client := matrixPasswordClient("matrix-special-trojan@example")
	client.Password = "tr/oj=an pass"
	inbound := matrixInbound(model.Trojan, 12016, client)
	service := &SubService{address: "vpn.example", remarkModel: "-ieo"}

	link := service.genTrojanLink(inbound, client.Email)

	if !strings.Contains(link, "trojan://tr%2Foj%3Dan%20pass@vpn.example:12016") {
		t.Fatalf("trojan password userinfo is not URI-encoded safely: %s", link)
	}
}

func TestHysteriaExternalProxyLinkKeepsUdpHopMport(t *testing.T) {
	client := matrixHysteriaClient("matrix-external-hy2@example")
	inbound := matrixHysteriaInbound(12015, client)
	var stream map[string]any
	if err := json.Unmarshal([]byte(inbound.StreamSettings), &stream); err != nil {
		t.Fatalf("stream settings invalid: %v", err)
	}
	stream["externalProxy"] = []map[string]any{{
		"forceTls": "same",
		"dest":     "edge.example",
		"port":     443,
		"remark":   "edge",
	}}
	streamJSON, _ := json.Marshal(stream)
	inbound.StreamSettings = string(streamJSON)
	service := &SubService{address: "vpn.example", remarkModel: "-ieo"}

	link := service.genHysteriaLink(inbound, client.Email)

	if !strings.Contains(link, "hysteria2://matrix-hy2-auth@edge.example:443") {
		t.Fatalf("hysteria external proxy endpoint missing: %s", link)
	}
	if !strings.Contains(link, "mport=40000-45000") {
		t.Fatalf("hysteria external proxy link should keep UDP hop mport: %s", link)
	}
}

func TestNormalizeClientNodeReturnsClientMetadata(t *testing.T) {
	client := matrixClient("matrix-normalized@example")
	inbound := matrixInboundWithSettings(model.VLESS, 12008, map[string]any{
		"decryption": "none",
		"encryption": "none",
		"clients":    []model.Client{client},
	}, map[string]any{
		"network":  "tcp",
		"security": "none",
	})
	service := &SubService{address: "vpn.example"}

	node, ok := service.normalizeClientNode(inbound, client.Email)
	if !ok {
		t.Fatalf("normalizeClientNode returned false")
	}
	if node.Protocol != model.VLESS || node.Address != "vpn.example" || node.Port != 12008 {
		t.Fatalf("normalized node protocol/address/port = %s/%s/%d", node.Protocol, node.Address, node.Port)
	}
	if node.Client.Email != client.Email || node.StreamNetwork != "tcp" || node.Security != "none" {
		t.Fatalf("normalized node metadata mismatch: %#v", node)
	}
	if node.Settings["encryption"] != "none" {
		t.Fatalf("normalized node settings encryption = %v, want none", node.Settings["encryption"])
	}
}

func TestNormalizeClientNodeRejectsMissingClient(t *testing.T) {
	missingEmail := "missing@example"
	presentClient := matrixClient("matrix-present@example")
	tests := []struct {
		name    string
		inbound *model.Inbound
		link    func(*SubService, *model.Inbound, string) string
	}{
		{
			name:    "vmess",
			inbound: matrixInbound(model.VMESS, 12009, presentClient),
			link:    (*SubService).genVmessLink,
		},
		{
			name:    "vless",
			inbound: matrixInbound(model.VLESS, 12010, presentClient),
			link:    (*SubService).genVlessLink,
		},
		{
			name:    "trojan",
			inbound: matrixInbound(model.Trojan, 12011, presentClient),
			link:    (*SubService).genTrojanLink,
		},
		{
			name: "shadowsocks",
			inbound: matrixInboundWithSettings(model.Shadowsocks, 12012, map[string]any{
				"method":  "aes-128-gcm",
				"clients": []model.Client{presentClient},
			}, map[string]any{
				"network":  "tcp",
				"security": "none",
			}),
			link: (*SubService).genShadowsocksLink,
		},
		{
			name:    "hysteria2",
			inbound: matrixHysteriaInbound(12013, matrixHysteriaClient("matrix-present@example")),
			link:    (*SubService).genHysteriaLink,
		},
	}
	service := &SubService{address: "vpn.example"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := service.normalizeClientNode(tt.inbound, missingEmail); ok {
				t.Fatalf("normalizeClientNode should reject missing client")
			}
			if link := tt.link(service, tt.inbound, missingEmail); link != "" {
				t.Fatalf("generated link with missing client = %q, want empty string", link)
			}
		})
	}
}

func TestResolveRequestIgnoresMalformedForwardedHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "http://panel.example/sub/sub-id", nil)
	request.Host = "safe.example:2096"
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("X-Forwarded-Host", "evil.example/path@attacker")
	context.Request = request

	scheme, host, hostWithPort, hostHeader := (&SubService{}).ResolveRequest(context)

	if scheme != "https" {
		t.Fatalf("scheme = %q, want https", scheme)
	}
	if host != "safe.example" {
		t.Fatalf("host = %q, want safe.example", host)
	}
	if hostWithPort != "safe.example:2096" {
		t.Fatalf("hostWithPort = %q, want safe.example:2096", hostWithPort)
	}
	if hostHeader != "safe.example" {
		t.Fatalf("hostHeader = %q, want safe.example", hostHeader)
	}
}

func TestSubServiceWithRequestContextDoesNotMutateOriginal(t *testing.T) {
	original := &SubService{
		address:     "old.example",
		datepicker:  "gregorian",
		showInfo:    true,
		remarkModel: "-ieo",
	}

	scoped := original.withRequestContext("new.example", "jalali")

	if scoped == original {
		t.Fatalf("request scoped service should be a copy")
	}
	if scoped.address != "new.example" || scoped.datepicker != "jalali" {
		t.Fatalf("request scoped service has address=%q datepicker=%q", scoped.address, scoped.datepicker)
	}
	if original.address != "old.example" || original.datepicker != "gregorian" {
		t.Fatalf("original service was mutated: address=%q datepicker=%q", original.address, original.datepicker)
	}
	if !scoped.showInfo || scoped.remarkModel != "-ieo" {
		t.Fatalf("request scoped service should preserve static configuration")
	}
}

func TestBrowserSubscriptionRequestReturnsPlainTextWithoutLegacyVisualPage(t *testing.T) {
	setupSubscriptionDiagnosticDB(t)
	client := matrixClient("matrix-subpage@example")
	inbound := matrixInbound(model.VLESS, 13014, client)
	inbound.Enable = true
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("base_path", "/sub/")
	})
	NewSUBController(router.Group("/"), "/sub/", "/json/", "/clash/", true, true, false, true, "-ieo", "12", "", "", "", "", "", "", "", "", true, "")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/sub/matrix-sub", nil)
	request.Host = "vpn.example:2096"
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/plain") {
		t.Fatalf("Content-Type = %q, want plain subscription output", contentType)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "vless://") {
		t.Fatalf("subscription output missing link:\n%s", body)
	}
	for _, retired := range []string{`id="qrcode"`, "/assets/", "<script", "<html"} {
		if strings.Contains(body, retired) {
			t.Fatalf("subscription output still contains retired visual page fragment %q:\n%s", retired, body)
		}
	}
}

func TestSubJsonExternalProxyForceTLSUsesIndependentStreamCopies(t *testing.T) {
	client := matrixClient("matrix-external@example")
	inbound := matrixInboundWithSettings(model.VLESS, 12006, map[string]any{
		"decryption": "none",
		"encryption": "none",
		"clients":    []model.Client{client},
	}, map[string]any{
		"network":  "tcp",
		"security": "tls",
		"tlsSettings": map[string]any{
			"serverName": "base.example",
			"settings": map[string]any{
				"fingerprint": "chrome",
			},
		},
		"externalProxy": []map[string]any{
			{
				"forceTls": "none",
				"dest":     "plain.example",
				"port":     443,
				"remark":   "plain",
			},
			{
				"forceTls": "tls",
				"dest":     "tls.example",
				"port":     8443,
				"remark":   "tls",
			},
		},
	})
	jsonService := &SubJsonService{
		configJson: map[string]any{},
		SubService: &SubService{remarkModel: "-ieo"},
	}

	configs := jsonService.getConfig(inbound, client, "vpn.example")
	if len(configs) != 2 {
		t.Fatalf("external proxy configs = %d, want 2", len(configs))
	}

	firstStream, ok := firstMatrixOutbound(t, configs[0])["streamSettings"].(map[string]any)
	if !ok {
		t.Fatalf("first config streamSettings has unexpected shape")
	}
	secondStream, ok := firstMatrixOutbound(t, configs[1])["streamSettings"].(map[string]any)
	if !ok {
		t.Fatalf("second config streamSettings has unexpected shape")
	}

	if firstStream["security"] != "none" {
		t.Fatalf("first external proxy security = %v, want none", firstStream["security"])
	}
	if _, ok := firstStream["tlsSettings"]; ok {
		t.Fatalf("first external proxy should not include tlsSettings: %#v", firstStream)
	}
	if secondStream["security"] != "tls" {
		t.Fatalf("second external proxy security = %v, want tls", secondStream["security"])
	}
	secondTLS, ok := secondStream["tlsSettings"].(map[string]any)
	if !ok {
		t.Fatalf("second external proxy should keep tlsSettings: %#v", secondStream)
	}
	if secondTLS["serverName"] != "base.example" {
		t.Fatalf("second external proxy tlsSettings.serverName = %v, want base.example", secondTLS["serverName"])
	}
}

func TestExternalProxyMalformedEntriesDoNotPanic(t *testing.T) {
	client := matrixClient("matrix-malformed@example")
	inbound := matrixInboundWithSettings(model.VLESS, 12007, map[string]any{
		"decryption": "none",
		"encryption": "none",
		"clients":    []model.Client{client},
	}, map[string]any{
		"network":  "tcp",
		"security": "none",
		"externalProxy": []any{
			"not-a-map",
			map[string]any{"forceTls": "same", "dest": "", "port": 443, "remark": "empty-dest"},
			map[string]any{"forceTls": "same", "dest": "valid.example", "port": "bad-port", "remark": "bad-port"},
			map[string]any{"forceTls": "same", "dest": "valid.example", "port": 9443, "remark": "valid"},
		},
	})
	subService := &SubService{address: "vpn.example", remarkModel: "-ieo"}
	jsonService := &SubJsonService{configJson: map[string]any{}, SubService: subService}

	link := subService.genVlessLink(cloneMatrixInbound(inbound), client.Email)
	if !strings.Contains(link, "valid.example:9443") {
		t.Fatalf("plain subscription should keep valid external proxy only, got %q", link)
	}

	configs := jsonService.getConfig(cloneMatrixInbound(inbound), client, "vpn.example")
	if len(configs) != 1 {
		t.Fatalf("json subscription configs = %d, want 1 valid external proxy config", len(configs))
	}
	outbound := firstMatrixOutbound(t, configs[0])
	settings, ok := outbound["settings"].(map[string]any)
	if !ok {
		t.Fatalf("json outbound settings has unexpected shape: %#v", outbound["settings"])
	}
	vnext, ok := settings["vnext"].([]any)
	if !ok || len(vnext) != 1 {
		t.Fatalf("json outbound vnext has unexpected shape: %#v", settings["vnext"])
	}
	server, ok := vnext[0].(map[string]any)
	if !ok || server["address"] != "valid.example" {
		t.Fatalf("json outbound address = %#v, want valid.example", vnext[0])
	}
}

func parseMatrixOutbound(t *testing.T, outboundJSON []byte) map[string]any {
	t.Helper()
	var outbound map[string]any
	if err := json.Unmarshal(outboundJSON, &outbound); err != nil {
		t.Fatalf("outbound JSON invalid: %v", err)
	}
	return outbound
}

func proxyMatrixOutboundsByProtocol(t *testing.T, jsonText string) map[string]map[string]any {
	t.Helper()
	var configs []map[string]any
	if err := json.Unmarshal([]byte(jsonText), &configs); err != nil {
		var single map[string]any
		if singleErr := json.Unmarshal([]byte(jsonText), &single); singleErr != nil {
			t.Fatalf("proxy json subscription invalid: array error=%v single error=%v\n%s", err, singleErr, jsonText)
		}
		configs = []map[string]any{single}
	}
	outbounds := make(map[string]map[string]any)
	for _, config := range configs {
		items, ok := config["outbounds"].([]any)
		if !ok || len(items) == 0 {
			t.Fatalf("proxy config missing outbounds: %#v", config)
		}
		outbound, ok := items[0].(map[string]any)
		if !ok {
			t.Fatalf("proxy outbound has unexpected shape: %#v", items[0])
		}
		protocol, _ := outbound["protocol"].(string)
		outbounds[protocol] = outbound
	}
	for _, protocol := range []string{"http", "socks"} {
		if outbounds[protocol] == nil {
			t.Fatalf("proxy json subscription missing %s outbound: %#v", protocol, outbounds)
		}
	}
	return outbounds
}

func assertProxyMatrixServerUser(t *testing.T, outbound map[string]any, user string, pass string) {
	t.Helper()
	settings, ok := outbound["settings"].(map[string]any)
	if !ok {
		t.Fatalf("proxy outbound settings has unexpected shape: %#v", outbound["settings"])
	}
	servers, ok := settings["servers"].([]any)
	if !ok || len(servers) != 1 {
		t.Fatalf("proxy outbound servers = %#v, want one server", settings["servers"])
	}
	server, ok := servers[0].(map[string]any)
	if !ok {
		t.Fatalf("proxy server has unexpected shape: %#v", servers[0])
	}
	users, ok := server["users"].([]any)
	if !ok || len(users) != 1 {
		t.Fatalf("proxy server users = %#v, want one authenticated user", server["users"])
	}
	gotUser, ok := users[0].(map[string]any)
	if !ok {
		t.Fatalf("proxy user has unexpected shape: %#v", users[0])
	}
	if gotUser["user"] != user || gotUser["pass"] != pass {
		t.Fatalf("proxy user = %#v, want user=%q pass=%q", gotUser, user, pass)
	}
}
