package singbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	panelcore "github.com/superaddmin/SuperXray-gui/v2/core"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 4 && os.Args[1] == "run" && os.Args[2] == "-c" && os.Getenv("SUPERXRAY_TEST_SINGBOX_CHILD") == "1" {
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestRestartWaitsForOldProcessAndStopKeepsStatusClean(t *testing.T) {
	t.Setenv("SUPERXRAY_TEST_SINGBOX_CHILD", "1")
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := NewAdapter(Options{BinaryPath: binary, ConfigPath: configPath, LogDir: dir})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	first, err := adapter.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Stop(context.Background())
	second, err := adapter.Restart(ctx)
	if err != nil || second.PID == first.PID {
		t.Fatalf("restart returned %+v, %v after PID %d", second, err, first.PID)
	}
	if _, err := adapter.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	adapter.mu.Lock()
	reaped := adapter.cmd == nil
	adapter.mu.Unlock()
	status, err := adapter.Status(ctx)
	if err != nil || !reaped || status.State != panelcore.StateStopped || status.ErrorMsg != "" {
		t.Fatalf("stop completed with reaped=%v, status=%+v, err=%v", reaped, status, err)
	}
}

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
