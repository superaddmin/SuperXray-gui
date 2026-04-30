package sub

import "github.com/superaddmin/SuperXray-gui/v2/database/model"

func subscriptionClientProtocols() []model.Protocol {
	return []model.Protocol{
		model.VMESS,
		model.VLESS,
		model.Trojan,
		model.Shadowsocks,
		model.Hysteria,
		model.Hysteria2,
	}
}

func subscriptionPeerProtocols() []model.Protocol {
	return []model.Protocol{
		model.WireGuard,
	}
}
