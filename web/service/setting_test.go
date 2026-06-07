package service

import (
	"path/filepath"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
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
