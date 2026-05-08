package sub

import "strings"

type subscriptionFormat string

const (
	subscriptionFormatURI       subscriptionFormat = "uri"
	subscriptionFormatJSON      subscriptionFormat = "json"
	subscriptionFormatClash     subscriptionFormat = "clash"
	subscriptionFormatWireGuard subscriptionFormat = "wireguard"
)

type subscriptionProfile struct {
	Name   string
	Format subscriptionFormat
}

var subscriptionProfiles = map[string]subscriptionProfile{
	"generic":      {Name: "generic", Format: subscriptionFormatURI},
	"v2rayn":       {Name: "v2rayn", Format: subscriptionFormatURI},
	"shadowrocket": {Name: "shadowrocket", Format: subscriptionFormatURI},
	"stash":        {Name: "stash", Format: subscriptionFormatClash},
	"mihomo":       {Name: "mihomo", Format: subscriptionFormatClash},
	"xray":         {Name: "xray", Format: subscriptionFormatJSON},
	"wireguard":    {Name: "wireguard", Format: subscriptionFormatWireGuard},
}

var subscriptionProfileAliases = map[string]string{
	"":             "generic",
	"default":      "generic",
	"general":      "generic",
	"clash":        "mihomo",
	"clash-meta":   "mihomo",
	"clashmeta":    "mihomo",
	"mihomo-party": "mihomo",
	"xray-core":    "xray",
	"wg":           "wireguard",
}

// subscriptionTargetProfile 将请求 target 参数归一化为内部客户端订阅 Profile。
func subscriptionTargetProfile(target string) subscriptionProfile {
	key := strings.ToLower(strings.TrimSpace(target))
	if alias, ok := subscriptionProfileAliases[key]; ok {
		key = alias
	}
	profile, ok := subscriptionProfiles[key]
	if !ok {
		return subscriptionProfiles["generic"]
	}
	return profile
}

// resolveTargetSubscriptionFormat 根据 profile 和已启用的订阅格式决定当前请求实际输出格式。
func resolveTargetSubscriptionFormat(profile subscriptionProfile, jsonEnabled bool, clashEnabled bool) subscriptionFormat {
	switch profile.Format {
	case subscriptionFormatJSON:
		if jsonEnabled {
			return subscriptionFormatJSON
		}
	case subscriptionFormatClash:
		if clashEnabled {
			return subscriptionFormatClash
		}
	}
	return subscriptionFormatURI
}
