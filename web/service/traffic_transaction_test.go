package service

import (
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

func TestInboundServiceAddTrafficRollsBackOnClientSaveError(t *testing.T) {
	setupInboundServiceTestDB(t)
	db := database.GetDB()
	inbound := createTrafficTestInbound(t, "client-save-error")
	client := xray.ClientTraffic{
		InboundId: inbound.Id,
		Email:     "client-save-error@example.test",
		Enable:    true,
		Up:        10,
		Down:      20,
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatalf("create client traffic: %v", err)
	}
	if err := db.Exec(`
		CREATE TRIGGER fail_client_traffic_save
		BEFORE UPDATE ON client_traffics
		WHEN OLD.email = 'client-save-error@example.test'
		BEGIN
			SELECT RAISE(ABORT, 'forced client traffic save failure');
		END;
	`).Error; err != nil {
		t.Fatalf("create client traffic failure trigger: %v", err)
	}

	err, _ := (&InboundService{}).AddTraffic(
		[]*xray.Traffic{{IsInbound: true, Tag: inbound.Tag, Up: 3, Down: 4}},
		[]*xray.ClientTraffic{{Email: client.Email, Up: 5, Down: 6}},
	)
	if err == nil || !strings.Contains(err.Error(), "forced client traffic save failure") {
		t.Fatalf("AddTraffic() error = %v, want forced client save failure", err)
	}

	assertInboundTraffic(t, inbound.Id, 0, 0)
	var storedClient xray.ClientTraffic
	if err := db.First(&storedClient, client.Id).Error; err != nil {
		t.Fatalf("reload client traffic: %v", err)
	}
	if storedClient.Up != 10 || storedClient.Down != 20 {
		t.Fatalf("client traffic after rollback = %d/%d, want 10/20", storedClient.Up, storedClient.Down)
	}
}

func TestInboundServiceAddTrafficRollsBackOnMaintenanceError(t *testing.T) {
	setupInboundServiceTestDB(t)
	setTrafficTestProcess(t, nil)
	db := database.GetDB()
	inbound := createTrafficTestInbound(t, "maintenance-error")
	if err := db.Migrator().DropTable(&xray.ClientTraffic{}); err != nil {
		t.Fatalf("drop client traffic table: %v", err)
	}

	err, _ := (&InboundService{}).AddTraffic(
		[]*xray.Traffic{{IsInbound: true, Tag: inbound.Tag, Up: 7, Down: 8}},
		nil,
	)
	if err == nil {
		t.Fatal("AddTraffic() error = nil, want maintenance query failure")
	}

	assertInboundTraffic(t, inbound.Id, 0, 0)
}

func TestInboundServiceAddTrafficReturnsCommitError(t *testing.T) {
	setupInboundServiceTestDB(t)
	setTrafficTestProcess(t, nil)
	inbound := createTrafficTestInbound(t, "inbound-commit-error")
	installDeferredCommitFailure(t, "inbounds", "UPDATE OF up", "fail_inbound_traffic_commit")

	err, _ := (&InboundService{}).AddTraffic(
		[]*xray.Traffic{{IsInbound: true, Tag: inbound.Tag, Up: 11, Down: 12}},
		nil,
	)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Fatalf("AddTraffic() error = %v, want deferred foreign key commit failure", err)
	}

	assertInboundTraffic(t, inbound.Id, 0, 0)
}

func TestOutboundServiceAddTrafficReturnsCommitError(t *testing.T) {
	setupInboundServiceTestDB(t)
	db := database.GetDB()
	outbound := model.OutboundTraffics{Tag: "outbound-commit-error"}
	if err := db.Create(&outbound).Error; err != nil {
		t.Fatalf("create outbound traffic: %v", err)
	}
	installDeferredCommitFailure(t, "outbound_traffics", "UPDATE OF up", "fail_outbound_traffic_commit")

	err, _ := (&OutboundService{}).AddTraffic(
		[]*xray.Traffic{{IsOutbound: true, Tag: outbound.Tag, Up: 13, Down: 14}},
		nil,
	)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Fatalf("AddTraffic() error = %v, want deferred foreign key commit failure", err)
	}

	var stored model.OutboundTraffics
	if err := db.First(&stored, outbound.Id).Error; err != nil {
		t.Fatalf("reload outbound traffic: %v", err)
	}
	if stored.Up != 0 || stored.Down != 0 || stored.Total != 0 {
		t.Fatalf("outbound traffic after failed commit = %d/%d/%d, want 0/0/0", stored.Up, stored.Down, stored.Total)
	}
}

func createTrafficTestInbound(t *testing.T, tag string) *model.Inbound {
	t.Helper()
	inbound := &model.Inbound{
		UserId:         1,
		Remark:         tag,
		Enable:         true,
		TrafficReset:   "never",
		Protocol:       model.VLESS,
		Settings:       `{"clients":[]}`,
		StreamSettings: `{}`,
		Tag:            tag,
		Sniffing:       `{}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound %q: %v", tag, err)
	}
	return inbound
}

func assertInboundTraffic(t *testing.T, id int, wantUp, wantDown int64) {
	t.Helper()
	var stored model.Inbound
	if err := database.GetDB().First(&stored, id).Error; err != nil {
		t.Fatalf("reload inbound: %v", err)
	}
	if stored.Up != wantUp || stored.Down != wantDown {
		t.Fatalf("inbound traffic after rollback = %d/%d, want %d/%d", stored.Up, stored.Down, wantUp, wantDown)
	}
}

func installDeferredCommitFailure(t *testing.T, table, event, trigger string) {
	t.Helper()
	db := database.GetDB()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	statements := []string{
		"PRAGMA foreign_keys = ON",
		"CREATE TABLE traffic_commit_parent (id INTEGER PRIMARY KEY)",
		"CREATE TABLE traffic_commit_child (parent_id INTEGER, FOREIGN KEY(parent_id) REFERENCES traffic_commit_parent(id) DEFERRABLE INITIALLY DEFERRED)",
		"CREATE TRIGGER " + trigger + " AFTER " + event + " ON " + table + " BEGIN INSERT INTO traffic_commit_child(parent_id) VALUES (1); END",
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("install deferred commit failure %q: %v", statement, err)
		}
	}
}

func setTrafficTestProcess(t *testing.T, process *xray.Process) {
	t.Helper()
	oldProcess := p
	p = process
	t.Cleanup(func() {
		p = oldProcess
	})
}
