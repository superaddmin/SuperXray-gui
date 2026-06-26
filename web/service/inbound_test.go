package service

import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/op/go-logging"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
	"gorm.io/gorm"
)

func TestInboundServiceUsesRepositoryBoundaryForReadPaths(t *testing.T) {
	setupInboundServiceTestDB(t)

	repo := &fakeInboundRepository{
		inbounds: []*model.Inbound{
			{
				Id:             1,
				UserId:         7,
				Remark:         "one",
				Protocol:       model.VLESS,
				Port:           443,
				Tag:            "one",
				Settings:       `{"clients":[{"id":"uuid-1","email":"client@example.test","subId":"sub-1","tgId":42}]}`,
				StreamSettings: `{"network":"tcp","security":"tls"}`,
				TrafficReset:   "daily",
				ClientStats: []xray.ClientTraffic{
					{Id: 2, Email: "CLIENT@example.test", Up: 50, Down: 5, LastOnline: 77},
					{Id: 1, Email: "occupied@example.test", InboundId: 1, Up: 200, Down: 100, LastOnline: 99},
					{Id: 3, Email: "client@example.test", InboundId: 1, Up: 7, Down: 3, LastOnline: 55},
				},
			},
			{
				Id:             2,
				UserId:         8,
				Remark:         "two",
				Protocol:       model.Shadowsocks,
				Port:           8443,
				Tag:            "two",
				Settings:       `{"clients":[{"id":"uuid-2","email":"secondary@example.test","subId":"sub-2"}]}`,
				StreamSettings: `{"network":"tcp"}`,
				TrafficReset:   "never",
				ClientStats: []xray.ClientTraffic{
					{Email: "secondary@example.test", Up: 1, Down: 1, LastOnline: 11},
				},
			},
		},
		options: []database.InboundOptionRecord{
			{ID: 1, Remark: "one", Tag: "one", Protocol: string(model.VLESS), Port: 443, TLSFlowCapable: true},
			{ID: 2, Remark: "two", Tag: "two", Protocol: string(model.Shadowsocks), Port: 8443, SSMethod: "2022-blake3-aes-128-gcm"},
		},
		clientEmails: []string{"occupied@example.test"},
		portExists:   true,
	}
	svc := NewInboundService(repo)

	userInbounds, err := svc.GetInbounds(7)
	if err != nil {
		t.Fatalf("GetInbounds() returned error: %v", err)
	}
	if len(repo.listByUserIDCalls) != 1 || repo.listByUserIDCalls[0] != 7 {
		t.Fatalf("ListByUserID calls = %v, want [7]", repo.listByUserIDCalls)
	}
	if len(userInbounds) != 1 {
		t.Fatalf("GetInbounds() returned %d inbounds, want 1", len(userInbounds))
	}
	if got := userInbounds[0].ClientStats[0].UUID; got != "uuid-1" {
		t.Fatalf("ClientStats[0].UUID = %q, want uuid-1", got)
	}
	if got := userInbounds[0].ClientStats[0].SubId; got != "sub-1" {
		t.Fatalf("ClientStats[0].SubId = %q, want sub-1", got)
	}

	allInbounds, err := svc.GetAllInbounds()
	if err != nil {
		t.Fatalf("GetAllInbounds() returned error: %v", err)
	}
	if !repo.listAllCalled {
		t.Fatal("GetAllInbounds() did not call ListAll")
	}
	if len(allInbounds) != 2 {
		t.Fatalf("GetAllInbounds() returned %d inbounds, want 2", len(allInbounds))
	}
	if got := allInbounds[1].ClientStats[0].UUID; got != "uuid-2" {
		t.Fatalf("allInbounds[1].ClientStats[0].UUID = %q, want uuid-2", got)
	}

	inbound, err := svc.GetInbound(1)
	if err != nil {
		t.Fatalf("GetInbound() returned error: %v", err)
	}
	if len(repo.getCalls) != 1 || repo.getCalls[0] != 1 {
		t.Fatalf("Get calls = %v, want [1]", repo.getCalls)
	}
	if inbound.Id != 1 {
		t.Fatalf("GetInbound().Id = %d, want 1", inbound.Id)
	}

	options, err := svc.GetInboundOptions(7)
	if err != nil {
		t.Fatalf("GetInboundOptions() returned error: %v", err)
	}
	if len(repo.listOptionsByUserIDCalls) != 1 || repo.listOptionsByUserIDCalls[0] != 7 {
		t.Fatalf("ListOptionsByUserID calls = %v, want [7]", repo.listOptionsByUserIDCalls)
	}
	if len(options) != 2 {
		t.Fatalf("GetInboundOptions() returned %d options, want 2", len(options))
	}
	if !options[0].TlsFlowCapable || options[1].SsMethod != "2022-blake3-aes-128-gcm" {
		t.Fatalf("GetInboundOptions() returned %+v, want repository option metadata", options)
	}

	resetInbounds, err := svc.GetInboundsByTrafficReset("daily")
	if err != nil {
		t.Fatalf("GetInboundsByTrafficReset() returned error: %v", err)
	}
	if len(repo.listByTrafficResetCalls) != 1 || repo.listByTrafficResetCalls[0] != "daily" {
		t.Fatalf("ListByTrafficReset calls = %v, want [daily]", repo.listByTrafficResetCalls)
	}
	if len(resetInbounds) != 1 || resetInbounds[0].Id != 1 {
		t.Fatalf("GetInboundsByTrafficReset() returned %+v, want inbound 1", resetInbounds)
	}

	exists, err := svc.checkPortExist("", 443, 9)
	if err != nil {
		t.Fatalf("checkPortExist() returned error: %v", err)
	}
	if !exists {
		t.Fatal("checkPortExist() returned false, want true")
	}
	if len(repo.portExistsCalls) != 1 {
		t.Fatalf("PortExists call count = %d, want 1", len(repo.portExistsCalls))
	}
	if got := repo.portExistsCalls[0]; got != (portExistsCall{listen: "", port: 443, ignoreID: 9}) {
		t.Fatalf("PortExists call = %+v, want wildcard listen/443/9", got)
	}

	duplicateEmail, err := svc.checkEmailExistForInbound(&model.Inbound{
		Settings: `{"clients":[{"id":"new-client","email":"occupied@example.test"}]}`,
	})
	if err != nil {
		t.Fatalf("checkEmailExistForInbound() returned error: %v", err)
	}
	if duplicateEmail != "occupied@example.test" {
		t.Fatalf("checkEmailExistForInbound() = %q, want occupied@example.test", duplicateEmail)
	}
	if repo.listClientEmailsCalls != 1 {
		t.Fatalf("ListClientEmails call count = %d, want 1", repo.listClientEmailsCalls)
	}

	traffic, inbound, err := svc.GetClientInboundByEmail("occupied@example.test")
	if err != nil {
		t.Fatalf("GetClientInboundByEmail() returned error: %v", err)
	}
	if traffic == nil || inbound == nil {
		t.Fatalf("GetClientInboundByEmail() returned nil values: traffic=%v inbound=%v", traffic, inbound)
	}
	if traffic.Email != "occupied@example.test" || inbound.Id != 1 {
		t.Fatalf("GetClientInboundByEmail() returned traffic=%+v inbound=%+v, want email occupied@example.test and inbound 1", traffic, inbound)
	}

	traffic, inbound, err = svc.GetClientInboundByTrafficID(1)
	if err != nil {
		t.Fatalf("GetClientInboundByTrafficID() returned error: %v", err)
	}
	if traffic == nil || inbound == nil {
		t.Fatalf("GetClientInboundByTrafficID() returned nil values: traffic=%v inbound=%v", traffic, inbound)
	}
	if traffic.Id != 1 || inbound.Id != 1 {
		t.Fatalf("GetClientInboundByTrafficID() returned traffic=%+v inbound=%+v, want both id 1", traffic, inbound)
	}

	lastOnline, err := svc.GetClientsLastOnline()
	if err != nil {
		t.Fatalf("GetClientsLastOnline() returned error: %v", err)
	}
	if lastOnline["occupied@example.test"] != 99 || lastOnline["CLIENT@example.test"] != 77 {
		t.Fatalf("GetClientsLastOnline() returned %+v, want occupied/client timestamps", lastOnline)
	}

	validEmails, extraEmails, err := svc.FilterAndSortClientEmails([]string{"missing@example.test", "CLIENT@example.test", "occupied@example.test"})
	if err != nil {
		t.Fatalf("FilterAndSortClientEmails() returned error: %v", err)
	}
	if len(validEmails) != 2 || validEmails[0] != "occupied@example.test" || validEmails[1] != "CLIENT@example.test" {
		t.Fatalf("FilterAndSortClientEmails() valid emails = %v, want occupied then CLIENT", validEmails)
	}
	if len(extraEmails) != 1 || extraEmails[0] != "missing@example.test" {
		t.Fatalf("FilterAndSortClientEmails() extra emails = %v, want missing@example.test", extraEmails)
	}

	searchResults, err := svc.SearchInbounds("one")
	if err != nil {
		t.Fatalf("SearchInbounds() returned error: %v", err)
	}
	if len(searchResults) != 1 || searchResults[0].Id != 1 {
		t.Fatalf("SearchInbounds() returned %+v, want inbound 1", searchResults)
	}

	clients, err := svc.GetClients(repo.inbounds[0])
	if err != nil {
		t.Fatalf("GetClients() returned error: %v", err)
	}
	if len(clients) != 1 || clients[0].ID != "uuid-1" || clients[0].TgID != 42 {
		t.Fatalf("GetClients() returned %+v, want uuid-1 tgId 42", clients)
	}

	traffics, err := svc.GetClientTrafficTgBot(42)
	if err != nil {
		t.Fatalf("GetClientTrafficTgBot() returned error: %v", err)
	}
	if len(traffics) != 1 {
		t.Fatalf("GetClientTrafficTgBot() returned %d traffics, want 1", len(traffics))
	}
	if traffics[0].Email != "client@example.test" || traffics[0].UUID != "uuid-1" || traffics[0].SubId != "sub-1" {
		t.Fatalf("GetClientTrafficTgBot() returned %+v, want client@example.test with uuid-1/sub-1", traffics[0])
	}
	if repo.listAllCalls != 2 {
		t.Fatalf("ListAll call count = %d, want 2", repo.listAllCalls)
	}
	if len(repo.listClientTrafficsByEmailsCalls) != 2 {
		t.Fatalf("ListClientTrafficsByEmails calls = %v, want 2 calls", repo.listClientTrafficsByEmailsCalls)
	}
	if got := repo.listClientTrafficsByEmailsCalls[1]; len(got) != 1 || got[0] != "client@example.test" {
		t.Fatalf("second ListClientTrafficsByEmails call = %v, want [client@example.test]", got)
	}

	needRestart, err := svc.ResetClientTraffic(1, "client@example.test")
	if err != nil {
		t.Fatalf("ResetClientTraffic() returned error: %v", err)
	}
	if needRestart {
		t.Fatal("ResetClientTraffic() returned needRestart=true, want false")
	}
	if len(repo.saveClientTrafficCalls) != 1 {
		t.Fatalf("SaveClientTraffic call count = %d, want 1", len(repo.saveClientTrafficCalls))
	}
	if got := repo.saveClientTrafficCalls[0]; got.Email != "client@example.test" || got.Up != 0 || got.Down != 0 || !got.Enable {
		t.Fatalf("SaveClientTraffic payload = %+v, want client@example.test reset", got)
	}
}

func TestInboundServiceGetInboundTagsAndMigrationCleanupUseRepository(t *testing.T) {
	setupInboundServiceTestDB(t)

	repo := &fakeInboundRepository{
		inbounds: []*model.Inbound{
			{Id: 1, Tag: "tag-one"},
			{Id: 2, Tag: "tag-two"},
		},
	}
	svc := NewInboundService(repo)

	tagsJSON, err := svc.GetInboundTags()
	if err != nil {
		t.Fatalf("GetInboundTags() returned error: %v", err)
	}
	if repo.listTagsCalls != 1 {
		t.Fatalf("GetInboundTags() ListTags call count = %d, want 1", repo.listTagsCalls)
	}
	var tags []string
	if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
		t.Fatalf("GetInboundTags() returned invalid JSON: %v", err)
	}
	if len(tags) != 2 || tags[0] != "tag-one" || tags[1] != "tag-two" {
		t.Fatalf("GetInboundTags() returned %v, want [tag-one tag-two]", tags)
	}

	svc.MigrationRemoveOrphanedTraffics()
	if repo.deleteOrphanedClientTrafficsCalls != 1 {
		t.Fatalf("MigrationRemoveOrphanedTraffics() call count = %d, want 1", repo.deleteOrphanedClientTrafficsCalls)
	}
}

func TestInboundServiceDelInboundUsesRepositoryLookupAndDelete(t *testing.T) {
	setupInboundServiceTestDB(t)

	repo := &fakeInboundRepository{
		inbounds: []*model.Inbound{
			{
				Id:          7,
				Remark:      "delete-me",
				Enable:      false,
				Tag:         "delete-me-tag",
				Settings:    `{"clients":[{"id":"delete-client","email":"delete@example.test"}]}`,
				ClientStats: []xray.ClientTraffic{{Id: 9, Email: "delete@example.test", InboundId: 7}},
			},
		},
	}
	svc := NewInboundService(repo)

	needRestart, err := svc.DelInbound(7)
	if err != nil {
		t.Fatalf("DelInbound() returned error: %v", err)
	}
	if needRestart {
		t.Fatal("DelInbound() returned needRestart=true, want false")
	}
	if len(repo.getCalls) != 1 || repo.getCalls[0] != 7 {
		t.Fatalf("DelInbound() Get calls = %v, want [7]", repo.getCalls)
	}
	if len(repo.deleteClientTrafficsCalls) != 1 || repo.deleteClientTrafficsCalls[0] != 7 {
		t.Fatalf("DelInbound() DeleteClientTrafficsByInboundID calls = %v, want [7]", repo.deleteClientTrafficsCalls)
	}
	if len(repo.deleteByIDCalls) != 1 || repo.deleteByIDCalls[0] != 7 {
		t.Fatalf("DelInbound() DeleteByID calls = %v, want [7]", repo.deleteByIDCalls)
	}
}

func TestInboundServiceAddInboundUsesRepositorySave(t *testing.T) {
	setupInboundServiceTestDB(t)

	repo := &fakeInboundRepository{}
	svc := NewInboundService(repo)

	inbound := &model.Inbound{
		Id:             31,
		Remark:         "create-repository-save",
		Enable:         false,
		Protocol:       model.VLESS,
		Port:           10031,
		Tag:            "create-repository-save",
		Settings:       `{"clients":[{"id":"00000000-0000-4000-8000-000000000031","email":"created@example.test","enable":true}]}`,
		StreamSettings: `{"network":"tcp","security":"none"}`,
		Sniffing:       `{}`,
	}

	_, needRestart, err := svc.AddInbound(inbound)
	if err != nil {
		t.Fatalf("AddInbound() returned error: %v", err)
	}
	if needRestart {
		t.Fatal("AddInbound() returned needRestart=true, want false for disabled inbound")
	}
	if len(repo.saveInboundCalls) != 1 || repo.saveInboundCalls[0].Id != 31 {
		t.Fatalf("AddInbound() SaveInbound calls = %+v, want inbound 31", repo.saveInboundCalls)
	}
	if len(repo.createClientTrafficCalls) != 1 || repo.createClientTrafficCalls[0].Email != "created@example.test" {
		t.Fatalf("AddInbound() CreateClientTraffic calls = %+v, want created@example.test", repo.createClientTrafficCalls)
	}
}

func TestInboundServiceClientWritePathsUseRepositorySave(t *testing.T) {
	setupInboundServiceTestDB(t)

	repo := &fakeInboundRepository{
		inbounds: []*model.Inbound{
			{
				Id:             44,
				Remark:         "client-repository-save",
				Enable:         false,
				Protocol:       model.VLESS,
				Port:           10044,
				Tag:            "client-repository-save",
				Settings:       `{"clients":[{"id":"00000000-0000-4000-8000-000000000044","email":"old@example.test","enable":false}]}`,
				StreamSettings: `{"network":"tcp","security":"none"}`,
				Sniffing:       `{}`,
				ClientStats: []xray.ClientTraffic{
					{Id: 44, Email: "old@example.test", InboundId: 44, Enable: false},
				},
			},
		},
	}
	svc := NewInboundService(repo)

	_, err := svc.AddInboundClient(&model.Inbound{
		Id:       44,
		Settings: `{"clients":[{"id":"00000000-0000-4000-8000-000000000045","email":"added@example.test","enable":false}]}`,
	})
	if err != nil {
		t.Fatalf("AddInboundClient() returned error: %v", err)
	}
	if len(repo.saveInboundCalls) != 1 {
		t.Fatalf("AddInboundClient() SaveInbound call count = %d, want 1", len(repo.saveInboundCalls))
	}
	if !strings.Contains(repo.saveInboundCalls[0].Settings, "added@example.test") {
		t.Fatalf("AddInboundClient() saved settings = %s, want added@example.test", repo.saveInboundCalls[0].Settings)
	}

	_, err = svc.UpdateInboundClient(&model.Inbound{
		Id:       44,
		Settings: `{"clients":[{"id":"00000000-0000-4000-8000-000000000044","email":"updated@example.test","enable":false}]}`,
	}, "00000000-0000-4000-8000-000000000044")
	if err != nil {
		t.Fatalf("UpdateInboundClient() returned error: %v", err)
	}
	if len(repo.saveInboundCalls) != 2 {
		t.Fatalf("UpdateInboundClient() SaveInbound call count = %d, want 2", len(repo.saveInboundCalls))
	}
	if !strings.Contains(repo.saveInboundCalls[1].Settings, "updated@example.test") {
		t.Fatalf("UpdateInboundClient() saved settings = %s, want updated@example.test", repo.saveInboundCalls[1].Settings)
	}
	if len(repo.updateClientTrafficCalls) != 1 || repo.updateClientTrafficCalls[0].email != "old@example.test" {
		t.Fatalf("UpdateInboundClient() UpdateClientTraffic calls = %+v, want old@example.test", repo.updateClientTrafficCalls)
	}
}

func TestInboundServiceTrafficAggregationUsesRepositoryBoundary(t *testing.T) {
	setupInboundServiceTestDB(t)

	repo := &fakeInboundRepository{
		inbounds: []*model.Inbound{
			{
				Id:       55,
				Remark:   "traffic-repository-boundary",
				Enable:   false,
				Protocol: model.VLESS,
				Port:     10055,
				Tag:      "traffic-tag",
				Settings: `{"clients":[{"id":"00000000-0000-4000-8000-000000000055","email":"metered@example.test","enable":true,"expiryTime":1000}]}`,
				ClientStats: []xray.ClientTraffic{
					{
						Id:         55,
						InboundId:  55,
						Email:      "metered@example.test",
						Enable:     true,
						Up:         10,
						Down:       20,
						AllTime:    30,
						ExpiryTime: -1000,
					},
				},
			},
		},
	}
	svc := NewInboundService(repo)
	tx := database.GetDB()

	if err := svc.addInboundTraffic(tx, []*xray.Traffic{
		{IsInbound: true, Tag: "traffic-tag", Up: 3, Down: 4},
		{IsInbound: false, Tag: "ignored", Up: 9, Down: 9},
	}); err != nil {
		t.Fatalf("addInboundTraffic() returned error: %v", err)
	}
	if len(repo.addInboundTrafficCalls) != 1 {
		t.Fatalf("addInboundTraffic() repository call count = %d, want 1", len(repo.addInboundTrafficCalls))
	}
	if got := repo.addInboundTrafficCalls[0]; got != (inboundTrafficCall{tag: "traffic-tag", upload: 3, download: 4}) {
		t.Fatalf("addInboundTraffic() repository call = %+v, want traffic-tag 3/4", got)
	}

	if err := svc.addClientTraffic(tx, []*xray.ClientTraffic{
		{Email: "metered@example.test", Up: 5, Down: 6},
	}); err != nil {
		t.Fatalf("addClientTraffic() returned error: %v", err)
	}
	if len(repo.listClientTrafficsByEmailsTxCalls) != 1 {
		t.Fatalf("addClientTraffic() ListClientTrafficsByEmailsTx call count = %d, want 1", len(repo.listClientTrafficsByEmailsTxCalls))
	}
	if got := repo.listClientTrafficsByEmailsTxCalls[0]; len(got) != 1 || got[0] != "metered@example.test" {
		t.Fatalf("addClientTraffic() ListClientTrafficsByEmailsTx call = %v, want [metered@example.test]", got)
	}
	if len(repo.listInboundsByIDsCalls) != 1 {
		t.Fatalf("adjustTraffics() ListInboundsByIDs call count = %d, want 1", len(repo.listInboundsByIDsCalls))
	}
	if got := repo.listInboundsByIDsCalls[0]; len(got) != 1 || got[0] != 55 {
		t.Fatalf("adjustTraffics() ListInboundsByIDs call = %v, want [55]", got)
	}
	if len(repo.saveInboundsCalls) != 1 {
		t.Fatalf("adjustTraffics() SaveInbounds call count = %d, want 1", len(repo.saveInboundsCalls))
	}
	if len(repo.saveClientTrafficsCalls) != 1 || len(repo.saveClientTrafficsCalls[0]) != 1 {
		t.Fatalf("addClientTraffic() SaveClientTraffics calls = %+v, want one metered traffic", repo.saveClientTrafficsCalls)
	}
	savedTraffic := repo.saveClientTrafficsCalls[0][0]
	if savedTraffic.Email != "metered@example.test" || savedTraffic.Up != 15 || savedTraffic.Down != 26 || savedTraffic.AllTime != 41 {
		t.Fatalf("addClientTraffic() saved traffic = %+v, want metered@example.test up/down/all_time 15/26/41", savedTraffic)
	}
	if !strings.Contains(repo.saveInboundsCalls[0][0].Settings, "updated_at") {
		t.Fatalf("adjustTraffics() saved inbound settings = %s, want updated_at backfill", repo.saveInboundsCalls[0][0].Settings)
	}
}

func TestInboundServiceTrafficMaintenanceUsesRepositoryBoundary(t *testing.T) {
	setupInboundServiceTestDB(t)
	oldProcess := p
	p = nil
	t.Cleanup(func() {
		p = oldProcess
	})

	expiredAt := time.Now().UnixMilli() - 1000
	repo := &fakeInboundRepository{
		inbounds: []*model.Inbound{
			{
				Id:       66,
				Remark:   "maintenance-repository-boundary",
				Enable:   true,
				Protocol: model.VLESS,
				Port:     10066,
				Tag:      "maintenance-tag",
				Settings: `{"clients":[{"id":"00000000-0000-4000-8000-000000000066","email":"renew@example.test","enable":true,"expiryTime":1}]}`,
				Up:       80,
				Down:     30,
				Total:    100,
				ClientStats: []xray.ClientTraffic{
					{
						Id:         66,
						InboundId:  66,
						Email:      "renew@example.test",
						Enable:     false,
						Up:         9,
						Down:       8,
						Total:      100,
						Reset:      1,
						ExpiryTime: expiredAt,
					},
					{
						Id:        67,
						InboundId: 66,
						Email:     "depleted-maintenance@example.test",
						Enable:    true,
						Up:        60,
						Down:      0,
						Total:     50,
						Reset:     0,
					},
				},
			},
		},
	}
	svc := NewInboundService(repo)
	tx := database.GetDB()

	needRestart, count, err := svc.autoRenewClients(tx)
	if err != nil {
		t.Fatalf("autoRenewClients() returned error: %v", err)
	}
	if needRestart {
		t.Fatal("autoRenewClients() needRestart=true, want false without process")
	}
	if count != 1 {
		t.Fatalf("autoRenewClients() count = %d, want 1", count)
	}
	if repo.listRenewableClientTrafficsCalls != 1 {
		t.Fatalf("autoRenewClients() ListRenewableClientTraffics call count = %d, want 1", repo.listRenewableClientTrafficsCalls)
	}
	if len(repo.listInboundsByIDsCalls) != 1 {
		t.Fatalf("autoRenewClients() ListInboundsByIDs call count = %d, want 1", len(repo.listInboundsByIDsCalls))
	}
	if len(repo.saveInboundsCalls) != 1 {
		t.Fatalf("autoRenewClients() SaveInbounds call count = %d, want 1", len(repo.saveInboundsCalls))
	}
	if len(repo.saveClientTrafficsCalls) != 1 || len(repo.saveClientTrafficsCalls[0]) != 1 {
		t.Fatalf("autoRenewClients() SaveClientTraffics calls = %+v, want one renewed traffic", repo.saveClientTrafficsCalls)
	}
	renewedTraffic := repo.saveClientTrafficsCalls[0][0]
	if renewedTraffic.Email != "renew@example.test" || renewedTraffic.Up != 0 || renewedTraffic.Down != 0 || !renewedTraffic.Enable || renewedTraffic.ExpiryTime <= expiredAt {
		t.Fatalf("autoRenewClients() saved traffic = %+v, want reset and future expiry", renewedTraffic)
	}

	needRestart, count, err = svc.disableInvalidInbounds(tx)
	if err != nil {
		t.Fatalf("disableInvalidInbounds() returned error: %v", err)
	}
	if needRestart {
		t.Fatal("disableInvalidInbounds() needRestart=true, want false without process")
	}
	if count != 1 {
		t.Fatalf("disableInvalidInbounds() count = %d, want 1", count)
	}
	if repo.listInvalidInboundTagsCalls != 0 {
		t.Fatalf("disableInvalidInbounds() ListInvalidInboundTags call count = %d, want 0 without process", repo.listInvalidInboundTagsCalls)
	}
	if repo.disableInvalidInboundsCalls != 1 {
		t.Fatalf("disableInvalidInbounds() DisableInvalidInbounds call count = %d, want 1", repo.disableInvalidInboundsCalls)
	}

	needRestart, count, err = svc.disableInvalidClients(tx)
	if err != nil {
		t.Fatalf("disableInvalidClients() returned error: %v", err)
	}
	if needRestart {
		t.Fatal("disableInvalidClients() needRestart=true, want false without process")
	}
	if count != 1 {
		t.Fatalf("disableInvalidClients() count = %d, want 1", count)
	}
	if repo.listInvalidClientTrafficTargetsCalls != 0 {
		t.Fatalf("disableInvalidClients() ListInvalidClientTrafficTargets call count = %d, want 0 without process", repo.listInvalidClientTrafficTargetsCalls)
	}
	if repo.disableInvalidClientTrafficsCalls != 1 {
		t.Fatalf("disableInvalidClients() DisableInvalidClientTraffics call count = %d, want 1", repo.disableInvalidClientTrafficsCalls)
	}
}

type fakeInboundRepository struct {
	inbounds                          []*model.Inbound
	options                           []database.InboundOptionRecord
	clientEmails                      []string
	portExists                        bool
	getCalls                          []int
	listByUserIDCalls                 []int
	listAllCalls                      int
	listAllCalled                     bool
	listOptionsByUserIDCalls          []int
	listByTrafficResetCalls           []string
	listClientEmailsCalls             int
	searchByRemarkCalls               []string
	portExistsCalls                   []portExistsCall
	listClientTrafficsByEmailsCalls   [][]string
	listTagsCalls                     int
	resetClientTrafficCalls           []string
	resetAllClientTrafficsCalls       []int
	resetAllTrafficsCalled            bool
	resetInboundTrafficCalls          []int
	deleteOrphanedClientTrafficsCalls int
	addInboundTrafficCalls            []inboundTrafficCall
	listClientTrafficsByEmailsTxCalls [][]string
	listInboundsByIDsCalls            [][]int
	listRenewableClientTrafficsCalls  int
	listInvalidInboundTagsCalls       int
	disableInvalidInboundsCalls       int
	listInvalidClientTrafficTargetsCalls int
	disableInvalidClientTrafficsCalls    int
	saveInboundCalls                  []model.Inbound
	saveInboundsCalls                 [][]model.Inbound
	saveClientTrafficCalls            []xray.ClientTraffic
	saveClientTrafficsCalls           [][]xray.ClientTraffic
	createClientTrafficCalls          []xray.ClientTraffic
	updateClientTrafficCalls          []updateClientTrafficCall
	updateClientIPCalls               []ipUpdateCall
	updateTrafficUsageCalls           []usageUpdateCall
	deleteClientTrafficCalls          []string
	deleteClientTrafficsCalls         []int
	deleteClientIPCalls               []string
	deleteByIDCalls                   []int
	clearClientIPCalls                []string
}

type updateClientTrafficCall struct {
	email  string
	client model.Client
}

type ipUpdateCall struct {
	oldEmail string
	newEmail string
}

type usageUpdateCall struct {
	email    string
	upload   int64
	download int64
}

type inboundTrafficCall struct {
	tag      string
	upload   int64
	download int64
}

type portExistsCall struct {
	listen   string
	port     int
	ignoreID int
}

func (r *fakeInboundRepository) Get(id int) (*model.Inbound, error) {
	r.getCalls = append(r.getCalls, id)
	for _, inbound := range r.inbounds {
		if inbound.Id == id {
			return copyInboundForRepositoryTest(inbound), nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeInboundRepository) ListByUserID(userID int) ([]*model.Inbound, error) {
	r.listByUserIDCalls = append(r.listByUserIDCalls, userID)
	inbounds := make([]*model.Inbound, 0)
	for _, inbound := range r.inbounds {
		if inbound.UserId == userID {
			inbounds = append(inbounds, copyInboundForRepositoryTest(inbound))
		}
	}
	return inbounds, nil
}

func (r *fakeInboundRepository) ListAll() ([]*model.Inbound, error) {
	r.listAllCalled = true
	r.listAllCalls++
	inbounds := make([]*model.Inbound, 0, len(r.inbounds))
	for _, inbound := range r.inbounds {
		inbounds = append(inbounds, copyInboundForRepositoryTest(inbound))
	}
	return inbounds, nil
}

func (r *fakeInboundRepository) ListOptionsByUserID(userID int) ([]database.InboundOptionRecord, error) {
	r.listOptionsByUserIDCalls = append(r.listOptionsByUserIDCalls, userID)
	return append([]database.InboundOptionRecord(nil), r.options...), nil
}

func (r *fakeInboundRepository) ListByTrafficReset(period string) ([]*model.Inbound, error) {
	r.listByTrafficResetCalls = append(r.listByTrafficResetCalls, period)
	inbounds := make([]*model.Inbound, 0)
	for _, inbound := range r.inbounds {
		if inbound.TrafficReset == period {
			inbounds = append(inbounds, copyInboundForRepositoryTest(inbound))
		}
	}
	return inbounds, nil
}

func (r *fakeInboundRepository) PortExists(listen string, port int, ignoreID int) (bool, error) {
	r.portExistsCalls = append(r.portExistsCalls, portExistsCall{listen: listen, port: port, ignoreID: ignoreID})
	return r.portExists, nil
}

func (r *fakeInboundRepository) ListClientEmails() ([]string, error) {
	r.listClientEmailsCalls++
	return append([]string(nil), r.clientEmails...), nil
}

func (r *fakeInboundRepository) ListClientTrafficsLastOnline() ([]xray.ClientTraffic, error) {
	traffics := make([]xray.ClientTraffic, 0)
	for _, inbound := range r.inbounds {
		traffics = append(traffics, inbound.ClientStats...)
	}
	return append([]xray.ClientTraffic(nil), traffics...), nil
}

func (r *fakeInboundRepository) ListClientTrafficsByEmails(emails []string) ([]xray.ClientTraffic, error) {
	r.listClientTrafficsByEmailsCalls = append(r.listClientTrafficsByEmailsCalls, append([]string(nil), emails...))
	allowed := make(map[string]struct{}, len(emails))
	for _, email := range emails {
		allowed[email] = struct{}{}
	}
	traffics := make([]xray.ClientTraffic, 0)
	for _, inbound := range r.inbounds {
		for _, clientTraffic := range inbound.ClientStats {
			if _, ok := allowed[clientTraffic.Email]; ok {
				traffics = append(traffics, clientTraffic)
			}
		}
	}
	return traffics, nil
}

func (r *fakeInboundRepository) ListClientTrafficsByEmailsTx(_ *gorm.DB, emails []string) ([]*xray.ClientTraffic, error) {
	r.listClientTrafficsByEmailsTxCalls = append(r.listClientTrafficsByEmailsTxCalls, append([]string(nil), emails...))
	allowed := make(map[string]struct{}, len(emails))
	for _, email := range emails {
		allowed[email] = struct{}{}
	}
	traffics := make([]*xray.ClientTraffic, 0)
	for _, inbound := range r.inbounds {
		for _, clientTraffic := range inbound.ClientStats {
			if _, ok := allowed[clientTraffic.Email]; ok {
				copy := clientTraffic
				traffics = append(traffics, &copy)
			}
		}
	}
	return traffics, nil
}

func (r *fakeInboundRepository) ListInboundsByIDs(_ *gorm.DB, ids []int) ([]*model.Inbound, error) {
	r.listInboundsByIDsCalls = append(r.listInboundsByIDsCalls, append([]int(nil), ids...))
	allowed := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	inbounds := make([]*model.Inbound, 0)
	for _, inbound := range r.inbounds {
		if _, ok := allowed[inbound.Id]; ok {
			inbounds = append(inbounds, copyInboundForRepositoryTest(inbound))
		}
	}
	return inbounds, nil
}

func (r *fakeInboundRepository) ListRenewableClientTraffics(_ *gorm.DB, now int64) ([]*xray.ClientTraffic, error) {
	r.listRenewableClientTrafficsCalls++
	traffics := make([]*xray.ClientTraffic, 0)
	for _, inbound := range r.inbounds {
		for _, clientTraffic := range inbound.ClientStats {
			if clientTraffic.Reset > 0 && clientTraffic.ExpiryTime > 0 && clientTraffic.ExpiryTime <= now {
				copy := clientTraffic
				traffics = append(traffics, &copy)
			}
		}
	}
	return traffics, nil
}

func (r *fakeInboundRepository) ListInvalidInboundTags(_ *gorm.DB, now int64) ([]string, error) {
	r.listInvalidInboundTagsCalls++
	tags := make([]string, 0)
	for _, inbound := range r.inbounds {
		if inbound.Enable && ((inbound.Total > 0 && inbound.Up+inbound.Down >= inbound.Total) || (inbound.ExpiryTime > 0 && inbound.ExpiryTime <= now)) {
			tags = append(tags, inbound.Tag)
		}
	}
	return tags, nil
}

func (r *fakeInboundRepository) DisableInvalidInbounds(_ *gorm.DB, now int64) (int64, error) {
	r.disableInvalidInboundsCalls++
	var count int64
	for i := range r.inbounds {
		inbound := r.inbounds[i]
		if inbound.Enable && ((inbound.Total > 0 && inbound.Up+inbound.Down >= inbound.Total) || (inbound.ExpiryTime > 0 && inbound.ExpiryTime <= now)) {
			r.inbounds[i].Enable = false
			count++
		}
	}
	return count, nil
}

func (r *fakeInboundRepository) ListInvalidClientTrafficTargets(_ *gorm.DB, now int64) ([]database.ClientTrafficTarget, error) {
	r.listInvalidClientTrafficTargetsCalls++
	targets := make([]database.ClientTrafficTarget, 0)
	for _, inbound := range r.inbounds {
		for _, clientTraffic := range inbound.ClientStats {
			if clientTraffic.Enable && ((clientTraffic.Total > 0 && clientTraffic.Up+clientTraffic.Down >= clientTraffic.Total) || (clientTraffic.ExpiryTime > 0 && clientTraffic.ExpiryTime <= now)) {
				targets = append(targets, database.ClientTrafficTarget{
					Tag:   inbound.Tag,
					Email: clientTraffic.Email,
				})
			}
		}
	}
	return targets, nil
}

func (r *fakeInboundRepository) DisableInvalidClientTraffics(_ *gorm.DB, now int64) (int64, error) {
	r.disableInvalidClientTrafficsCalls++
	var count int64
	for i := range r.inbounds {
		for j := range r.inbounds[i].ClientStats {
			clientTraffic := &r.inbounds[i].ClientStats[j]
			if clientTraffic.Enable && ((clientTraffic.Total > 0 && clientTraffic.Up+clientTraffic.Down >= clientTraffic.Total) || (clientTraffic.ExpiryTime > 0 && clientTraffic.ExpiryTime <= now)) {
				clientTraffic.Enable = false
				count++
			}
		}
	}
	return count, nil
}

func (r *fakeInboundRepository) ListClientTrafficsByClientID(id string) ([]xray.ClientTraffic, error) {
	targetEmails := make(map[string]struct{})
	for _, inbound := range r.inbounds {
		var settings struct {
			Clients []map[string]any `json:"clients"`
		}
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			continue
		}
		for _, client := range settings.Clients {
			if clientID, _ := client["id"].(string); clientID == id {
				if email, _ := client["email"].(string); email != "" {
					targetEmails[email] = struct{}{}
				}
			}
		}
	}

	traffics := make([]xray.ClientTraffic, 0)
	for _, inbound := range r.inbounds {
		for _, clientTraffic := range inbound.ClientStats {
			if _, ok := targetEmails[clientTraffic.Email]; ok {
				traffics = append(traffics, clientTraffic)
			}
		}
	}
	return traffics, nil
}

func (r *fakeInboundRepository) SearchByRemark(query string) ([]*model.Inbound, error) {
	r.searchByRemarkCalls = append(r.searchByRemarkCalls, query)
	inbounds := make([]*model.Inbound, 0)
	for _, inbound := range r.inbounds {
		if strings.Contains(inbound.Remark, query) {
			inbounds = append(inbounds, copyInboundForRepositoryTest(inbound))
		}
	}
	return inbounds, nil
}

func (r *fakeInboundRepository) ListTags() ([]string, error) {
	r.listTagsCalls++
	tags := make([]string, 0, len(r.inbounds))
	for _, inbound := range r.inbounds {
		tags = append(tags, inbound.Tag)
	}
	return tags, nil
}

func (r *fakeInboundRepository) DeleteOrphanedClientTraffics() error {
	r.deleteOrphanedClientTrafficsCalls++
	return nil
}

func (r *fakeInboundRepository) AddInboundTrafficByTag(_ *gorm.DB, tag string, upload int64, download int64) error {
	r.addInboundTrafficCalls = append(r.addInboundTrafficCalls, inboundTrafficCall{tag: tag, upload: upload, download: download})
	for i := range r.inbounds {
		if r.inbounds[i].Tag == tag {
			r.inbounds[i].Up += upload
			r.inbounds[i].Down += download
			r.inbounds[i].AllTime += upload + download
		}
	}
	return nil
}

func (r *fakeInboundRepository) CreateClientTraffic(_ *gorm.DB, clientTraffic *xray.ClientTraffic) error {
	r.createClientTrafficCalls = append(r.createClientTrafficCalls, *clientTraffic)
	copy := *clientTraffic
	for _, inbound := range r.inbounds {
		if inbound.Id == clientTraffic.InboundId {
			inbound.ClientStats = append(inbound.ClientStats, copy)
			break
		}
	}
	return nil
}

func (r *fakeInboundRepository) UpdateClientTrafficByEmail(_ *gorm.DB, email string, client *model.Client) error {
	r.updateClientTrafficCalls = append(r.updateClientTrafficCalls, updateClientTrafficCall{email: email, client: *client})
	for i := range r.inbounds {
		for j := range r.inbounds[i].ClientStats {
			if strings.EqualFold(r.inbounds[i].ClientStats[j].Email, email) {
				r.inbounds[i].ClientStats[j].Email = client.Email
				r.inbounds[i].ClientStats[j].Enable = client.Enable
				r.inbounds[i].ClientStats[j].Total = client.TotalGB
				r.inbounds[i].ClientStats[j].ExpiryTime = client.ExpiryTime
				r.inbounds[i].ClientStats[j].Reset = client.Reset
			}
		}
	}
	return nil
}

func (r *fakeInboundRepository) UpdateInboundClientIPs(_ *gorm.DB, oldEmail string, newEmail string) error {
	r.updateClientIPCalls = append(r.updateClientIPCalls, ipUpdateCall{oldEmail: oldEmail, newEmail: newEmail})
	return nil
}

func (r *fakeInboundRepository) UpdateClientTrafficUsageByEmail(email string, upload int64, download int64) error {
	r.updateTrafficUsageCalls = append(r.updateTrafficUsageCalls, usageUpdateCall{email: email, upload: upload, download: download})
	for i := range r.inbounds {
		for j := range r.inbounds[i].ClientStats {
			if strings.EqualFold(r.inbounds[i].ClientStats[j].Email, email) {
				r.inbounds[i].ClientStats[j].Up = upload
				r.inbounds[i].ClientStats[j].Down = download
			}
		}
	}
	return nil
}

func (r *fakeInboundRepository) ResetClientTrafficByEmail(email string) error {
	r.resetClientTrafficCalls = append(r.resetClientTrafficCalls, email)
	for i := range r.inbounds {
		for j := range r.inbounds[i].ClientStats {
			if strings.EqualFold(r.inbounds[i].ClientStats[j].Email, email) {
				r.inbounds[i].ClientStats[j].Enable = true
				r.inbounds[i].ClientStats[j].Up = 0
				r.inbounds[i].ClientStats[j].Down = 0
			}
		}
	}
	return nil
}

func (r *fakeInboundRepository) ResetAllClientTraffics(id int) error {
	r.resetAllClientTrafficsCalls = append(r.resetAllClientTrafficsCalls, id)
	for i := range r.inbounds {
		if id == -1 || r.inbounds[i].Id == id {
			for j := range r.inbounds[i].ClientStats {
				r.inbounds[i].ClientStats[j].Enable = true
				r.inbounds[i].ClientStats[j].Up = 0
				r.inbounds[i].ClientStats[j].Down = 0
			}
			r.inbounds[i].LastTrafficResetTime = 123456789
		}
	}
	return nil
}

func (r *fakeInboundRepository) ResetAllTraffics() error {
	r.resetAllTrafficsCalled = true
	for i := range r.inbounds {
		if r.inbounds[i].UserId > 0 {
			r.inbounds[i].Up = 0
			r.inbounds[i].Down = 0
		}
	}
	return nil
}

func (r *fakeInboundRepository) ResetInboundTraffic(id int) error {
	r.resetInboundTrafficCalls = append(r.resetInboundTrafficCalls, id)
	for i := range r.inbounds {
		if r.inbounds[i].Id == id {
			r.inbounds[i].Up = 0
			r.inbounds[i].Down = 0
		}
	}
	return nil
}

func (r *fakeInboundRepository) DeleteClientTrafficByEmail(_ *gorm.DB, email string) error {
	r.deleteClientTrafficCalls = append(r.deleteClientTrafficCalls, email)
	for i := range r.inbounds {
		filtered := r.inbounds[i].ClientStats[:0]
		for _, stat := range r.inbounds[i].ClientStats {
			if !strings.EqualFold(stat.Email, email) {
				filtered = append(filtered, stat)
			}
		}
		r.inbounds[i].ClientStats = filtered
	}
	return nil
}

func (r *fakeInboundRepository) DeleteClientTrafficsByInboundID(_ *gorm.DB, inboundID int) error {
	r.deleteClientTrafficsCalls = append(r.deleteClientTrafficsCalls, inboundID)
	for i := range r.inbounds {
		if r.inbounds[i].Id == inboundID {
			r.inbounds[i].ClientStats = nil
		}
	}
	return nil
}

func (r *fakeInboundRepository) DeleteInboundClientIPsByEmail(_ *gorm.DB, email string) error {
	r.deleteClientIPCalls = append(r.deleteClientIPCalls, email)
	return nil
}

func (r *fakeInboundRepository) FindInboundClientIps(clientEmail string) (*model.InboundClientIps, error) {
	for _, inbound := range r.inbounds {
		for _, clientTraffic := range inbound.ClientStats {
			if strings.EqualFold(clientTraffic.Email, clientEmail) {
				return &model.InboundClientIps{
					ClientEmail: clientEmail,
					Ips:         `[{"ip":"192.0.2.1","timestamp":42}]`,
				}, nil
			}
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeInboundRepository) FindInboundBySettingsContains(query string) (*model.Inbound, error) {
	for _, inbound := range r.inbounds {
		if strings.Contains(inbound.Settings, query) {
			return copyInboundForRepositoryTest(inbound), nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeInboundRepository) ClearInboundClientIps(clientEmail string) error {
	r.clearClientIPCalls = append(r.clearClientIPCalls, clientEmail)
	return nil
}

func (r *fakeInboundRepository) DeleteByID(_ *gorm.DB, id int) error {
	r.deleteByIDCalls = append(r.deleteByIDCalls, id)
	filtered := r.inbounds[:0]
	for _, inbound := range r.inbounds {
		if inbound.Id != id {
			filtered = append(filtered, inbound)
		}
	}
	r.inbounds = filtered
	return nil
}

func (r *fakeInboundRepository) GetClientTrafficByID(id int) (*xray.ClientTraffic, error) {
	for _, inbound := range r.inbounds {
		for _, clientTraffic := range inbound.ClientStats {
			if clientTraffic.Id == id {
				copy := clientTraffic
				return &copy, nil
			}
		}
	}
	return nil, nil
}

func (r *fakeInboundRepository) GetClientTrafficByEmail(email string) (*xray.ClientTraffic, error) {
	for _, inbound := range r.inbounds {
		for _, clientTraffic := range inbound.ClientStats {
			if clientTraffic.Email == email {
				copy := clientTraffic
				return &copy, nil
			}
		}
	}
	return nil, nil
}

func (r *fakeInboundRepository) Save(_ *model.Inbound) error {
	return nil
}

func (r *fakeInboundRepository) SaveInbound(_ *gorm.DB, inbound *model.Inbound) error {
	if inbound.Id == 0 {
		inbound.Id = len(r.inbounds) + 1
	}
	saved := copyInboundForRepositoryTest(inbound)
	r.saveInboundCalls = append(r.saveInboundCalls, *saved)
	for i := range r.inbounds {
		if r.inbounds[i].Id == inbound.Id {
			r.inbounds[i] = copyInboundForRepositoryTest(inbound)
			return nil
		}
	}
	r.inbounds = append(r.inbounds, copyInboundForRepositoryTest(inbound))
	return nil
}

func (r *fakeInboundRepository) SaveInbounds(tx *gorm.DB, inbounds []*model.Inbound) error {
	saved := make([]model.Inbound, 0, len(inbounds))
	for _, inbound := range inbounds {
		if err := r.SaveInbound(tx, inbound); err != nil {
			return err
		}
		saved = append(saved, *copyInboundForRepositoryTest(inbound))
	}
	r.saveInboundsCalls = append(r.saveInboundsCalls, saved)
	return nil
}

func (r *fakeInboundRepository) SaveClientTraffic(traffic *xray.ClientTraffic) error {
	r.saveClientTrafficCalls = append(r.saveClientTrafficCalls, *traffic)
	copy := *traffic
	for i := range r.inbounds {
		for j := range r.inbounds[i].ClientStats {
			if r.inbounds[i].ClientStats[j].Id == traffic.Id || strings.EqualFold(r.inbounds[i].ClientStats[j].Email, traffic.Email) {
				r.inbounds[i].ClientStats[j] = copy
			}
		}
	}
	return nil
}

func (r *fakeInboundRepository) SaveClientTraffics(_ *gorm.DB, traffics []*xray.ClientTraffic) error {
	saved := make([]xray.ClientTraffic, 0, len(traffics))
	for _, traffic := range traffics {
		if traffic == nil {
			continue
		}
		saved = append(saved, *traffic)
		if err := r.SaveClientTraffic(traffic); err != nil {
			return err
		}
	}
	r.saveClientTrafficsCalls = append(r.saveClientTrafficsCalls, saved)
	return nil
}

func copyInboundForRepositoryTest(inbound *model.Inbound) *model.Inbound {
	copied := *inbound
	if inbound.ClientStats != nil {
		copied.ClientStats = append([]xray.ClientTraffic(nil), inbound.ClientStats...)
	}
	return &copied
}

func setupInboundServiceTestDB(t *testing.T) {
	t.Helper()
	dbDir := t.TempDir()
	t.Setenv("XUI_LOG_FOLDER", filepath.Join(dbDir, "logs"))
	logger.InitLogger(logging.ERROR)
	oldProcess := p
	p = xray.NewProcess(&xray.Config{})
	if err := database.InitDB(filepath.Join(dbDir, "SuperXray.db")); err != nil {
		t.Fatalf("database.InitDB failed: %v", err)
	}
	t.Cleanup(func() {
		p = oldProcess
		if err := database.CloseDB(); err != nil {
			t.Logf("database.CloseDB warning: %v", err)
		}
		logger.CloseLogger()
	})
}

func TestInboundServiceGetClientsIgnoresNonClientSettingsFields(t *testing.T) {
	inbound := &model.Inbound{
		Settings: `{
			"clients": [
				{
					"id": "00000000-0000-4000-8000-000000000000",
					"email": "test@example",
					"enable": true,
					"tgId": 0
				}
			],
			"decryption": "none",
			"encryption": "none"
		}`,
	}

	clients, err := (&InboundService{}).GetClients(inbound)
	if err != nil {
		t.Fatalf("GetClients returned error: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("GetClients returned %d clients, want 1", len(clients))
	}
	if clients[0].Email != "test@example" {
		t.Fatalf("client email = %q, want %q", clients[0].Email, "test@example")
	}
}

func TestInboundSettingsEntryRejectsMissingClientsArray(t *testing.T) {
	_, _, err := parseInboundSettingsEntry(`{"method":"2022-blake3-aes-128-gcm"}`)
	if err == nil {
		t.Fatal("parseInboundSettingsEntry returned nil error")
	}
	if !strings.Contains(err.Error(), "clients") {
		t.Fatalf("error = %q, want mention clients", err.Error())
	}
}

func TestInboundSettingsEntryRejectsNonArrayClients(t *testing.T) {
	_, _, err := parseInboundSettingsEntry(`{"clients":"not-an-array"}`)
	if err == nil {
		t.Fatal("parseInboundSettingsEntry returned nil error")
	}
	if !strings.Contains(err.Error(), "clients") {
		t.Fatalf("error = %q, want mention clients", err.Error())
	}
}

func TestInboundSettingsEntryReturnsRawClients(t *testing.T) {
	settings, rawClients, err := parseInboundSettingsEntry(`{
		"clients": [
			{"id":"00000000-0000-4000-8000-000000000000","email":"a@example","enable":true}
		],
		"decryption": "none"
	}`)
	if err != nil {
		t.Fatalf("parseInboundSettingsEntry returned error: %v", err)
	}
	if len(rawClients) != 1 {
		t.Fatalf("rawClients length = %d, want 1", len(rawClients))
	}
	if settings["decryption"] != "none" {
		t.Fatalf("settings decryption = %v, want none", settings["decryption"])
	}
}

func TestBuildTargetClientFromSourceShadowsocksLegacyUsesInboundMethod(t *testing.T) {
	targetInbound := &model.Inbound{
		Protocol: model.Shadowsocks,
		Settings: `{"method":"chacha20-ietf-poly1305","password":"server-pass","clients":[]}`,
	}

	client, err := (&InboundService{}).buildTargetClientFromSource(
		model.Client{ID: "source-id", Password: "source-pass", Auth: "source-auth", Email: "source@example"},
		targetInbound,
		"copied@example",
		"",
	)
	if err != nil {
		t.Fatalf("buildTargetClientFromSource returned error: %v", err)
	}

	if client.Method != "chacha20-ietf-poly1305" {
		t.Fatalf("client method = %q, want inbound method", client.Method)
	}
	decodedPassword, err := base64.RawURLEncoding.DecodeString(client.Password)
	if err != nil {
		t.Fatalf("legacy Shadowsocks password is not URL-safe base64: %v", err)
	}
	if len(decodedPassword) != generatedCredentialBytes {
		t.Fatalf("legacy Shadowsocks password decoded length = %d, want %d", len(decodedPassword), generatedCredentialBytes)
	}
	if client.ID != "" || client.Auth != "" {
		t.Fatalf("client kept source credentials: id=%q auth=%q", client.ID, client.Auth)
	}
}

func TestBuildTargetClientFromSourceShadowsocks2022UsesClientKey(t *testing.T) {
	targetInbound := &model.Inbound{
		Protocol: model.Shadowsocks,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"server-key","clients":[]}`,
	}

	client, err := (&InboundService{}).buildTargetClientFromSource(
		model.Client{ID: "source-id", Password: "source-pass", Auth: "source-auth", Email: "source@example"},
		targetInbound,
		"copied@example",
		"",
	)
	if err != nil {
		t.Fatalf("buildTargetClientFromSource returned error: %v", err)
	}

	if client.Method != "" {
		t.Fatalf("2022 client method = %q, want empty", client.Method)
	}
	decoded, err := base64.StdEncoding.DecodeString(client.Password)
	if err != nil {
		t.Fatalf("2022 Shadowsocks password is not base64: %v", err)
	}
	if len(decoded) != 16 {
		t.Fatalf("2022 aes-128 client key length = %d, want 16", len(decoded))
	}
	if client.ID != "" || client.Auth != "" {
		t.Fatalf("client kept source credentials: id=%q auth=%q", client.ID, client.Auth)
	}
}

func TestBuildTargetClientFromSourceLiteralHysteria2UsesAuthCredential(t *testing.T) {
	targetInbound := &model.Inbound{
		Protocol: model.Hysteria2,
		Settings: `{"version":2,"clients":[]}`,
	}

	client, err := (&InboundService{}).buildTargetClientFromSource(
		model.Client{ID: "source-id", Password: "source-pass", Auth: "source-auth", Email: "source@example"},
		targetInbound,
		"copied@example",
		"",
	)
	if err != nil {
		t.Fatalf("buildTargetClientFromSource returned error: %v", err)
	}

	if client.Auth == "" {
		t.Fatalf("literal hysteria2 copied client auth is empty: %#v", client)
	}
	if client.ID != "" || client.Password != "" {
		t.Fatalf("literal hysteria2 copied client kept non-HY2 credentials: id=%q password=%q", client.ID, client.Password)
	}
}

func TestGetClientPrimaryKeyLiteralHysteria2UsesAuth(t *testing.T) {
	client := model.Client{ID: "id-value", Auth: "auth-value", Password: "password-value"}

	got := (&InboundService{}).getClientPrimaryKey(model.Hysteria2, client)

	if got != client.Auth {
		t.Fatalf("literal hysteria2 primary key = %q, want auth %q", got, client.Auth)
	}
}

func TestGetClientPrimaryKeyByEmailLiteralHysteria2UsesAuth(t *testing.T) {
	clients := []model.Client{
		{Email: "other@example", ID: "other-id", Auth: "other-auth", Password: "other-password"},
		{Email: "hy2@example", ID: "", Auth: "hy2-auth", Password: "hy2-password"},
	}

	got := (&InboundService{}).getClientPrimaryKeyByEmail(model.Hysteria2, clients, "hy2@example")

	if got != "hy2-auth" {
		t.Fatalf("literal hysteria2 primary key by email = %q, want auth", got)
	}
}

func TestXrayRuntimeProtocolNormalizesLiteralHysteria2(t *testing.T) {
	if got := xrayRuntimeProtocol(model.Hysteria2); got != string(model.Hysteria) {
		t.Fatalf("xrayRuntimeProtocol(hysteria2) = %q, want %q", got, model.Hysteria)
	}
	if got := xrayRuntimeProtocol(model.VLESS); got != string(model.VLESS) {
		t.Fatalf("xrayRuntimeProtocol(vless) = %q, want %q", got, model.VLESS)
	}
}

func TestNormalizeShadowsocksSettingsFillsLegacyClientMethod(t *testing.T) {
	settings, err := normalizeShadowsocksSettingsText(`{
		"method":"chacha20-ietf-poly1305",
		"password":"stale-server-password",
		"clients":[{"email":"ss@example","password":"client-password","enable":true}]
	}`)
	if err != nil {
		t.Fatalf("normalizeShadowsocksSettingsText returned error: %v", err)
	}
	var parsed struct {
		Password string         `json:"password"`
		Clients  []model.Client `json:"clients"`
	}
	if err := json.Unmarshal([]byte(settings), &parsed); err != nil {
		t.Fatalf("normalized settings are invalid JSON: %v", err)
	}
	if parsed.Password != "" {
		t.Fatalf("normalized legacy settings kept server password: %q", parsed.Password)
	}
	if len(parsed.Clients) != 1 || parsed.Clients[0].Method != "chacha20-ietf-poly1305" {
		t.Fatalf("normalized legacy client method = %#v, want chacha20-ietf-poly1305", parsed.Clients)
	}
}

func TestNormalizeShadowsocksSettingsCanonicalizesLegacyCipherAliases(t *testing.T) {
	settings, err := normalizeShadowsocksSettingsText(`{
		"method":"CHACHA20_POLY1305",
		"password":"stale-server-password",
		"clients":[{"method":"CHACHA20_POLY1305","email":"ss@example","password":"client-password","enable":true}]
	}`)
	if err != nil {
		t.Fatalf("normalizeShadowsocksSettingsText returned error: %v", err)
	}
	var parsed struct {
		Method  string         `json:"method"`
		Clients []model.Client `json:"clients"`
	}
	if err := json.Unmarshal([]byte(settings), &parsed); err != nil {
		t.Fatalf("normalized settings are invalid JSON: %v", err)
	}
	if parsed.Method != "chacha20-poly1305" {
		t.Fatalf("normalized method = %q, want chacha20-poly1305", parsed.Method)
	}
	if len(parsed.Clients) != 1 || parsed.Clients[0].Method != "chacha20-poly1305" {
		t.Fatalf("normalized legacy client method = %#v, want chacha20-poly1305", parsed.Clients)
	}
}

func TestXrayAPISyncLockSerializesCalls(t *testing.T) {
	var wg sync.WaitGroup
	var running int32
	var maxRunning int32

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			withXrayAPISyncLock(func() {
				current := atomic.AddInt32(&running, 1)
				for {
					max := atomic.LoadInt32(&maxRunning)
					if current <= max || atomic.CompareAndSwapInt32(&maxRunning, max, current) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&running, -1)
			})
		}()
	}

	wg.Wait()
	if maxRunning != 1 {
		t.Fatalf("max concurrent Xray API calls = %d, want 1", maxRunning)
	}
}
