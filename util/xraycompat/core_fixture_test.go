package xraycompat

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/xtls/xray-core/infra/conf"
	xraytls "github.com/xtls/xray-core/transport/internet/tls"
)

func TestLegacyTLSPinsBuildWithSameTrustInTargetCore(t *testing.T) {
	raw := `{"tlsSettings":{"pinnedPeerCertSha256":["` + strings.Repeat("01", 32) + `","` + strings.Repeat("02", 32) + `"]}}`
	normalized, _, err := NormalizeStream(raw)
	if err != nil {
		t.Fatal(err)
	}
	var stream struct {
		TLS conf.TLSConfig `json:"tlsSettings"`
	}
	if err := json.Unmarshal([]byte(normalized), &stream); err != nil {
		t.Fatal(err)
	}
	built, err := stream.TLS.Build()
	if err != nil {
		t.Fatalf("target core did not retain both certificate pins: %v", err)
	}
	tlsConfig := built.(*xraytls.Config)
	if len(tlsConfig.PinnedPeerCertSha256) != 2 || len(tlsConfig.PinnedPeerCertSha256[0]) != 32 || tlsConfig.PinnedPeerCertSha256[0][0] != 1 || tlsConfig.PinnedPeerCertSha256[1][0] != 2 {
		t.Fatal("target core changed the trusted certificate hashes")
	}
}

// Build through the pinned Xray v26.10.10 loader, rather than merely proving
// that two panel-side serializers agree on an unsupported JSON shape.
func TestMigratedInboundFixturesBuildInTargetCore(t *testing.T) {
	certs := fixtureCertificate(t)
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	reality := map[string]any{"dest": "127.0.0.1:443", "serverNames": []string{"fixture.test"}, "privateKey": base64.RawURLEncoding.EncodeToString(key.Bytes()), "shortIds": []string{"0123456789abcdef"}}
	transports := []struct {
		name   string
		stream map[string]any
	}{
		{"tcp", map[string]any{"network": "tcp"}},
		{"tcp_xmc_profiles", map[string]any{"network": "tcp", "finalmask": map[string]any{"tcp": []any{map[string]any{"type": "xmc", "settings": map[string]any{"password": "fixture-password", "profiles": []any{map[string]any{"username": "Fixture_1", "uuid": "00000000-0000-4000-8000-000000000001", "texturesValue": "fixture-value", "texturesSignature": "fixture-signature"}}}}}}}},
		{"ws_tls", map[string]any{"network": "ws", "wsSettings": map[string]any{"path": "/fixture"}, "security": "tls", "tlsSettings": certs}},
		{"grpc_tls", map[string]any{"network": "grpc", "grpcSettings": map[string]any{"serviceName": "fixture"}, "security": "tls", "tlsSettings": certs}},
		{"xhttp_tls_legacy", map[string]any{"method": "xhttp", "xhttpSettings": map[string]any{"path": "/fixture", "extra": map[string]any{"sessionPlacement": "header", "sessionKey": "X-Session", "xmux": map[string]any{"maxConcurrency": "16-32"}}}, "security": "tls", "tlsSettings": certs}},
		{"tcp_reality_legacy", map[string]any{"network": "tcp", "security": "reality", "realitySettings": reality, "sockopt": map[string]any{"V6Only": false}}},
		{"hysteria_udp_hop", map[string]any{"network": "hysteria", "hysteriaSettings": map[string]any{"version": 2}, "security": "tls", "tlsSettings": certs, "finalmask": map[string]any{"quicParams": map[string]any{"udpHop": map[string]any{"ports": "20000-30000", "interval": 30}}, "udp": []any{map[string]any{"type": "salamander", "settings": map[string]any{"password": "fixture-obfuscation"}}}}}},
	}
	for _, tc := range transports {
		t.Run(tc.name, func(t *testing.T) {
			rawStream, _ := json.Marshal(tc.stream)
			protocol := "vless"
			settings := `{"decryption":"none","clients":[{"id":"00000000-0000-4000-8000-000000000001","email":"fixture"}]}`
			if tc.name == "hysteria_udp_hop" {
				protocol = "hysteria"
				settings = `{"version":2,"clients":[{"auth":"fixture-auth","email":"fixture"}]}`
			}
			settings, stream, _, err := NormalizeInbound(protocol, settings, string(rawStream))
			if err != nil {
				t.Fatal(err)
			}
			stream, err = InboundRuntimeStream(stream)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(stream, `"intervalRemote"`) {
				t.Fatal("client UDP hop leaked into listener")
			}
			wire, _ := json.Marshal(map[string]any{"protocol": protocol, "listen": "127.0.0.1", "port": 19443, "settings": json.RawMessage(settings), "streamSettings": json.RawMessage(stream)})
			var cfg conf.InboundDetourConfig
			if err := json.Unmarshal(wire, &cfg); err != nil {
				t.Fatal(err)
			}
			if _, err := cfg.Build(); err != nil {
				t.Fatalf("target core rejects migrated %s: %v", tc.name, err)
			}
		})
	}
}

func TestSessionLengthFixturesMatchTargetCore(t *testing.T) {
	for _, value := range []string{`16`, `"16"`, `"32-16"`, `"8-16"`} {
		t.Run(value, func(t *testing.T) {
			raw := `{"network":"xhttp","xhttpSettings":{"sessionIDTable":"Base62","sessionIDLength":` + value + `}}`
			got, _, err := NormalizeStream(raw)
			if err != nil {
				t.Fatal(err)
			}
			var cfg conf.StreamConfig
			if err := json.Unmarshal([]byte(got), &cfg); err != nil {
				t.Fatal(err)
			}
			if _, err := cfg.Build(); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, value := range []string{`2147483647`, `"1-2147483647"`} {
		var target conf.Int32Range
		if err := json.Unmarshal([]byte(value), &target); err != nil {
			t.Fatal(err)
		}
		if target.To != 2147483647 || target.From <= 0 {
			t.Fatal(target)
		}
	}
}

func TestMigratedOutboundAndDNSFixturesBuildInTargetCore(t *testing.T) {
	fixtures := []string{
		`{"outbounds":[{"protocol":"freedom","tag":"direct","targetStrategy":"UseIPv4","settings":{"domainStrategy":"UseIPv6"},"streamSettings":{"sockopt":{"domainStrategy":"AsIs","addressPortStrategy":"none"}}}]}`,
		`{"outbounds":[{"protocol":"dns","tag":"dns","settings":{"nonIPQuery":"reject","blockTypes":[0,16]}}]}`,
		`{"outbounds":[{"protocol":"freedom","tag":"direct","settings":{},"proxySettings":{"tag":"proxy"}},{"protocol":"socks","tag":"proxy","settings":{"servers":[{"address":"127.0.0.1","port":1080}]}}]}`,
		`{"dns":{"servers":["localhost",{"address":"https://dns.example.test/dns-query","skipFallback":true}]},"routing":{"rules":[{"type":"field","inboundTag":["api"],"outboundTag":"api"},{"type":"field","network":"udp","port":"53","outboundTag":"direct"}]},"outbounds":[{"protocol":"freedom","tag":"direct"}]}`,
	}
	for _, raw := range fixtures {
		t.Run(raw, func(t *testing.T) {
			got, _, err := NormalizeConfig(raw)
			if err != nil {
				t.Fatal(err)
			}
			var cfg conf.Config
			if err := json.Unmarshal([]byte(got), &cfg); err != nil {
				t.Fatal(err)
			}
			if _, err := cfg.Build(); err != nil {
				t.Fatalf("target core rejects migrated fixture: %v", err)
			}
		})
	}
}

func fixtureCertificate(t *testing.T) map[string]any {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "fixture.test"}, DNSNames: []string{"fixture.test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := strings.Split(strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))), "\n")
	keyPEM := strings.Split(strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pk}))), "\n")
	return map[string]any{"certificates": []any{map[string]any{"certificate": certPEM, "key": keyPEM}}}
}
