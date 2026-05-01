package xray

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetBinaryNameUsesWindowsExecutableSuffix(t *testing.T) {
	name := GetBinaryName()

	if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
		t.Fatalf("GetBinaryName() = %q, want Windows executable suffix", name)
	}
	if runtime.GOOS != "windows" && strings.HasSuffix(name, ".exe") {
		t.Fatalf("GetBinaryName() = %q, non-Windows binaries should not use .exe suffix", name)
	}
}
