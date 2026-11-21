package job

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	ldaputil "github.com/superaddmin/SuperXray-gui/v2/util/ldap"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

func TestLDAPAutoDeletePreservesClientsThroughPartialAndChangedDirectories(t *testing.T) {
	if err := database.InitDB(filepath.Join(t.TempDir(), "panel.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := database.CloseDB(); err != nil {
			t.Error(err)
		}
	})
	db := database.GetDB()
	settings := database.NewRepositories(db).Settings
	for key, value := range map[string]string{"ldapEnable": "true", "ldapAutoDelete": "true", "ldapInboundTags": "directory", "ldapAutoCreate": "false"} {
		if err := settings.Save(key, value); err != nil {
			t.Fatal(err)
		}
	}
	initial := []string{"alice", "ben", "carol", "dan", "erin"}
	clients := make([]model.Client, 0, len(initial))
	for _, email := range initial {
		clients = append(clients, model.Client{Email: email, ID: "fixture-" + email, Enable: true})
	}
	payload, err := json.Marshal(map[string]any{"clients": clients, "decryption": "none"})
	if err != nil {
		t.Fatal(err)
	}
	inbound := model.Inbound{Tag: "directory", Protocol: model.VLESS, Settings: string(payload)}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	for _, email := range initial {
		if err := db.Create(&xray.ClientTraffic{InboundId: inbound.Id, Email: email}).Error; err != nil {
			t.Fatal(err)
		}
	}
	job := NewLdapSyncJob()
	observed := func(emails []string) map[string]bool {
		flags := make(map[string]bool, len(emails))
		for _, email := range emails {
			flags[email] = true
		}
		return flags
	}
	partial := initial[:2]
	replacement := []string{"fern", "greg", "hana", "ian", "jia"}
	for _, step := range []struct {
		label           string
		directory, want []string
		confirmed       bool
	}{
		{"cold empty", nil, initial, false},
		{"cold partial", partial, initial, false},
		{"cold replacement", replacement, initial, false},
		{"trusted baseline", initial, initial, true},
		{"first partial", partial, initial, true},
		{"second partial", partial, initial, true},
		{"same size but other emails", replacement, initial, true},
		{"restored with one departed user", initial[:4], initial[:4], true},
	} {
		job.run(func(ldaputil.Config) (map[string]bool, error) {
			return observed(step.directory), nil
		})
		got, err := job.inboundService.GetInbound(inbound.Id)
		if err != nil {
			t.Fatal(err)
		}
		stored, err := job.inboundService.GetClients(got)
		if err != nil {
			t.Fatal(err)
		}
		var emails []string
		for _, client := range stored {
			emails = append(emails, client.Email)
		}
		var stats []string
		if err := db.Model(&xray.ClientTraffic{}).Where("inbound_id = ?", inbound.Id).Order("email").Pluck("email", &stats).Error; err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(emails, step.want) || !slices.Equal(stats, step.want) || job.directoryConfirmed != step.confirmed || !maps.Equal(job.referenceEmails, observed(step.want)) {
			t.Fatalf("%s: clients=%v stats=%v reference=%v confirmed=%t", step.label, emails, stats, job.referenceEmails, job.directoryConfirmed)
		}
	}
}
