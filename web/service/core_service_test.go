package service

import (
	"context"
	"path/filepath"
	"testing"

	panelcore "github.com/superaddmin/SuperXray-gui/v2/core"
)

func TestCoreServiceRegistersDefaultXrayAndExperimentalSingBox(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XUI_BIN_FOLDER", dir)
	t.Setenv("XUI_LOG_FOLDER", filepath.Join(dir, "logs"))
	t.Setenv("SUPERXRAY_SING_BOX_BINARY", filepath.Join(dir, "sing-box.exe"))
	t.Setenv("SUPERXRAY_SING_BOX_CONFIG", filepath.Join(dir, "sing-box-config.json"))

	service := NewCoreService()
	instances, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(instances) != 2 {
		t.Fatalf("len(instances) = %d, want 2", len(instances))
	}

	byID := map[string]panelcore.Instance{}
	for _, instance := range instances {
		byID[instance.ID] = instance
	}
	if byID["default-xray"].CoreType != panelcore.CoreTypeXray {
		t.Fatalf("default-xray core type = %q, want %q", byID["default-xray"].CoreType, panelcore.CoreTypeXray)
	}
	if byID["default-xray"].Capabilities.LifecycleViaCoreManager {
		t.Fatal("default-xray must keep legacy lifecycle owner")
	}
	if byID["experimental-sing-box"].CoreType != panelcore.CoreTypeSingBox {
		t.Fatalf("experimental-sing-box core type = %q, want %q", byID["experimental-sing-box"].CoreType, panelcore.CoreTypeSingBox)
	}
	if byID["experimental-sing-box"].Status.State != panelcore.StateNotInstalled {
		t.Fatalf("sing-box state = %q, want %q", byID["experimental-sing-box"].Status.State, panelcore.StateNotInstalled)
	}
}

func TestDefaultXrayAdapterRejectsCoreManagerLifecycle(t *testing.T) {
	service := NewCoreService()

	if _, err := service.Start(context.Background(), "default-xray"); err == nil {
		t.Fatal("expected CoreManager lifecycle to be rejected for default-xray")
	}
}
