package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"
	"github.com/xlzd/gotp"
)

func TestSettingsUpdateAcceptsSeparateConfirmationCodeForJSONAndForm(t *testing.T) {
	for _, contentType := range []string{"application/json", "application/x-www-form-urlencoded"} {
		t.Run(contentType, func(t *testing.T) {
			if err := database.InitDB(filepath.Join(t.TempDir(), "settings.db")); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = database.CloseDB() })
			svc := &service.SettingService{}
			secret := "JBSWY3DPEHPK3PXP"
			if err := svc.SetTwoFactorToken(secret); err != nil {
				t.Fatal(err)
			}
			if err := svc.SetTwoFactorEnable(true); err != nil {
				t.Fatal(err)
			}
			controller := &SettingController{}
			router := gin.New()
			// The handler is exercised directly; auth/CSRF have separate middleware tests.
			router.POST("/update", controller.updateSetting)
			values := url.Values{"webPort": {"2053"}, "subPort": {"2096"}, "timeLocation": {"Local"}, "twoFactorEnable": {"false"}, "twoFactorToken": {""}, "twoFactorCode": {gotp.NewDefaultTOTP(secret).Now()}}
			body := values.Encode()
			if contentType == "application/json" {
				data, err := json.Marshal(map[string]any{"webPort": 2053, "subPort": 2096, "timeLocation": "Local", "twoFactorEnable": false, "twoFactorToken": "", "twoFactorCode": values.Get("twoFactorCode")})
				if err != nil {
					t.Fatal(err)
				}
				body = string(data)
			}
			request := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
			request.Header.Set("Content-Type", contentType)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			var result struct {
				Success bool `json:"success"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if !result.Success {
				t.Fatalf("confirmed settings update failed: %s", response.Body.String())
			}
			if enabled, err := svc.GetTwoFactorEnable(); err != nil || enabled {
				t.Fatal("confirmed disable did not persist")
			}
		})
	}
}
