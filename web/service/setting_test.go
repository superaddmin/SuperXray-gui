package service

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/web/entity"
)

func TestSettingServiceFallsBackToDefaultForEmptyRequiredSetting(t *testing.T) {
	setupSettingServiceTestDB(t)
	if err := database.GetDB().Create(&model.Setting{Key: "webPort", Value: ""}).Error; err != nil {
		t.Fatalf("create empty webPort setting failed: %v", err)
	}

	port, err := (&SettingService{}).GetPort()
	if err != nil {
		t.Fatalf("GetPort returned error for empty persisted webPort: %v", err)
	}
	if port != 2053 {
		t.Fatalf("GetPort = %d, want default 2053 for empty persisted webPort", port)
	}
}

func TestSettingServiceDoesNotFallbackToSecuritySensitiveDefaultsForEmptyValues(t *testing.T) {
	setupSettingServiceTestDB(t)
	settings := []*model.Setting{
		{Key: "subEnable", Value: ""},
		{Key: "subShowInfo", Value: ""},
		{Key: "subPath", Value: ""},
	}
	if err := database.GetDB().Create(&settings).Error; err != nil {
		t.Fatalf("create empty sensitive settings failed: %v", err)
	}

	if enabled, err := (&SettingService{}).GetSubEnable(); err == nil || enabled {
		t.Fatalf("GetSubEnable = %v, %v; want parse error and no default true fallback", enabled, err)
	}
	if showInfo, err := (&SettingService{}).GetSubShowInfo(); err == nil || showInfo {
		t.Fatalf("GetSubShowInfo = %v, %v; want parse error and no default true fallback", showInfo, err)
	}
	subPath, err := (&SettingService{}).GetSubPath()
	if err != nil {
		t.Fatalf("GetSubPath returned error for empty persisted subPath: %v", err)
	}
	if subPath != "" {
		t.Fatalf("GetSubPath = %q, want empty value instead of default /sub/ fallback", subPath)
	}
}

func TestSettingServiceGetAllSettingFallsBackForEmptyRequiredSetting(t *testing.T) {
	setupSettingServiceTestDB(t)
	if err := database.GetDB().Create(&model.Setting{Key: "webPort", Value: ""}).Error; err != nil {
		t.Fatalf("create empty webPort setting failed: %v", err)
	}

	allSetting, err := (&SettingService{}).GetAllSetting()
	if err != nil {
		t.Fatalf("GetAllSetting returned error for empty persisted webPort: %v", err)
	}
	if allSetting.WebPort != 2053 {
		t.Fatalf("GetAllSetting WebPort = %d, want default 2053", allSetting.WebPort)
	}
}

func TestSettingServicePanelProxyDefaultsPersistsAndBuildsClient(t *testing.T) {
	setupSettingServiceTestDB(t)
	settingSvc := &SettingService{}

	proxyURL, err := settingSvc.GetPanelProxy()
	if err != nil {
		t.Fatalf("GetPanelProxy default returned error: %v", err)
	}
	if proxyURL != "" {
		t.Fatalf("GetPanelProxy default = %q, want empty", proxyURL)
	}

	if err := settingSvc.SetPanelProxy("http://127.0.0.1:18080"); err != nil {
		t.Fatalf("SetPanelProxy returned error: %v", err)
	}
	allSetting, err := settingSvc.GetAllSetting()
	if err != nil {
		t.Fatalf("GetAllSetting returned error: %v", err)
	}
	if allSetting.PanelProxy != "http://127.0.0.1:18080" {
		t.Fatalf("AllSetting.PanelProxy = %q", allSetting.PanelProxy)
	}

	client := settingSvc.NewProxiedHTTPClient(9 * time.Second)
	if client.Timeout != 9*time.Second {
		t.Fatalf("NewProxiedHTTPClient timeout = %v, want 9s", client.Timeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("NewProxiedHTTPClient transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("NewProxiedHTTPClient did not configure HTTP proxy transport")
	}
}

func TestSettingServiceUpdateAllSettingRejectsInvalidPanelProxy(t *testing.T) {
	setupSettingServiceTestDB(t)
	settings := validAllSettingForServiceTest()
	settings.PanelProxy = "socks5://"

	if err := (&SettingService{}).UpdateAllSetting(settings); err == nil {
		t.Fatal("UpdateAllSetting accepted invalid panelProxy")
	}
}

func setupSettingServiceTestDB(t *testing.T) {
	t.Helper()
	dbDir := t.TempDir()
	if err := database.InitDB(filepath.Join(dbDir, "SuperXray.db")); err != nil {
		t.Fatalf("database.InitDB failed: %v", err)
	}
	t.Cleanup(func() {
		if err := database.CloseDB(); err != nil {
			t.Logf("database.CloseDB warning: %v", err)
		}
	})
}

func validAllSettingForServiceTest() *entity.AllSetting {
	return &entity.AllSetting{
		WebPort:      2053,
		SubPort:      2096,
		WebBasePath:  "/super/",
		SubPath:      "/sub/",
		SubJsonPath:  "/json/",
		SubClashPath: "/clash/",
		TimeLocation: "Local",
	}
}
