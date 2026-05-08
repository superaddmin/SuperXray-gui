package sub

import "testing"

func TestSubscriptionTargetProfileDefaultsToGeneric(t *testing.T) {
	profile := subscriptionTargetProfile("")
	if profile.Name != "generic" {
		t.Fatalf("profile name = %q, want generic", profile.Name)
	}
	if profile.Format != subscriptionFormatURI {
		t.Fatalf("profile format = %q, want uri", profile.Format)
	}
}

func TestSubscriptionTargetProfileNormalizesKnownTargets(t *testing.T) {
	tests := []struct {
		input  string
		name   string
		format subscriptionFormat
	}{
		{input: "v2rayN", name: "v2rayn", format: subscriptionFormatURI},
		{input: "shadowrocket", name: "shadowrocket", format: subscriptionFormatURI},
		{input: "stash", name: "stash", format: subscriptionFormatClash},
		{input: "mihomo", name: "mihomo", format: subscriptionFormatClash},
		{input: "clash-meta", name: "mihomo", format: subscriptionFormatClash},
		{input: "xray", name: "xray", format: subscriptionFormatJSON},
		{input: "wireguard", name: "wireguard", format: subscriptionFormatWireGuard},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			profile := subscriptionTargetProfile(tt.input)
			if profile.Name != tt.name || profile.Format != tt.format {
				t.Fatalf("profile = %#v, want name=%q format=%q", profile, tt.name, tt.format)
			}
		})
	}
}

func TestSubscriptionTargetProfileFallsBackForUnknownTarget(t *testing.T) {
	profile := subscriptionTargetProfile("unknown-client")
	if profile.Name != "generic" || profile.Format != subscriptionFormatURI {
		t.Fatalf("profile = %#v, want generic uri", profile)
	}
}

func TestResolveTargetSubscriptionFormat(t *testing.T) {
	tests := []struct {
		name         string
		target       string
		jsonEnabled  bool
		clashEnabled bool
		want         subscriptionFormat
	}{
		{name: "default uri", target: "", jsonEnabled: true, clashEnabled: true, want: subscriptionFormatURI},
		{name: "xray to json", target: "xray", jsonEnabled: true, clashEnabled: true, want: subscriptionFormatJSON},
		{name: "mihomo to clash", target: "mihomo", jsonEnabled: true, clashEnabled: true, want: subscriptionFormatClash},
		{name: "stash to clash", target: "stash", jsonEnabled: true, clashEnabled: true, want: subscriptionFormatClash},
		{name: "json disabled fallback", target: "xray", jsonEnabled: false, clashEnabled: true, want: subscriptionFormatURI},
		{name: "clash disabled fallback", target: "mihomo", jsonEnabled: true, clashEnabled: false, want: subscriptionFormatURI},
		{name: "wireguard keeps uri entry", target: "wireguard", jsonEnabled: true, clashEnabled: true, want: subscriptionFormatURI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := subscriptionTargetProfile(tt.target)
			if got := resolveTargetSubscriptionFormat(profile, tt.jsonEnabled, tt.clashEnabled); got != tt.want {
				t.Fatalf("target format = %q, want %q", got, tt.want)
			}
		})
	}
}
