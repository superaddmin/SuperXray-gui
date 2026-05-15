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
	host := "vpn.example"
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
			linkContains: []string{"hysteria2://matrix-hy2-auth@", "obfs=salamander", "obfs-password=matrix-obfs-pass"},
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
			}

			proxies := clashService.getProxies(cloneMatrixInbound(tc.inbound), tc.client, host)
			if len(proxies) == 0 {
				t.Fatalf("clash subscription for %s returned no proxies", tc.name)
			}
			if proxies[0]["type"] != tc.clashType {
				t.Fatalf("clash type for %s = %v, want %s", tc.name, proxies[0]["type"], tc.clashType)
			}
		})
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
	settings := map[string]any{
		"clients": []model.Client{client},
	}
	if protocol == model.VLESS {
		settings["decryption"] = "none"
		settings["encryption"] = "none"
	}
	return matrixInboundWithSettings(protocol, port, settings, matrixTCPStream())
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
				"fingerprint":   "chrome",
			},
		},
		"finalmask": map[string]any{
			"udp": []map[string]any{{
				"type": "salamander",
				"settings": map[string]any{
					"password": "matrix-obfs-pass",
				},
			}},
		},
	})
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

func TestBrowserSubscriptionRequestRendersLegacyVisualPageWithMountedAssets(t *testing.T) {
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
	router.SetFuncMap(map[string]any{
		"i18n": func(key string, params ...string) string {
			return key
		},
	})
	if err := setEmbeddedTemplates(router); err != nil {
		t.Fatalf("setEmbeddedTemplates failed: %v", err)
	}
	NewSUBController(router.Group("/"), "/sub/", "/json/", "/clash/", true, true, false, true, "-ieo", "12", "", "", "", "", "", "", "", "", true, "")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/sub/matrix-sub", nil)
	request.Host = "vpn.example:2096"
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`id="qrcode"`,
		`id="qrcode-subjson"`,
		`id="qrcode-subclash"`,
		`src="/sub/assets/qrcode/qrious2.min.js`,
		`src="/sub/assets/js/subscription.js`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("subscription visual page missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "/sub/matrix-sub/assets/") {
		t.Fatalf("subscription visual page uses unmounted subId asset prefix:\n%s", body)
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
