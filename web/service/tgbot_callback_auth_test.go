package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/web/global"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

func TestTgbotCallbackAuthorizationAndHashedSelfLinks(t *testing.T) {
	setupSettingServiceTestDB(t)
	requests := setupTgbotCallbackTestAPI(t)
	const senderID = int64(42)
	const chatID = int64(4242)
	longEmail := strings.Repeat("long", 16) + "@example.test"
	clients := []model.Client{
		{Email: "own@example.test", TgID: senderID, SubID: "own-sub"},
		{Email: longEmail, TgID: senderID, SubID: "long-sub"},
		{Email: "other@example.test", TgID: 99, SubID: "other-sub"},
		{Email: "unbound@example.test", SubID: "unbound-sub"},
	}
	settings, err := json.Marshal(map[string]any{"clients": clients})
	if err != nil {
		t.Fatal(err)
	}
	inbound := &model.Inbound{Id: 1, Protocol: model.VLESS, Settings: string(settings)}
	for _, client := range clients {
		inbound.ClientStats = append(inbound.ClientStats, xray.ClientTraffic{Email: client.Email, InboundId: inbound.Id})
	}
	tg := &Tgbot{inboundService: *NewInboundService(&fakeInboundRepository{inbounds: []*model.Inbound{inbound}})}

	for _, test := range []struct {
		name    string
		data    string
		hashed  bool
		admin   bool
		allowed bool
	}{
		{"backup denied", "get_backup", false, false, false},
		{"traffic reset denied", "reset_all_traffics_c", false, false, false},
		{"wizard denied", "add_client", false, false, false},
		{"logs denied", "get_banlogs", false, false, false},
		{"usage denied", "get_usage", false, false, false},
		{"online users denied", "onlines", false, false, false},
		{"inbounds denied", "inbounds", false, false, false},
		{"hashed backup denied", "get_backup", true, false, false},
		{"unknown denied", "future_admin_action", false, false, false},
		{"foreign subscription denied", "client_sub_links other@example.test", false, false, false},
		{"foreign individual links denied", "client_individual_links other@example.test", false, false, false},
		{"foreign QR denied", "client_qr_links other@example.test", false, false, false},
		{"hashed foreign links denied", "client_sub_links other@example.test", true, false, false},
		{"unbound links denied", "client_sub_links unbound@example.test", false, false, false},
		{"missing client denied", "client_sub_links missing@example.test", false, false, false},
		{"empty email denied", "client_sub_links ", false, false, false},
		{"unknown hash denied", strings.Repeat("0", 64), false, false, false},
		{"self help allowed", "client_commands", false, false, true},
		{"self links allowed", "client_sub_links own@example.test", false, false, true},
		{"long self links allowed", "client_sub_links " + longEmail, true, false, true},
		{"admin links allowed", "client_sub_links other@example.test", false, true, true},
		{"admin wizard allowed", "add_client_ch_default_email", false, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := test.data
			if test.hashed {
				data = hashStorage.SaveHash(data)
			}
			query := &telego.CallbackQuery{
				ID: "callback", Data: data, From: telego.User{ID: senderID},
				Message: &telego.Message{MessageID: 1, Chat: telego.Chat{ID: chatID}},
			}
			setUserState(chatID, "awaiting_comment")
			defer deleteUserState(chatID)
			before := requests.Load()
			tg.answerCallback(query, test.admin)
			if got := requests.Load() > before; got != test.allowed {
				t.Fatalf("outbound Telegram request = %v, want %v", got, test.allowed)
			}
			if query.Data != data {
				t.Fatal("callback dispatch mutated the original update")
			}
			if !test.admin {
				if state, ok := getUserState(chatID); !ok || state != "awaiting_comment" {
					t.Fatalf("non-admin callback changed admin wizard state: %q, %v", state, ok)
				}
			}
		})
	}
	for _, data := range []string{"client_traffic", "client_commands", "client_sub_links", "client_individual_links", "client_qr_links"} {
		if !tg.isClientSelfCallbackAllowed(data, senderID) {
			t.Errorf("self-service menu %q denied", data)
		}
		if tg.isClientSelfCallbackAllowed(data, 0) {
			t.Errorf("self-service menu %q accepted without a sender", data)
		}
	}
}

func TestTgbotDelayedMessageDeletionPreservesNextWizardStep(t *testing.T) {
	requests := setupTgbotCallbackTestAPI(t)
	const chatID = int64(4242)
	setUserState(chatID, "awaiting_comment")
	t.Cleanup(func() { deleteUserState(chatID) })
	(&Tgbot{}).deleteMessageAfterDelay(chatID, 1, 0)
	if requests.Load() != 1 {
		t.Fatal("old message was not deleted")
	}
	if state, ok := getUserState(chatID); !ok || state != "awaiting_comment" {
		t.Fatalf("delayed deletion cleared the next wizard state: %q, %v", state, ok)
	}
}

func TestTgbotKeyboardAndCommandResetRequireAdmin(t *testing.T) {
	clientStateMutex.Lock()
	defer clientStateMutex.Unlock()
	const chatID = int64(4242)
	oldAdmins, oldEmail := adminIds, client_Email
	adminIds = []int64{42}
	t.Cleanup(func() {
		adminIds, client_Email = oldAdmins, oldEmail
		deleteUserState(chatID)
	})
	for _, from := range []*telego.User{nil, {ID: 99}, {ID: 42}} {
		setUserState(chatID, "awaiting_comment")
		client_Email = "pending@example.test"
		isAdmin := from != nil && from.ID == 42
		if got := resetClientAddStateForAdmin(chatID, from); got != isAdmin {
			t.Fatalf("reset authorization = %v, want %v", got, isAdmin)
		}
		state, exists := getUserState(chatID)
		if isAdmin {
			if exists || client_Email != "" {
				t.Fatal("admin could not cancel the current wizard")
			}
		} else if !exists || state != "awaiting_comment" || client_Email != "pending@example.test" {
			t.Fatal("non-admin keyboard or command reset changed the wizard")
		}
	}
}

func setupTgbotCallbackTestAPI(t *testing.T) *atomic.Int32 {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		result := any(true)
		if strings.HasSuffix(r.URL.Path, "/sendMessage") {
			result = map[string]any{"message_id": 1, "chat": map[string]any{"id": 4242, "type": "private"}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result})
	}))
	t.Cleanup(server.Close)
	testBot, err := telego.NewBot("1:"+strings.Repeat("A", 35), telego.WithAPIServer(server.URL), telego.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	oldBot, oldRunning, oldStorage := bot, isRunning, hashStorage
	bot, isRunning, hashStorage = testBot, true, global.NewHashStorage(time.Minute)
	t.Cleanup(func() { bot, isRunning, hashStorage = oldBot, oldRunning, oldStorage })
	return &requests
}
