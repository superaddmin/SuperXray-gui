package database

import (
	"path/filepath"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestInitDBCreatesMigrationMetadata(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "x-ui.db")

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := CloseDB(); err != nil {
			t.Logf("CloseDB warning: %v", err)
		}
	})

	gdb := GetDB()
	if gdb == nil {
		t.Fatal("GetDB() returned nil")
	}

	if !gdb.Migrator().HasTable("schema_migrations") {
		t.Fatal("schema_migrations table was not created")
	}
	if !gdb.Migrator().HasTable("migration_events") {
		t.Fatal("migration_events table was not created")
	}

	var migrationCount int64
	if err := gdb.Table("schema_migrations").
		Where("version = ? AND name = ? AND status = ?", "202606260001", "baseline-auto-migrate", "applied").
		Count(&migrationCount).Error; err != nil {
		t.Fatalf("count schema_migrations failed: %v", err)
	}
	if migrationCount != 1 {
		t.Fatalf("baseline schema migration count = %d, want 1", migrationCount)
	}

	var eventCount int64
	if err := gdb.Table("migration_events").
		Where("version = ? AND direction = ? AND status = ?", "202606260001", "up", "applied").
		Count(&eventCount).Error; err != nil {
		t.Fatalf("count migration_events failed: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("baseline migration event count = %d, want 1", eventCount)
	}
}

func TestInitDBSeederIsIdempotentAndDoesNotRehash(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "x-ui.db")
	if err := InitDB(dbPath); err != nil {
		t.Fatalf("first InitDB() failed: %v", err)
	}
	var first model.User
	if err := GetDB().Where("username = ?", "admin").First(&first).Error; err != nil {
		t.Fatalf("load default user: %v", err)
	}
	firstHash := first.Password
	if err := CloseDB(); err != nil {
		t.Fatalf("close after first init: %v", err)
	}
	if err := InitDB(dbPath); err != nil {
		t.Fatalf("second InitDB() failed: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB() })
	var second model.User
	if err := GetDB().Where("username = ?", "admin").First(&second).Error; err != nil {
		t.Fatalf("load user after retry: %v", err)
	}
	if second.Password != firstHash {
		t.Fatalf("password hash changed on idempotent init")
	}
	var count int64
	if err := GetDB().Model(&model.HistoryOfSeeders{}).Where("seeder_name = ?", "UserPasswordHash").Count(&count).Error; err != nil {
		t.Fatalf("count seeder history: %v", err)
	}
	if count != 1 {
		t.Fatalf("password seeder history rows = %d, want 1", count)
	}
}

func TestInitDBNormalizationFailureRestoresExistingConnection(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "x-ui.db")
	if err := InitDB(dbPath); err != nil {
		t.Fatalf("initial InitDB() failed: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB() })
	if err := GetDB().Create(&model.Inbound{
		Tag:            "legacy-invalid-ss",
		Enable:         true,
		Port:           12345,
		Protocol:       model.Shadowsocks,
		Settings:       `{"method":"none","clients":[]}`,
		StreamSettings: `{}`,
	}).Error; err != nil {
		t.Fatalf("seed invalid inbound: %v", err)
	}
	previous := GetDB()
	if err := InitDB(dbPath); err == nil {
		t.Fatal("InitDB() unexpectedly accepted removed Shadowsocks cipher")
	}
	if GetDB() != previous {
		t.Fatal("failed InitDB() did not restore the previous connection")
	}
	var inbound model.Inbound
	if err := GetDB().Where("tag = ?", "legacy-invalid-ss").First(&inbound).Error; err != nil {
		t.Fatalf("previous connection unusable after failed init: %v", err)
	}
	if inbound.Settings != `{"method":"none","clients":[]}` {
		t.Fatalf("failed normalization changed original settings: %s", inbound.Settings)
	}
}

func TestPrepareDBDoesNotReplaceGlobalConnection(t *testing.T) {
	activePath := filepath.Join(t.TempDir(), "active.db")
	candidatePath := filepath.Join(t.TempDir(), "candidate.db")
	if err := InitDB(activePath); err != nil {
		t.Fatalf("InitDB() failed: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB() })
	previous := GetDB()
	candidate, err := PrepareDB(candidatePath)
	if err != nil {
		t.Fatalf("PrepareDB() failed: %v", err)
	}
	pool, err := candidate.DB()
	if err != nil {
		t.Fatalf("candidate pool: %v", err)
	}
	defer pool.Close()
	if GetDB() != previous {
		t.Fatal("PrepareDB() replaced global connection")
	}
}
