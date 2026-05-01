package service

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestRandomURLSafeCredentialUsesRequestedRandomBytes(t *testing.T) {
	credential := randomURLSafeCredential(generatedCredentialBytes)

	if strings.ContainsAny(credential, "+/=") {
		t.Fatalf("credential %q is not URL-safe base64 without padding", credential)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(credential)
	if err != nil {
		t.Fatalf("DecodeString returned error: %v", err)
	}
	if len(decoded) != generatedCredentialBytes {
		t.Fatalf("decoded length = %d, want %d", len(decoded), generatedCredentialBytes)
	}
}

func TestRandomShadowsocksCredentialLegacyUsesUnifiedStrongSecret(t *testing.T) {
	credential := randomShadowsocksCredential("chacha20-ietf-poly1305")

	decoded, err := base64.RawURLEncoding.DecodeString(credential)
	if err != nil {
		t.Fatalf("DecodeString returned error: %v", err)
	}
	if len(decoded) != generatedCredentialBytes {
		t.Fatalf("decoded length = %d, want %d", len(decoded), generatedCredentialBytes)
	}
}

func TestRandomShadowsocksCredential2022KeepsProtocolKeyLength(t *testing.T) {
	credential := randomShadowsocksCredential(shadowsocks2022AES128GCM)

	decoded, err := base64.StdEncoding.DecodeString(credential)
	if err != nil {
		t.Fatalf("DecodeString returned error: %v", err)
	}
	if len(decoded) != 16 {
		t.Fatalf("decoded length = %d, want 16", len(decoded))
	}
}

func TestGenerateRandomCredentialForPasswordProtocolsUsesUnifiedStrongSecret(t *testing.T) {
	credential := (&InboundService{}).generateRandomCredential(model.Trojan)

	decoded, err := base64.RawURLEncoding.DecodeString(credential)
	if err != nil {
		t.Fatalf("DecodeString returned error: %v", err)
	}
	if len(decoded) != generatedCredentialBytes {
		t.Fatalf("decoded length = %d, want %d", len(decoded), generatedCredentialBytes)
	}
}
