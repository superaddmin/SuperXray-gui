package database

import (
	"path/filepath"
	"testing"
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
