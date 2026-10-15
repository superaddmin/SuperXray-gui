package xray

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTemplateInboundExtensionsRoundTripAndTriggerReload(t *testing.T) {
	raw := `{"inbounds":[{"port":12345,"protocol":"socks","settings":{},"vendor":{"large":9007199254740993},"allocate":{"strategy":"always"}}]}`
	var config Config
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(config)
	if err != nil || !strings.Contains(string(encoded), "9007199254740993") || !strings.Contains(string(encoded), `"allocate":{"strategy":"always"}`) {
		t.Fatalf("template inbound lost extension data: %s, %v", encoded, err)
	}
	var other Config
	if err := json.Unmarshal(encoded, &other); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &config); err != nil || !config.Equals(&other) {
		t.Fatalf("canonical round trip changed config: %v", err)
	}
	other.InboundConfigs[0].Extra["vendor"] = json.RawMessage(`{"large":9007199254740994}`)
	if config.Equals(&other) {
		t.Fatal("changed inbound extension would skip runtime reload")
	}
}
