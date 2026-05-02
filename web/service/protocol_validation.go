package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func validateInboundProtocolConfig(inbound *model.Inbound) error {
	if inbound == nil {
		return fmt.Errorf("inbound is nil")
	}

	clients, err := parseInboundClients(inbound.Settings)
	if err != nil {
		return fmt.Errorf("invalid %s settings: %w", inbound.Protocol, err)
	}
	return validateInboundProtocolClients(inbound.Protocol, inbound.Settings, inbound.StreamSettings, clients)
}

func validateInboundProtocolClients(protocol model.Protocol, settingsText string, streamSettingsText string, clients []model.Client) error {
	settings := map[string]any{}
	if strings.TrimSpace(settingsText) != "" {
		if err := json.Unmarshal([]byte(settingsText), &settings); err != nil {
			return fmt.Errorf("invalid %s settings: %w", protocol, err)
		}
	}

	streamSettings := map[string]any{}
	if strings.TrimSpace(streamSettingsText) != "" {
		if err := json.Unmarshal([]byte(streamSettingsText), &streamSettings); err != nil {
			return fmt.Errorf("invalid %s stream settings: %w", protocol, err)
		}
	}

	if protocol == model.Shadowsocks {
		if err := validateShadowsocksSettings(settings, clients); err != nil {
			return fmt.Errorf("%s settings: %w", protocol, err)
		}
	}
	for _, client := range clients {
		if err := validateProtocolClient(protocol, settings, streamSettings, client); err != nil {
			if client.Email != "" {
				return fmt.Errorf("%s client %q: %w", protocol, client.Email, err)
			}
			return fmt.Errorf("%s client: %w", protocol, err)
		}
	}
	if err := validateInboundProtocolStream(protocol, streamSettings); err != nil {
		return fmt.Errorf("%s stream settings: %w", protocol, err)
	}
	return nil
}

func validateInboundProtocolStream(protocol model.Protocol, streamSettings map[string]any) error {
	security, _ := streamSettings["security"].(string)
	if protocol == model.Hysteria || protocol == model.Hysteria2 {
		if security != "tls" {
			return fmt.Errorf("hysteria requires tls security")
		}
	}
	if security != "tls" {
		return nil
	}
	return validateTLSCertificates(streamSettings)
}

func validateTLSCertificates(streamSettings map[string]any) error {
	tlsSettings, _ := streamSettings["tlsSettings"].(map[string]any)
	certificates, _ := tlsSettings["certificates"].([]any)
	if len(certificates) == 0 {
		return fmt.Errorf("tls certificate is required")
	}

	for _, entry := range certificates {
		cert, ok := entry.(map[string]any)
		if !ok {
			return fmt.Errorf("tls certificate entry is invalid")
		}
		if _, hasCertFile := cert["certificateFile"]; hasCertFile {
			if isBlankString(cert["certificateFile"]) || isBlankString(cert["keyFile"]) {
				return fmt.Errorf("tls certificate file paths are incomplete")
			}
			continue
		}
		if hasNonBlankStringValue(cert["certificate"]) && hasNonBlankStringValue(cert["key"]) {
			continue
		}
		return fmt.Errorf("tls certificate content is incomplete")
	}
	return nil
}

func isBlankString(value any) bool {
	text, _ := value.(string)
	return strings.TrimSpace(text) == ""
}

func hasNonBlankStringValue(value any) bool {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case []any:
		for _, item := range v {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				return true
			}
		}
	}
	return false
}

func validateProtocolClient(protocol model.Protocol, settings map[string]any, streamSettings map[string]any, client model.Client) error {
	switch protocol {
	case model.VMESS:
		return validateClientUUID(client.ID)
	case model.VLESS:
		if err := validateClientUUID(client.ID); err != nil {
			return err
		}
		return validateVLESSFlow(streamSettings, client.Flow)
	case model.Trojan:
		if strings.TrimSpace(client.Password) == "" {
			return fmt.Errorf("password is required")
		}
	case model.Shadowsocks:
		return validateShadowsocksClient(settings, client)
	case model.Hysteria, model.Hysteria2:
		if strings.TrimSpace(client.Auth) == "" {
			return fmt.Errorf("auth is required")
		}
	}
	return nil
}

func validateShadowsocksSettings(settings map[string]any, clients []model.Client) error {
	method, _ := settings["method"].(string)
	method = normalizeShadowsocksMethodName(method)
	if method == "" {
		return fmt.Errorf("shadowsocks method is required")
	}

	if !isShadowsocks2022Method(method) {
		return nil
	}

	serverPassword, _ := settings["password"].(string)
	if err := validateShadowsocks2022Key(method, serverPassword); err != nil {
		return fmt.Errorf("shadowsocks server key: %w", err)
	}
	if method == shadowsocks2022Blake3Chacha20Poly1305 && len(clients) > 0 {
		return fmt.Errorf("shadowsocks 2022 chacha20-poly1305 does not support multi-user clients")
	}
	return nil
}

func validateClientUUID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("uuid is required")
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("uuid is invalid")
	}
	return nil
}

func validateVLESSFlow(streamSettings map[string]any, flow string) error {
	if strings.TrimSpace(flow) == "" {
		return nil
	}

	network, _ := streamSettings["network"].(string)
	security, _ := streamSettings["security"].(string)
	if network != "tcp" || (security != "tls" && security != "reality") {
		return fmt.Errorf("flow requires tcp with tls or reality")
	}
	return nil
}

func validateShadowsocksClient(settings map[string]any, client model.Client) error {
	method, _ := settings["method"].(string)
	method = normalizeShadowsocksMethodName(method)
	if method == "" {
		return fmt.Errorf("shadowsocks method is required")
	}

	if !isShadowsocks2022Method(method) {
		clientMethod := normalizeShadowsocksMethodName(client.Method)
		if clientMethod == "" {
			return fmt.Errorf("shadowsocks client method is required")
		}
		if clientMethod != method {
			return fmt.Errorf("shadowsocks client method %q does not match inbound method %q", clientMethod, method)
		}
		if strings.TrimSpace(client.Password) == "" {
			return fmt.Errorf("shadowsocks password is required")
		}
		return nil
	}

	if strings.TrimSpace(client.Method) != "" {
		return fmt.Errorf("shadowsocks 2022 client method must be empty")
	}
	if err := validateShadowsocks2022Key(method, client.Password); err != nil {
		return fmt.Errorf("shadowsocks client key: %w", err)
	}
	return nil
}

func validateShadowsocks2022Key(method string, key string) error {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(key))
	if err != nil {
		return fmt.Errorf("must be base64")
	}
	expectedBytes := shadowsocksKeyBytes(method)
	if len(decoded) != expectedBytes {
		return fmt.Errorf("decoded length = %d, want %d", len(decoded), expectedBytes)
	}
	return nil
}
