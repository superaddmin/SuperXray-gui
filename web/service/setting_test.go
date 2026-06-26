package service

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/web/entity"
	"gorm.io/gorm"
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

func TestSettingServiceUsesRepositoryBoundary(t *testing.T) {
	repo := newFakeSettingRepository()
	repo.settings["webPort"] = &model.Setting{Key: "webPort", Value: "2053"}
	settingSvc := NewSettingService(repo)

	if err := settingSvc.setString("panelProxy", "http://127.0.0.1:18080"); err != nil {
		t.Fatalf("setString through repository returned error: %v", err)
	}
	if repo.settings["panelProxy"].Value != "http://127.0.0.1:18080" {
		t.Fatalf("panelProxy persisted through repository = %q", repo.settings["panelProxy"].Value)
	}

	panelProxy, err := settingSvc.getString("panelProxy")
	if err != nil {
		t.Fatalf("getString through repository returned error: %v", err)
	}
	if panelProxy != "http://127.0.0.1:18080" {
		t.Fatalf("getString through repository = %q", panelProxy)
	}

	allSetting, err := settingSvc.GetAllSetting()
	if err != nil {
		t.Fatalf("GetAllSetting through repository returned error: %v", err)
	}
	if allSetting.WebPort != 2053 {
		t.Fatalf("GetAllSetting WebPort = %d, want 2053", allSetting.WebPort)
	}
	if len(repo.allExceptKeys) != 1 || repo.allExceptKeys[0] != "xrayTemplateConfig" {
		t.Fatalf("GetAllSetting excluded keys = %v, want [xrayTemplateConfig]", repo.allExceptKeys)
	}

	if err := settingSvc.ResetSettings(); err != nil {
		t.Fatalf("ResetSettings through repository returned error: %v", err)
	}
	if !repo.deleteAllCalled {
		t.Fatal("ResetSettings did not call repository DeleteAll")
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

type fakeSettingRepository struct {
	settings        map[string]*model.Setting
	allExceptKeys   []string
	deleteAllCalled bool
}

func newFakeSettingRepository() *fakeSettingRepository {
	return &fakeSettingRepository{
		settings: make(map[string]*model.Setting),
	}
}

func (r *fakeSettingRepository) Get(key string) (*model.Setting, error) {
	setting, ok := r.settings[key]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *setting
	return &copy, nil
}

func (r *fakeSettingRepository) Save(key string, value string) error {
	r.settings[key] = &model.Setting{
		Key:   key,
		Value: value,
	}
	return nil
}

func (r *fakeSettingRepository) AllExcept(keys ...string) ([]*model.Setting, error) {
	r.allExceptKeys = append([]string(nil), keys...)
	excluded := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		excluded[key] = struct{}{}
	}

	settings := make([]*model.Setting, 0, len(r.settings))
	for key, setting := range r.settings {
		if _, ok := excluded[key]; ok {
			continue
		}
		copy := *setting
		settings = append(settings, &copy)
	}
	return settings, nil
}

func (r *fakeSettingRepository) DeleteAll() error {
	r.deleteAllCalled = true
	r.settings = make(map[string]*model.Setting)
	return nil
}
