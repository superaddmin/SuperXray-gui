package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestXrayLogsToleratePartialLinesAndInvalidCount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XUI_BIN_FOLDER", dir)
	logPath := filepath.Join(dir, "access.log")
	data := "partial\n2026/10/15 10:00:00 from\n2026/10/15 10:00:01 accepted\n2026/10/15 10:00:02 email:\n2026/10/15 10:00:03 from 127.0.0.1 accepted tcp:example.test:443 [in -> direct] email: fixture\n"
	if err := os.WriteFile(logPath, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	config, _ := json.Marshal(map[string]any{"log": map[string]string{"access": logPath}})
	if err := os.WriteFile(filepath.Join(dir, "config.json"), config, 0o600); err != nil {
		t.Fatal(err)
	}
	s := &ServerService{}
	for _, count := range []string{"-1", "0", "bad", "10001"} {
		if entries := s.GetXrayLogs(count, "", "true", "true", "true", nil, nil); len(entries) != 0 {
			t.Fatalf("invalid count %s accepted", count)
		}
	}
	entries := s.GetXrayLogs("1", "", "true", "true", "true", []string{"direct"}, nil)
	if len(entries) != 1 || entries[0].Email != "fixture" || entries[0].ToAddress != "tcp:example.test:443" {
		t.Fatalf("failed to read valid last entry after partial lines: %+v", entries)
	}
}
