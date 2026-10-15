package job

import "testing"

func TestLDAPConfiguredValueParsing(t *testing.T) {
	if got := truthyValuesOrDefault(""); len(got) != len(DefaultTruthyValues) {
		t.Fatalf("default truthy values = %#v", got)
	}
	if got := splitConfiguredTags(""); got != nil {
		t.Fatalf("empty inbound tag setting = %#v, want nil", got)
	}
	got := splitConfiguredTags(" inbound-a, ,inbound-b ")
	if len(got) != 2 || got[0] != "inbound-a" || got[1] != "inbound-b" {
		t.Fatalf("configured tags = %#v", got)
	}
}
