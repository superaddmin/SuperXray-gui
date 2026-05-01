package service

import (
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestValidateInboundProtocolConfigRejectsVMessInvalidUUID(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.VMESS,
		`{"clients":[{"id":"not-a-uuid","email":"vmess@example","enable":true}]}`,
		`{"network":"tcp","security":"none"}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "uuid")
}

func TestValidateInboundProtocolConfigRejectsVLESSFlowOutsideTCPWithTLSOrReality(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.VLESS,
		`{"clients":[{"id":"00000000-0000-4000-8000-000000000000","email":"vless@example","flow":"xtls-rprx-vision","enable":true}],"decryption":"none","encryption":"none"}`,
		`{"network":"ws","security":"tls"}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "flow")
}

func TestValidateInboundProtocolConfigRejectsTrojanEmptyPassword(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.Trojan,
		`{"clients":[{"password":"","email":"trojan@example","enable":true}]}`,
		`{"network":"tcp","security":"tls"}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "password")
}

func TestValidateInboundProtocolConfigRejectsShadowsocks2022InvalidClientKey(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.Shadowsocks,
		`{"method":"2022-blake3-aes-128-gcm","password":"MDEyMzQ1Njc4OWFiY2RlZg==","clients":[{"email":"ss@example","password":"short","enable":true}]}`,
		`{"network":"tcp","security":"none"}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "shadowsocks")
}

func TestValidateInboundProtocolConfigRejectsShadowsocks2022InvalidServerKeyWithoutClients(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.Shadowsocks,
		`{"method":"2022-blake3-chacha20-poly1305","password":"short","clients":[]}`,
		`{"network":"tcp","security":"none"}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "server key")
}

func TestValidateInboundProtocolConfigRejectsShadowsocks2022ClientMethod(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.Shadowsocks,
		`{"method":"2022-blake3-aes-128-gcm","password":"MDEyMzQ1Njc4OWFiY2RlZg==","clients":[{"method":"chacha20-ietf-poly1305","email":"ss@example","password":"MDEyMzQ1Njc4OWFiY2RlZg==","enable":true}]}`,
		`{"network":"tcp","security":"none"}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "method")
}

func TestValidateInboundProtocolConfigRejectsShadowsocks2022ChachaMultiUser(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.Shadowsocks,
		`{"method":"2022-blake3-chacha20-poly1305","password":"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=","clients":[{"email":"ss@example","password":"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=","enable":true}]}`,
		`{"network":"tcp","security":"none"}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "multi-user")
}

func TestValidateInboundProtocolConfigRejectsShadowsocksLegacyClientMissingMethod(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.Shadowsocks,
		`{"method":"chacha20-ietf-poly1305","clients":[{"email":"ss@example","password":"legacy-password","enable":true}]}`,
		`{"network":"tcp","security":"none"}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "method")
}

func TestValidateInboundProtocolConfigRejectsHysteriaEmptyAuth(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.Hysteria,
		`{"version":2,"clients":[{"auth":"","email":"hy@example","enable":true}]}`,
		`{"network":"hysteria","security":"tls"}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "auth")
}

func TestValidateInboundProtocolConfigRejectsHysteriaEmptyTLSCertificate(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.Hysteria,
		`{"version":2,"clients":[{"auth":"hy-auth","email":"hy@example","enable":true}]}`,
		`{"network":"hysteria","security":"tls","tlsSettings":{"certificates":[{"certificateFile":"","keyFile":"","usage":"encipherment"}]}}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "tls certificate")
}

func TestValidateInboundProtocolConfigRejectsHysteriaWithoutTLS(t *testing.T) {
	inbound := inboundForProtocolValidation(
		model.Hysteria2,
		`{"version":2,"clients":[{"auth":"hy-auth","email":"hy@example","enable":true}]}`,
		`{"network":"hysteria","security":"none"}`,
	)

	err := validateInboundProtocolConfig(inbound)
	assertValidationErrorContains(t, err, "requires tls")
}

func TestValidateInboundProtocolConfigAcceptsValidMainstreamClients(t *testing.T) {
	cases := []*model.Inbound{
		inboundForProtocolValidation(
			model.VLESS,
			`{"clients":[{"id":"00000000-0000-4000-8000-000000000000","email":"vless@example","flow":"xtls-rprx-vision","enable":true}],"decryption":"none","encryption":"none"}`,
			`{"network":"tcp","security":"reality"}`,
		),
		inboundForProtocolValidation(
			model.Shadowsocks,
			`{"method":"chacha20-ietf-poly1305","clients":[{"method":"chacha20-ietf-poly1305","email":"ss@example","password":"legacy-password","enable":true}]}`,
			`{"network":"tcp","security":"none"}`,
		),
		inboundForProtocolValidation(
			model.Shadowsocks,
			`{"method":"2022-blake3-chacha20-poly1305","password":"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=","clients":[]}`,
			`{"network":"tcp","security":"none"}`,
		),
		inboundForProtocolValidation(
			model.Hysteria2,
			`{"version":2,"clients":[{"auth":"hy-auth","email":"hy@example","enable":true}]}`,
			`{"network":"hysteria","security":"tls","tlsSettings":{"certificates":[{"certificate":["-----BEGIN CERTIFICATE-----","MIIB","-----END CERTIFICATE-----"],"key":["-----BEGIN PRIVATE KEY-----","MIIB","-----END PRIVATE KEY-----"],"usage":"encipherment"}]}}`,
		),
	}

	for _, inbound := range cases {
		if err := validateInboundProtocolConfig(inbound); err != nil {
			t.Fatalf("validateInboundProtocolConfig(%s) returned error: %v", inbound.Protocol, err)
		}
	}
}

func inboundForProtocolValidation(protocol model.Protocol, settings string, streamSettings string) *model.Inbound {
	return &model.Inbound{
		Protocol:       protocol,
		Settings:       settings,
		StreamSettings: streamSettings,
	}
}

func assertValidationErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("validateInboundProtocolConfig returned nil error, want %q", want)
	}
	if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(want)) {
		t.Fatalf("error = %q, want contain %q", err.Error(), want)
	}
}
