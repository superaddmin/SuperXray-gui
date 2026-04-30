package sub

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

const (
	wireguardTestServerPrivate = "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE="
	wireguardTestServerPublic  = "QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo1Njc4OTA="
	wireguardTestPeerPrivate   = "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY="
	wireguardTestPeerPublic    = "emFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6MTIzNDU="
	wireguardTestPSK           = "cHNoYXJlZGtleTEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI="
)

func TestWireguardPeersBySubIDFindsEnabledPeer(t *testing.T) {
	inbound := wireguardInboundForTest(`[
		{
			"email": "wg@example",
			"enable": true,
			"subId": "sub-123",
			"privateKey": "` + wireguardTestPeerPrivate + `",
			"publicKey": "` + wireguardTestPeerPublic + `",
			"preSharedKey": "` + wireguardTestPSK + `",
			"allowedIPs": ["10.0.0.2/32"],
			"keepAlive": 25
		},
		{
			"email": "disabled@example",
			"enable": false,
			"subId": "sub-123",
			"privateKey": "` + wireguardTestPeerPrivate + `",
			"publicKey": "` + wireguardTestPeerPublic + `",
			"allowedIPs": ["10.0.0.3/32"]
		}
	]`)

	peers, err := wireguardPeersBySubID(inbound, "sub-123")
	if err != nil {
		t.Fatalf("wireguardPeersBySubID returned error: %v", err)
	}
	if len(peers) != 1 {
		t.Fatalf("wireguardPeersBySubID returned %d peers, want 1", len(peers))
	}
	if peers[0].Email != "wg@example" {
		t.Fatalf("peer email = %q, want wg@example", peers[0].Email)
	}
}

func TestWireguardPeersBySubIDSkipsInvalidPeerMaterial(t *testing.T) {
	inbound := wireguardInboundForTest(`[
		{
			"email": "bad-key@example",
			"enable": true,
			"subId": "sub-123",
			"privateKey": "not-base64",
			"publicKey": "` + wireguardTestPeerPublic + `",
			"allowedIPs": ["10.0.0.2/32"]
		},
		{
			"email": "bad-ip@example",
			"enable": true,
			"subId": "sub-123",
			"privateKey": "` + wireguardTestPeerPrivate + `",
			"publicKey": "` + wireguardTestPeerPublic + `",
			"allowedIPs": ["not-an-ip"]
		}
	]`)

	peers, err := wireguardPeersBySubID(inbound, "sub-123")
	if err != nil {
		t.Fatalf("wireguardPeersBySubID returned error: %v", err)
	}
	if len(peers) != 0 {
		t.Fatalf("wireguardPeersBySubID returned %d peers, want 0", len(peers))
	}
}

func TestGenWireguardConfigIncludesPeerAndServerKeys(t *testing.T) {
	inbound := wireguardInboundForTest(`[{
		"email": "wg@example",
		"enable": true,
		"subId": "sub-123",
		"privateKey": "` + wireguardTestPeerPrivate + `",
		"publicKey": "` + wireguardTestPeerPublic + `",
		"preSharedKey": "` + wireguardTestPSK + `",
		"allowedIPs": ["10.0.0.2/32"],
		"keepAlive": 25
	}]`)
	peers, err := wireguardPeersBySubID(inbound, "sub-123")
	if err != nil {
		t.Fatalf("wireguardPeersBySubID returned error: %v", err)
	}
	service := &SubService{address: "vpn.example", remarkModel: "-ieo"}

	config := service.genWireguardConfig(inbound, peers[0])

	for _, want := range []string{
		"[Interface]",
		"PrivateKey = " + wireguardTestPeerPrivate,
		"Address = 10.0.0.2/32",
		"MTU = 1420",
		"# wg-wg@example",
		"[Peer]",
		"PublicKey = " + wireguardTestServerPublic,
		"Endpoint = vpn.example:51820",
		"PresharedKey = " + wireguardTestPSK,
		"PersistentKeepalive = 25",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("wireguard config missing %q:\n%s", want, config)
		}
	}
}

func TestSubJsonServiceGenWireguardOutbound(t *testing.T) {
	inbound := wireguardInboundForTest(`[{
		"email": "wg@example",
		"enable": true,
		"subId": "sub-123",
		"privateKey": "` + wireguardTestPeerPrivate + `",
		"publicKey": "` + wireguardTestPeerPublic + `",
		"preSharedKey": "` + wireguardTestPSK + `",
		"allowedIPs": ["10.0.0.2/32"],
		"keepAlive": 25
	}]`)
	peers, err := wireguardPeersBySubID(inbound, "sub-123")
	if err != nil {
		t.Fatalf("wireguardPeersBySubID returned error: %v", err)
	}

	outbound := (&SubJsonService{}).genWireguard(inbound, peers[0], "vpn.example")
	var got map[string]any
	if err := json.Unmarshal(outbound, &got); err != nil {
		t.Fatalf("wireguard outbound JSON invalid: %v", err)
	}

	if got["protocol"] != "wireguard" {
		t.Fatalf("protocol = %v, want wireguard", got["protocol"])
	}
	settings := got["settings"].(map[string]any)
	if settings["secretKey"] != wireguardTestPeerPrivate {
		t.Fatalf("secretKey = %v, want peer private key", settings["secretKey"])
	}
	peersJSON := settings["peers"].([]any)
	firstPeer := peersJSON[0].(map[string]any)
	if firstPeer["endpoint"] != "vpn.example:51820" {
		t.Fatalf("endpoint = %v, want vpn.example:51820", firstPeer["endpoint"])
	}
	if firstPeer["publicKey"] != wireguardTestServerPublic {
		t.Fatalf("publicKey = %v, want server public key", firstPeer["publicKey"])
	}
}

func TestSubClashServiceBuildWireguardProxy(t *testing.T) {
	inbound := wireguardInboundForTest(`[{
		"email": "wg@example",
		"enable": true,
		"subId": "sub-123",
		"privateKey": "` + wireguardTestPeerPrivate + `",
		"publicKey": "` + wireguardTestPeerPublic + `",
		"preSharedKey": "` + wireguardTestPSK + `",
		"allowedIPs": ["10.0.0.2/32"],
		"keepAlive": 25
	}]`)
	peers, err := wireguardPeersBySubID(inbound, "sub-123")
	if err != nil {
		t.Fatalf("wireguardPeersBySubID returned error: %v", err)
	}
	service := &SubClashService{SubService: &SubService{remarkModel: "-ieo"}}

	proxy := service.buildWireguardProxy(inbound, peers[0], "vpn.example", "")

	if proxy["type"] != "wireguard" {
		t.Fatalf("type = %v, want wireguard", proxy["type"])
	}
	if proxy["server"] != "vpn.example" {
		t.Fatalf("server = %v, want vpn.example", proxy["server"])
	}
	if proxy["private-key"] != wireguardTestPeerPrivate {
		t.Fatalf("private-key = %v, want peer private key", proxy["private-key"])
	}
	if proxy["public-key"] != wireguardTestServerPublic {
		t.Fatalf("public-key = %v, want server public key", proxy["public-key"])
	}
}

func wireguardInboundForTest(peersJSON string) *model.Inbound {
	return &model.Inbound{
		Protocol: model.WireGuard,
		Listen:   "",
		Port:     51820,
		Remark:   "wg",
		Settings: `{
			"mtu": 1420,
			"secretKey": "` + wireguardTestServerPrivate + `",
			"pubKey": "` + wireguardTestServerPublic + `",
			"peers": ` + peersJSON + `,
			"noKernelTun": false
		}`,
		StreamSettings: `{"network":"tcp","security":"none"}`,
	}
}
