package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestInboundServiceGetClientsIgnoresNonClientSettingsFields(t *testing.T) {
	inbound := &model.Inbound{
		Settings: `{
			"clients": [
				{
					"id": "00000000-0000-4000-8000-000000000000",
					"email": "test@example",
					"enable": true,
					"tgId": 0
				}
			],
			"decryption": "none",
			"encryption": "none"
		}`,
	}

	clients, err := (&InboundService{}).GetClients(inbound)
	if err != nil {
		t.Fatalf("GetClients returned error: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("GetClients returned %d clients, want 1", len(clients))
	}
	if clients[0].Email != "test@example" {
		t.Fatalf("client email = %q, want %q", clients[0].Email, "test@example")
	}
}

func TestInboundSettingsEntryRejectsMissingClientsArray(t *testing.T) {
	_, _, err := parseInboundSettingsEntry(`{"method":"2022-blake3-aes-128-gcm"}`)
	if err == nil {
		t.Fatal("parseInboundSettingsEntry returned nil error")
	}
	if !strings.Contains(err.Error(), "clients") {
		t.Fatalf("error = %q, want mention clients", err.Error())
	}
}

func TestInboundSettingsEntryRejectsNonArrayClients(t *testing.T) {
	_, _, err := parseInboundSettingsEntry(`{"clients":"not-an-array"}`)
	if err == nil {
		t.Fatal("parseInboundSettingsEntry returned nil error")
	}
	if !strings.Contains(err.Error(), "clients") {
		t.Fatalf("error = %q, want mention clients", err.Error())
	}
}

func TestInboundSettingsEntryReturnsRawClients(t *testing.T) {
	settings, rawClients, err := parseInboundSettingsEntry(`{
		"clients": [
			{"id":"00000000-0000-4000-8000-000000000000","email":"a@example","enable":true}
		],
		"decryption": "none"
	}`)
	if err != nil {
		t.Fatalf("parseInboundSettingsEntry returned error: %v", err)
	}
	if len(rawClients) != 1 {
		t.Fatalf("rawClients length = %d, want 1", len(rawClients))
	}
	if settings["decryption"] != "none" {
		t.Fatalf("settings decryption = %v, want none", settings["decryption"])
	}
}

func TestBuildTargetClientFromSourceShadowsocksLegacyUsesInboundMethod(t *testing.T) {
	targetInbound := &model.Inbound{
		Protocol: model.Shadowsocks,
		Settings: `{"method":"chacha20-ietf-poly1305","password":"server-pass","clients":[]}`,
	}

	client, err := (&InboundService{}).buildTargetClientFromSource(
		model.Client{ID: "source-id", Password: "source-pass", Auth: "source-auth", Email: "source@example"},
		targetInbound,
		"copied@example",
		"",
	)
	if err != nil {
		t.Fatalf("buildTargetClientFromSource returned error: %v", err)
	}

	if client.Method != "chacha20-ietf-poly1305" {
		t.Fatalf("client method = %q, want inbound method", client.Method)
	}
	decodedPassword, err := base64.RawURLEncoding.DecodeString(client.Password)
	if err != nil {
		t.Fatalf("legacy Shadowsocks password is not URL-safe base64: %v", err)
	}
	if len(decodedPassword) != generatedCredentialBytes {
		t.Fatalf("legacy Shadowsocks password decoded length = %d, want %d", len(decodedPassword), generatedCredentialBytes)
	}
	if client.ID != "" || client.Auth != "" {
		t.Fatalf("client kept source credentials: id=%q auth=%q", client.ID, client.Auth)
	}
}

func TestBuildTargetClientFromSourceShadowsocks2022UsesClientKey(t *testing.T) {
	targetInbound := &model.Inbound{
		Protocol: model.Shadowsocks,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"server-key","clients":[]}`,
	}

	client, err := (&InboundService{}).buildTargetClientFromSource(
		model.Client{ID: "source-id", Password: "source-pass", Auth: "source-auth", Email: "source@example"},
		targetInbound,
		"copied@example",
		"",
	)
	if err != nil {
		t.Fatalf("buildTargetClientFromSource returned error: %v", err)
	}

	if client.Method != "" {
		t.Fatalf("2022 client method = %q, want empty", client.Method)
	}
	decoded, err := base64.StdEncoding.DecodeString(client.Password)
	if err != nil {
		t.Fatalf("2022 Shadowsocks password is not base64: %v", err)
	}
	if len(decoded) != 16 {
		t.Fatalf("2022 aes-128 client key length = %d, want 16", len(decoded))
	}
	if client.ID != "" || client.Auth != "" {
		t.Fatalf("client kept source credentials: id=%q auth=%q", client.ID, client.Auth)
	}
}

func TestBuildTargetClientFromSourceLiteralHysteria2UsesAuthCredential(t *testing.T) {
	targetInbound := &model.Inbound{
		Protocol: model.Hysteria2,
		Settings: `{"version":2,"clients":[]}`,
	}

	client, err := (&InboundService{}).buildTargetClientFromSource(
		model.Client{ID: "source-id", Password: "source-pass", Auth: "source-auth", Email: "source@example"},
		targetInbound,
		"copied@example",
		"",
	)
	if err != nil {
		t.Fatalf("buildTargetClientFromSource returned error: %v", err)
	}

	if client.Auth == "" {
		t.Fatalf("literal hysteria2 copied client auth is empty: %#v", client)
	}
	if client.ID != "" || client.Password != "" {
		t.Fatalf("literal hysteria2 copied client kept non-HY2 credentials: id=%q password=%q", client.ID, client.Password)
	}
}

func TestGetClientPrimaryKeyLiteralHysteria2UsesAuth(t *testing.T) {
	client := model.Client{ID: "id-value", Auth: "auth-value", Password: "password-value"}

	got := (&InboundService{}).getClientPrimaryKey(model.Hysteria2, client)

	if got != client.Auth {
		t.Fatalf("literal hysteria2 primary key = %q, want auth %q", got, client.Auth)
	}
}

func TestGetClientPrimaryKeyByEmailLiteralHysteria2UsesAuth(t *testing.T) {
	clients := []model.Client{
		{Email: "other@example", ID: "other-id", Auth: "other-auth", Password: "other-password"},
		{Email: "hy2@example", ID: "", Auth: "hy2-auth", Password: "hy2-password"},
	}

	got := (&InboundService{}).getClientPrimaryKeyByEmail(model.Hysteria2, clients, "hy2@example")

	if got != "hy2-auth" {
		t.Fatalf("literal hysteria2 primary key by email = %q, want auth", got)
	}
}

func TestXrayRuntimeProtocolNormalizesLiteralHysteria2(t *testing.T) {
	if got := xrayRuntimeProtocol(model.Hysteria2); got != string(model.Hysteria) {
		t.Fatalf("xrayRuntimeProtocol(hysteria2) = %q, want %q", got, model.Hysteria)
	}
	if got := xrayRuntimeProtocol(model.VLESS); got != string(model.VLESS) {
		t.Fatalf("xrayRuntimeProtocol(vless) = %q, want %q", got, model.VLESS)
	}
}

func TestNormalizeShadowsocksSettingsFillsLegacyClientMethod(t *testing.T) {
	settings, err := normalizeShadowsocksSettingsText(`{
		"method":"chacha20-ietf-poly1305",
		"password":"stale-server-password",
		"clients":[{"email":"ss@example","password":"client-password","enable":true}]
	}`)
	if err != nil {
		t.Fatalf("normalizeShadowsocksSettingsText returned error: %v", err)
	}
	var parsed struct {
		Password string         `json:"password"`
		Clients  []model.Client `json:"clients"`
	}
	if err := json.Unmarshal([]byte(settings), &parsed); err != nil {
		t.Fatalf("normalized settings are invalid JSON: %v", err)
	}
	if parsed.Password != "" {
		t.Fatalf("normalized legacy settings kept server password: %q", parsed.Password)
	}
	if len(parsed.Clients) != 1 || parsed.Clients[0].Method != "chacha20-ietf-poly1305" {
		t.Fatalf("normalized legacy client method = %#v, want chacha20-ietf-poly1305", parsed.Clients)
	}
}

func TestNormalizeShadowsocksSettingsCanonicalizesLegacyCipherAliases(t *testing.T) {
	settings, err := normalizeShadowsocksSettingsText(`{
		"method":"CHACHA20_POLY1305",
		"password":"stale-server-password",
		"clients":[{"method":"CHACHA20_POLY1305","email":"ss@example","password":"client-password","enable":true}]
	}`)
	if err != nil {
		t.Fatalf("normalizeShadowsocksSettingsText returned error: %v", err)
	}
	var parsed struct {
		Method  string         `json:"method"`
		Clients []model.Client `json:"clients"`
	}
	if err := json.Unmarshal([]byte(settings), &parsed); err != nil {
		t.Fatalf("normalized settings are invalid JSON: %v", err)
	}
	if parsed.Method != "chacha20-poly1305" {
		t.Fatalf("normalized method = %q, want chacha20-poly1305", parsed.Method)
	}
	if len(parsed.Clients) != 1 || parsed.Clients[0].Method != "chacha20-poly1305" {
		t.Fatalf("normalized legacy client method = %#v, want chacha20-poly1305", parsed.Clients)
	}
}

func TestXrayAPISyncLockSerializesCalls(t *testing.T) {
	var wg sync.WaitGroup
	var running int32
	var maxRunning int32

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			withXrayAPISyncLock(func() {
				current := atomic.AddInt32(&running, 1)
				for {
					max := atomic.LoadInt32(&maxRunning)
					if current <= max || atomic.CompareAndSwapInt32(&maxRunning, max, current) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&running, -1)
			})
		}()
	}

	wg.Wait()
	if maxRunning != 1 {
		t.Fatalf("max concurrent Xray API calls = %d, want 1", maxRunning)
	}
}
