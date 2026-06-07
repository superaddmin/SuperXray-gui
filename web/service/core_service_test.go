package service

import (
	"context"
	"errors"
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
	if byID["default-xray"].Capabilities.Validate ||
		byID["default-xray"].Capabilities.Start ||
		byID["default-xray"].Capabilities.Stop ||
		byID["default-xray"].Capabilities.Restart {
		t.Fatalf("default-xray lifecycle capabilities must stay false: %#v", byID["default-xray"].Capabilities)
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

	tests := []struct {
		name string
		call func(context.Context) (panelcore.LifecycleResult, error)
	}{
		{name: "validate", call: func(ctx context.Context) (panelcore.LifecycleResult, error) {
			return service.Validate(ctx, "default-xray")
		}},
		{name: "start", call: func(ctx context.Context) (panelcore.LifecycleResult, error) {
			return service.Start(ctx, "default-xray")
		}},
		{name: "stop", call: func(ctx context.Context) (panelcore.LifecycleResult, error) {
			return service.Stop(ctx, "default-xray")
		}},
		{name: "restart", call: func(ctx context.Context) (panelcore.LifecycleResult, error) {
			return service.Restart(ctx, "default-xray")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.call(context.Background())
			if !errors.Is(err, panelcore.ErrLifecycleUnsupported) {
				t.Fatalf("%s error = %v, want ErrLifecycleUnsupported", tt.name, err)
			}
			if result.State != panelcore.StateError {
				t.Fatalf("%s state = %q, want %q", tt.name, result.State, panelcore.StateError)
			}
			if result.ErrorMsg != panelcore.ErrLifecycleUnsupported.Error() {
				t.Fatalf("%s error message = %q, want %q", tt.name, result.ErrorMsg, panelcore.ErrLifecycleUnsupported.Error())
			}
		})
	}
}
