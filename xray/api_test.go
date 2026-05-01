package xray

import (
	"testing"

	"github.com/xtls/xray-core/proxy/shadowsocks"
)

func TestShadowsocksCipherTypeFromStringSupportsLegacyAEAD(t *testing.T) {
	cases := map[string]shadowsocks.CipherType{
		"aes-128-gcm":             shadowsocks.CipherType_AES_128_GCM,
		"aead_aes_128_gcm":        shadowsocks.CipherType_AES_128_GCM,
		"aes-256-gcm":             shadowsocks.CipherType_AES_256_GCM,
		"aead_aes_256_gcm":        shadowsocks.CipherType_AES_256_GCM,
		"chacha20-poly1305":       shadowsocks.CipherType_CHACHA20_POLY1305,
		"chacha20-ietf-poly1305":  shadowsocks.CipherType_CHACHA20_POLY1305,
		"xchacha20-poly1305":      shadowsocks.CipherType_XCHACHA20_POLY1305,
		"xchacha20-ietf-poly1305": shadowsocks.CipherType_XCHACHA20_POLY1305,
		"2022-blake3-aes-128-gcm": shadowsocks.CipherType_NONE,
		"2022-blake3-aes-256-gcm": shadowsocks.CipherType_NONE,
	}

	for method, want := range cases {
		if got := shadowsocksCipherTypeFromString(method); got != want {
			t.Fatalf("shadowsocksCipherTypeFromString(%q) = %v, want %v", method, got, want)
		}
	}
}
