package model

import "testing"

func TestProtocolConstantsCoverFrontendInboundProtocols(t *testing.T) {
	if Tun != Protocol("tun") {
		t.Fatalf("Tun protocol = %q, want tun", Tun)
	}
}
