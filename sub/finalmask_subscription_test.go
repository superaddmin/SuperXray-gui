package sub

import (
	"strings"
	"testing"
)

func TestMarshalFinalMaskSkipsFragmentWithEmptyLength(t *testing.T) {
	out, ok := marshalFinalMask(map[string]any{
		"tcp": []any{
			map[string]any{
				"type": "fragment",
				"settings": map[string]any{
					"packets": "1-3",
					"length":  "",
				},
			},
		},
	})

	if ok {
		t.Fatalf("marshalFinalMask ok = true, payload = %q; want empty invalid fragment omitted", out)
	}
}

func TestMarshalFinalMaskSkipsFragmentWithZeroMinimumLength(t *testing.T) {
	out, ok := marshalFinalMask(map[string]any{
		"tcp": []any{
			map[string]any{
				"type": "fragment",
				"settings": map[string]any{
					"packets": "1-3",
					"length":  "0-200",
				},
			},
		},
	})

	if ok {
		t.Fatalf("marshalFinalMask ok = true, payload = %q; want zero-min invalid fragment omitted", out)
	}
}

func TestMarshalFinalMaskKeepsValidFragmentLength(t *testing.T) {
	out, ok := marshalFinalMask(map[string]any{
		"tcp": []any{
			map[string]any{
				"type": "fragment",
				"settings": map[string]any{
					"packets": "1-3",
					"length":  "100-200",
				},
			},
		},
	})

	if !ok {
		t.Fatal("marshalFinalMask ok = false, want valid fragment retained")
	}
	if !strings.Contains(out, `"length":"100-200"`) {
		t.Fatalf("marshalFinalMask payload = %q, want valid length preserved", out)
	}
}

func TestHysteriaHopPortsReadsFinalMaskUdpHopPorts(t *testing.T) {
	ports := hysteriaHopPorts(map[string]any{
		"finalmask": map[string]any{
			"quicParams": map[string]any{
				"udpHop": map[string]any{
					"ports": " 40000-45000 ",
				},
			},
		},
	})

	if ports != "40000-45000" {
		t.Fatalf("hysteriaHopPorts = %q, want trimmed UDP hop ports", ports)
	}
}

func TestHysteriaHopPortsSkipsMissingUdpHopPorts(t *testing.T) {
	ports := hysteriaHopPorts(map[string]any{
		"finalmask": map[string]any{
			"quicParams": map[string]any{
				"udpHop": map[string]any{},
			},
		},
	})

	if ports != "" {
		t.Fatalf("hysteriaHopPorts = %q, want empty missing UDP hop ports", ports)
	}
}
