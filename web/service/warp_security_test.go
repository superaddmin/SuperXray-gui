package service

import (
	"encoding/json"
	"testing"
)

func TestBuildWarpRegistrationPayloadEscapesJSON(t *testing.T) {
	body, err := buildWarpRegistrationPayload(`pub"key`, "2026-04-28T00:00:00.000Z", `host"name`)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("payload is invalid JSON: %v\n%s", err, body)
	}

	if parsed["key"] != `pub"key` {
		t.Fatalf("key = %q", parsed["key"])
	}
	if parsed["name"] != `host"name` {
		t.Fatalf("name = %q", parsed["name"])
	}
}

func TestBuildWarpLicensePayloadEscapesJSON(t *testing.T) {
	body, err := buildWarpLicensePayload(`license"key`)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("payload is invalid JSON: %v\n%s", err, body)
	}

	if parsed["license"] != `license"key` {
		t.Fatalf("license = %q", parsed["license"])
	}
}
