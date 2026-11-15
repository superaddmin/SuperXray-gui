package database

import (
	"path/filepath"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/crypto"
)

func TestInitDBDoesNotSeedPanelCredentials(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "x-ui.db")
	for range 2 {
		if err := InitDB(dbPath); err != nil {
			t.Fatalf("InitDB: %v", err)
		}
		var count int64
		if err := GetDB().Model(&model.User{}).Count(&count).Error; err != nil {
			t.Fatalf("count users: %v", err)
		}
		if err := CloseDB(); err != nil {
			t.Fatalf("CloseDB: %v", err)
		}
		if count != 0 {
			t.Fatalf("fresh database contains %d users; credentials must be configured explicitly", count)
		}
	}
}

func TestPrepareDBPreservesExistingPanelCredentials(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "import-candidate.db")
	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB() })
	hash, err := crypto.HashPasswordAsBcrypt("admin")
	if err != nil {
		t.Fatalf("hash existing credential: %v", err)
	}
	existing := model.User{Username: "admin", Password: hash}
	if err := GetDB().Create(&existing).Error; err != nil {
		t.Fatalf("create existing user: %v", err)
	}
	if err := CloseDB(); err != nil {
		t.Fatalf("CloseDB: %v", err)
	}
	candidate, err := PrepareDB(dbPath)
	if err != nil {
		t.Fatalf("PrepareDB: %v", err)
	}
	t.Cleanup(func() { _ = closeGormDB(candidate) })
	var stored model.User
	if err := candidate.First(&stored, existing.Id).Error; err != nil {
		t.Fatalf("read imported user: %v", err)
	}
	if stored.Username != existing.Username || stored.Password != existing.Password {
		t.Fatal("candidate preparation changed existing credentials")
	}
}
