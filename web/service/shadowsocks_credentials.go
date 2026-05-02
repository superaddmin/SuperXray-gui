package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

const (
	shadowsocks2022Prefix                 = "2022-"
	shadowsocks2022AES128GCM              = "2022-blake3-aes-128-gcm"
	shadowsocks2022Blake3Chacha20Poly1305 = "2022-blake3-chacha20-poly1305"
	generatedCredentialBytes              = 32
	shadowsocksDefaultKeyBytes            = 32
)

func randomURLSafeCredential(keyBytes int) string {
	array := make([]byte, keyBytes)
	if _, err := rand.Read(array); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(array)
}

func normalizeShadowsocksMethodName(method string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(method), "_", "-"))
}

func isShadowsocks2022Method(method string) bool {
	return strings.HasPrefix(normalizeShadowsocksMethodName(method), shadowsocks2022Prefix)
}

func shadowsocksKeyBytes(method string) int {
	method = normalizeShadowsocksMethodName(method)
	if method == shadowsocks2022AES128GCM {
		return 16
	}
	return shadowsocksDefaultKeyBytes
}

func shadowsocksMethodFromSettings(settingsText string) string {
	var settings map[string]any
	if err := json.Unmarshal([]byte(settingsText), &settings); err != nil {
		return ""
	}
	method, _ := settings["method"].(string)
	return normalizeShadowsocksMethodName(method)
}

func normalizeShadowsocksInboundSettings(inbound *model.Inbound) error {
	if inbound == nil || inbound.Protocol != model.Shadowsocks {
		return nil
	}

	normalized, err := normalizeShadowsocksSettingsText(inbound.Settings)
	if err != nil {
		return err
	}
	inbound.Settings = normalized
	return nil
}

func normalizeShadowsocksSettingsText(settingsText string) (string, error) {
	var settings map[string]any
	if err := json.Unmarshal([]byte(settingsText), &settings); err != nil {
		return settingsText, err
	}
	if settings == nil {
		return settingsText, nil
	}

	normalizeShadowsocksSettingsMap(settings)

	normalized, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return settingsText, err
	}
	return string(normalized), nil
}

func normalizeShadowsocksSettingsMap(settings map[string]any) {
	method, _ := settings["method"].(string)
	method = normalizeShadowsocksMethodName(method)
	if method == "" {
		return
	}
	settings["method"] = method

	if !isShadowsocks2022Method(method) {
		delete(settings, "password")
	}

	rawClients, ok := settings["clients"].([]any)
	if !ok {
		return
	}

	if method == shadowsocks2022Blake3Chacha20Poly1305 {
		settings["clients"] = []any{}
		return
	}

	normalizeShadowsocksClientEntries(method, rawClients)
	settings["clients"] = rawClients
}

func normalizeShadowsocksClientEntries(method string, clients []any) {
	method = normalizeShadowsocksMethodName(method)
	if method == "" {
		return
	}

	for i, rawClient := range clients {
		client, ok := rawClient.(map[string]any)
		if !ok {
			continue
		}
		if isShadowsocks2022Method(method) {
			delete(client, "method")
		} else {
			client["method"] = method
		}
		clients[i] = client
	}
}

func randomShadowsocksCredential(method string) string {
	if !isShadowsocks2022Method(method) {
		return randomURLSafeCredential(generatedCredentialBytes)
	}

	keyBytes := shadowsocksKeyBytes(method)
	array := make([]byte, keyBytes)
	if _, err := rand.Read(array); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(array)
}
