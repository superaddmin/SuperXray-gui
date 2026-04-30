package sub

import (
	"slices"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestSubscriptionProtocolMatrix(t *testing.T) {
	clientProtocols := subscriptionClientProtocols()
	for _, protocol := range []model.Protocol{
		model.VMESS,
		model.VLESS,
		model.Trojan,
		model.Shadowsocks,
		model.Hysteria,
		model.Hysteria2,
	} {
		if !slices.Contains(clientProtocols, protocol) {
			t.Fatalf("client subscription protocols missing %s", protocol)
		}
	}

	peerProtocols := subscriptionPeerProtocols()
	if !slices.Contains(peerProtocols, model.WireGuard) {
		t.Fatalf("peer subscription protocols missing %s", model.WireGuard)
	}
}
