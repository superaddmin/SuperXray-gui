package service

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/config"
	"github.com/superaddmin/SuperXray-gui/v2/util/json_util"
)

func TestResolveXrayLogPathsConfinesAccessAndErrorToLogFolder(t *testing.T) {
	logDir := t.TempDir()
	t.Setenv("XUI_LOG_FOLDER", logDir)

	input := json_util.RawMessage(`{
		"loglevel": "warning",
		"access": "/tmp/../../escape/access.log",
		"error": "../outside/error.log",
		"dnsLog": false
	}`)

	out := resolveXrayLogPaths(input)

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("resolveXrayLogPaths returned invalid JSON: %v", err)
	}

	if got["access"] != filepath.Join(config.GetLogFolder(), "access.log") {
		t.Fatalf("access path = %q, want confined basename in log folder", got["access"])
	}
	if got["error"] != filepath.Join(config.GetLogFolder(), "error.log") {
		t.Fatalf("error path = %q, want confined basename in log folder", got["error"])
	}
	if got["loglevel"] != "warning" || got["dnsLog"] != false {
		t.Fatalf("unrelated log fields changed: %#v", got)
	}
}

func TestResolveXrayLogPathsLeavesDisabledValues(t *testing.T) {
	t.Setenv("XUI_LOG_FOLDER", t.TempDir())

	input := json_util.RawMessage(`{"access":"none","error":"","loglevel":"info"}`)
	out := resolveXrayLogPaths(input)

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("resolveXrayLogPaths returned invalid JSON: %v", err)
	}

	if got["access"] != "none" {
		t.Fatalf("access path = %q, want none", got["access"])
	}
	if got["error"] != "" {
		t.Fatalf("error path = %q, want empty string", got["error"])
	}
}
