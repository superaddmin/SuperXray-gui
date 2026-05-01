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
	if len(client.Password) != 16 {
		t.Fatalf("legacy Shadowsocks password length = %d, want 16", len(client.Password))
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
