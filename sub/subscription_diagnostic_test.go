package sub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestDiagnoseSubscriptionInboundsReportsOutputAndSkippedNodes(t *testing.T) {
	client := matrixClient("diagnose-vless@example")
	unsupportedClient := matrixClient("diagnose-http@example")
	unsupportedClient.SubID = client.SubID
	inbounds := []*model.Inbound{
		matrixInbound(model.VLESS, 13001, client),
		matrixInbound(model.HTTP, 13002, unsupportedClient),
	}

	diagnostic := diagnoseSubscriptionInbounds(inbounds, client.SubID, subscriptionFormatURI)

	if diagnostic.TotalInbounds != 2 {
		t.Fatalf("total inbounds = %d, want 2", diagnostic.TotalInbounds)
	}
	if diagnostic.OutputNodes != 1 {
		t.Fatalf("output nodes = %d, want 1", diagnostic.OutputNodes)
	}
	if diagnostic.SkippedNodes != 1 {
		t.Fatalf("skipped nodes = %d, want 1", diagnostic.SkippedNodes)
	}
	if len(diagnostic.SkipReasons) != 1 || diagnostic.SkipReasons[0].Protocol != string(model.HTTP) {
		t.Fatalf("skip reasons = %#v, want one http skip", diagnostic.SkipReasons)
	}
	if len(diagnostic.Warnings) == 0 {
		t.Fatalf("warnings should explain skipped nodes")
	}
}

func TestDiagnoseSubscriptionInboundsReportsEmptySubscription(t *testing.T) {
	diagnostic := diagnoseSubscriptionInbounds(nil, "missing-sub", subscriptionFormatURI)

	if diagnostic.TotalInbounds != 0 || diagnostic.OutputNodes != 0 || diagnostic.SkippedNodes != 0 {
		t.Fatalf("empty diagnostic counts = %#v, want all zero", diagnostic)
	}
	if len(diagnostic.Warnings) != 1 {
		t.Fatalf("empty diagnostic warnings = %#v, want one readable warning", diagnostic.Warnings)
	}
}

func TestSubscriptionDiagnoseEndpointReturnsJSON(t *testing.T) {
	setupSubscriptionDiagnosticDB(t)
	client := matrixClient("diagnose-endpoint@example")
	inbound := matrixInbound(model.VLESS, 13003, client)
	inbound.Enable = true
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewSUBController(router.Group("/"), "/sub/", "/json/", "/clash/", true, true, false, true, "-ieo", "12", "", "", "", "", "", "", "", "", true, "")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/sub/matrix-sub/diagnose?target=xray", nil)
	request.Host = "vpn.example"
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var diagnostic SubscriptionDiagnostic
	if err := json.Unmarshal(recorder.Body.Bytes(), &diagnostic); err != nil {
		t.Fatalf("diagnostic JSON invalid: %v", err)
	}
	if diagnostic.Format != string(subscriptionFormatJSON) || diagnostic.OutputNodes != 1 {
		t.Fatalf("diagnostic = %#v, want json format with one output", diagnostic)
	}
}

func setupSubscriptionDiagnosticDB(t *testing.T) {
	t.Helper()
	dbDir := t.TempDir()
	t.Setenv("XUI_DB_FOLDER", dbDir)
	if err := database.InitDB(filepath.Join(dbDir, "SuperXray.db")); err != nil {
		t.Fatalf("database.InitDB failed: %v", err)
	}
	t.Cleanup(func() {
		if err := database.CloseDB(); err != nil {
			t.Logf("database.CloseDB warning: %v", err)
		}
	})
}
