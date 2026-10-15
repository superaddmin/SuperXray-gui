package sub

import (
	"strings"
	"testing"
)

func TestMarshalClashYAMLQuotesAmbiguousStrings(t *testing.T) {
	data, err := marshalClashYAML(map[string]any{
		"proxies": []map[string]any{{"name": "2351e1", "password": "1234"}},
	})
	if err != nil {
		t.Fatalf("marshal clash yaml: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "'2351e1'") || !strings.Contains(text, "'1234'") {
		t.Fatalf("ambiguous strings were not quoted: %s", text)
	}
}

func TestMarshalClashYAMLLeavesOrdinaryStringsUnchanged(t *testing.T) {
	data, err := marshalClashYAML(map[string]any{"name": "example.test"})
	if err != nil {
		t.Fatalf("marshal clash yaml: %v", err)
	}
	if strings.Contains(string(data), "'example.test'") {
		t.Fatalf("ordinary string was unnecessarily quoted: %s", data)
	}
}
