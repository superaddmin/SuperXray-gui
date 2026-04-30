package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

const (
	shadowsocks2022Prefix      = "2022-"
	shadowsocks2022AES128GCM   = "2022-blake3-aes-128-gcm"
	shadowsocksLegacyPassLen   = 16
	shadowsocksDefaultKeyBytes = 32
)

func isShadowsocks2022Method(method string) bool {
	return strings.HasPrefix(method, shadowsocks2022Prefix)
}

func shadowsocksKeyBytes(method string) int {
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
	return method
}

func randomShadowsocksCredential(method string) string {
	if !isShadowsocks2022Method(method) {
		return strings.ReplaceAll(uuid.NewString(), "-", "")[:shadowsocksLegacyPassLen]
	}

	keyBytes := shadowsocksKeyBytes(method)
	array := make([]byte, keyBytes)
	if _, err := rand.Read(array); err != nil {
		fallback := strings.ReplaceAll(uuid.NewString(), "-", "")
		return base64.StdEncoding.EncodeToString([]byte(fallback[:keyBytes]))
	}
	return base64.StdEncoding.EncodeToString(array)
}
