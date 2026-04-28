package job

import (
	"encoding/json"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestClientToJSONEscapesSpecialCharacters(t *testing.T) {
	job := &LdapSyncJob{}
	got := job.clientToJSON(model.Client{
		ID:       `id"with\chars`,
		Password: `pass"with\chars`,
		Email:    `email"with\chars@example.test`,
		Enable:   true,
		LimitIP:  2,
		TotalGB:  1024,
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("clientToJSON returned invalid JSON: %v\n%s", err, got)
	}

	if parsed["email"] != `email"with\chars@example.test` {
		t.Fatalf("email = %q", parsed["email"])
	}
	if parsed["password"] != `pass"with\chars` {
		t.Fatalf("password = %q", parsed["password"])
	}
}
