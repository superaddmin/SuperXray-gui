package service

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"strconv"
	"sync"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestTgbotUserStateStoreConcurrentAccess(t *testing.T) {
	const workers = 32
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			chatID := int64(i)
			state := "state-" + strconv.Itoa(i)
			setUserState(chatID, state)
			if got, ok := getUserState(chatID); !ok || got != state {
				t.Errorf("getUserState(%d) = %q, %v", chatID, got, ok)
			}
			deleteUserState(chatID)
			if _, ok := getUserState(chatID); ok {
				t.Errorf("state for chat %d was not deleted", chatID)
			}
		}(i)
	}

	wg.Wait()
}

func TestClientAddStateIsIsolatedByChat(t *testing.T) {
	clientStateMutex.Lock()
	defer clientStateMutex.Unlock()

	resetClientAddGlobals()
	receiver_inbound_ID = 1
	client_Email = "first@example.test"
	saveClientAddState(101)

	resetClientAddGlobals()
	receiver_inbound_ID = 2
	client_Email = "second@example.test"
	saveClientAddState(202)

	loadClientAddState(101)
	if receiver_inbound_ID != 1 || client_Email != "first@example.test" {
		t.Fatalf("chat 101 state = inbound %d email %q", receiver_inbound_ID, client_Email)
	}

	loadClientAddState(202)
	if receiver_inbound_ID != 2 || client_Email != "second@example.test" {
		t.Fatalf("chat 202 state = inbound %d email %q", receiver_inbound_ID, client_Email)
	}

	resetClientAddGlobals()
	saveClientAddState(101)
	saveClientAddState(202)
}

func TestTgbotNumericHelpersClampOverflow(t *testing.T) {
	if got := saturatingAddUint64(math.MaxUint64, 1); got != math.MaxUint64 {
		t.Fatalf("saturatingAddUint64 overflow = %d, want %d", got, uint64(math.MaxUint64))
	}
	if got := telegramRequestID(int(math.MaxInt32) + 1); got != math.MaxInt32 {
		t.Fatalf("telegramRequestID overflow = %d, want %d", got, int32(math.MaxInt32))
	}
}

func TestTgbotPrepareShadowsocksClientDefaultsUsesLegacyInboundMethod(t *testing.T) {
	clientStateMutex.Lock()
	defer clientStateMutex.Unlock()

	resetClientAddGlobals()
	inbound := &model.Inbound{
		Protocol: model.Shadowsocks,
		Settings: `{"method":"chacha20-ietf-poly1305","password":"server-pass","clients":[]}`,
	}

	tg := &Tgbot{}
	tg.prepareShadowsocksClientDefaults(inbound)

	if client_Method != "chacha20-ietf-poly1305" {
		t.Fatalf("client method = %q, want inbound method", client_Method)
	}
	if len(client_ShPassword) != 16 {
		t.Fatalf("legacy client password length = %d, want 16", len(client_ShPassword))
	}

	client_Email = "ss-legacy@example.test"
	jsonText, err := tg.BuildJSONForProtocol(model.Shadowsocks)
	if err != nil {
		t.Fatalf("BuildJSONForProtocol returned error: %v", err)
	}
	client := decodeTgbotClientJSON(t, jsonText)
	if client["method"] != "chacha20-ietf-poly1305" {
		t.Fatalf("json method = %v, want inbound method", client["method"])
	}
	if client["password"] != client_ShPassword {
		t.Fatalf("json password did not use generated client password")
	}
}

func TestTgbotPrepareShadowsocksClientDefaultsUses2022Key(t *testing.T) {
	clientStateMutex.Lock()
	defer clientStateMutex.Unlock()

	resetClientAddGlobals()
	inbound := &model.Inbound{
		Protocol: model.Shadowsocks,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"server-key","clients":[]}`,
	}

	tg := &Tgbot{}
	tg.prepareShadowsocksClientDefaults(inbound)

	if client_Method != "" {
		t.Fatalf("2022 client method = %q, want empty", client_Method)
	}
	decoded, err := base64.StdEncoding.DecodeString(client_ShPassword)
	if err != nil {
		t.Fatalf("2022 client password is not base64: %v", err)
	}
	if len(decoded) != 16 {
		t.Fatalf("2022 aes-128 client key length = %d, want 16", len(decoded))
	}

	client_Email = "ss-2022@example.test"
	jsonText, err := tg.BuildJSONForProtocol(model.Shadowsocks)
	if err != nil {
		t.Fatalf("BuildJSONForProtocol returned error: %v", err)
	}
	client := decodeTgbotClientJSON(t, jsonText)
	if client["method"] != "" {
		t.Fatalf("json method = %v, want empty for 2022", client["method"])
	}
	if client["password"] != client_ShPassword {
		t.Fatalf("json password did not use generated 2022 client key")
	}
}

func decodeTgbotClientJSON(t *testing.T, jsonText string) map[string]any {
	t.Helper()

	var payload struct {
		Clients []map[string]any `json:"clients"`
	}
	if err := json.Unmarshal([]byte(jsonText), &payload); err != nil {
		t.Fatalf("client JSON is invalid: %v", err)
	}
	if len(payload.Clients) != 1 {
		t.Fatalf("client JSON has %d clients, want 1", len(payload.Clients))
	}
	return payload.Clients[0]
}
