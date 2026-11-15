package service

import (
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestUpdateInboundClearsAutomaticQuotaDisable(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "manual-disable"
		if enabled {
			name = "manual-enable"
		}
		t.Run(name, func(t *testing.T) {
			setupInboundServiceTestDB(t)
			db := database.GetDB()
			inbound := inboundWriteFixture(name, "manual@example.test")
			inbound.Up, inbound.Total = 100, 100
			inbound.QuotaDisabled = true
			if err := db.Create(inbound).Error; err != nil {
				t.Fatal(err)
			}
			inbound.Enable = enabled
			svc := &InboundService{}
			if _, _, err := svc.UpdateInbound(inbound); err != nil {
				t.Fatal(err)
			}
			var stored model.Inbound
			if err := db.First(&stored, inbound.Id).Error; err != nil || stored.QuotaDisabled || stored.Enable != enabled {
				t.Fatalf("explicit update did not own enable state: %+v, %v", stored, err)
			}
			if err := svc.ResetInboundTraffic(inbound.Id); err != nil {
				t.Fatal(err)
			}
			if err := db.First(&stored, inbound.Id).Error; err != nil || stored.Enable != enabled || stored.QuotaDisabled || stored.Up != 0 {
				t.Fatalf("reset changed an explicitly selected enable state: %+v, %v", stored, err)
			}
		})
	}
}

func TestFailedManualUpdatePreservesAutomaticQuotaDisable(t *testing.T) {
	setupInboundServiceTestDB(t)
	db := database.GetDB()
	inbound := inboundWriteFixture("quota-disabled", "manual@example.test")
	inbound.Up, inbound.Total = 100, 100
	inbound.QuotaDisabled = true
	if err := db.Create(inbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER fail_manual_update BEFORE UPDATE ON inbounds BEGIN SELECT RAISE(ABORT, 'forced update failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := (&InboundService{}).UpdateInbound(inbound); err == nil {
		t.Fatal("expected manual update failure")
	}
	var stored model.Inbound
	if err := db.First(&stored, inbound.Id).Error; err != nil || !stored.QuotaDisabled || stored.Enable {
		t.Fatalf("failed update lost automatic quota disable: %+v, %v", stored, err)
	}
}
