package singbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	panelcore "github.com/superaddmin/SuperXray-gui/v2/core"
)

func TestAdapterStatusReportsNotInstalledWhenBinaryMissing(t *testing.T) {
	dir := t.TempDir()
	adapter := NewAdapter(Options{
		BinaryPath: filepath.Join(dir, "sing-box.exe"),
		ConfigPath: filepath.Join(dir, "sing-box-config.json"),
		LogDir:     dir,
	})

	status, err := adapter.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.State != panelcore.StateNotInstalled {
		t.Fatalf("status state = %q, want %q", status.State, panelcore.StateNotInstalled)
	}
	if status.ErrorMsg == "" {
		t.Fatal("expected missing binary status to include an error message")
	}
}

func TestAdapterValidateReportsNotConfiguredWhenConfigMissing(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "sing-box.exe")
	if err := os.WriteFile(binaryPath, []byte("placeholder"), 0o700); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	adapter := NewAdapter(Options{
		BinaryPath: binaryPath,
		ConfigPath: filepath.Join(dir, "sing-box-config.json"),
		LogDir:     dir,
	})

	result, err := adapter.Validate(context.Background())
	if !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("Validate() error = %v, want ErrConfigNotFound", err)
	}
	if result.State != panelcore.StateNotConfigured {
		t.Fatalf("result state = %q, want %q", result.State, panelcore.StateNotConfigured)
	}
}
