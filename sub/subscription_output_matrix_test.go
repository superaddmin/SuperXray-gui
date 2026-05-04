package sub

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
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

func parseMatrixOutbound(t *testing.T, outboundJSON []byte) map[string]any {
	t.Helper()
	var outbound map[string]any
	if err := json.Unmarshal(outboundJSON, &outbound); err != nil {
		t.Fatalf("outbound JSON invalid: %v", err)
	}
	return outbound
}
