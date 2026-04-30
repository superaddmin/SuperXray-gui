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

	for _, client := range clients {
		if err := validateProtocolClient(protocol, settings, streamSettings, client); err != nil {
			if client.Email != "" {
				return fmt.Errorf("%s client %q: %w", protocol, client.Email, err)
			}
			return fmt.Errorf("%s client: %w", protocol, err)
		}
	}
	return nil
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
	method = strings.TrimSpace(method)
	if method == "" {
		return fmt.Errorf("shadowsocks method is required")
	}

	if !isShadowsocks2022Method(method) {
		if strings.TrimSpace(client.Password) == "" {
			return fmt.Errorf("shadowsocks password is required")
		}
		return nil
	}

	serverPassword, _ := settings["password"].(string)
	if err := validateShadowsocks2022Key(method, serverPassword); err != nil {
		return fmt.Errorf("shadowsocks server key: %w", err)
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
