package service

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
	"gorm.io/gorm"
)

func TestCandidateMigrationIsTransactionalAndUsesCandidateDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "candidate.db")
	gdb, err := database.PrepareDB(path)
	if err != nil {
		t.Fatalf("PrepareDB() failed: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("candidate DB handle failed: %v", err)
	}
	defer sqlDB.Close()

	inbound := &model.Inbound{
		UserId:         1,
		Remark:         "candidate-migration",
		Enable:         true,
		Port:           24001,
		Protocol:       model.VLESS,
		Settings:       `{"clients":[{"id":"candidate-client","email":"candidate@example.test"}]}`,
		StreamSettings: `{}`,
		Tag:            "0.0.0.0:candidate-migration",
		Up:             4,
		Down:           6,
		AllTime:        0,
	}
	if err := gdb.Create(inbound).Error; err != nil {
		t.Fatalf("create candidate inbound failed: %v", err)
	}
	clientTraffic := &xray.ClientTraffic{
		InboundId: inbound.Id,
		Email:     "candidate@example.test",
		Up:        2,
		Down:      5,
		AllTime:   0,
	}
	if err := gdb.Create(clientTraffic).Error; err != nil {
		t.Fatalf("create candidate client traffic failed: %v", err)
	}
	orphan := &xray.ClientTraffic{InboundId: 0, Email: "orphan@example.test"}
	if err := gdb.Create(orphan).Error; err != nil {
		t.Fatalf("create orphan traffic failed: %v", err)
	}

	service := NewInboundService(database.NewRepositories(gdb).Inbounds)
	if err := gdb.Transaction(func(tx *gorm.DB) error {
		if err := service.migrationRequirementsOn(tx); err != nil {
			return err
		}
		return migrationRemoveOrphanedClientTrafficsOn(tx)
	}); err != nil {
		t.Fatalf("candidate migration failed: %v", err)
	}

	var storedInbound model.Inbound
	if err := gdb.First(&storedInbound, inbound.Id).Error; err != nil {
		t.Fatalf("reload candidate inbound failed: %v", err)
	}
	if storedInbound.AllTime != 10 || storedInbound.Tag != inbound.Tag {
		t.Fatalf("candidate inbound after migration = all_time=%d tag=%q, want 10 and unchanged tag", storedInbound.AllTime, storedInbound.Tag)
	}
	var storedTraffic xray.ClientTraffic
	if err := gdb.First(&storedTraffic, clientTraffic.Id).Error; err != nil {
		t.Fatalf("reload candidate client traffic failed: %v", err)
	}
	if storedTraffic.AllTime != 7 {
		t.Fatalf("candidate client traffic all_time=%d, want 7", storedTraffic.AllTime)
	}
	var orphanCount int64
	if err := gdb.Model(&xray.ClientTraffic{}).Where("email = ?", orphan.Email).Count(&orphanCount).Error; err != nil {
		t.Fatalf("count orphan traffic failed: %v", err)
	}
	if orphanCount != 0 {
		t.Fatalf("orphan traffic rows = %d, want 0", orphanCount)
	}
}

func TestPanelMigrationMergesMultiDomainWithoutLosingExtensions(t *testing.T) {
	gdb, err := database.PrepareDB(filepath.Join(t.TempDir(), "domains.db"))
	if err != nil {
		t.Fatal(err)
	}
	pool, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	inbound := model.Inbound{
		UserId: 1, Protocol: model.VLESS, Tag: "domain-migration", Port: 443, Settings: `{"clients":[]}`,
		StreamSettings: `{"security":"tls","future":9007199254740993,"tlsSettings":{"settings":{"fingerprint":"chrome","domains":[{"domain":"legacy.example.test","future":9007199254740993},{"domain":"shared.example.test"}]}},"externalProxy":[{"dest":"custom.example.test","port":8443,"forceTls":"tls","vendor":9007199254740993},{"dest":"shared.example.test","port":443,"forceTls":"same"}]}`,
	}
	if err := gdb.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Transaction(migratePanelDatabase); err != nil {
		t.Fatal(err)
	}
	var first model.Inbound
	if err := gdb.First(&first, inbound.Id).Error; err != nil {
		t.Fatal(err)
	}
	var stream map[string]any
	decoder := json.NewDecoder(strings.NewReader(first.StreamSettings))
	decoder.UseNumber()
	if err := decoder.Decode(&stream); err != nil {
		t.Fatal(err)
	}
	proxies := stream["externalProxy"].([]any)
	if len(proxies) != 3 {
		t.Fatalf("merged proxies=%d, want 3 without duplicating shared endpoint", len(proxies))
	}
	custom := proxies[0].(map[string]any)
	legacy := proxies[2].(map[string]any)
	if custom["dest"] != "custom.example.test" || custom["forceTls"] != "tls" || custom["port"] != json.Number("8443") {
		t.Fatalf("existing proxy changed: %#v", custom)
	}
	for _, number := range []any{stream["future"], custom["vendor"], legacy["future"]} {
		if number != json.Number("9007199254740993") {
			t.Fatalf("unknown numeric extension lost precision: %v", number)
		}
	}
	settings := stream["tlsSettings"].(map[string]any)["settings"].(map[string]any)
	if _, exists := settings["domains"]; exists || settings["fingerprint"] != "chrome" {
		t.Fatal("migration did not remove only legacy domains")
	}
	if err := gdb.Transaction(migratePanelDatabase); err != nil {
		t.Fatal(err)
	}
	var second model.Inbound
	if err := gdb.First(&second, inbound.Id).Error; err != nil {
		t.Fatal(err)
	}
	if first.StreamSettings != second.StreamSettings {
		t.Fatal("repeated migration changed merged proxies")
	}
}

func TestPanelMigrationRejectsInvalidExistingExternalProxyAtomically(t *testing.T) {
	gdb, err := database.PrepareDB(filepath.Join(t.TempDir(), "invalid-domains.db"))
	if err != nil {
		t.Fatal(err)
	}
	pool, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	inbound := model.Inbound{
		UserId: 1, Protocol: model.VLESS, Tag: "domain-conflict", Port: 443, Up: 10, Settings: `{"clients":[]}`,
		StreamSettings: `{"security":"tls","tlsSettings":{"settings":{"domains":[{"domain":"legacy.example.test"}]}},"externalProxy":{"unknown":"preserve"}}`,
	}
	if err := gdb.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Transaction(migratePanelDatabase); err == nil || !strings.Contains(err.Error(), "externalProxy must be an array") {
		t.Fatalf("expected actionable migration error, got %v", err)
	}
	var stored model.Inbound
	if err := gdb.First(&stored, inbound.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.StreamSettings != inbound.StreamSettings || stored.Settings != inbound.Settings || stored.AllTime != 0 {
		t.Fatal("invalid external proxy partially changed historical data")
	}
}

func TestPanelDatabaseMigrationRollsBackAllStagesOnHistoricalFailure(t *testing.T) {
	setupInboundServiceTestDB(t)
	active := database.GetDB()
	candidate := filepath.Join(t.TempDir(), "candidate.db")
	gdb, err := database.PrepareDB(candidate)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	inbound := model.Inbound{
		UserId: 1, Protocol: model.VLESS, Tag: "0.0.0.0:preserved", Port: 24002,
		Settings:       `{"clients":[{"id":"fixture-id","email":"fixture@example.test"}]}`,
		StreamSettings: `{"network":"xhttp","xhttpSettings":{"sessionPlacement":"header","sessionKey":"X-Fixture"}}`,
		Up:             10,
	}
	if err := gdb.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	orphan := xray.ClientTraffic{InboundId: 0, Email: "orphan@example.test"}
	if err := gdb.Create(&orphan).Error; err != nil {
		t.Fatal(err)
	}
	// Compatibility updates the stream first; fail the subsequent historical
	// usage backfill to prove both stages share the same transaction.
	if err := gdb.Exec(`CREATE TRIGGER reject_history BEFORE UPDATE OF all_time ON inbounds BEGIN SELECT RAISE(ABORT, 'history failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if err := InitDB(candidate); err == nil || !strings.Contains(err.Error(), "history failure") {
		t.Fatalf("expected historical migration failure, got %v", err)
	}
	if database.GetDB() != active {
		t.Fatal("failed initialization replaced the active connection")
	}
	var users int64
	if err := active.Model(&model.User{}).Count(&users).Error; err != nil || users == 0 {
		t.Fatalf("original connection is no longer usable: users=%d err=%v", users, err)
	}
	var stored model.Inbound
	if err := gdb.First(&stored, inbound.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Settings != inbound.Settings || stored.StreamSettings != inbound.StreamSettings || stored.AllTime != 0 || stored.Tag != inbound.Tag {
		t.Fatal("failed migration partially changed inbound data")
	}
	var trafficCount int64
	if err := gdb.Model(&xray.ClientTraffic{}).Count(&trafficCount).Error; err != nil || trafficCount != 1 {
		t.Fatalf("failed migration changed traffic rows: count=%d err=%v", trafficCount, err)
	}
}

func TestPanelDatabaseMigrationIsIdempotentAcrossStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "startup.db")
	gdb, err := database.PrepareDB(path)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	inbound := model.Inbound{
		UserId: 1, Protocol: model.VLESS, Tag: "0.0.0.0:preserved", Port: 24003,
		Settings:       `{"clients":[{"id":"fixture-id","email":"fixture@example.test","enable":true}],"future":9007199254740993}`,
		StreamSettings: `{"network":"xhttp","xhttpSettings":{"sessionPlacement":"header","sessionKey":"X-Fixture"}}`,
		Up:             10,
	}
	if err := gdb.Create(&inbound).Error; err != nil {
		_ = pool.Close()
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.CloseDB() })
	if err := InitDB(path); err != nil {
		t.Fatal(err)
	}
	var first model.Inbound
	if err := database.GetDB().First(&first, inbound.Id).Error; err != nil {
		t.Fatal(err)
	}
	if first.AllTime != 10 || first.Tag != inbound.Tag || !strings.Contains(first.StreamSettings, "sessionIDPlacement") || !strings.Contains(first.Settings, "9007199254740993") {
		t.Fatal("startup did not preserve and migrate the historical record")
	}
	if err := InitDB(path); err != nil {
		t.Fatal(err)
	}
	var second model.Inbound
	if err := database.GetDB().First(&second, inbound.Id).Error; err != nil {
		t.Fatal(err)
	}
	if first.Settings != second.Settings || first.StreamSettings != second.StreamSettings || first.AllTime != second.AllTime || first.Tag != second.Tag {
		t.Fatal("repeated startup changed migrated data")
	}
	var count int64
	if err := database.GetDB().Model(&xray.ClientTraffic{}).Where("email = ?", "fixture@example.test").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("repeated startup duplicated client traffic: count=%d err=%v", count, err)
	}
}
