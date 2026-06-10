package model

import "testing"

func TestIsHysteria(t *testing.T) {
	cases := []struct {
		in   Protocol
		want bool
	}{
		{Hysteria, true},
		{Hysteria2, true},
		{VLESS, false},
		{Shadowsocks, false},
		{Protocol(""), false},
		{Protocol("hysteria3"), false},
	}
	for _, c := range cases {
		if got := IsHysteria(c.in); got != c.want {
			t.Errorf("IsHysteria(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestGenXrayInboundConfigNormalizesLiteralHysteria2Protocol(t *testing.T) {
	inbound := &Inbound{
		Listen:         "127.0.0.1",
		Port:           443,
		Protocol:       Hysteria2,
		Settings:       `{"version":2,"clients":[{"auth":"secret","email":"hy2@example"}]}`,
		StreamSettings: `{"network":"hysteria","security":"tls"}`,
		Tag:            "literal-hy2",
		Sniffing:       `{}`,
	}

	config := inbound.GenXrayInboundConfig()

	if config.Protocol != string(Hysteria) {
		t.Fatalf("GenXrayInboundConfig protocol = %q, want %q", config.Protocol, Hysteria)
	}
}
