package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/superaddmin/SuperXray-gui/v2/config"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/util/json_util"
	"github.com/superaddmin/SuperXray-gui/v2/util/xraycompat"
	"github.com/superaddmin/SuperXray-gui/v2/xray"

	"go.uber.org/atomic"
)

var (
	p                 *xray.Process
	xrayProcessMu     sync.RWMutex
	lock              sync.Mutex
	isNeedXrayRestart atomic.Bool // Indicates that restart was requested for Xray
	isManuallyStopped atomic.Bool // Indicates that Xray was stopped manually from the panel
)

func currentXrayProcess() *xray.Process {
	xrayProcessMu.RLock()
	defer xrayProcessMu.RUnlock()
	return p
}

func setXrayProcess(process *xray.Process) {
	xrayProcessMu.Lock()
	p = process
	xrayProcessMu.Unlock()
}

// A zero port lets the API report an ordinary initialization error while the
// core is absent, instead of dereferencing a nil global process.
func currentXrayAPIPort() int {
	if process := currentXrayProcess(); process != nil {
		return process.GetAPIPort()
	}
	return 0
}

// XrayService provides business logic for Xray process management.
// It handles starting, stopping, restarting Xray, and managing its configuration.
type XrayService struct {
	inboundService InboundService
	settingService SettingService
}

// IsXrayRunning checks if the Xray process is currently running.
func (s *XrayService) IsXrayRunning() bool {
	p := currentXrayProcess()
	return p != nil && p.IsRunning()
}

// GetXrayErr returns the error from the Xray process, if any.
func (s *XrayService) GetXrayErr() error {
	p := currentXrayProcess()
	if p == nil {
		return nil
	}

	err := p.GetErr()
	if err == nil {
		return nil
	}

	if runtime.GOOS == "windows" && err.Error() == "exit status 1" {
		// exit status 1 on Windows means that Xray process was killed
		// as we kill process to stop in on Windows, this is not an error
		return nil
	}

	return err
}

// GetXrayResult returns the result string from the Xray process.
func (s *XrayService) GetXrayResult() string {
	p := currentXrayProcess()
	if p == nil || p.IsRunning() {
		return ""
	}

	result := p.GetResult()

	if runtime.GOOS == "windows" && result == "exit status 1" {
		// exit status 1 on Windows means that Xray process was killed
		// as we kill process to stop in on Windows, this is not an error
		return ""
	}

	return result
}

// GetXrayVersion returns the version of the running Xray process.
func (s *XrayService) GetXrayVersion() string {
	p := currentXrayProcess()
	if p == nil {
		return "Unknown"
	}
	return p.GetVersion()
}

// RemoveIndex removes an element at the specified index from a slice.
// Returns a new slice with the element removed.
func RemoveIndex(s []any, index int) []any {
	return append(s[:index], s[index+1:]...)
}

// GetXrayConfig retrieves and builds the Xray configuration from settings and inbounds.
func (s *XrayService) GetXrayConfig() (*xray.Config, error) {
	return s.getXrayConfig(true)
}

// getXrayConfig builds the effective configuration. updateTraffic is disabled
// while validating an imported candidate database so previewing a migration
// cannot mutate the currently active database or traffic counters.
func (s *XrayService) getXrayConfig(updateTraffic bool) (*xray.Config, error) {
	templateConfig, err := s.settingService.GetXrayConfigTemplate()
	if err != nil {
		return nil, err
	}

	normalizedTemplate, _, normalizeErr := xraycompat.NormalizeConfig(templateConfig)
	if normalizeErr != nil {
		return nil, fmt.Errorf("invalid Xray template for v26.10.10: %w", normalizeErr)
	}
	xrayConfig := &xray.Config{}
	err = json.Unmarshal([]byte(normalizedTemplate), xrayConfig)
	if err != nil {
		return nil, err
	}
	xrayConfig.LogConfig = resolveXrayLogPaths(xrayConfig.LogConfig)
	for i := range xrayConfig.InboundConfigs {
		inbound := &xrayConfig.InboundConfigs[i]
		stream, err := xraycompat.InboundRuntimeStream(string(inbound.StreamSettings))
		if err != nil {
			return nil, fmt.Errorf("template inbound %q cannot be built: %w", inbound.Tag, err)
		}
		inbound.StreamSettings = json_util.RawMessage(stream)
	}

	if updateTraffic {
		if err, _ := s.inboundService.AddTraffic(nil, nil); err != nil {
			return nil, err
		}
	}

	inbounds, err := s.inboundService.GetAllInbounds()
	if err != nil {
		return nil, err
	}
	for _, storedInbound := range inbounds {
		inbound := *storedInbound
		if !inbound.Enable {
			continue
		}
		normalizedSettings, normalizedStream, _, normalizeErr := xraycompat.NormalizeInbound(string(inbound.Protocol.XrayProtocol()), inbound.Settings, inbound.StreamSettings)
		if normalizeErr != nil {
			return nil, fmt.Errorf("inbound %q is incompatible with Xray v26.10.10: %w", inbound.Tag, normalizeErr)
		}
		inbound.Settings = normalizedSettings
		inbound.StreamSettings = normalizedStream
		settings := map[string]any{}
		decoder := json.NewDecoder(strings.NewReader(inbound.Settings))
		decoder.UseNumber()
		if err := decoder.Decode(&settings); err != nil {
			return nil, err
		}
		if inbound.Protocol == model.Shadowsocks {
			normalizeShadowsocksSettingsMap(settings)
		}
		clients, ok := settings["clients"].([]any)
		if ok {
			// Fast O(N) lookup map for client traffic enablement
			clientStats := inbound.ClientStats
			enableMap := make(map[string]bool, len(clientStats))
			for _, clientTraffic := range clientStats {
				enableMap[clientTraffic.Email] = clientTraffic.Enable
			}

			// filter and clean clients
			var final_clients []any
			for _, client := range clients {
				c, ok := client.(map[string]any)
				if !ok {
					continue
				}

				email, _ := c["email"].(string)

				// check users active or not via stats
				if enable, exists := enableMap[email]; exists && !enable {
					logger.Infof("Remove Inbound User %s due to expiration or traffic limit", email)
					continue
				}

				// check manual disabled flag
				if manualEnable, ok := c["enable"].(bool); ok && !manualEnable {
					continue
				}

				// Only remove panel bookkeeping. Core extensions and future
				// client fields must survive runtime regeneration unchanged.
				for _, key := range []string{"enable", "limitIp", "totalGB", "expiryTime", "tgId", "subId", "comment", "reset", "created_at", "updated_at"} {
					delete(c, key)
				}
				if flow, ok := c["flow"].(string); ok && flow == "xtls-rprx-vision-udp443" {
					c["flow"] = "xtls-rprx-vision"
				}
				final_clients = append(final_clients, any(c))
			}

			settings["clients"] = final_clients
			modifiedSettings, err := json.MarshalIndent(settings, "", "  ")
			if err != nil {
				return nil, err
			}

			inbound.Settings = string(modifiedSettings)
		}

		if len(inbound.StreamSettings) > 0 {
			// Unmarshal stream JSON
			var stream map[string]any
			decoder := json.NewDecoder(strings.NewReader(inbound.StreamSettings))
			decoder.UseNumber()
			if err := decoder.Decode(&stream); err != nil {
				return nil, err
			}

			// Remove the "settings" field under "tlsSettings" and "realitySettings"
			tlsSettings, ok1 := stream["tlsSettings"].(map[string]any)
			realitySettings, ok2 := stream["realitySettings"].(map[string]any)
			if ok1 || ok2 {
				if ok1 {
					delete(tlsSettings, "settings")
				} else if ok2 {
					delete(realitySettings, "settings")
				}
			}

			delete(stream, "externalProxy")

			newStream, err := json.MarshalIndent(stream, "", "  ")
			if err != nil {
				return nil, err
			}
			inbound.StreamSettings = string(newStream)
		}

		inboundConfig, buildErr := inbound.BuildXrayInboundConfig()
		if buildErr != nil {
			return nil, fmt.Errorf("inbound %q cannot be built: %w", inbound.Tag, buildErr)
		}
		xrayConfig.InboundConfigs = append(xrayConfig.InboundConfigs, *inboundConfig)
	}
	return xrayConfig, nil
}

// resolveXrayLogPaths confines Xray log.access/log.error values to the panel
// log folder. This preserves "" and "none" as disabled values, but reduces any
// user-provided path to its basename to prevent absolute-path or traversal
// writes outside config.GetLogFolder().
func resolveXrayLogPaths(logCfg json_util.RawMessage) json_util.RawMessage {
	if len(logCfg) == 0 {
		return logCfg
	}

	var parsed map[string]any
	if err := json.Unmarshal(logCfg, &parsed); err != nil {
		return logCfg
	}

	changed := false
	for _, key := range []string{"access", "error"} {
		value, ok := parsed[key].(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || strings.EqualFold(trimmed, "none") {
			continue
		}

		base := path.Base(filepath.ToSlash(trimmed))
		if base == "" || base == "." || base == ".." || base == "/" {
			continue
		}

		confined := filepath.Join(config.GetLogFolder(), base)
		if confined == value {
			continue
		}
		parsed[key] = confined
		changed = true
	}

	if !changed {
		return logCfg
	}
	out, err := json.Marshal(parsed)
	if err != nil {
		return logCfg
	}
	return out
}

// GetXrayTraffic fetches the current traffic statistics from the running Xray process.
func (s *XrayService) GetXrayTraffic() ([]*xray.Traffic, []*xray.ClientTraffic, error) {
	p := currentXrayProcess()
	if p == nil || !p.IsRunning() {
		err := errors.New("xray is not running")
		logger.Debug("Attempted to fetch Xray traffic, but Xray is not running:", err)
		return nil, nil, err
	}
	apiPort := p.GetAPIPort()
	var api xray.XrayAPI
	if err := api.Init(apiPort); err != nil {
		logger.Debug("Failed to initialize Xray API:", err)
		return nil, nil, err
	}
	defer api.Close()

	traffic, clientTraffic, err := api.GetTraffic(true)
	if err != nil {
		logger.Debug("Failed to fetch Xray traffic:", err)
		return nil, nil, err
	}
	return traffic, clientTraffic, nil
}

// RestartXray restarts the Xray process, optionally forcing a restart even if config unchanged.
func (s *XrayService) RestartXray(isForce bool) error {
	runtimeMaintenanceMu.Lock()
	defer runtimeMaintenanceMu.Unlock()
	return s.restartXray(isForce)
}

func (s *XrayService) restartXray(isForce bool) (err error) {
	lock.Lock()
	defer lock.Unlock()
	defer func() {
		if err != nil {
			isNeedXrayRestart.Store(true)
		}
	}()
	if !isForce && isManuallyStopped.Load() {
		isNeedXrayRestart.Store(true)
		return nil
	}
	logger.Debug("restart Xray, force:", isForce)

	xrayConfig, err := s.GetXrayConfig()
	if err != nil {
		return err
	}

	previous := currentXrayProcess()
	wasRunning := previous != nil && previous.IsRunning()
	if wasRunning {
		if !isForce && previous.GetConfig().Equals(xrayConfig) && !isNeedXrayRestart.Load() {
			logger.Debug("It does not need to restart Xray")
			return nil
		}
	}
	// Parsing/building with the installed core must succeed before stopping a
	// healthy instance; syntactically valid JSON alone does not validate Xray.
	if err := xray.ValidateConfig(xrayConfig); err != nil {
		return fmt.Errorf("preflight Xray restart: %w", err)
	}
	if wasRunning {
		if err := previous.Stop(); err != nil {
			return err
		}
	}

	isManuallyStopped.Store(false)
	next := xray.NewProcess(xrayConfig)
	err = next.Start()
	if err != nil {
		// A valid configuration can still fail to start because of a transient
		// listener/resource error. Restore the previous process before returning
		// the failure so a restart never leaves the panel without its old core.
		if wasRunning {
			if restoreErr := previous.Start(); restoreErr != nil {
				return errors.Join(err, fmt.Errorf("restore previous Xray process: %w", restoreErr))
			}
		} else {
			setXrayProcess(next)
		}
		return err
	}
	setXrayProcess(next)

	return nil
}

// StopXray stops the running Xray process.
func (s *XrayService) StopXray() error {
	runtimeMaintenanceMu.Lock()
	defer runtimeMaintenanceMu.Unlock()
	return s.stopXray()
}

func (s *XrayService) stopXray() error {
	lock.Lock()
	defer lock.Unlock()
	isManuallyStopped.Store(true)
	logger.Debug("Attempting to stop Xray...")
	p := currentXrayProcess()
	if p != nil && p.IsRunning() {
		return p.Stop()
	}
	return errors.New("xray is not running")
}

// SetToNeedRestart marks that Xray needs to be restarted.
func (s *XrayService) SetToNeedRestart() {
	isNeedXrayRestart.Store(true)
}

// IsNeedRestartAndSetFalse checks if restart is needed and resets the flag to false.
func (s *XrayService) IsNeedRestartAndSetFalse() bool {
	if isManuallyStopped.Load() {
		return false
	}
	return isNeedXrayRestart.CompareAndSwap(true, false)
}

// DidXrayCrash checks if Xray crashed by verifying it's not running and wasn't manually stopped.
func (s *XrayService) DidXrayCrash() bool {
	return !s.IsXrayRunning() && !isManuallyStopped.Load()
}
