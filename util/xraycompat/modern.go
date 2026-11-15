package xraycompat

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
)

// Script routing is exclusive in Xray 26.10.21. Reject conflicts instead of
// silently discarding the administrator's saved rules or DNS policy.
func validateLuaConfig(cfg map[string]any) error {
	routing := stringMap(cfg["routing"])
	if stringValue(routing["script"]) != "" {
		rules, _ := routing["rules"].([]any)
		if len(rules) > 0 || stringValue(routing["domainStrategy"]) != "" {
			return errors.New("routing.script cannot be combined with rules or domainStrategy")
		}
	}
	dns := stringMap(cfg["dns"])
	if stringValue(dns["script"]) == "" {
		return nil
	}
	for _, key := range []string{"disableFallback", "disableFallbackIfMatch", "enableParallelQuery"} {
		if enabled, _ := dns[key].(bool); enabled {
			return fmt.Errorf("dns.script cannot be combined with %s", key)
		}
	}
	servers, _ := dns["servers"].([]any)
	for _, raw := range servers {
		server := stringMap(raw)
		domains, _ := server["domains"].([]any)
		skip, _ := server["skipFallback"].(bool)
		final, _ := server["finalQuery"].(bool)
		if len(domains) > 0 || skip || final {
			return errors.New("dns.script cannot be combined with server domains, skipFallback or finalQuery")
		}
	}
	return nil
}

func validateXDriveInbound(protocol, rawSettings, rawStream string) error {
	var stream map[string]any
	if strings.TrimSpace(rawStream) == "" || decode(rawStream, &stream) != nil || stream["network"] != "xdrive" {
		return nil
	}
	if protocol != "vless" {
		return errors.New("XDRIVE transport requires VLESS")
	}
	var settings map[string]any
	if err := decode(rawSettings, &settings); err != nil {
		return err
	}
	if value := stringValue(settings["decryption"]); value == "" || value == "none" {
		return errors.New("XDRIVE transport requires VLESS decryption")
	}
	clients, _ := settings["clients"].([]any)
	for _, raw := range clients {
		if stringValue(stringMap(raw)["flow"]) != "" {
			return errors.New("XDRIVE transport does not support VLESS flow")
		}
	}
	return nil
}

func validateXDriveOutbound(protocol string, settings, stream map[string]any, mux any) error {
	if stream["network"] != "xdrive" {
		return nil
	}
	if protocol != "vless" {
		return errors.New("XDRIVE transport requires VLESS")
	}
	users := []map[string]any{settings}
	if servers, ok := settings["vnext"].([]any); ok {
		users = nil
		for _, raw := range servers {
			accounts, _ := stringMap(raw)["users"].([]any)
			for _, account := range accounts {
				users = append(users, stringMap(account))
			}
		}
	}
	if len(users) == 0 {
		return errors.New("XDRIVE transport requires a VLESS account")
	}
	for _, user := range users {
		if value := stringValue(user["encryption"]); value == "" || value == "none" {
			return errors.New("XDRIVE transport requires VLESS encryption")
		}
		if stringValue(user["flow"]) != "" || numberValue(user["testpre"]) != 0 {
			return errors.New("XDRIVE transport does not support VLESS flow or testpre")
		}
		// Reverse opens its own mux; ordinary users must explicitly enable it.
		if reverse, ok := user["reverse"].(map[string]any); ok && reverse != nil {
			continue
		}
		raw, err := json.Marshal(mux)
		if err != nil || !XDriveMuxValid(string(raw)) {
			return errors.New("XDRIVE transport requires enabled mux, non-negative concurrency and xudpConcurrency, and xudpProxyUDP443 other than skip")
		}
	}
	return nil
}

// XDriveMuxValid reports whether mux will carry every XDRIVE connection.
func XDriveMuxValid(raw string) bool {
	var mux struct {
		Enabled         bool   `json:"enabled"`
		Concurrency     int16  `json:"concurrency"`
		XUDPConcurrency int16  `json:"xudpConcurrency"`
		XUDPProxyUDP443 string `json:"xudpProxyUDP443"`
	}
	if json.Unmarshal([]byte(raw), &mux) != nil || !mux.Enabled || mux.Concurrency < 0 || mux.XUDPConcurrency < 0 {
		return false
	}
	return mux.XUDPProxyUDP443 == "" || mux.XUDPProxyUDP443 == "reject" || mux.XUDPProxyUDP443 == "allow"
}

// ClientFinalMask projects server masks for subscriptions without changing the
// stored map. Source-address filters only apply to listeners and are private.
func ClientFinalMask(finalmask map[string]any) map[string]any {
	if finalmask == nil {
		return nil
	}
	result := maps.Clone(finalmask)
	for _, key := range []string{"tcp", "udp"} {
		masks, ok := finalmask[key].([]any)
		if !ok {
			continue
		}
		kept := make([]any, 0, len(masks))
		for _, raw := range masks {
			if !strings.EqualFold(stringValue(stringMap(raw)["type"]), "filter") {
				kept = append(kept, raw)
			}
		}
		if len(kept) == 0 {
			delete(result, key)
		} else {
			result[key] = kept
		}
	}
	return result
}

func normalizeLegacyKCPMask(mask map[string]any) (bool, error) {
	typ := stringValue(mask["type"])
	header, isHeader := map[string]string{"header-dns": "dns", "header-dtls": "dtls", "header-srtp": "srtp", "header-utp": "utp", "header-wechat": "wechat", "header-wireguard": "wireguard"}[typ]
	if !isHeader && typ != "mkcp-original" && typ != "mkcp-aes128gcm" {
		return false, nil
	}
	settings := stringMap(mask["settings"])
	if settings == nil {
		if raw, exists := mask["settings"]; exists && raw != nil {
			return false, errors.New("legacy KCP mask settings must be an object")
		}
		settings = map[string]any{}
	}
	if existing, exists := settings["header"]; exists && existing != header {
		return false, errors.New("legacy KCP mask header conflicts with its type")
	}
	if isHeader {
		settings["header"] = header
	}
	legacyKey := ""
	if typ == "mkcp-aes128gcm" {
		if stringValue(settings["password"]) == "" {
			return false, errors.New("legacy KCP AES mask needs a non-empty password; an empty value would disable encryption")
		}
		legacyKey = "password"
	}
	if typ == "header-dns" {
		legacyKey = "domain"
	}
	if legacyKey != "" {
		if value, exists := settings[legacyKey]; exists {
			if _, ok := value.(string); !ok {
				return false, errors.New("legacy KCP mask value must be a string")
			}
			if current, exists := settings["value"]; exists && !reflect.DeepEqual(current, value) {
				return false, errors.New("legacy KCP mask value conflicts with its legacy field")
			}
			settings["value"] = value
			delete(settings, legacyKey)
		}
	} else if value := stringValue(settings["value"]); value != "" && !isHeader {
		return false, errors.New("original KCP mask cannot gain an encryption value")
	}
	mask["type"] = "mkcp-legacy"
	if len(settings) > 0 {
		mask["settings"] = settings
	}
	return true, nil
}
