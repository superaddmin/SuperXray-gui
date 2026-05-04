package core

import (
	"context"
	"errors"
	"testing"
)

func TestManagerListsRegisteredInstancesWithCurrentStatus(t *testing.T) {
	manager := NewManager()
	adapter := &fakeAdapter{
		instance: Instance{
			ID:               "default-xray",
			Name:             "default-xray",
			DisplayName:      "Default Xray",
			CoreType:         CoreTypeXray,
			Mode:             "legacy",
			Source:           "legacy-inbound-table",
			LifecycleOwner:   "legacy-xray-service",
			Capabilities:     Capabilities{Read: true, Write: false, LifecycleViaCoreManager: false},
			WriteSupported:   false,
			ManagerAttached:  false,
			ExperimentalOnly: false,
		},
		status: Status{State: StateRunning, Version: "26.3.27"},
	}

	if err := manager.Register(adapter); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	instances, err := manager.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("len(instances) = %d, want 1", len(instances))
	}
	got := instances[0]
	if got.ID != "default-xray" {
		t.Fatalf("instance ID = %q, want default-xray", got.ID)
	}
	if got.Status.State != StateRunning {
		t.Fatalf("status state = %q, want %q", got.Status.State, StateRunning)
	}
	if got.Capabilities.LifecycleViaCoreManager {
		t.Fatal("default-xray must not be controlled by CoreManager lifecycle")
	}
}

func TestManagerRejectsDuplicateAdapters(t *testing.T) {
	manager := NewManager()
	adapter := &fakeAdapter{instance: Instance{ID: "experimental-sing-box"}}

	if err := manager.Register(adapter); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if err := manager.Register(adapter); !errors.Is(err, ErrInstanceAlreadyRegistered) {
		t.Fatalf("second Register() error = %v, want ErrInstanceAlreadyRegistered", err)
	}
}

func TestManagerReturnsNotFoundForUnknownInstance(t *testing.T) {
	manager := NewManager()

	if _, err := manager.Get(context.Background(), "missing"); !errors.Is(err, ErrInstanceNotFound) {
		t.Fatalf("Get() error = %v, want ErrInstanceNotFound", err)
	}
	if _, err := manager.Start(context.Background(), "missing"); !errors.Is(err, ErrInstanceNotFound) {
		t.Fatalf("Start() error = %v, want ErrInstanceNotFound", err)
	}
}

type fakeAdapter struct {
	instance Instance
	status   Status
}

func (a *fakeAdapter) Instance() Instance {
	return a.instance
}

func (a *fakeAdapter) Status(context.Context) (Status, error) {
	return a.status, nil
}

func (a *fakeAdapter) Validate(context.Context) (LifecycleResult, error) {
	return LifecycleResult{State: a.status.State}, nil
}

func (a *fakeAdapter) Start(context.Context) (LifecycleResult, error) {
	return LifecycleResult{State: StateRunning}, nil
}

func (a *fakeAdapter) Stop(context.Context) (LifecycleResult, error) {
	return LifecycleResult{State: StateStopped}, nil
}

func (a *fakeAdapter) Restart(context.Context) (LifecycleResult, error) {
	return LifecycleResult{State: StateRunning}, nil
}
