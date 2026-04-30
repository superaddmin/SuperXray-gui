package sub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/json_util"
)

type wireguardSettings struct {
	MTU         int             `json:"mtu"`
	SecretKey   string          `json:"secretKey"`
	PublicKey   string          `json:"pubKey"`
	NoKernelTun bool            `json:"noKernelTun"`
	Peers       []wireguardPeer `json:"-"`
}

type wireguardPeer struct {
	Email        string   `json:"email,omitempty"`
	Enable       bool     `json:"enable"`
	SubID        string   `json:"subId,omitempty"`
	PrivateKey   string   `json:"privateKey"`
	PublicKey    string   `json:"publicKey"`
	PreSharedKey string   `json:"preSharedKey,omitempty"`
	AllowedIPs   []string `json:"allowedIPs"`
	KeepAlive    int      `json:"keepAlive,omitempty"`
}

func wireguardPeersBySubID(inbound *model.Inbound, subID string) ([]wireguardPeer, error) {
	settings, err := wireguardSettingsFromInbound(inbound)
	if err != nil {
		return nil, err
	}

	peers := make([]wireguardPeer, 0, len(settings.Peers))
	for _, peer := range settings.Peers {
		if peer.SubID != subID || !peer.Enable || !peer.isUsable(settings) {
			continue
		}
		peers = append(peers, peer)
	}
	return peers, nil
}

func wireguardSettingsFromInbound(inbound *model.Inbound) (wireguardSettings, error) {
	if inbound == nil {
		return wireguardSettings{}, fmt.Errorf("inbound is nil")
	}

	var raw struct {
		MTU         int               `json:"mtu"`
		SecretKey   string            `json:"secretKey"`
		PublicKey   string            `json:"pubKey"`
		NoKernelTun bool              `json:"noKernelTun"`
		Peers       []json.RawMessage `json:"peers"`
	}
	if err := json.Unmarshal([]byte(inbound.Settings), &raw); err != nil {
		return wireguardSettings{}, err
	}

	settings := wireguardSettings{
		MTU:         raw.MTU,
		SecretKey:   raw.SecretKey,
		PublicKey:   raw.PublicKey,
		NoKernelTun: raw.NoKernelTun,
		Peers:       make([]wireguardPeer, 0, len(raw.Peers)),
	}
	for _, rawPeer := range raw.Peers {
		peer, err := decodeWireguardPeer(rawPeer)
		if err != nil {
			return wireguardSettings{}, err
		}
		settings.Peers = append(settings.Peers, peer)
	}
	return settings, nil
}

func decodeWireguardPeer(rawPeer json.RawMessage) (wireguardPeer, error) {
	var peer wireguardPeer
	if err := json.Unmarshal(rawPeer, &peer); err != nil {
		return wireguardPeer{}, err
	}

	var fields map[string]any
	if err := json.Unmarshal(rawPeer, &fields); err != nil {
		return wireguardPeer{}, err
	}
	if _, ok := fields["enable"]; !ok {
		peer.Enable = true
	}
	return peer, nil
}

func (p wireguardPeer) isUsable(settings wireguardSettings) bool {
	if !validWireguardKey(settings.PublicKey) || !validWireguardKey(p.PrivateKey) || !validWireguardKey(p.PublicKey) {
		return false
	}
	if p.PreSharedKey != "" && !validWireguardKey(p.PreSharedKey) {
		return false
	}
	return firstUsableAllowedIP(p.AllowedIPs) != ""
}

func validWireguardKey(key string) bool {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(key))
	return err == nil && len(decoded) == 32
}

func firstUsableAllowedIP(allowedIPs []string) string {
	for _, allowedIP := range allowedIPs {
		allowedIP = strings.TrimSpace(allowedIP)
		if allowedIP == "" {
			continue
		}
		if _, err := netip.ParsePrefix(allowedIP); err == nil {
			return allowedIP
		}
		if addr, err := netip.ParseAddr(allowedIP); err == nil {
			if addr.Is4() {
				return allowedIP + "/32"
			}
			return allowedIP + "/128"
		}
	}
	return ""
}

func (s *SubService) genWireguardConfig(inbound *model.Inbound, peer wireguardPeer) string {
	settings, err := wireguardSettingsFromInbound(inbound)
	if err != nil || !peer.isUsable(settings) {
		return ""
	}

	address := s.resolveInboundAddress(inbound)
	clientAddress := firstUsableAllowedIP(peer.AllowedIPs)
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString("PrivateKey = " + peer.PrivateKey + "\n")
	b.WriteString("Address = " + clientAddress + "\n")
	b.WriteString("DNS = 1.1.1.1, 1.0.0.1\n")
	if settings.MTU > 0 {
		b.WriteString(fmt.Sprintf("MTU = %d\n", settings.MTU))
	}
	b.WriteString("\n# " + s.genRemark(inbound, peer.Email, "") + "\n")
	b.WriteString("[Peer]\n")
	b.WriteString("PublicKey = " + settings.PublicKey + "\n")
	b.WriteString("AllowedIPs = 0.0.0.0/0, ::/0\n")
	b.WriteString(fmt.Sprintf("Endpoint = %s:%d", address, inbound.Port))
	if peer.PreSharedKey != "" {
		b.WriteString("\nPresharedKey = " + peer.PreSharedKey)
	}
	if peer.KeepAlive > 0 {
		b.WriteString(fmt.Sprintf("\nPersistentKeepalive = %d\n", peer.KeepAlive))
	}
	return b.String()
}

func (s *SubJsonService) genWireguard(inbound *model.Inbound, peer wireguardPeer, host string) json_util.RawMessage {
	settings, err := wireguardSettingsFromInbound(inbound)
	if err != nil || !peer.isUsable(settings) {
		return nil
	}

	address := inbound.Listen
	if address == "" || address == "0.0.0.0" || address == "::" || address == "::0" {
		address = host
	}
	outboundSettings := map[string]any{
		"secretKey": peer.PrivateKey,
		"address":   []string{firstUsableAllowedIP(peer.AllowedIPs)},
		"peers": []map[string]any{{
			"publicKey": settings.PublicKey,
			"endpoint":  fmt.Sprintf("%s:%d", address, inbound.Port),
		}},
	}
	if settings.MTU > 0 {
		outboundSettings["mtu"] = settings.MTU
	}
	if settings.NoKernelTun {
		outboundSettings["noKernelTun"] = true
	}
	if peer.PreSharedKey != "" {
		outboundSettings["peers"].([]map[string]any)[0]["preSharedKey"] = peer.PreSharedKey
	}
	if peer.KeepAlive > 0 {
		outboundSettings["peers"].([]map[string]any)[0]["keepAlive"] = peer.KeepAlive
	}

	outbound := Outbound{
		Protocol: "wireguard",
		Tag:      "proxy",
		Settings: outboundSettings,
	}
	result, _ := json.MarshalIndent(outbound, "", "  ")
	return result
}

func (s *SubClashService) buildWireguardProxy(inbound *model.Inbound, peer wireguardPeer, host string, extraRemark string) map[string]any {
	settings, err := wireguardSettingsFromInbound(inbound)
	if err != nil || !peer.isUsable(settings) {
		return nil
	}

	address := inbound.Listen
	if address == "" || address == "0.0.0.0" || address == "::" || address == "::0" {
		address = host
	}
	proxy := map[string]any{
		"name":        s.SubService.genRemark(inbound, peer.Email, extraRemark),
		"type":        "wireguard",
		"server":      address,
		"port":        inbound.Port,
		"ip":          firstUsableAllowedIP(peer.AllowedIPs),
		"private-key": peer.PrivateKey,
		"public-key":  settings.PublicKey,
		"allowed-ips": []string{"0.0.0.0/0", "::/0"},
		"udp":         true,
	}
	if settings.MTU > 0 {
		proxy["mtu"] = settings.MTU
	}
	if peer.PreSharedKey != "" {
		proxy["pre-shared-key"] = peer.PreSharedKey
	}
	if peer.KeepAlive > 0 {
		proxy["persistent-keepalive"] = peer.KeepAlive
	}
	return proxy
}
