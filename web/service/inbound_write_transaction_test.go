package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

func TestAddInboundNeverOverwritesSuppliedPrimaryKeys(t *testing.T) {
	setupInboundServiceTestDB(t)
	db := database.GetDB()
	original := inboundWriteFixture("original", "original@example.test")
	if _, _, err := (&InboundService{}).AddInbound(original); err != nil {
		t.Fatal(err)
	}
	var originalTraffic xray.ClientTraffic
	if err := db.Where("inbound_id = ?", original.Id).First(&originalTraffic).Error; err != nil {
		t.Fatal(err)
	}
	copy := inboundWriteFixture("copy", "copy@example.test")
	copy.Id, copy.Port = original.Id, original.Port+1
	copy.ClientStats = []xray.ClientTraffic{{Id: originalTraffic.Id, InboundId: original.Id, Email: "copy@example.test", Up: 11, Down: 13, AllTime: 24}}
	if _, _, err := (&InboundService{}).AddInbound(copy); err != nil {
		t.Fatal(err)
	}
	if copy.Id == original.Id || copy.ClientStats[0].Id == originalTraffic.Id || copy.ClientStats[0].InboundId != copy.Id {
		t.Fatalf("supplied primary keys were reused: %+v", copy)
	}
	var stored model.Inbound
	if err := db.First(&stored, original.Id).Error; err != nil || stored.Tag != original.Tag || stored.Settings != original.Settings {
		t.Fatalf("original inbound changed: %+v, %v", stored, err)
	}
	var traffic xray.ClientTraffic
	if err := db.First(&traffic, originalTraffic.Id).Error; err != nil || traffic.Email != originalTraffic.Email || traffic.InboundId != original.Id {
		t.Fatalf("original traffic changed: %+v, %v", traffic, err)
	}
	traffic = xray.ClientTraffic{}
	if err := db.First(&traffic, copy.ClientStats[0].Id).Error; err != nil || traffic.Up != 11 || traffic.Down != 13 || traffic.AllTime != 24 {
		t.Fatalf("imported traffic history lost: %+v, %v", traffic, err)
	}
}

func TestAddInboundRollsBackAllClientStatistics(t *testing.T) {
	setupInboundServiceTestDB(t)
	db := database.GetDB()
	if err := db.Exec(`CREATE TRIGGER fail_second_client BEFORE INSERT ON client_traffics WHEN NEW.email = 'second@example.test' BEGIN SELECT RAISE(ABORT, 'forced client insertion failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	inbound := inboundWriteFixture("rollback", "first@example.test")
	inbound.Settings = `{"clients":[{"id":"00000000-0000-4000-8000-000000000001","email":"first@example.test"},{"id":"00000000-0000-4000-8000-000000000002","email":"second@example.test"}]}`
	if _, _, err := (&InboundService{}).AddInbound(inbound); err == nil || !strings.Contains(err.Error(), "forced client insertion failure") {
		t.Fatalf("expected client insertion error: %v", err)
	}
	assertInboundWriteCounts(t, 0, 0)
}

func TestAddInboundRejectsExistingTrafficWithoutReassigningIt(t *testing.T) {
	setupInboundServiceTestDB(t)
	db := database.GetDB()
	original := inboundWriteFixture("original", "original@example.test")
	if _, _, err := (&InboundService{}).AddInbound(original); err != nil {
		t.Fatal(err)
	}
	inbound := inboundWriteFixture("duplicate-traffic", "new@example.test")
	inbound.Port++
	inbound.ClientStats = []xray.ClientTraffic{{Email: "original@example.test", InboundId: original.Id}}
	if _, _, err := (&InboundService{}).AddInbound(inbound); err == nil {
		t.Fatal("duplicate traffic email was accepted")
	}
	assertInboundWriteCounts(t, 1, 1)
	var traffic xray.ClientTraffic
	if err := db.First(&traffic).Error; err != nil || traffic.InboundId != original.Id {
		t.Fatalf("existing traffic was reassigned: %+v, %v", traffic, err)
	}
}

func TestAddInboundReturnsCommitFailure(t *testing.T) {
	setupInboundServiceTestDB(t)
	installDeferredCommitFailure(t, "inbounds", "INSERT", "fail_inbound_insert_commit")
	if _, _, err := (&InboundService{}).AddInbound(inboundWriteFixture("commit", "commit@example.test")); err == nil || !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Fatalf("expected deferred commit failure: %v", err)
	}
	assertInboundWriteCounts(t, 0, 0)
}

func TestInboundWriteOperationsRollBackOnSaveOrCommitFailure(t *testing.T) {
	for _, operation := range []string{"update-inbound", "add-client", "update-client"} {
		for _, failure := range []string{"save", "commit"} {
			t.Run(operation+"/"+failure, func(t *testing.T) {
				setupInboundServiceTestDB(t)
				db := database.GetDB()
				// An old client without an email exercises AddClientStat during update.
				original := inboundWriteFixture("old", "")
				if err := db.Create(original).Error; err != nil {
					t.Fatal(err)
				}
				if failure == "save" {
					if err := db.Exec(`CREATE TRIGGER fail_inbound_save BEFORE UPDATE ON inbounds BEGIN SELECT RAISE(ABORT, 'forced inbound save failure'); END`).Error; err != nil {
						t.Fatal(err)
					}
				} else {
					installDeferredCommitFailure(t, "inbounds", "UPDATE", "fail_inbound_update_commit")
				}
				updated := inboundWriteFixture("changed", "new@example.test")
				updated.Id = original.Id
				service := &InboundService{}
				var err error
				switch operation {
				case "update-inbound":
					_, _, err = service.UpdateInbound(updated)
				case "add-client":
					_, err = service.AddInboundClient(updated)
				case "update-client":
					_, err = service.UpdateInboundClient(updated, "00000000-0000-4000-8000-000000000001")
				}
				if err == nil || (failure == "save" && !strings.Contains(err.Error(), "forced inbound save failure")) || (failure == "commit" && !strings.Contains(strings.ToLower(err.Error()), "foreign key")) {
					t.Fatalf("expected %s failure, got %v", failure, err)
				}
				var stored model.Inbound
				if err := db.First(&stored, original.Id).Error; err != nil || stored.Settings != original.Settings || stored.Remark != original.Remark {
					t.Fatalf("failed write changed inbound: %+v, %v", stored, err)
				}
				assertInboundWriteCounts(t, 1, 0)
			})
		}
	}
}

func TestInboundWritesPreserveUnknownFieldsAndLargeIntegers(t *testing.T) {
	setupInboundServiceTestDB(t)
	service := &InboundService{}
	inbound := inboundWriteFixture("extensions", "extension@example.test")
	inbound.Settings = `{"vendor":9007199254740993,"clients":[{"id":"00000000-0000-4000-8000-000000000001","email":"extension@example.test","enable":true,"vendor":{"large":9007199254740993},"created_at":123}]}`
	if _, _, err := service.AddInbound(inbound); err != nil {
		t.Fatal(err)
	}
	assertExtensions := func(stage string, settings string) {
		t.Helper()
		if strings.Count(settings, "9007199254740993") != 2 || !strings.Contains(settings, `"created_at": 123`) {
			t.Fatalf("%s lost extension data or timestamp: %s", stage, settings)
		}
	}
	assertExtensions("create", inbound.Settings)
	inbound.Remark = "updated"
	if _, _, err := service.UpdateInbound(inbound); err != nil {
		t.Fatal(err)
	}
	stored, err := service.GetInbound(inbound.Id)
	if err != nil {
		t.Fatal(err)
	}
	assertExtensions("update", stored.Settings)
	runtimeInbound, err := service.buildRuntimeInboundForAPI(nil, stored)
	if err != nil {
		t.Fatal(err)
	}
	assertExtensions("runtime API", runtimeInbound.Settings)
	for _, operation := range []string{"append", "replace"} {
		payload := &model.Inbound{Id: inbound.Id, Settings: `{"clients":[{"id":"00000000-0000-4000-8000-000000000002","email":"second@example.test","enable":false,"vendor":9007199254740995}]}`}
		var err error
		if operation == "append" {
			_, err = service.AddInboundClient(payload)
		} else {
			_, err = service.UpdateInboundClient(payload, "00000000-0000-4000-8000-000000000002")
		}
		if err != nil {
			t.Fatal(err)
		}
		stored, err = service.GetInbound(inbound.Id)
		if err != nil {
			t.Fatal(err)
		}
		assertExtensions(operation, stored.Settings)
		if !strings.Contains(stored.Settings, "9007199254740995") {
			t.Fatalf("%s lost new client extension: %s", operation, stored.Settings)
		}
	}
}

func TestConcurrentClientAppendsPreserveEveryClient(t *testing.T) {
	setupInboundServiceTestDB(t)
	inbound := inboundWriteFixture("concurrent", "")
	inbound.Settings = `{"clients":[]}`
	if _, _, err := (&InboundService{}).AddInbound(inbound); err != nil {
		t.Fatal(err)
	}
	const count = 8
	var workers sync.WaitGroup
	results := make(chan error, count)
	start := make(chan struct{})
	for i := 0; i < count; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			payload := &model.Inbound{Id: inbound.Id, Settings: fmt.Sprintf(`{"clients":[{"id":"00000000-0000-4000-8000-%012d","email":"client%d@example.test","enable":false}]}`, i+1, i)}
			_, err := (&InboundService{}).AddInboundClient(payload)
			results <- err
		}(i)
	}
	close(start)
	workers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	stored, err := (&InboundService{}).GetInbound(inbound.Id)
	if err != nil {
		t.Fatal(err)
	}
	var settings struct {
		Clients []model.Client `json:"clients"`
	}
	if err := json.Unmarshal([]byte(stored.Settings), &settings); err != nil || len(settings.Clients) != count {
		t.Fatalf("concurrent append lost clients: %d, %v", len(settings.Clients), err)
	}
	assertInboundWriteCounts(t, 1, count)
}

func TestShadowsocksInboundWritesPreserveUnknownFieldsAndLargeIntegers(t *testing.T) {
	setupInboundServiceTestDB(t)
	service := &InboundService{}
	inbound := inboundWriteFixture("ss-extensions", "ss@example.test")
	inbound.Protocol = model.Shadowsocks
	inbound.Settings = `{"method":"aes_128_gcm","vendor":9007199254740993,"clients":[{"email":"ss@example.test","password":"test-password","vendor":{"large":9007199254740993}}]}`
	for _, stage := range []string{"create", "update"} {
		var err error
		if stage == "create" {
			_, _, err = service.AddInbound(inbound)
		} else {
			_, _, err = service.UpdateInbound(inbound)
		}
		if err != nil {
			t.Fatal(err)
		}
		stored, err := service.GetInbound(inbound.Id)
		if err != nil || strings.Count(stored.Settings, "9007199254740993") != 2 || !strings.Contains(stored.Settings, `"method": "aes-128-gcm"`) {
			t.Fatalf("%s lost shadowsocks extension data: %+v, %v", stage, stored, err)
		}
	}
}

func inboundWriteFixture(tag, email string) *model.Inbound {
	return &model.Inbound{
		UserId: 1, Remark: tag, Port: 19991, Protocol: model.VLESS, Tag: tag,
		Settings:       fmt.Sprintf(`{"clients":[{"id":"00000000-0000-4000-8000-000000000001","email":%q,"enable":false}]}`, email),
		StreamSettings: `{"network":"tcp","security":"none"}`, Sniffing: `{}`,
	}
}

func assertInboundWriteCounts(t *testing.T, wantInbounds, wantTraffics int64) {
	t.Helper()
	var inbounds, traffics int64
	db := database.GetDB()
	if err := db.Model(&model.Inbound{}).Count(&inbounds).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&xray.ClientTraffic{}).Count(&traffics).Error; err != nil {
		t.Fatal(err)
	}
	if inbounds != wantInbounds || traffics != wantTraffics {
		t.Fatalf("persisted rows = %d inbounds, %d client statistics; want %d/%d", inbounds, traffics, wantInbounds, wantTraffics)
	}
}
