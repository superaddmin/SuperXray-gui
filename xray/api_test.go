package xray

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/proxy/shadowsocks"
	"github.com/xtls/xray-core/proxy/shadowsocks_2022"
	"google.golang.org/grpc"
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
	}

	for method, want := range cases {
		got, ok := shadowsocksCipherTypeFromString(method)
		if !ok || got != want {
			t.Fatalf("shadowsocksCipherTypeFromString(%q) = (%v, %v), want (%v, true)", method, got, ok, want)
		}
	}
	if _, ok := shadowsocksCipherTypeFromString("2022-blake3-aes-128-gcm"); ok {
		t.Fatal("SS-2022 cipher must be handled by its dedicated account")
	}
}

type recordingHandler struct {
	command.HandlerServiceClient
	t        *testing.T
	requests []*command.AlterInboundRequest
}

func (h *recordingHandler) AlterInbound(ctx context.Context, req *command.AlterInboundRequest, _ ...grpc.CallOption) (*command.AlterInboundResponse, error) {
	h.t.Helper()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 5*time.Second {
		h.t.Fatal("mutation must have a bounded deadline")
	}
	h.requests = append(h.requests, req)
	return &command.AlterInboundResponse{}, nil
}

func TestSS2022MutationUsesAccountAndRejectsUnknownCipher(t *testing.T) {
	h := &recordingHandler{t: t}
	var client command.HandlerServiceClient = h
	api := XrayAPI{HandlerServiceClient: &client}
	password := base64.StdEncoding.EncodeToString(make([]byte, 16))
	user := map[string]any{"email": "fixture@example.test", "cipher": "2022-blake3-aes-128-gcm", "password": password}
	if err := api.AddUser("shadowsocks", "fixture", user); err != nil {
		t.Fatal(err)
	}
	operation, err := h.requests[0].Operation.GetInstance()
	if err != nil {
		t.Fatal(err)
	}
	account, err := operation.(*command.AddUserOperation).User.Account.GetInstance()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := account.(*shadowsocks_2022.Account)
	if !ok || got.Key != password {
		t.Fatalf("SS2022 mutation account type = %T", account)
	}
	user["cipher"] = "2022-blake3-unsupported"
	if err := api.AddUser("shadowsocks", "fixture", user); err == nil {
		t.Fatal("unknown SS2022 cipher accepted")
	}
	if len(h.requests) != 1 {
		t.Fatal("unsupported cipher changed core users")
	}
}

func TestUninitializedHandlerReturnsError(t *testing.T) {
	api := &XrayAPI{}
	for _, err := range []error{api.AddInbound([]byte(`{}`)), api.DelInbound("fixture"), api.AddUser("vless", "fixture", nil), api.RemoveUser("fixture", "fixture")} {
		if err == nil {
			t.Fatal("uninitialized API was accepted")
		}
	}
}
