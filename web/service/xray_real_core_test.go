package service

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/json_util"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

func isolatedRealCore(t *testing.T) {
	t.Helper()
	binary := os.Getenv("SUPERXRAY_TEST_XRAY_BINARY")
	if binary == "" {
		t.Skip("set SUPERXRAY_TEST_XRAY_BINARY to run isolated real-core integration")
	}
	source, err := os.Open(binary)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	bin := t.TempDir()
	destination, err := os.OpenFile(filepath.Join(bin, xray.GetBinaryName()), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o700)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		t.Fatalf("copy isolated core: %v, %v", copyErr, closeErr)
	}
	t.Setenv("XUI_BIN_FOLDER", bin)
	t.Setenv("XUI_LOG_FOLDER", t.TempDir())
}

func TestRealXrayRestartPreservesWorkingRuntime(t *testing.T) {
	isolatedRealCore(t)
	setupInboundServiceTestDB(t)
	previousManual := isManuallyStopped.Load()
	setXrayProcess(nil)
	service := &XrayService{}
	t.Cleanup(func() {
		if service.IsXrayRunning() {
			if err := service.StopXray(); err != nil {
				t.Errorf("stop isolated core: %v", err)
			}
		}
		isManuallyStopped.Store(previousManual)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	template := func(port int, protocol string) string {
		return fmt.Sprintf(`{"log":{"loglevel":"error"},"inbounds":[{"listen":"127.0.0.1","port":%d,"protocol":"socks","settings":{"auth":"noauth","udp":false}}],"outbounds":[{"protocol":%q,"tag":"direct"}]}`, port, protocol)
	}
	setTemplate := func(t *testing.T, value string) {
		t.Helper()
		if err := service.settingService.setString("xrayTemplateConfig", value); err != nil {
			t.Fatal(err)
		}
	}
	setTemplate(t, template(port, "freedom"))
	if err := service.RestartXray(true); err != nil {
		t.Fatalf("start baseline core: %v", err)
	}
	baseline := currentXrayProcess()
	baselineJSON, err := os.ReadFile(xray.GetConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	assertBaseline := func(t *testing.T) {
		t.Helper()
		if currentXrayProcess() != baseline || !baseline.IsRunning() {
			t.Fatal("failed restart discarded the working process")
		}
		actual, err := os.ReadFile(xray.GetConfigPath())
		if err != nil || !bytes.Equal(actual, baselineJSON) {
			t.Fatalf("failed restart replaced working configuration: %v", err)
		}
		connection, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprint(port)), time.Second)
		if err != nil {
			t.Fatalf("original listener is unavailable: %v", err)
		}
		if err := connection.Close(); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("schema_rejected_before_stop", func(t *testing.T) {
		setTemplate(t, template(port, "unsupported-fixture-protocol"))
		if err := service.RestartXray(true); err == nil || !strings.Contains(err.Error(), "preflight") {
			t.Fatalf("expected preflight rejection, got %v", err)
		}
		assertBaseline(t)
	})
	t.Run("occupied_port_restores_old_runtime", func(t *testing.T) {
		busy, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer busy.Close()
		setTemplate(t, template(busy.Addr().(*net.TCPAddr).Port, "freedom"))
		if err := service.RestartXray(true); err == nil {
			t.Fatal("port conflict unexpectedly succeeded")
		}
		assertBaseline(t)
	})
}

func TestRealXrayMigratedTransportFixtures(t *testing.T) {
	isolatedRealCore(t)
	certificate, err := (&ServerService{}).GetNewSelfSignedCert("fixture.test")
	if err != nil {
		t.Fatal(err)
	}
	tls := map[string]any{"certificates": []any{map[string]any{"certificate": strings.Split(strings.TrimSpace(certificate.Cert), "\n"), "key": strings.Split(strings.TrimSpace(certificate.Key), "\n")}}}
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	fixtures := map[string]map[string]any{
		"tcp":              {"network": "tcp"},
		"ws_tls":           {"network": "ws", "wsSettings": map[string]any{"path": "/fixture"}, "security": "tls", "tlsSettings": tls},
		"grpc_tls":         {"network": "grpc", "grpcSettings": map[string]any{"serviceName": "fixture"}, "security": "tls", "tlsSettings": tls},
		"legacy_xhttp_tls": {"method": "xhttp", "xhttpSettings": map[string]any{"path": "/fixture", "sessionPlacement": "header", "sessionKey": "X-Fixture", "sessionIDLength": "8-16"}, "security": "tls", "tlsSettings": tls},
		"legacy_reality":   {"network": "tcp", "security": "reality", "realitySettings": map[string]any{"dest": "127.0.0.1:443", "serverNames": []string{"fixture.test"}, "privateKey": base64.RawURLEncoding.EncodeToString(key.Bytes()), "shortIds": []string{"0123456789abcdef"}}, "sockopt": map[string]any{"V6Only": false}},
		"hysteria_udp_hop": {"network": "hysteria", "hysteriaSettings": map[string]any{"version": 2}, "security": "tls", "tlsSettings": tls, "finalmask": map[string]any{"quicParams": map[string]any{"udpHop": map[string]any{"ports": "20000-30000", "interval": 30}}, "udp": []any{map[string]any{"type": "salamander", "settings": map[string]any{"password": "fixture-obfuscation"}}}}},
	}
	for name, stream := range fixtures {
		t.Run(name, func(t *testing.T) {
			streamJSON, err := json.Marshal(stream)
			if err != nil {
				t.Fatal(err)
			}
			inbound := model.Inbound{Listen: "127.0.0.1", Port: 19443, Protocol: model.VLESS, Settings: `{"decryption":"none","clients":[{"id":"00000000-0000-4000-8000-000000000001"}]}`, StreamSettings: string(streamJSON), Tag: name}
			if name == "hysteria_udp_hop" {
				inbound.Protocol = model.Protocol("hysteria2")
				inbound.Settings = `{"version":2,"clients":[{"auth":"fixture-auth"}]}`
			}
			generated, err := inbound.BuildXrayInboundConfig()
			if err != nil {
				t.Fatal(err)
			}
			config := &xray.Config{InboundConfigs: []xray.InboundConfig{*generated}, OutboundConfigs: json_util.RawMessage(`[{"protocol":"freedom"}]`)}
			if err := xray.ValidateConfig(config); err != nil {
				t.Fatalf("real core rejected migrated fixture: %v", err)
			}
		})
	}
}
