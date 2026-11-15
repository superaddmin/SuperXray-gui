package sub

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestClashClaudeRulesPrecedeCustomAndProviderRoutes(t *testing.T) {
	for _, full := range []bool{false, true} {
		for _, routing := range []bool{false, true} {
			t.Run(fmt.Sprintf("full=%t/routing=%t", full, routing), func(t *testing.T) {
				service := NewSubClashService(&SubService{}).
					WithFullTemplate(full).
					WithRoutingSettings(routing, "DOMAIN-SUFFIX,anthropic.com,DIRECT\nDOMAIN-SUFFIX,example.com,DIRECT\nMATCH,DIRECT")
				config := service.buildClashConfig([]map[string]any{
					{"name": "node-a", "type": "http", "server": "node.example.com", "port": 443},
					{"name": "node-b", "type": "socks5", "server": "203.0.113.2", "port": 443, "udp": true},
				})
				assertClashClaudePolicy(t, config, "CLAUDE", []string{"node-a", "node-b"})
				if got := slices.Contains(config.Rules, "DOMAIN-SUFFIX,example.com,DIRECT"); got != routing {
					t.Fatalf("custom route present = %t, want %t", got, routing)
				}
				if got := config.DNS != nil && config.RuleProviders != nil && config.Sniffer != nil; got != full {
					t.Fatalf("full template sections present = %t, want %t", got, full)
				}
				if full && !slices.Contains(config.Rules, "DOMAIN,node.example.com,DIRECT") {
					t.Fatal("full template lost node reachability exception")
				}
				if config.Rules[len(config.Rules)-1] != "MATCH,PROXY" {
					t.Fatal("general traffic lost its existing fallback")
				}
			})
		}
	}
}

func TestClashClaudeGroupAvoidsProxyNameCollisions(t *testing.T) {
	config := NewSubClashService(&SubService{}).buildClashConfig([]map[string]any{
		{"name": "CLAUDE", "type": "http", "server": "one.example.com", "port": 443},
		{"name": "CLAUDE-2", "type": "http", "server": "two.example.com", "port": 443},
	})
	assertClashClaudePolicy(t, config, "CLAUDE-3", []string{"CLAUDE", "CLAUDE-2"})
}

func TestGetClashIncludesClaudePolicyInBothTemplates(t *testing.T) {
	setupSubscriptionDiagnosticDB(t)
	const subID = "claude-policy-test-sub"
	client := matrixClient("claude-policy@example")
	client.SubID = subID
	inbound := matrixInbound(model.VLESS, 13209, client)
	inbound.Listen = "node.example.com"
	inbound.Enable = true
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	for _, full := range []bool{false, true} {
		t.Run(fmt.Sprintf("full=%t", full), func(t *testing.T) {
			service := NewSubClashService(&SubService{remarkModel: "-ieo"}).
				WithFullTemplate(full).
				WithRoutingRules("MATCH,DIRECT")
			text, _, err := service.GetClash(subID, "node.example.com")
			if err != nil {
				t.Fatalf("GetClash: %v", err)
			}
			var config ClashConfig
			if err := yaml.Unmarshal([]byte(text), &config); err != nil {
				t.Fatalf("decode generated YAML: %v", err)
			}
			if len(config.Proxies) != 1 {
				t.Fatalf("generated %d proxies, want 1", len(config.Proxies))
			}
			name, _ := config.Proxies[0]["name"].(string)
			assertClashClaudePolicy(t, config, "CLAUDE", []string{name})
		})
	}
}

func assertClashClaudePolicy(t *testing.T, config ClashConfig, groupName string, nodes []string) {
	t.Helper()
	// Literal expectations intentionally include apex suffixes and only the
	// confirmed telemetry hosts, so a shared-vendor wildcard cannot slip in.
	matchers := []string{
		"DOMAIN-SUFFIX,claude.ai",
		"DOMAIN-SUFFIX,claude.com",
		"DOMAIN-SUFFIX,anthropic.com",
		"DOMAIN-SUFFIX,claudeusercontent.com",
		"DOMAIN,o1158394.ingest.us.sentry.io",
		"DOMAIN,http-intake.logs.us5.datadoghq.com",
		"DOMAIN,browser-intake-us5-datadoghq.com",
	}
	if len(config.Rules) < 2*len(matchers) {
		t.Fatalf("missing fixed Claude policy: %v", config.Rules)
	}
	for i, matcher := range matchers {
		if config.Rules[2*i] != matcher+","+groupName || config.Rules[2*i+1] != matcher+",REJECT" {
			t.Fatalf("matcher %s must route to %s then reject unsupported UDP, got %v", matcher, groupName, config.Rules[2*i:2*i+2])
		}
	}
	groups := make(map[string]map[string]any)
	for _, group := range config.ProxyGroups {
		name, _ := group["name"].(string)
		if _, duplicate := groups[name]; duplicate {
			t.Fatalf("duplicate proxy group %q", name)
		}
		groups[name] = group
	}
	claude, ok := groups[groupName]
	if !ok || claude["type"] != "select" {
		t.Fatalf("missing Claude selector %q: %v", groupName, config.ProxyGroups)
	}
	// Marshal/unmarshal also exercises the serialized membership contract.
	data, err := yaml.Marshal(claude["proxies"])
	if err != nil {
		t.Fatal(err)
	}
	var members []string
	if err := yaml.Unmarshal(data, &members); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(members, nodes) || slices.Contains(members, "DIRECT") || slices.Contains(members, "PROXY") {
		t.Fatalf("Claude selector members = %v, want only nodes %v", members, nodes)
	}
	for _, proxy := range config.Proxies {
		if proxy["name"] == groupName {
			t.Fatalf("Claude group collides with proxy %q", groupName)
		}
	}
	if data, err := yaml.Marshal(groups["PROXY"]["proxies"]); err != nil || !strings.Contains(string(data), "DIRECT") {
		t.Fatal("general PROXY selector must retain its existing DIRECT option")
	}
}
