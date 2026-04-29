package service

import (
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
