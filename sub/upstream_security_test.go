package sub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestWireguardConfigRejectsInjectedLines(t *testing.T) {
	for _, field := range []string{"remark", "email", "address", "privateKey", "publicKey", "preSharedKey"} {
		for _, control := range []string{"\n", "\r", "\x00", "\x7f"} {
			t.Run(field+"/"+strings.ReplaceAll(control, "\n", "LF"), func(t *testing.T) {
				inbound := wireguardInboundForTest(`[]`)
				peer := wireguardPeer{Email: "peer", Enable: true, PrivateKey: wireguardTestPeerPrivate,
					PublicKey: wireguardTestPeerPublic, PreSharedKey: wireguardTestPSK, AllowedIPs: []string{"10.0.0.2/32"}}
				service := &SubService{address: "vpn.example", remarkModel: "-ieo"}
				switch field {
				case "remark":
					inbound.Remark += control + "[Interface]"
				case "email":
					peer.Email += control + "PostUp = injected"
				case "address":
					service.address += control + "PostUp = injected"
				case "privateKey":
					peer.PrivateKey = peer.PrivateKey[:8] + control + peer.PrivateKey[8:]
				case "preSharedKey":
					peer.PreSharedKey = peer.PreSharedKey[:8] + control + peer.PreSharedKey[8:]
				case "publicKey":
					var settings map[string]any
					if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
						t.Fatal(err)
					}
					settings["pubKey"] = wireguardTestServerPublic[:8] + control + wireguardTestServerPublic[8:]
					data, err := json.Marshal(settings)
					if err != nil {
						t.Fatal(err)
					}
					inbound.Settings = string(data)
				}
				if got := service.genWireguardConfig(inbound, peer); got != "" {
					t.Fatal("unsafe WireGuard config was rendered")
				}
			})
		}
	}
}

func TestSubscriptionRoutesPreventCaching(t *testing.T) {
	setupSubscriptionDiagnosticDB(t)
	inbound := matrixInbound(model.VLESS, 13443, matrixClient("cache-test@example"))
	inbound.Enable = true
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	NewSUBController(router.Group("/"), "/sub/", "/json/", "/clash/", true, true, false, true, "-ieo", "12", "", "", "", "", "", "", "", "", true, "", false)
	for _, prefix := range []string{"/sub/", "/json/", "/clash/"} {
		for _, suffix := range []string{"matrix-sub", "matrix-sub/diagnose", "missing-sub", "missing-sub/diagnose"} {
			t.Run(prefix+suffix, func(t *testing.T) {
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, prefix+suffix, nil))
				if got := response.Header().Get("Cache-Control"); got != "no-store" {
					t.Fatalf("Cache-Control = %q, want no-store (status %d)", got, response.Code)
				}
			})
		}
	}
}
