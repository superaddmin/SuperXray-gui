package entity

import "testing"

func TestAllSettingCheckValidAcceptsEmptyPanelProxy(t *testing.T) {
	settings := validAllSettingForTest()
	settings.PanelProxy = ""

	if err := settings.CheckValid(); err != nil {
		t.Fatalf("CheckValid rejected empty PanelProxy: %v", err)
	}
}

func TestAllSettingCheckValidRejectsInvalidPanelProxy(t *testing.T) {
	for _, proxyURL := range []string{"ftp://127.0.0.1:21", "http://", "socks5://"} {
		t.Run(proxyURL, func(t *testing.T) {
			settings := validAllSettingForTest()
			settings.PanelProxy = proxyURL

			if err := settings.CheckValid(); err == nil {
				t.Fatalf("CheckValid accepted invalid PanelProxy %q", proxyURL)
			}
		})
	}
}

func TestAllSettingCheckValidAcceptsSupportedPanelProxy(t *testing.T) {
	for _, proxyURL := range []string{
		"http://127.0.0.1:18080",
		"https://proxy.example.test:18443",
		"socks5://user:password@127.0.0.1:1080",
		"socks5h://127.0.0.1:1080",
	} {
		t.Run(proxyURL, func(t *testing.T) {
			settings := validAllSettingForTest()
			settings.PanelProxy = proxyURL

			if err := settings.CheckValid(); err != nil {
				t.Fatalf("CheckValid rejected supported PanelProxy %q: %v", proxyURL, err)
			}
		})
	}
}

func validAllSettingForTest() *AllSetting {
	return &AllSetting{
		WebPort:      2053,
		SubPort:      2096,
		WebBasePath:  "/super/",
		SubPath:      "/sub/",
		SubJsonPath:  "/json/",
		SubClashPath: "/clash/",
		TimeLocation: "Local",
	}
}
