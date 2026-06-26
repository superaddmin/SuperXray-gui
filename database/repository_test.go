package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
	"gorm.io/gorm"
)

func initRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

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
	return gdb
}

func TestRepositoriesReadAndWriteCurrentGORMModels(t *testing.T) {
	gdb := initRepositoryTestDB(t)
	repos := NewRepositories(gdb)

	user, err := repos.Users.First()
	if err != nil {
		t.Fatalf("Users.First() failed: %v", err)
	}
	if user.Username != defaultUsername {
		t.Fatalf("Users.First().Username = %q, want %q", user.Username, defaultUsername)
	}
	if err := repos.Users.UpdateCredentials(user.Id, "phase2-admin", "phase2-password"); err != nil {
		t.Fatalf("Users.UpdateCredentials() failed: %v", err)
	}
	user, err = repos.Users.FindByUsername("phase2-admin")
	if err != nil {
		t.Fatalf("Users.FindByUsername() failed: %v", err)
	}
	if user.Password != "phase2-password" {
		t.Fatalf("Users.FindByUsername().Password = %q, want %q", user.Password, "phase2-password")
	}

	if err := repos.Settings.Save("phase2Repository", "enabled"); err != nil {
		t.Fatalf("Settings.Save() create failed: %v", err)
	}
	setting, err := repos.Settings.Get("phase2Repository")
	if err != nil {
		t.Fatalf("Settings.Get() failed: %v", err)
	}
	if setting.Value != "enabled" {
		t.Fatalf("Settings.Get().Value = %q, want %q", setting.Value, "enabled")
	}

	if err := repos.Settings.Save("phase2Repository", "updated"); err != nil {
		t.Fatalf("Settings.Save() update failed: %v", err)
	}
	setting, err = repos.Settings.Get("phase2Repository")
	if err != nil {
		t.Fatalf("Settings.Get() after update failed: %v", err)
	}
	if setting.Value != "updated" {
		t.Fatalf("Settings.Get().Value after update = %q, want %q", setting.Value, "updated")
	}

	inbound := &model.Inbound{
		UserId:         user.Id,
		Remark:         "phase2-repository",
		Enable:         true,
		Port:           12345,
		Protocol:       model.VLESS,
		Settings:       `{"clients":[{"id":"phase2-client","email":"phase2@example.test"}]}`,
		StreamSettings: `{"network":"tcp","security":"tls"}`,
		Tag:            "phase2-repository",
		Sniffing:       `{}`,
		TrafficReset:   "daily",
	}
	if err := gdb.Create(inbound).Error; err != nil {
		t.Fatalf("create inbound fixture failed: %v", err)
	}
	otherInbound := &model.Inbound{
		UserId:         user.Id,
		Remark:         "phase2-repository-other",
		Enable:         true,
		Listen:         "127.0.0.1",
		Port:           12346,
		Protocol:       model.Shadowsocks,
		Settings:       `{"method":"2022-blake3-aes-128-gcm","clients":[{"id":"phase2-other-client","email":"other@example.test"}]}`,
		StreamSettings: `{"network":"tcp"}`,
		Tag:            "phase2-repository-other",
		Sniffing:       `{}`,
		TrafficReset:   "never",
	}
	if err := gdb.Create(otherInbound).Error; err != nil {
		t.Fatalf("create other inbound fixture failed: %v", err)
	}
	if err := gdb.Create(&xray.ClientTraffic{
		InboundId: inbound.Id,
		Enable:    true,
		Email:     "phase2@example.test",
	}).Error; err != nil {
		t.Fatalf("create client traffic fixture failed: %v", err)
	}
	clientTraffic := &xray.ClientTraffic{
		InboundId:  inbound.Id,
		Enable:     true,
		Email:      "phase2-lookup@example.test",
		Up:         123,
		Down:       456,
		Total:      789,
		Reset:      7,
		LastOnline: 42,
	}
	if err := gdb.Create(clientTraffic).Error; err != nil {
		t.Fatalf("create lookup client traffic fixture failed: %v", err)
	}
	if err := gdb.Create(&model.InboundClientIps{
		ClientEmail: clientTraffic.Email,
		Ips:         `[{"ip":"192.0.2.1","timestamp":42}]`,
	}).Error; err != nil {
		t.Fatalf("create inbound client ips fixture failed: %v", err)
	}

	storedInbound, err := repos.Inbounds.Get(inbound.Id)
	if err != nil {
		t.Fatalf("Inbounds.Get() failed: %v", err)
	}
	if len(storedInbound.ClientStats) != 0 {
		t.Fatalf("Inbounds.Get().ClientStats length = %d, want 0", len(storedInbound.ClientStats))
	}

	options, err := repos.Inbounds.ListOptionsByUserID(user.Id)
	if err != nil {
		t.Fatalf("Inbounds.ListOptionsByUserID() failed: %v", err)
	}
	if len(options) != 2 {
		t.Fatalf("Inbounds.ListOptionsByUserID() returned %d options, want 2", len(options))
	}
	if options[0].ID != inbound.Id || options[0].Protocol != string(model.VLESS) || !options[0].TLSFlowCapable {
		t.Fatalf("first option = %+v, want vless TLS-flow-capable inbound", options[0])
	}
	if options[1].ID != otherInbound.Id || options[1].SSMethod != "2022-blake3-aes-128-gcm" {
		t.Fatalf("second option = %+v, want shadowsocks method", options[1])
	}

	resetInbounds, err := repos.Inbounds.ListByTrafficReset("daily")
	if err != nil {
		t.Fatalf("Inbounds.ListByTrafficReset() failed: %v", err)
	}
	if len(resetInbounds) != 1 || resetInbounds[0].Id != inbound.Id {
		t.Fatalf("Inbounds.ListByTrafficReset() returned %+v, want inbound %d", resetInbounds, inbound.Id)
	}

	portExists, err := repos.Inbounds.PortExists("", inbound.Port, 0)
	if err != nil {
		t.Fatalf("Inbounds.PortExists() failed: %v", err)
	}
	if !portExists {
		t.Fatal("Inbounds.PortExists() returned false, want true for wildcard listen")
	}
	portExists, err = repos.Inbounds.PortExists("127.0.0.1", otherInbound.Port, otherInbound.Id)
	if err != nil {
		t.Fatalf("Inbounds.PortExists() with ignore failed: %v", err)
	}
	if portExists {
		t.Fatal("Inbounds.PortExists() returned true, want false when ignoring the matching inbound")
	}

	clientEmails, err := repos.Inbounds.ListClientEmails()
	if err != nil {
		t.Fatalf("Inbounds.ListClientEmails() failed: %v", err)
	}
	emailSet := make(map[string]bool, len(clientEmails))
	for _, email := range clientEmails {
		emailSet[email] = true
	}
	if !emailSet["phase2@example.test"] || !emailSet["other@example.test"] {
		t.Fatalf("Inbounds.ListClientEmails() returned %v, want phase2@example.test and other@example.test", clientEmails)
	}

	inbounds, err := repos.Inbounds.ListByUserID(user.Id)
	if err != nil {
		t.Fatalf("Inbounds.ListByUserID() failed: %v", err)
	}
	if len(inbounds) != 2 {
		t.Fatalf("Inbounds.ListByUserID() returned %d inbounds, want 2", len(inbounds))
	}
	if len(inbounds[0].ClientStats) != 2 {
		t.Fatalf("Inbounds.ListByUserID()[0].ClientStats length = %d, want 2", len(inbounds[0].ClientStats))
	}

	allInbounds, err := repos.Inbounds.ListAll()
	if err != nil {
		t.Fatalf("Inbounds.ListAll() failed: %v", err)
	}
	if len(allInbounds) != 2 {
		t.Fatalf("Inbounds.ListAll() returned %d inbounds, want 2", len(allInbounds))
	}
	if len(allInbounds[0].ClientStats) != 2 {
		t.Fatalf("Inbounds.ListAll()[0].ClientStats length = %d, want 2", len(allInbounds[0].ClientStats))
	}

	storedTraffic, err := repos.Inbounds.GetClientTrafficByID(clientTraffic.Id)
	if err != nil {
		t.Fatalf("Inbounds.GetClientTrafficByID() failed: %v", err)
	}
	if storedTraffic.Email != clientTraffic.Email || storedTraffic.InboundId != inbound.Id {
		t.Fatalf("Inbounds.GetClientTrafficByID() returned %+v, want %+v", storedTraffic, clientTraffic)
	}

	storedTraffic, err = repos.Inbounds.GetClientTrafficByEmail(clientTraffic.Email)
	if err != nil {
		t.Fatalf("Inbounds.GetClientTrafficByEmail() failed: %v", err)
	}
	if storedTraffic.Id != clientTraffic.Id || storedTraffic.Total != clientTraffic.Total {
		t.Fatalf("Inbounds.GetClientTrafficByEmail() returned %+v, want %+v", storedTraffic, clientTraffic)
	}

	if err := repos.Inbounds.UpdateClientTrafficUsageByEmail(clientTraffic.Email, 777, 888); err != nil {
		t.Fatalf("Inbounds.UpdateClientTrafficUsageByEmail() failed: %v", err)
	}
	storedTraffic, err = repos.Inbounds.GetClientTrafficByEmail(clientTraffic.Email)
	if err != nil {
		t.Fatalf("Inbounds.GetClientTrafficByEmail() after usage update failed: %v", err)
	}
	if storedTraffic.Up != 777 || storedTraffic.Down != 888 {
		t.Fatalf("Inbounds.UpdateClientTrafficUsageByEmail() persisted %+v, want up/down 777/888", storedTraffic)
	}

	clientIPs, err := repos.Inbounds.FindInboundClientIps(clientTraffic.Email)
	if err != nil {
		t.Fatalf("Inbounds.FindInboundClientIps() failed: %v", err)
	}
	if clientIPs == nil || clientIPs.Ips != `[{"ip":"192.0.2.1","timestamp":42}]` {
		t.Fatalf("Inbounds.FindInboundClientIps() returned %+v, want stored IP payload", clientIPs)
	}
	if err := repos.Inbounds.ClearInboundClientIps(clientTraffic.Email); err != nil {
		t.Fatalf("Inbounds.ClearInboundClientIps() failed: %v", err)
	}
	clientIPs, err = repos.Inbounds.FindInboundClientIps(clientTraffic.Email)
	if err != nil {
		t.Fatalf("Inbounds.FindInboundClientIps() after clear failed: %v", err)
	}
	if clientIPs.Ips != "" {
		t.Fatalf("Inbounds.ClearInboundClientIps() left ips=%q, want empty string", clientIPs.Ips)
	}

	lastOnlineRows, err := repos.Inbounds.ListClientTrafficsLastOnline()
	if err != nil {
		t.Fatalf("Inbounds.ListClientTrafficsLastOnline() failed: %v", err)
	}
	if len(lastOnlineRows) < 2 {
		t.Fatalf("Inbounds.ListClientTrafficsLastOnline() returned %d rows, want at least 2", len(lastOnlineRows))
	}
	lastOnlineByEmail := make(map[string]int64, len(lastOnlineRows))
	for _, row := range lastOnlineRows {
		lastOnlineByEmail[row.Email] = row.LastOnline
	}
	if lastOnlineByEmail[clientTraffic.Email] != 42 {
		t.Fatalf("Inbounds.ListClientTrafficsLastOnline()[%q] = %d, want 42", clientTraffic.Email, lastOnlineByEmail[clientTraffic.Email])
	}

	filteredRows, err := repos.Inbounds.ListClientTrafficsByEmails([]string{clientTraffic.Email, "phase2@example.test", "missing@example.test"})
	if err != nil {
		t.Fatalf("Inbounds.ListClientTrafficsByEmails() failed: %v", err)
	}
	if len(filteredRows) != 2 {
		t.Fatalf("Inbounds.ListClientTrafficsByEmails() returned %d rows, want 2", len(filteredRows))
	}
	filteredEmailSet := make(map[string]bool, len(filteredRows))
	for _, row := range filteredRows {
		filteredEmailSet[row.Email] = true
	}
	if !filteredEmailSet[clientTraffic.Email] || !filteredEmailSet["phase2@example.test"] {
		t.Fatalf("Inbounds.ListClientTrafficsByEmails() returned %v, want matching emails", filteredRows)
	}

	searchInbound := &model.Inbound{
		UserId:         999,
		Remark:         "phase2-search-target",
		Enable:         true,
		Port:           12347,
		Protocol:       model.VLESS,
		Settings:       `{"clients":[{"id":"search-client","email":"search@example.test"}]}`,
		StreamSettings: `{"network":"tcp","security":"tls"}`,
		Tag:            "phase2-search-target",
		Sniffing:       `{}`,
		TrafficReset:   "never",
	}
	if err := gdb.Create(searchInbound).Error; err != nil {
		t.Fatalf("create search inbound fixture failed: %v", err)
	}
	if err := gdb.Create(&xray.ClientTraffic{
		InboundId:  searchInbound.Id,
		Enable:     true,
		Email:      "search@example.test",
		Up:         999,
		Down:       1,
		LastOnline: 99,
	}).Error; err != nil {
		t.Fatalf("create search client traffic fixture failed: %v", err)
	}

	searchResults, err := repos.Inbounds.SearchByRemark("search-target")
	if err != nil {
		t.Fatalf("Inbounds.SearchByRemark() failed: %v", err)
	}
	if len(searchResults) != 1 || searchResults[0].Id != searchInbound.Id {
		t.Fatalf("Inbounds.SearchByRemark() returned %+v, want inbound %d", searchResults, searchInbound.Id)
	}
	if len(searchResults[0].ClientStats) != 1 || searchResults[0].ClientStats[0].Email != "search@example.test" {
		t.Fatalf("Inbounds.SearchByRemark() did not preload client stats: %+v", searchResults[0].ClientStats)
	}

	tx := gdb.Begin()
	if err := repos.Inbounds.AddInboundTrafficByTag(tx, inbound.Tag, 10, 20); err != nil {
		t.Fatalf("Inbounds.AddInboundTrafficByTag() failed: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit AddInboundTrafficByTag transaction failed: %v", err)
	}
	storedInbound, err = repos.Inbounds.Get(inbound.Id)
	if err != nil {
		t.Fatalf("Inbounds.Get() after AddInboundTrafficByTag failed: %v", err)
	}
	if storedInbound.Up != 10 || storedInbound.Down != 20 || storedInbound.AllTime != 30 {
		t.Fatalf("Inbounds.AddInboundTrafficByTag() stored up/down/all_time = %d/%d/%d, want 10/20/30", storedInbound.Up, storedInbound.Down, storedInbound.AllTime)
	}

	tx = gdb.Begin()
	txTraffics, err := repos.Inbounds.ListClientTrafficsByEmailsTx(tx, []string{clientTraffic.Email, "phase2@example.test", "missing@example.test"})
	if err != nil {
		t.Fatalf("Inbounds.ListClientTrafficsByEmailsTx() failed: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback ListClientTrafficsByEmailsTx transaction failed: %v", err)
	}
	if len(txTraffics) != 2 {
		t.Fatalf("Inbounds.ListClientTrafficsByEmailsTx() returned %d rows, want 2", len(txTraffics))
	}
	txTrafficEmailSet := make(map[string]bool, len(txTraffics))
	for _, traffic := range txTraffics {
		txTrafficEmailSet[traffic.Email] = true
	}
	if !txTrafficEmailSet[clientTraffic.Email] || !txTrafficEmailSet["phase2@example.test"] {
		t.Fatalf("Inbounds.ListClientTrafficsByEmailsTx() returned %+v, want matching emails", txTraffics)
	}

	tx = gdb.Begin()
	txInbounds, err := repos.Inbounds.ListInboundsByIDs(tx, []int{inbound.Id, otherInbound.Id, 999999})
	if err != nil {
		t.Fatalf("Inbounds.ListInboundsByIDs() failed: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback ListInboundsByIDs transaction failed: %v", err)
	}
	if len(txInbounds) != 2 {
		t.Fatalf("Inbounds.ListInboundsByIDs() returned %d rows, want 2", len(txInbounds))
	}
	txInboundIDSet := make(map[int]bool, len(txInbounds))
	for _, row := range txInbounds {
		txInboundIDSet[row.Id] = true
	}
	if !txInboundIDSet[inbound.Id] || !txInboundIDSet[otherInbound.Id] {
		t.Fatalf("Inbounds.ListInboundsByIDs() returned %+v, want both fixture inbounds", txInbounds)
	}

	tx = gdb.Begin()
	enableRows, err := repos.Inbounds.ListClientTrafficEnableByInboundID(tx, inbound.Id)
	if err != nil {
		t.Fatalf("Inbounds.ListClientTrafficEnableByInboundID() failed: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback ListClientTrafficEnableByInboundID transaction failed: %v", err)
	}
	enableByEmail := make(map[string]bool, len(enableRows))
	for _, row := range enableRows {
		enableByEmail[row.Email] = row.Enable
	}
	if _, ok := enableByEmail["phase2@example.test"]; !ok {
		t.Fatalf("Inbounds.ListClientTrafficEnableByInboundID() returned %+v, want phase2@example.test", enableRows)
	}
	if _, ok := enableByEmail[clientTraffic.Email]; !ok {
		t.Fatalf("Inbounds.ListClientTrafficEnableByInboundID() returned %+v, want %s", enableRows, clientTraffic.Email)
	}
	trafficEnabled, err := repos.Inbounds.IsClientTrafficEnabledByEmail(gdb, "phase2@example.test")
	if err != nil {
		t.Fatalf("Inbounds.IsClientTrafficEnabledByEmail() failed: %v", err)
	}
	if !trafficEnabled {
		t.Fatal("Inbounds.IsClientTrafficEnabledByEmail() returned false, want true")
	}

	maintenanceNow := time.Now().Unix() * 1000
	depletedInbound := &model.Inbound{
		UserId:         user.Id,
		Up:             70,
		Down:           40,
		Total:          100,
		Remark:         "phase2-invalid-inbound",
		Enable:         true,
		Port:           12348,
		Protocol:       model.VLESS,
		Settings:       `{"clients":[{"id":"phase2-depleted-client","email":"phase2-depleted@example.test"}]}`,
		StreamSettings: `{"network":"tcp","security":"tls"}`,
		Tag:            "phase2-invalid-inbound",
		Sniffing:       `{}`,
		TrafficReset:   "never",
	}
	if err := gdb.Create(depletedInbound).Error; err != nil {
		t.Fatalf("create depleted inbound fixture failed: %v", err)
	}
	depletedTraffic := &xray.ClientTraffic{
		InboundId: depletedInbound.Id,
		Enable:    true,
		Email:     "phase2-depleted@example.test",
		Up:        80,
		Down:      30,
		Total:     100,
	}
	if err := gdb.Create(depletedTraffic).Error; err != nil {
		t.Fatalf("create depleted client traffic fixture failed: %v", err)
	}
	renewableTraffic := &xray.ClientTraffic{
		InboundId:  otherInbound.Id,
		Enable:     false,
		Email:      "phase2-renew@example.test",
		Up:         10,
		Down:       11,
		Total:      1000,
		Reset:      1,
		ExpiryTime: maintenanceNow - 1000,
	}
	if err := gdb.Create(renewableTraffic).Error; err != nil {
		t.Fatalf("create renewable client traffic fixture failed: %v", err)
	}

	tx = gdb.Begin()
	depletedGroups, err := repos.Inbounds.ListDepletedClientGroups(tx, -1, maintenanceNow)
	if err != nil {
		t.Fatalf("Inbounds.ListDepletedClientGroups() failed: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback ListDepletedClientGroups transaction failed: %v", err)
	}
	foundDepletedGroup := false
	for _, group := range depletedGroups {
		for _, email := range group.Emails {
			if group.InboundID == depletedInbound.Id && email == depletedTraffic.Email {
				foundDepletedGroup = true
				break
			}
		}
	}
	if !foundDepletedGroup {
		t.Fatalf("Inbounds.ListDepletedClientGroups() returned %+v, want inbound %d email %s", depletedGroups, depletedInbound.Id, depletedTraffic.Email)
	}

	tx = gdb.Begin()
	renewableTraffics, err := repos.Inbounds.ListRenewableClientTraffics(tx, maintenanceNow)
	if err != nil {
		t.Fatalf("Inbounds.ListRenewableClientTraffics() failed: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback ListRenewableClientTraffics transaction failed: %v", err)
	}
	if len(renewableTraffics) != 1 || renewableTraffics[0].Email != renewableTraffic.Email {
		t.Fatalf("Inbounds.ListRenewableClientTraffics() returned %+v, want %s", renewableTraffics, renewableTraffic.Email)
	}

	tx = gdb.Begin()
	invalidInboundTags, err := repos.Inbounds.ListInvalidInboundTags(tx, maintenanceNow)
	if err != nil {
		t.Fatalf("Inbounds.ListInvalidInboundTags() failed: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback ListInvalidInboundTags transaction failed: %v", err)
	}
	invalidInboundTagSet := make(map[string]bool, len(invalidInboundTags))
	for _, tag := range invalidInboundTags {
		invalidInboundTagSet[tag] = true
	}
	if !invalidInboundTagSet[depletedInbound.Tag] {
		t.Fatalf("Inbounds.ListInvalidInboundTags() returned %v, want %s", invalidInboundTags, depletedInbound.Tag)
	}

	tx = gdb.Begin()
	disabledInboundCount, err := repos.Inbounds.DisableInvalidInbounds(tx, maintenanceNow)
	if err != nil {
		t.Fatalf("Inbounds.DisableInvalidInbounds() failed: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit DisableInvalidInbounds transaction failed: %v", err)
	}
	if disabledInboundCount != 1 {
		t.Fatalf("Inbounds.DisableInvalidInbounds() count = %d, want 1", disabledInboundCount)
	}
	storedDepletedInbound, err := repos.Inbounds.Get(depletedInbound.Id)
	if err != nil {
		t.Fatalf("Inbounds.Get() after DisableInvalidInbounds failed: %v", err)
	}
	if storedDepletedInbound.Enable {
		t.Fatal("Inbounds.DisableInvalidInbounds() left depleted inbound enabled")
	}

	tx = gdb.Begin()
	invalidClientTargets, err := repos.Inbounds.ListInvalidClientTrafficTargets(tx, maintenanceNow)
	if err != nil {
		t.Fatalf("Inbounds.ListInvalidClientTrafficTargets() failed: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback ListInvalidClientTrafficTargets transaction failed: %v", err)
	}
	foundDepletedTarget := false
	for _, target := range invalidClientTargets {
		if target.Tag == depletedInbound.Tag && target.Email == depletedTraffic.Email {
			foundDepletedTarget = true
			break
		}
	}
	if !foundDepletedTarget {
		t.Fatalf("Inbounds.ListInvalidClientTrafficTargets() returned %+v, want %s/%s", invalidClientTargets, depletedInbound.Tag, depletedTraffic.Email)
	}

	tx = gdb.Begin()
	disabledClientCount, err := repos.Inbounds.DisableInvalidClientTraffics(tx, maintenanceNow)
	if err != nil {
		t.Fatalf("Inbounds.DisableInvalidClientTraffics() failed: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit DisableInvalidClientTraffics transaction failed: %v", err)
	}
	if disabledClientCount < 1 {
		t.Fatalf("Inbounds.DisableInvalidClientTraffics() count = %d, want at least 1", disabledClientCount)
	}
	storedDepletedTraffic, err := repos.Inbounds.GetClientTrafficByEmail(depletedTraffic.Email)
	if err != nil {
		t.Fatalf("Inbounds.GetClientTrafficByEmail() after DisableInvalidClientTraffics failed: %v", err)
	}
	if storedDepletedTraffic.Enable {
		t.Fatal("Inbounds.DisableInvalidClientTraffics() left depleted client traffic enabled")
	}

	tx = gdb.Begin()
	if err := repos.Inbounds.DeleteDepletedClientTraffics(tx, -1, maintenanceNow); err != nil {
		t.Fatalf("Inbounds.DeleteDepletedClientTraffics() failed: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit DeleteDepletedClientTraffics transaction failed: %v", err)
	}
	storedDepletedTraffic, err = repos.Inbounds.GetClientTrafficByEmail(depletedTraffic.Email)
	if err != nil {
		t.Fatalf("Inbounds.GetClientTrafficByEmail() after DeleteDepletedClientTraffics failed: %v", err)
	}
	if storedDepletedTraffic != nil {
		t.Fatalf("Inbounds.DeleteDepletedClientTraffics() left traffic %+v, want nil", storedDepletedTraffic)
	}

	tx = gdb.Begin()
	inbound.Remark = "phase2-save-inbound"
	if err := repos.Inbounds.SaveInbound(tx, inbound); err != nil {
		t.Fatalf("Inbounds.SaveInbound() failed: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit SaveInbound transaction failed: %v", err)
	}
	storedInbound, err = repos.Inbounds.Get(inbound.Id)
	if err != nil {
		t.Fatalf("Inbounds.Get() after SaveInbound failed: %v", err)
	}
	if storedInbound.Remark != "phase2-save-inbound" {
		t.Fatalf("Inbounds.SaveInbound() stored remark = %q, want phase2-save-inbound", storedInbound.Remark)
	}

	tx = gdb.Begin()
	inbound.Remark = "phase2-save-inbounds-one"
	otherInbound.Remark = "phase2-save-inbounds-two"
	if err := repos.Inbounds.SaveInbounds(tx, []*model.Inbound{inbound, otherInbound}); err != nil {
		t.Fatalf("Inbounds.SaveInbounds() failed: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit SaveInbounds transaction failed: %v", err)
	}
	storedInbound, err = repos.Inbounds.Get(inbound.Id)
	if err != nil {
		t.Fatalf("Inbounds.Get() after SaveInbounds failed: %v", err)
	}
	if storedInbound.Remark != "phase2-save-inbounds-one" {
		t.Fatalf("Inbounds.SaveInbounds() first remark = %q, want phase2-save-inbounds-one", storedInbound.Remark)
	}
	storedOtherInbound, err := repos.Inbounds.Get(otherInbound.Id)
	if err != nil {
		t.Fatalf("Inbounds.Get() other after SaveInbounds failed: %v", err)
	}
	if storedOtherInbound.Remark != "phase2-save-inbounds-two" {
		t.Fatalf("Inbounds.SaveInbounds() second remark = %q, want phase2-save-inbounds-two", storedOtherInbound.Remark)
	}

	tx = gdb.Begin()
	clientTraffic.Up = 1111
	clientTraffic.Down = 2222
	if err := repos.Inbounds.SaveClientTraffics(tx, []*xray.ClientTraffic{clientTraffic}); err != nil {
		t.Fatalf("Inbounds.SaveClientTraffics() failed: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit SaveClientTraffics transaction failed: %v", err)
	}
	storedTraffic, err = repos.Inbounds.GetClientTrafficByEmail(clientTraffic.Email)
	if err != nil {
		t.Fatalf("Inbounds.GetClientTrafficByEmail() after SaveClientTraffics failed: %v", err)
	}
	if storedTraffic.Up != 1111 || storedTraffic.Down != 2222 {
		t.Fatalf("Inbounds.SaveClientTraffics() stored %+v, want up/down 1111/2222", storedTraffic)
	}
}
