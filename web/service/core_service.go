package service

import (
	"context"
	"fmt"

	panelcore "github.com/superaddmin/SuperXray-gui/v2/core"
	"github.com/superaddmin/SuperXray-gui/v2/core/singbox"
)

type CoreService struct {
	manager *panelcore.Manager
}

func NewCoreService() *CoreService {
	manager := panelcore.NewCoreManager()
	mustRegisterCore(manager, &defaultXrayAdapter{xrayService: XrayService{}})
	mustRegisterCore(manager, singbox.NewAdapter(singbox.DefaultOptions()))
	return &CoreService{manager: manager}
}

func (s *CoreService) List(ctx context.Context) ([]panelcore.Instance, error) {
	return s.manager.List(ctx)
}

func (s *CoreService) Get(ctx context.Context, id string) (panelcore.Instance, error) {
	return s.manager.Get(ctx, id)
}

func (s *CoreService) Status(ctx context.Context, id string) (panelcore.Status, error) {
	return s.manager.Status(ctx, id)
}

func (s *CoreService) Validate(ctx context.Context, id string) (panelcore.LifecycleResult, error) {
	return s.manager.Validate(ctx, id)
}

func (s *CoreService) Start(ctx context.Context, id string) (panelcore.LifecycleResult, error) {
	return s.manager.Start(ctx, id)
}

func (s *CoreService) Stop(ctx context.Context, id string) (panelcore.LifecycleResult, error) {
	return s.manager.Stop(ctx, id)
}

func (s *CoreService) Restart(ctx context.Context, id string) (panelcore.LifecycleResult, error) {
	return s.manager.Restart(ctx, id)
}

func mustRegisterCore(manager *panelcore.Manager, adapter panelcore.Adapter) {
	if err := manager.Register(adapter); err != nil {
		panic(fmt.Sprintf("register core adapter: %v", err))
	}
}

type defaultXrayAdapter struct {
	xrayService XrayService
}

func (a *defaultXrayAdapter) Instance() panelcore.Instance {
	return panelcore.Instance{
		ID:             "default-xray",
		Name:           "default-xray",
		DisplayName:    "Default Xray",
		CoreType:       panelcore.CoreTypeXray,
		Mode:           "legacy",
		Source:         "legacy-inbound-table",
		LifecycleOwner: "legacy-xray-service",
		Capabilities: panelcore.Capabilities{
			Read:                    true,
			Write:                   false,
			Validate:                false,
			Start:                   false,
			Stop:                    false,
			Restart:                 false,
			LifecycleViaCoreManager: false,
		},
		WriteSupported:   false,
		ManagerAttached:  false,
		ExperimentalOnly: false,
	}
}

func (a *defaultXrayAdapter) Status(context.Context) (panelcore.Status, error) {
	status := panelcore.Status{
		State:   panelcore.StateStopped,
		Version: a.xrayService.GetXrayVersion(),
	}
	if a.xrayService.IsXrayRunning() {
		status.State = panelcore.StateRunning
		return status, nil
	}
	if err := a.xrayService.GetXrayErr(); err != nil {
		status.State = panelcore.StateError
	}
	status.ErrorMsg = a.xrayService.GetXrayResult()
	return status, nil
}

func (a *defaultXrayAdapter) Validate(context.Context) (panelcore.LifecycleResult, error) {
	return unsupportedDefaultXrayLifecycle()
}

func (a *defaultXrayAdapter) Start(context.Context) (panelcore.LifecycleResult, error) {
	return unsupportedDefaultXrayLifecycle()
}

func (a *defaultXrayAdapter) Stop(context.Context) (panelcore.LifecycleResult, error) {
	return unsupportedDefaultXrayLifecycle()
}

func (a *defaultXrayAdapter) Restart(context.Context) (panelcore.LifecycleResult, error) {
	return unsupportedDefaultXrayLifecycle()
}

func unsupportedDefaultXrayLifecycle() (panelcore.LifecycleResult, error) {
	return panelcore.LifecycleResult{
		State:    panelcore.StateError,
		ErrorMsg: panelcore.ErrLifecycleUnsupported.Error(),
	}, panelcore.ErrLifecycleUnsupported
}
