package sub

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
)

//go:embed rules/claude.txt
var clashClaudeRuleMatchers string

func (s *SubClashService) buildClashConfig(proxies []map[string]any) ClashConfig {
	proxyNames := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		if name, ok := proxy["name"].(string); ok && name != "" {
			proxyNames = append(proxyNames, name)
		}
	}

	// A separate selector keeps Claude on a real node even when PROXY is DIRECT.
	// Avoid colliding with subscription remarks used as proxy names.
	claudeGroup := "CLAUDE"
	for suffix := 2; slices.Contains(proxyNames, claudeGroup); suffix++ {
		claudeGroup = fmt.Sprintf("CLAUDE-%d", suffix)
	}
	generalProxyNames := append(slices.Clone(proxyNames), "DIRECT")
	config := ClashConfig{
		Proxies: proxies,
		ProxyGroups: []map[string]any{
			{"name": "PROXY", "type": "select", "proxies": generalProxyNames},
			{"name": claudeGroup, "type": "select", "proxies": proxyNames},
		},
		Rules: s.buildClashRules(),
	}
	if s.fullTemplate {
		nodeRules, nodeDomains := clashNodeAddressExceptions(proxies)
		config.Rules = s.buildFullClashRules(nodeRules)
		config.DNS = clashFullTemplateDNS(nodeDomains)
		config.RuleProviders = clashFullTemplateRuleProviders()
		config.Sniffer = clashFullTemplateSniffer()
	}

	// Apply to both templates, independently of the custom-routing switch. Domain
	// rules precede IP exceptions too, since nodes can share a CDN IP with Claude.
	config.Rules = append(clashClaudeRules(claudeGroup), config.Rules...)
	return config
}

func clashClaudeRules(group string) []string {
	var rules []string
	for _, line := range strings.Split(clashClaudeRuleMatchers, "\n") {
		matcher := strings.TrimSpace(line)
		if matcher == "" || strings.HasPrefix(matcher, "#") {
			continue
		}
		rules = append(rules, matcher+","+group, matcher+",REJECT")
	}
	// Mihomo skips a matching proxy rule when the selected node cannot carry UDP.
	// Its adjacent REJECT rule prevents falling through to a later DIRECT rule.
	return rules
}
