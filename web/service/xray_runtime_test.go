package service

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

func TestAutomaticRestartKeepsManualStopAndPendingRequest(t *testing.T) {
	manual, pending := isManuallyStopped.Load(), isNeedXrayRestart.Load()
	t.Cleanup(func() { isManuallyStopped.Store(manual); isNeedXrayRestart.Store(pending) })
	isManuallyStopped.Store(true)
	isNeedXrayRestart.Store(true)
	s := &XrayService{}
	if s.IsNeedRestartAndSetFalse() || !isNeedXrayRestart.Load() {
		t.Fatal("manual stop consumed the pending restart request")
	}
	if err := s.RestartXray(false); err != nil || !isManuallyStopped.Load() || !isNeedXrayRestart.Load() {
		t.Fatalf("automatic restart changed manual stop: %v", err)
	}
}

func TestFailedRestartRetainsPendingRequest(t *testing.T) {
	manual, pending := isManuallyStopped.Load(), isNeedXrayRestart.Load()
	t.Cleanup(func() { isManuallyStopped.Store(manual); isNeedXrayRestart.Store(pending) })
	isManuallyStopped.Store(false)
	isNeedXrayRestart.Store(false)
	repo := newFakeSettingRepository()
	if err := repo.Save("xrayTemplateConfig", `{"inbounds":`); err != nil {
		t.Fatal(err)
	}
	s := &XrayService{settingService: *NewSettingService(repo)}
	if err := s.RestartXray(false); err == nil || !isNeedXrayRestart.Load() {
		t.Fatalf("failed restart lost pending request: %v", err)
	}
}

func TestXrayConfigPreservesCoreClientExtensionsAndExactNumbers(t *testing.T) {
	settings := `{"decryption":"none","futureSettings":9007199254740993,"clients":[{"id":"fixture-id","email":"fixture@example.test","enable":true,"totalGB":42,"created_at":12,"testseed":[1,2],"testpre":3,"security":"auto","futureCore":{"value":9007199254740993}}]}`
	stream := `{"network":"tcp","security":"none","futureStream":{"value":9007199254740993}}`
	inbound := &model.Inbound{Enable: true, Protocol: model.VLESS, Tag: "fixture", Port: 12345, Settings: settings, StreamSettings: stream}
	settingRepo := newFakeSettingRepository()
	if err := settingRepo.Save("xrayTemplateConfig", `{"inbounds":[],"outbounds":[]}`); err != nil {
		t.Fatal(err)
	}
	service := &XrayService{
		settingService: *NewSettingService(settingRepo),
		inboundService: *NewInboundService(&fakeInboundRepository{inbounds: []*model.Inbound{inbound}}),
	}
	config, err := service.getXrayConfig(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.InboundConfigs) != 1 {
		t.Fatalf("generated %d inbounds, want 1", len(config.InboundConfigs))
	}
	var generated map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(config.InboundConfigs[0].Settings)))
	decoder.UseNumber()
	if err := decoder.Decode(&generated); err != nil {
		t.Fatal(err)
	}
	client := generated["clients"].([]any)[0].(map[string]any)
	for _, key := range []string{"testseed", "testpre", "security", "futureCore"} {
		if _, ok := client[key]; !ok {
			t.Errorf("core client extension %q was lost", key)
		}
	}
	for _, key := range []string{"enable", "totalGB", "created_at"} {
		if _, ok := client[key]; ok {
			t.Errorf("panel field %q leaked into runtime client", key)
		}
	}
	if generated["futureSettings"] != json.Number("9007199254740993") ||
		client["futureCore"].(map[string]any)["value"] != json.Number("9007199254740993") ||
		!strings.Contains(string(config.InboundConfigs[0].StreamSettings), "9007199254740993") {
		t.Fatal("runtime regeneration rounded unknown numeric fields")
	}
	if inbound.Settings != settings || inbound.StreamSettings != stream {
		t.Fatal("runtime regeneration mutated the repository model")
	}
}

func TestXrayProcessPublicationConcurrentReaders(t *testing.T) {
	previous := currentXrayProcess()
	t.Cleanup(func() { setXrayProcess(previous) })
	service := &XrayService{}
	inbounds := &InboundService{}
	process := xray.NewProcess(&xray.Config{})
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			for range 1000 {
				service.IsXrayRunning()
				service.GetXrayErr()
				service.GetXrayResult()
				service.GetXrayVersion()
				currentXrayAPIPort()
				inbounds.GetOnlineClients()
			}
		})
	}
	for range 1000 {
		setXrayProcess(process)
		setXrayProcess(nil)
	}
	workers.Wait()
}
