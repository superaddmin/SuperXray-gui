package sub

import (
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestGenShadowsocksLinkLegacyDoesNotRequireServerPassword(t *testing.T) {
	inbound := &model.Inbound{
		Protocol:       model.Shadowsocks,
		Port:           443,
		Settings:       `{"method":"chacha20-ietf-poly1305","clients":[{"email":"ss@example","password":"client-password","enable":true}]}`,
		StreamSettings: `{"network":"tcp","security":"none"}`,
	}

	link := (&SubService{address: "vpn.example", remarkModel: "-ieo"}).genShadowsocksLink(inbound, "ss@example")

	if !strings.HasPrefix(link, "ss://") {
		t.Fatalf("link = %q, want ss:// prefix", link)
	}
}
