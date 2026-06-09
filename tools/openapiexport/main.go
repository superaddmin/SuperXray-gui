package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

func main() {
	in := flag.String("in", "docs/openapi/panel-api.yaml", "OpenAPI YAML input")
	out := flag.String("out", "frontend/public/openapi.json", "OpenAPI JSON output")
	version := flag.String("version-file", "config/version", "version file")
	flag.Parse()

	if err := exportOpenAPI(*in, *out, *version); err != nil {
		fmt.Fprintln(os.Stderr, "openapiexport:", err)
		os.Exit(1)
	}
}

func exportOpenAPI(inPath string, outPath string, versionPath string) error {
	data, err := os.ReadFile(inPath) // #nosec G304 -- local build-time input path controlled by repository scripts.
	if err != nil {
		return fmt.Errorf("read %s: %w", inPath, err)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", inPath, err)
	}
	if _, ok := doc["openapi"].(string); !ok {
		return fmt.Errorf("missing openapi version")
	}
	if _, ok := doc["paths"].(map[string]any); !ok {
		return fmt.Errorf("missing paths")
	}

	if versionPath != "" {
		versionBytes, err := os.ReadFile(versionPath) // #nosec G304 -- local build-time version path controlled by repository scripts.
		if err != nil {
			return fmt.Errorf("read %s: %w", versionPath, err)
		}
		version := strings.TrimSpace(string(versionBytes))
		if version == "" {
			return fmt.Errorf("version file %s is empty", versionPath)
		}
		info, ok := doc["info"].(map[string]any)
		if !ok {
			info = map[string]any{}
			doc["info"] = info
		}
		info["version"] = version
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	out = append(out, '\n')

	if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(outPath, out, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}
