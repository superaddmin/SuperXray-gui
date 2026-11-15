package database

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestResetInboundTrafficRestoresOnlyQuotaDisabledInbounds(t *testing.T) {
	db := initRepositoryTestDB(t)
	repository := NewRepositories(db).Inbounds
	now := time.Now().UnixMilli()
	for _, test := range []struct {
		name                    string
		enabled                 bool
		quotaDisabled           bool
		up, down, total, expiry int64
		wantDisabledByQuota     bool
		want                    bool
	}{
		{"quota", true, false, 60, 40, 100, 0, true, true},
		{"over quota", true, false, 120, 0, 100, now + 3600000, true, true},
		{"manual below quota", false, false, 10, 0, 100, 0, false, false},
		{"manual at quota", false, false, 60, 40, 100, 0, false, false},
		{"manual over quota", false, false, 120, 0, 100, 0, false, false},
		{"manual unlimited", false, false, 100, 0, 0, 0, false, false},
		{"expired at quota", true, false, 100, 0, 100, now, false, false},
		{"expired below quota", true, false, 10, 0, 100, now - 3600000, false, false},
		{"expired after quota disable", false, true, 100, 0, 100, now - 3600000, true, false},
		{"enabled", true, false, 10, 0, 100, 0, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			inbound := &model.Inbound{Tag: test.name, Enable: test.enabled, QuotaDisabled: test.quotaDisabled, Up: test.up, Down: test.down, Total: test.total, ExpiryTime: test.expiry}
			if err := db.Create(inbound).Error; err != nil {
				t.Fatal(err)
			}
			if _, err := repository.DisableInvalidInbounds(nil, now); err != nil {
				t.Fatal(err)
			}
			var disabled model.Inbound
			if err := db.First(&disabled, inbound.Id).Error; err != nil {
				t.Fatal(err)
			}
			if disabled.QuotaDisabled != test.wantDisabledByQuota {
				t.Fatalf("quota disable marker = %v, want %v", disabled.QuotaDisabled, test.wantDisabledByQuota)
			}
			if err := repository.ResetInboundTraffic(inbound.Id); err != nil {
				t.Fatal(err)
			}
			var got model.Inbound
			if err := db.First(&got, inbound.Id).Error; err != nil {
				t.Fatal(err)
			}
			if got.Enable != test.want || got.Up != 0 || got.Down != 0 || got.QuotaDisabled {
				t.Fatalf("reset result: enable=%v up=%d down=%d quotaDisabled=%v", got.Enable, got.Up, got.Down, got.QuotaDisabled)
			}
		})
	}
}

func TestQuotaDisabledMigrationKeepsLegacyDisabledInboundsOff(t *testing.T) {
	db := initRepositoryTestDB(t)
	// Recreate the pre-upgrade schema and seed a disabled row whose reason
	// cannot be inferred safely from its counters.
	if err := db.Migrator().DropColumn(&model.Inbound{}, "QuotaDisabled"); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO inbounds (tag, enable, up, down, total, expiry_time) VALUES ('legacy-disabled', false, 100, 0, 100, 0)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := initModels(db); err != nil {
		t.Fatal(err)
	}
	var inbound model.Inbound
	if err := db.Where("tag = ?", "legacy-disabled").First(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	var markerCount int64
	if err := db.Model(&model.Inbound{}).Where("id = ? AND quota_disabled = ?", inbound.Id, false).Count(&markerCount).Error; err != nil || markerCount != 1 {
		t.Fatalf("migration did not default the legacy marker to false: count=%d err=%v", markerCount, err)
	}
	if err := NewRepositories(db).Inbounds.ResetInboundTraffic(inbound.Id); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&inbound, inbound.Id).Error; err != nil || inbound.Enable || inbound.QuotaDisabled || inbound.Up != 0 {
		t.Fatalf("legacy disabled inbound unexpectedly restored: %+v, %v", inbound, err)
	}
}

func TestQuotaDisabledMarkerCannotBeImportedOrExportedAsJSON(t *testing.T) {
	data, err := json.Marshal(&model.Inbound{QuotaDisabled: true})
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"QuotaDisabled", "quotaDisabled", "quota_disabled"} {
		if _, ok := values[name]; ok {
			t.Fatalf("internal quota marker exported as %q", name)
		}
	}
	var inbound model.Inbound
	if err := json.Unmarshal([]byte(`{"QuotaDisabled":true,"quotaDisabled":true,"quota_disabled":true}`), &inbound); err != nil || inbound.QuotaDisabled {
		t.Fatalf("internal quota marker accepted from JSON: %v", err)
	}
}
