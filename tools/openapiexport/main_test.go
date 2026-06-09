package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExportOpenAPIYAMLToJSONInjectsVersion(t *testing.T) {
	tmp := t.TempDir()
	in := filepath.Join(tmp, "panel-api.yaml")
	out := filepath.Join(tmp, "openapi.json")
	versionFile := filepath.Join(tmp, "version")

	spec := []byte("openapi: 3.1.0\ninfo:\n  title: Test\n  version: 0.0.0\npaths: {}\n")
	if err := os.WriteFile(in, spec, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(versionFile, []byte("3.0.19\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := exportOpenAPI(in, out, versionFile); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("generated output must be JSON: %v", err)
	}
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatalf("generated output must contain info object: %#v", doc["info"])
	}
	if info["version"] != "3.0.19" {
		t.Fatalf("version mismatch: %#v", info["version"])
	}
}
