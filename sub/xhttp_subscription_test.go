package sub

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyXhttpShareParamsResolvesNestedExtraSettings(t *testing.T) {
	xhttp := map[string]any{
		"path": "/xhttp",
		"mode": "packet-up",
		"extra": map[string]any{
			"headers":             map[string]any{"Host": "nested.example.com"},
			"xPaddingBytes":       "100-1000",
			"sessionPlacement":    "header",
			"sessionKey":          "X-Session",
			"uplinkDataPlacement": "cookie",
			"xmux": map[string]any{
				"maxConcurrency": "16-32",
			},
			"downloadSettings": map[string]any{"network": "xhttp"},
			"localExtension":   map[string]any{"keep": true},
		},
	}

	params := map[string]string{}
	applyXhttpShareParams(xhttp, params)
	if params["path"] != "/xhttp" || params["host"] != "nested.example.com" {
		t.Fatalf("path/host params = %#v", params)
	}
	if params["mode"] != "packet-up" || params["x_padding_bytes"] != "100-1000" {
		t.Fatalf("mode/padding params = %#v", params)
	}

	extra := map[string]any{}
	if err := json.Unmarshal([]byte(params["extra"]), &extra); err != nil {
		t.Fatalf("decode XHTTP extra: %v", err)
	}
	if extra["sessionPlacement"] != "header" || extra["uplinkDataPlacement"] != "cookie" {
		t.Fatalf("extra placement fields = %#v", extra)
	}
	if _, ok := extra["xmux"].(map[string]any); !ok {
		t.Fatalf("extra xmux = %#v", extra["xmux"])
	}
	if _, ok := extra["downloadSettings"].(map[string]any); !ok {
		t.Fatalf("download settings = %#v", extra["downloadSettings"])
	}
	if _, ok := extra["localExtension"]; ok {
		t.Fatalf("unknown extra field leaked into subscription: %#v", extra)
	}
}

func TestApplyXhttpShareParamsKeepsLegacyRootSettings(t *testing.T) {
	params := map[string]string{}
	applyXhttpShareParams(
		map[string]any{
			"path":                "/legacy",
			"mode":                "auto",
			"headers":             map[string]any{"host": "legacy.example.com"},
			"xPaddingBytes":       "80-600",
			"uplinkHTTPMethod":    "PUT",
			"sessionPlacement":    "query",
			"uplinkDataPlacement": "body",
		},
		params,
	)
	if params["host"] != "legacy.example.com" || params["x_padding_bytes"] != "80-600" {
		t.Fatalf("legacy params = %#v", params)
	}
	extra := map[string]any{}
	if err := json.Unmarshal([]byte(params["extra"]), &extra); err != nil {
		t.Fatalf("decode legacy XHTTP extra: %v", err)
	}
	if extra["uplinkHTTPMethod"] != "PUT" || extra["sessionPlacement"] != "query" {
		t.Fatalf("legacy extra = %#v", extra)
	}
}

func TestApplyVmessNetworkParamsIncludesXhttpExtra(t *testing.T) {
	obj := map[string]any{}
	applyVmessNetworkParams(
		map[string]any{
			"xhttpSettings": map[string]any{
				"path": "/vmess",
				"mode": "packet-up",
				"extra": map[string]any{
					"headers":          map[string]any{"Host": "vmess.example.com"},
					"seqPlacement":     "cookie",
					"uplinkHTTPMethod": "GET",
				},
			},
		},
		"xhttp",
		obj,
	)
	if obj["path"] != "/vmess" || obj["host"] != "vmess.example.com" {
		t.Fatalf("VMess XHTTP path/host = %#v", obj)
	}
	extraText, ok := obj["extra"].(string)
	if !ok || !strings.Contains(extraText, `"seqPlacement":"cookie"`) {
		t.Fatalf("VMess XHTTP extra = %#v", obj["extra"])
	}
	extra := map[string]any{}
	if err := json.Unmarshal([]byte(extraText), &extra); err != nil {
		t.Fatalf("decode VMess XHTTP extra: %v", err)
	}
	if extra["seqPlacement"] != "cookie" || extra["uplinkHTTPMethod"] != "GET" {
		t.Fatalf("VMess XHTTP extra fields = %#v", extra)
	}
	if headers, ok := extra["headers"].(map[string]any); !ok || headers["Host"] != "vmess.example.com" {
		t.Fatalf("VMess XHTTP headers = %#v", extra["headers"])
	}

	encoded, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("encode VMess object: %v", err)
	}
	link := "vmess://" + base64.StdEncoding.EncodeToString(encoded)
	if !strings.HasPrefix(link, "vmess://") {
		t.Fatalf("VMess link = %q", link)
	}
}
