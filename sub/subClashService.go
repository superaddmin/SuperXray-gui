package sub

import (
	"fmt"
	"net"
	"strings"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

const (
	clashFullTemplateProviderBase     = "https://raw.githubusercontent.com/ryuu-restart/agent-route/release/mihomo/"
	clashFullTemplateProviderInterval = 86400
)

type SubClashService struct {
	inboundService service.InboundService
	SubService     *SubService
	routingEnabled bool
	routingRules   string
	fullTemplate   bool
}

type ClashConfig struct {
	Proxies       []map[string]any `yaml:"proxies"`
	ProxyGroups   []map[string]any `yaml:"proxy-groups"`
	DNS           map[string]any   `yaml:"dns,omitempty"`
	RuleProviders map[string]any   `yaml:"rule-providers,omitempty"`
	Sniffer       map[string]any   `yaml:"sniffer,omitempty"`
	Rules         []string         `yaml:"rules"`
}

func NewSubClashService(subService *SubService) *SubClashService {
	return &SubClashService{SubService: subService}
}

func (s *SubClashService) WithRoutingRules(rules string) *SubClashService {
	return s.WithRoutingSettings(true, rules)
}

func (s *SubClashService) WithRoutingSettings(enabled bool, rules string) *SubClashService {
	s.routingEnabled = enabled
	s.routingRules = rules
	return s
}

// WithFullTemplate emits DNS, sniffer and remote rule providers for Mihomo clients.
func (s *SubClashService) WithFullTemplate(enabled bool) *SubClashService {
	s.fullTemplate = enabled
	return s
}

func (s *SubClashService) GetClash(subId string, host string) (string, string, error) {
	inbounds, err := s.SubService.getInboundsBySubId(subId)
	if err != nil || len(inbounds) == 0 {
		return "", "", err
	}

	var traffic xray.ClientTraffic
	var clientTraffics []xray.ClientTraffic
	var proxies []map[string]any

	for _, inbound := range inbounds {
		if inbound.Protocol == model.WireGuard {
			if len(inbound.Listen) > 0 && inbound.Listen[0] == '@' {
				listen, port, streamSettings, err := s.SubService.getFallbackMaster(inbound.Listen, inbound.StreamSettings)
				if err == nil {
					inbound.Listen = listen
					inbound.Port = port
					inbound.StreamSettings = streamSettings
				}
			}
			peers, err := wireguardPeersBySubID(inbound, subId)
			if err != nil {
				logger.Error("SubClashService - WireGuard peers: Unable to get peers from inbound")
				continue
			}
			for _, peer := range peers {
				if proxy := s.buildWireguardProxy(inbound, peer, host, ""); len(proxy) > 0 {
					proxies = append(proxies, proxy)
				}
			}
			continue
		}
		if isProxyAccountProtocol(inbound.Protocol) {
			accounts, err := proxyAccountsBySubID(inbound, subId)
			if err != nil {
				logger.Error("SubClashService - Proxy accounts: Unable to get accounts from inbound")
				continue
			}
			for _, account := range accounts {
				if proxy := s.buildProxyAccountProxy(inbound, account, host, ""); len(proxy) > 0 {
					proxies = append(proxies, proxy)
				}
			}
			continue
		}

		clients, err := s.inboundService.GetClients(inbound)
		if err != nil {
			logger.Error("SubClashService - GetClients: Unable to get clients from inbound")
		}
		if clients == nil {
			continue
		}
		if len(inbound.Listen) > 0 && inbound.Listen[0] == '@' {
			listen, port, streamSettings, err := s.SubService.getFallbackMaster(inbound.Listen, inbound.StreamSettings)
			if err == nil {
				inbound.Listen = listen
				inbound.Port = port
				inbound.StreamSettings = streamSettings
			}
		}
		for _, client := range clients {
			if client.Enable && client.SubID == subId {
				clientTraffics = append(clientTraffics, s.SubService.getClientTraffics(inbound.ClientStats, client.Email))
				proxies = append(proxies, s.getProxies(inbound, client, host)...)
			}
		}
	}

	if len(proxies) == 0 {
		return "", "", nil
	}

	for index, clientTraffic := range clientTraffics {
		if index == 0 {
			traffic.Up = clientTraffic.Up
			traffic.Down = clientTraffic.Down
			traffic.Total = clientTraffic.Total
			if clientTraffic.ExpiryTime > 0 {
				traffic.ExpiryTime = clientTraffic.ExpiryTime
			}
		} else {
			traffic.Up += clientTraffic.Up
			traffic.Down += clientTraffic.Down
			if traffic.Total == 0 || clientTraffic.Total == 0 {
				traffic.Total = 0
			} else {
				traffic.Total += clientTraffic.Total
			}
			if clientTraffic.ExpiryTime != traffic.ExpiryTime {
				traffic.ExpiryTime = 0
			}
		}
	}

	proxyNames := make([]string, 0, len(proxies)+1)
	for _, proxy := range proxies {
		if name, ok := proxy["name"].(string); ok && name != "" {
			proxyNames = append(proxyNames, name)
		}
	}
	proxyNames = append(proxyNames, "DIRECT")

	config := ClashConfig{
		Proxies: proxies,
		ProxyGroups: []map[string]any{{
			"name":    "PROXY",
			"type":    "select",
			"proxies": proxyNames,
		}},
		Rules: s.buildClashRules(),
	}
	if s.fullTemplate {
		nodeRules, nodeDomains := clashNodeAddressExceptions(proxies)
		config.Rules = s.buildFullClashRules(nodeRules)
		config.DNS = clashFullTemplateDNS(nodeDomains)
		config.RuleProviders = clashFullTemplateRuleProviders()
		config.Sniffer = clashFullTemplateSniffer()
	}

	finalYAML, err := marshalClashYAML(config)
	if err != nil {
		return "", "", err
	}

	header := fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d", traffic.Up, traffic.Down, traffic.Total, traffic.ExpiryTime/1000)
	return string(finalYAML), header, nil
}

func (s *SubClashService) buildClashRules() []string {
	if !s.routingEnabled {
		return []string{"MATCH,PROXY"}
	}
	rules := make([]string, 0)
	hasMatchProxy := false
	for _, line := range strings.Split(s.routingRules, "\n") {
		rule := strings.TrimSpace(line)
		if rule == "" || strings.HasPrefix(rule, "#") || !strings.Contains(rule, ",") {
			continue
		}
		if rule == "MATCH,PROXY" {
			hasMatchProxy = true
		}
		rules = append(rules, rule)
	}
	if !hasMatchProxy {
		rules = append(rules, "MATCH,PROXY")
	}
	return rules
}

// buildFullClashRules keeps user exceptions first, then node addresses, LAN,
// DNS resolver paths and the shared rule sets. MATCH is generated once at the end.
func (s *SubClashService) buildFullClashRules(nodeRules []string) []string {
	rules := make([]string, 0, len(nodeRules)+32)
	rules = append(rules, s.clashUserExceptionRules()...)
	rules = append(rules, nodeRules...)
	rules = append(rules,
		"IP-CIDR,192.168.0.0/16,DIRECT,no-resolve",
		"IP-CIDR,10.0.0.0/8,DIRECT,no-resolve",
		"IP-CIDR,172.16.0.0/12,DIRECT,no-resolve",
		"IP-CIDR,127.0.0.1/8,DIRECT,no-resolve",
		"IP-CIDR,169.254.0.0/16,DIRECT,no-resolve",
		"IP-CIDR6,fc00::/7,DIRECT,no-resolve",
		"IP-CIDR6,::1/128,DIRECT,no-resolve",
		"IP-CIDR,1.1.1.1/32,PROXY,no-resolve",
		"IP-CIDR,1.0.0.1/32,PROXY,no-resolve",
		"IP-CIDR,223.5.5.5/32,DIRECT,no-resolve",
		"IP-CIDR,223.6.6.6/32,DIRECT,no-resolve",
		"IP-CIDR,119.29.29.29/32,DIRECT,no-resolve",
		"DST-PORT,3478,PROXY",
		"DST-PORT,5349,PROXY",
		"DST-PORT,19302,PROXY",
		"RULE-SET,sr-private,DIRECT",
		"RULE-SET,sr-reject,REJECT",
		"RULE-SET,sr-proxy,PROXY",
		"RULE-SET,sr-direct,DIRECT",
		"RULE-SET,sr-cn-ip,DIRECT",
		"MATCH,PROXY",
	)
	return rules
}

// clashUserExceptionRules drops MATCH lines so they cannot bypass the built-in order.
func (s *SubClashService) clashUserExceptionRules() []string {
	if !s.routingEnabled {
		return nil
	}
	rules := make([]string, 0)
	for _, line := range strings.Split(s.routingRules, "\n") {
		rule := strings.TrimSpace(line)
		if rule == "" || strings.HasPrefix(rule, "#") || !strings.Contains(rule, ",") {
			continue
		}
		ruleType := strings.TrimSpace(strings.SplitN(rule, ",", 2)[0])
		if strings.EqualFold(ruleType, "MATCH") {
			continue
		}
		rules = append(rules, rule)
	}
	return rules
}

// clashNodeAddressExceptions keeps the proxies reachable without sending their
// own connection back through PROXY. Domain nodes also need a real DNS answer.
func clashNodeAddressExceptions(proxies []map[string]any) ([]string, []string) {
	rules := make([]string, 0, len(proxies))
	domains := make([]string, 0, len(proxies))
	seenRules := make(map[string]struct{}, len(proxies))
	seenDomains := make(map[string]struct{}, len(proxies))
	for _, proxy := range proxies {
		server, _ := proxy["server"].(string)
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}
		if ip := net.ParseIP(server); ip != nil {
			rule := "IP-CIDR," + ip.String() + "/32,DIRECT,no-resolve"
			if ip.To4() == nil {
				rule = "IP-CIDR6," + ip.String() + "/128,DIRECT,no-resolve"
			}
			if _, ok := seenRules[rule]; ok {
				continue
			}
			seenRules[rule] = struct{}{}
			rules = append(rules, rule)
			continue
		}
		host := strings.ToLower(server)
		rule := "DOMAIN," + host + ",DIRECT"
		if _, ok := seenRules[rule]; ok {
			continue
		}
		seenRules[rule] = struct{}{}
		rules = append(rules, rule)
		if _, ok := seenDomains[host]; ok {
			continue
		}
		seenDomains[host] = struct{}{}
		domains = append(domains, host)
	}
	return rules, domains
}

func clashFullTemplateDNS(nodeDomains []string) map[string]any {
	fakeIPFilter := []string{"rule-set:sr-private", "*.local", "*.lan"}
	fakeIPFilter = append(fakeIPFilter, nodeDomains...)
	return map[string]any{
		"enable":                          true,
		"enhanced-mode":                   "fake-ip",
		"fake-ip-range":                   "198.18.0.1/16",
		"respect-rules":                   true,
		"default-nameserver":              []string{"223.5.5.5", "119.29.29.29"},
		"proxy-server-nameserver":         []string{"https://223.5.5.5/dns-query"},
		"direct-nameserver":               []string{"https://223.5.5.5/dns-query", "https://223.6.6.6/dns-query"},
		"direct-nameserver-follow-policy": true,
		"nameserver":                      []string{"https://1.1.1.1/dns-query", "https://1.0.0.1/dns-query"},
		"nameserver-policy":               map[string]any{"rule-set:sr-private": []string{"system"}},
		"fake-ip-filter":                  fakeIPFilter,
	}
}

func clashFullTemplateRuleProviders() map[string]any {
	return map[string]any{
		"sr-private": clashFullTemplateRuleProvider("private.yaml", "domain"),
		"sr-reject":  clashFullTemplateRuleProvider("reject.yaml", "domain"),
		"sr-proxy":   clashFullTemplateRuleProvider("proxy.yaml", "domain"),
		"sr-direct":  clashFullTemplateRuleProvider("direct.yaml", "domain"),
		"sr-cn-ip":   clashFullTemplateRuleProvider("cn-ip.yaml", "ipcidr"),
	}
}

func clashFullTemplateRuleProvider(file string, behavior string) map[string]any {
	return map[string]any{
		"type":     "http",
		"behavior": behavior,
		"url":      clashFullTemplateProviderBase + file,
		"interval": clashFullTemplateProviderInterval,
		"proxy":    "PROXY",
	}
}

func clashFullTemplateSniffer() map[string]any {
	return map[string]any{
		"enable":        true,
		"parse-pure-ip": true,
		"sniff": map[string]any{
			"TLS":  map[string]any{"ports": []int{443, 8443}},
			"HTTP": map[string]any{"ports": []int{80, 8080, 8888}},
			"QUIC": map[string]any{"ports": []int{443, 8443}},
		},
	}
}

func (s *SubClashService) getProxies(inbound *model.Inbound, client model.Client, host string) []map[string]any {
	stream := s.streamData(inbound.StreamSettings)
	externalProxies, ok := stream["externalProxy"].([]any)
	if !ok || len(externalProxies) == 0 {
		externalProxies = []any{map[string]any{
			"forceTls": "same",
			"dest":     host,
			"port":     float64(inbound.Port),
			"remark":   "",
		}}
	}
	delete(stream, "externalProxy")

	proxies := make([]map[string]any, 0, len(externalProxies))
	for _, ep := range externalProxies {
		entry, ok := parseExternalProxyEntry(ep)
		if !ok {
			continue
		}
		workingInbound := *inbound
		workingInbound.Listen = entry.Dest
		workingInbound.Port = entry.Port
		workingStream := cloneMap(stream)

		switch entry.ForceTLS {
		case "tls":
			if workingStream["security"] != "tls" {
				workingStream["security"] = "tls"
				workingStream["tlsSettings"] = map[string]any{}
			}
		case "none":
			if workingStream["security"] != "none" {
				workingStream["security"] = "none"
				delete(workingStream, "tlsSettings")
				delete(workingStream, "realitySettings")
			}
		}

		proxy := s.buildProxy(&workingInbound, client, workingStream, entry.Remark)
		if len(proxy) > 0 {
			proxies = append(proxies, proxy)
		}
	}
	return proxies
}

func (s *SubClashService) buildProxy(inbound *model.Inbound, client model.Client, stream map[string]any, extraRemark string) map[string]any {
	// Hysteria has its own transport + TLS model, applyTransport /
	// applySecurity don't fit. IsHysteria also covers the literal
	// "hysteria2" protocol string (#4081).
	if model.IsHysteria(inbound.Protocol) {
		return s.buildHysteriaProxy(inbound, client, extraRemark)
	}

	proxy := map[string]any{
		"name":   s.SubService.genRemark(inbound, client.Email, extraRemark),
		"server": inbound.Listen,
		"port":   inbound.Port,
		"udp":    true,
	}

	network, _ := stream["network"].(string)

	switch inbound.Protocol {
	case model.VMESS:
		proxy["type"] = "vmess"
		proxy["uuid"] = client.ID
		proxy["alterId"] = 0
		cipher := client.Security
		if cipher == "" {
			cipher = "auto"
		}
		proxy["cipher"] = cipher
	case model.VLESS:
		proxy["type"] = "vless"
		proxy["uuid"] = client.ID
		if client.Flow != "" && network == "tcp" {
			proxy["flow"] = client.Flow
		}
		var inboundSettings map[string]any
		if !decodeJSONString(inbound.Settings, &inboundSettings, "vless inbound settings") {
			return nil
		}
		if encryption, ok := inboundSettings["encryption"].(string); ok {
			switch strings.TrimSpace(encryption) {
			case "packetaddr", "xudp":
				proxy["packet-encoding"] = encryption
			}
		}
	case model.Trojan:
		proxy["type"] = "trojan"
		proxy["password"] = client.Password
	case model.Shadowsocks:
		proxy["type"] = "ss"
		proxy["password"] = client.Password
		var inboundSettings map[string]any
		if !decodeJSONString(inbound.Settings, &inboundSettings, "shadowsocks inbound settings") {
			return nil
		}
		method, _ := inboundSettings["method"].(string)
		if method == "" {
			return nil
		}
		proxy["cipher"] = method
		if strings.HasPrefix(method, "2022") {
			if serverPassword, ok := inboundSettings["password"].(string); ok && serverPassword != "" {
				proxy["password"] = fmt.Sprintf("%s:%s", serverPassword, client.Password)
			}
		}
	default:
		return nil
	}
	if !s.applyTransport(proxy, network, stream) {
		return nil
	}

	security, _ := stream["security"].(string)
	if !s.applySecurity(proxy, security, stream) {
		return nil
	}

	return proxy
}

func (s *SubClashService) buildProxyAccountProxy(inbound *model.Inbound, account proxyAccount, host string, extraRemark string) map[string]any {
	server := inbound.Listen
	if server == "" || server == "0.0.0.0" || server == "::" || server == "::0" {
		server = host
	}
	proxy := map[string]any{
		"name":   s.SubService.genProxyAccountRemark(inbound, account, extraRemark),
		"server": server,
		"port":   inbound.Port,
	}
	if proxyAccountUsesAuth(account) {
		proxy["username"] = account.User
		proxy["password"] = account.Pass
	}
	switch inbound.Protocol {
	case model.HTTP:
		proxy["type"] = "http"
	case model.Mixed:
		proxy["type"] = "socks5"
		proxy["udp"] = mixedProxyUDPEnabled(inbound)
	default:
		return nil
	}
	return proxy
}

func mixedProxyUDPEnabled(inbound *model.Inbound) bool {
	var settings map[string]any
	if !decodeJSONString(inbound.Settings, &settings, "mixed proxy settings") {
		return false
	}
	udp, _ := settings["udp"].(bool)
	return udp
}

// buildHysteriaProxy produces a mihomo-compatible Clash entry for a
// Hysteria (v1) or Hysteria2 inbound. It reads `inbound.StreamSettings`
// directly instead of going through streamData/tlsData, because those
// helpers prune fields (like `allowInsecure` / the salamander obfs
// block) that the hysteria proxy wants preserved.
func (s *SubClashService) buildHysteriaProxy(inbound *model.Inbound, client model.Client, extraRemark string) map[string]any {
	var inboundSettings map[string]any
	if !decodeJSONString(inbound.Settings, &inboundSettings, "hysteria inbound settings") {
		return nil
	}

	proxyType := "hysteria2"
	authKey := "password"
	if v, ok := inboundSettings["version"].(float64); ok && int(v) == 1 {
		proxyType = "hysteria"
		authKey = "auth-str"
	}

	proxy := map[string]any{
		"name":   s.SubService.genRemark(inbound, client.Email, extraRemark),
		"type":   proxyType,
		"server": inbound.Listen,
		"port":   inbound.Port,
		"udp":    true,
		authKey:  client.Auth,
	}

	var rawStream map[string]any
	if !decodeJSONString(inbound.StreamSettings, &rawStream, "hysteria stream settings") {
		return nil
	}

	// TLS details — hysteria always uses TLS.
	if tlsSettings, ok := rawStream["tlsSettings"].(map[string]any); ok {
		if serverName, ok := tlsSettings["serverName"].(string); ok && serverName != "" {
			proxy["sni"] = serverName
		}
		if alpnList, ok := tlsSettings["alpn"].([]any); ok && len(alpnList) > 0 {
			out := make([]string, 0, len(alpnList))
			for _, a := range alpnList {
				if s, ok := a.(string); ok && s != "" {
					out = append(out, s)
				}
			}
			if len(out) > 0 {
				proxy["alpn"] = out
			}
		}
		if inner, ok := tlsSettings["settings"].(map[string]any); ok {
			if insecure, ok := inner["allowInsecure"].(bool); ok && insecure {
				proxy["skip-cert-verify"] = true
			}
			if fp, ok := inner["fingerprint"].(string); ok && fp != "" {
				proxy["client-fingerprint"] = fp
			}
		}
	}

	// Salamander obfs (Hysteria2). Read the same finalmask.udp[salamander]
	// block the subscription link generator uses.
	if finalmask, ok := rawStream["finalmask"].(map[string]any); ok {
		if udpMasks, ok := finalmask["udp"].([]any); ok {
			for _, m := range udpMasks {
				mask, _ := m.(map[string]any)
				if mask == nil || mask["type"] != "salamander" {
					continue
				}
				settings, _ := mask["settings"].(map[string]any)
				if pw, ok := settings["password"].(string); ok && pw != "" {
					proxy["obfs"] = "salamander"
					proxy["obfs-password"] = pw
					break
				}
			}
		}
	}

	if hopPorts := hysteriaHopPorts(rawStream); hopPorts != "" {
		proxy["ports"] = hopPorts
	}

	return proxy
}

func (s *SubClashService) applyTransport(proxy map[string]any, network string, stream map[string]any) bool {
	// Mihomo has no XMC transport. Dropping the mask would advertise a node
	// that cannot complete the server handshake; URI and Xray JSON retain it.
	for _, raw := range normalizedFinalMaskTCPMasks(stream["finalmask"]) {
		mask, _ := raw.(map[string]any)
		if mask["type"] == "xmc" {
			return false
		}
	}
	switch network {
	case "", "tcp":
		proxy["network"] = "tcp"
		tcp, _ := stream["tcpSettings"].(map[string]any)
		if tcp != nil {
			header, _ := tcp["header"].(map[string]any)
			if header != nil {
				typeStr, _ := header["type"].(string)
				if typeStr != "" && typeStr != "none" {
					if proxy["type"] == "ss" && typeStr == "http" && tcpHeaderAcceptsRootPath(header) {
						proxy["plugin"] = "obfs"
						opts := map[string]any{"mode": "http"}
						request, _ := header["request"].(map[string]any)
						headers, _ := request["headers"].(map[string]any)
						if host := searchHost(headers); host != "" {
							opts["host"] = host
						}
						proxy["plugin-opts"] = opts
						return true
					}
					return false
				}
			}
		}
		return true
	case "ws":
		proxy["network"] = "ws"
		ws, _ := stream["wsSettings"].(map[string]any)
		wsOpts := map[string]any{}
		if ws != nil {
			if path, ok := ws["path"].(string); ok && path != "" {
				wsOpts["path"] = path
			}
			host := ""
			if v, ok := ws["host"].(string); ok && v != "" {
				host = v
			} else if headers, ok := ws["headers"].(map[string]any); ok {
				host = searchHost(headers)
			}
			if host != "" {
				wsOpts["headers"] = map[string]any{"Host": host}
			}
		}
		if len(wsOpts) > 0 {
			proxy["ws-opts"] = wsOpts
		}
		return true
	case "grpc":
		proxy["network"] = "grpc"
		grpc, _ := stream["grpcSettings"].(map[string]any)
		grpcOpts := map[string]any{}
		if grpc != nil {
			if serviceName, ok := grpc["serviceName"].(string); ok && serviceName != "" {
				grpcOpts["grpc-service-name"] = serviceName
			}
		}
		if len(grpcOpts) > 0 {
			proxy["grpc-opts"] = grpcOpts
		}
		return true
	default:
		return false
	}
}

// Mihomo's simple-obfs always sends GET /. Xray must accept that path;
// malformed path values cannot be treated as an absent/default path.
func tcpHeaderAcceptsRootPath(header map[string]any) bool {
	request, _ := header["request"].(map[string]any)
	raw, present := request["path"]
	if !present || raw == nil {
		return true
	}
	switch paths := raw.(type) {
	case []any:
		if len(paths) == 0 {
			return true
		}
		for _, path := range paths {
			if path == "/" {
				return true
			}
		}
	case []string:
		if len(paths) == 0 {
			return true
		}
		for _, path := range paths {
			if path == "/" {
				return true
			}
		}
	}
	return false
}

func (s *SubClashService) applySecurity(proxy map[string]any, security string, stream map[string]any) bool {
	switch security {
	case "", "none":
		proxy["tls"] = false
		return true
	case "tls":
		proxy["tls"] = true
		tlsSettings, _ := stream["tlsSettings"].(map[string]any)
		if tlsSettings != nil {
			if serverName, ok := tlsSettings["serverName"].(string); ok && serverName != "" {
				proxy["servername"] = serverName
				switch proxy["type"] {
				case "trojan":
					proxy["sni"] = serverName
				}
			}
			if fingerprint, ok := tlsSettings["fingerprint"].(string); ok && fingerprint != "" {
				proxy["client-fingerprint"] = fingerprint
			}
		}
		return true
	case "reality":
		proxy["tls"] = true
		realitySettings, _ := stream["realitySettings"].(map[string]any)
		if realitySettings == nil {
			return false
		}
		if serverName := firstRealityString(realitySettings, "serverName", "serverNames"); serverName != "" {
			proxy["servername"] = serverName
		}
		realityOpts := map[string]any{}
		if publicKey := firstRealityString(realitySettings, "publicKey", "publicKey"); publicKey != "" {
			realityOpts["public-key"] = publicKey
		}
		if shortID := firstRealityString(realitySettings, "shortId", "shortIds"); shortID != "" {
			realityOpts["short-id"] = shortID
		}
		if len(realityOpts) > 0 {
			proxy["reality-opts"] = realityOpts
		}
		if fingerprint := firstRealityString(realitySettings, "fingerprint", "fingerprint"); fingerprint != "" {
			proxy["client-fingerprint"] = fingerprint
		}
		return true
	default:
		return false
	}
}

func firstRealityString(settings map[string]any, normalizedKey string, rawKey string) string {
	if value, ok := settings[normalizedKey].(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	if values, ok := settings[rawKey].([]any); ok {
		for _, value := range values {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				return text
			}
		}
	}
	clientSettings, _ := settings["settings"].(map[string]any)
	if value, ok := clientSettings[rawKey].(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return ""
}

func (s *SubClashService) streamData(stream string) map[string]any {
	var streamSettings map[string]any
	if !decodeJSONString(stream, &streamSettings, "clash stream settings") {
		return map[string]any{}
	}
	security, _ := streamSettings["security"].(string)
	switch security {
	case "tls":
		if tlsSettings, ok := streamSettings["tlsSettings"].(map[string]any); ok {
			streamSettings["tlsSettings"] = s.tlsData(tlsSettings)
		}
	case "reality":
		if realitySettings, ok := streamSettings["realitySettings"].(map[string]any); ok {
			streamSettings["realitySettings"] = s.realityData(realitySettings)
		}
	}
	delete(streamSettings, "sockopt")
	return streamSettings
}

func (s *SubClashService) tlsData(tData map[string]any) map[string]any {
	tlsData := make(map[string]any, 1)
	tlsClientSettings, _ := tData["settings"].(map[string]any)
	tlsData["serverName"] = tData["serverName"]
	tlsData["alpn"] = tData["alpn"]
	if fingerprint, ok := tlsClientSettings["fingerprint"].(string); ok {
		tlsData["fingerprint"] = fingerprint
	}
	return tlsData
}

func (s *SubClashService) realityData(rData map[string]any) map[string]any {
	rDataOut := make(map[string]any, 1)
	realityClientSettings, _ := rData["settings"].(map[string]any)
	if publicKey, ok := realityClientSettings["publicKey"].(string); ok {
		rDataOut["publicKey"] = publicKey
	}
	if fingerprint, ok := realityClientSettings["fingerprint"].(string); ok {
		rDataOut["fingerprint"] = fingerprint
	}
	if serverNames, ok := rData["serverNames"].([]any); ok && len(serverNames) > 0 {
		rDataOut["serverName"] = fmt.Sprint(serverNames[0])
	}
	if shortIDs, ok := rData["shortIds"].([]any); ok && len(shortIDs) > 0 {
		rDataOut["shortId"] = fmt.Sprint(shortIDs[0])
	}
	return rDataOut
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
