package xray

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/config"
	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/util/common"
)

// GetBinaryName returns the Xray binary filename for the current OS and architecture.
func GetBinaryName() string {
	arch := runtime.GOARCH
	if arch == "arm" {
		arch = "arm32"
	}
	name := fmt.Sprintf("xray-%s-%s", runtime.GOOS, arch)
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// GetBinaryPath returns the full path to the Xray binary executable.
func GetBinaryPath() string {
	return config.GetBinFolderPath() + "/" + GetBinaryName()
}

// GetConfigPath returns the path to the Xray configuration file in the binary folder.
func GetConfigPath() string {
	return config.GetBinFolderPath() + "/config.json"
}

// GetGeositePath returns the path to the geosite data file used by Xray.
func GetGeositePath() string {
	return config.GetBinFolderPath() + "/geosite.dat"
}

// GetGeoipPath returns the path to the geoip data file used by Xray.
func GetGeoipPath() string {
	return config.GetBinFolderPath() + "/geoip.dat"
}

// GetIPLimitLogPath returns the path to the IP limit log file.
func GetIPLimitLogPath() string {
	return config.GetLogFolder() + "/3xipl.log"
}

// GetIPLimitBannedLogPath returns the path to the banned IP log file.
func GetIPLimitBannedLogPath() string {
	return config.GetLogFolder() + "/3xipl-banned.log"
}

// GetIPLimitBannedPrevLogPath returns the path to the previous banned IP log file.
func GetIPLimitBannedPrevLogPath() string {
	return config.GetLogFolder() + "/3xipl-banned.prev.log"
}

// GetAccessPersistentLogPath returns the path to the persistent access log file.
func GetAccessPersistentLogPath() string {
	return config.GetLogFolder() + "/3xipl-ap.log"
}

// GetAccessPersistentPrevLogPath returns the path to the previous persistent access log file.
func GetAccessPersistentPrevLogPath() string {
	return config.GetLogFolder() + "/3xipl-ap.prev.log"
}

// GetAccessLogPath reads the Xray config and returns the access log file path.
func GetAccessLogPath() (string, error) {
	config, err := os.ReadFile(GetConfigPath())
	if err != nil {
		logger.Warningf("Failed to read configuration file: %s", err)
		return "", err
	}

	jsonConfig := map[string]any{}
	err = json.Unmarshal([]byte(config), &jsonConfig)
	if err != nil {
		logger.Warningf("Failed to parse JSON configuration: %s", err)
		return "", err
	}

	if jsonLog, ok := jsonConfig["log"].(map[string]any); ok {
		if accessLogPath, ok := jsonLog["access"].(string); ok {
			return accessLogPath, nil
		}
	}
	return "", err
}

// stopProcess calls Stop on the given Process instance.
func stopProcess(p *Process) {
	if !p.IsRunning() {
		return
	}
	if err := p.Stop(); err != nil {
		logger.Warning("Failed to stop Xray process finalizer:", err)
	}
}

// Process wraps an Xray process instance and provides management methods.
type Process struct {
	*process
}

// NewProcess creates a new Xray process and sets up cleanup on garbage collection.
func NewProcess(xrayConfig *Config) *Process {
	p := &Process{newProcess(xrayConfig)}
	runtime.SetFinalizer(p, stopProcess)
	return p
}

// NewTestProcess creates a new Xray process that uses a specific config file path.
// Used for test runs (e.g. outbound test) so the main config.json is not overwritten.
// The config file at configPath is removed when the process is stopped.
func NewTestProcess(xrayConfig *Config, configPath string) *Process {
	p := &Process{newTestProcess(xrayConfig, configPath)}
	runtime.SetFinalizer(p, stopProcess)
	return p
}

type process struct {
	mu          sync.RWMutex
	lifecycleMu sync.Mutex
	cmd         *exec.Cmd
	done        chan struct{}

	version string
	apiPort int

	onlineClients []string

	config     *Config
	configPath string // if set, use this path instead of GetConfigPath() and remove on Stop
	logWriter  *LogWriter
	exitErr    error
	startTime  time.Time
}

// newProcess creates a new internal process struct for Xray.
func newProcess(config *Config) *process {
	return &process{
		version:   "Unknown",
		config:    config,
		logWriter: NewLogWriter(),
	}
}

// newTestProcess creates a process that writes and runs with a specific config path.
func newTestProcess(config *Config, configPath string) *process {
	p := newProcess(config)
	p.configPath = configPath
	return p
}

// IsRunning returns true if the Xray process is currently running.
func (p *process) IsRunning() bool {
	p.mu.RLock()
	cmd, done := p.cmd, p.done
	p.mu.RUnlock()
	if cmd == nil || cmd.Process == nil {
		return false
	}
	if done != nil {
		select {
		case <-done:
			return false
		default:
		}
	}
	return true
}

// GetErr returns the last error encountered by the Xray process.
func (p *process) GetErr() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exitErr
}

// GetResult returns the last log line or error from the Xray process.
func (p *process) GetResult() string {
	p.mu.RLock()
	exitErr := p.exitErr
	p.mu.RUnlock()
	lastLine := p.logWriter.LastLine()
	if len(lastLine) == 0 && exitErr != nil {
		return exitErr.Error()
	}
	return lastLine
}

// GetVersion returns the version string of the Xray process.
func (p *process) GetVersion() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.version
}

// GetAPIPort returns the API port used by the Xray process.
func (p *Process) GetAPIPort() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.apiPort
}

// GetConfig returns the configuration used by the Xray process.
func (p *Process) GetConfig() *Config {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// GetOnlineClients returns the list of online clients for the Xray process.
func (p *Process) GetOnlineClients() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]string(nil), p.onlineClients...)
}

// SetOnlineClients sets the list of online clients for the Xray process.
func (p *Process) SetOnlineClients(users []string) {
	p.mu.Lock()
	p.onlineClients = append([]string(nil), users...)
	p.mu.Unlock()
}

// GetUptime returns the uptime of the Xray process in seconds.
func (p *Process) GetUptime() uint64 {
	p.mu.RLock()
	started := p.startTime
	p.mu.RUnlock()
	if started.IsZero() {
		return 0
	}
	return uint64(time.Since(started).Seconds())
}

// refreshAPIPort updates the API port from the inbound configs.
func (p *process) refreshAPIPort() {
	port := 0
	for _, inbound := range p.config.InboundConfigs {
		if inbound.Tag == "api" {
			port = inbound.Port
			break
		}
	}
	p.mu.Lock()
	p.apiPort = port
	p.mu.Unlock()
}

// refreshVersion updates the version string by running the Xray binary with -version.
func (p *process) refreshVersion() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// #nosec G204 -- binary path is resolved by project configuration and no shell is invoked.
	cmd := exec.CommandContext(ctx, GetBinaryPath(), "-version")
	var output cappedBuffer
	output.limit = 1024
	cmd.Stdout = &output
	cmd.Stderr = &output
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	version := "Unknown"
	if err != nil {
		version = "Unknown"
	} else {
		datas := bytes.Split(output.Bytes(), []byte(" "))
		if len(datas) <= 1 {
			version = "Unknown"
		} else {
			version = string(datas[1])
		}
	}
	p.mu.Lock()
	p.version = version
	p.mu.Unlock()
}

// Start launches the Xray process with the current configuration.
func (p *process) Start() (err error) {
	p.lifecycleMu.Lock()
	defer p.lifecycleMu.Unlock()
	if p.IsRunning() {
		return errors.New("xray is already running")
	}

	defer func() {
		if err != nil {
			logger.Error("Failure in running xray-core process: ", err)
			p.mu.Lock()
			p.exitErr = err
			p.mu.Unlock()
		}
	}()

	data, err := json.MarshalIndent(p.config, "", "  ")
	if err != nil {
		return common.NewErrorf("Failed to generate XRAY configuration files: %v", err)
	}
	// Service callers also preflight before stopping the previous core. Keep
	// this check for standalone/test processes and to catch changed assets.
	if err := ValidateConfig(p.config); err != nil {
		return common.NewErrorf("Xray configuration rejected by core: %v", err)
	}

	err = os.MkdirAll(config.GetLogFolder(), 0o750)
	if err != nil {
		logger.Warningf("Failed to create log folder: %s", err)
	}

	configPath := GetConfigPath()
	if p.configPath != "" {
		configPath = p.configPath
	}
	p.refreshVersion()
	p.refreshAPIPort()
	// #nosec G204 -- binary path and config path are local project-controlled values; no shell is invoked.
	cmd := exec.Command(GetBinaryPath(), "-c", configPath)
	if err := p.startWithConfig(configPath, data, cmd); err != nil {
		return err
	}
	return nil
}

// startWithConfig restores the last runtime file when launching fails or the
// child exits during its initial startup window (for example, a busy port).
func (p *process) startWithConfig(configPath string, data []byte, cmd *exec.Cmd) (err error) {
	// #nosec G304 -- configuration path comes from the panel bin directory or a private test process.
	previous, readErr := os.ReadFile(configPath)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("read previous Xray configuration: %w", readErr)
	}
	if err := writeConfigAtomic(configPath, data); err != nil {
		return fmt.Errorf("write Xray configuration: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}
		var restoreErr error
		if readErr == nil {
			restoreErr = writeConfigAtomic(configPath, previous)
		} else {
			restoreErr = os.Remove(configPath)
			if errors.Is(restoreErr, os.ErrNotExist) {
				restoreErr = nil
			}
		}
		if restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("restore previous Xray configuration: %w", restoreErr))
		}
	}()
	if err := p.startCommand(cmd); err != nil {
		return fmt.Errorf("start Xray process: %w", err)
	}
	p.mu.RLock()
	done := p.done
	p.mu.RUnlock()
	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-done:
		return fmt.Errorf("xray exited during startup: %w", p.startupError())
	case <-timer.C:
		return nil
	}
}

func (p *process) startupError() error {
	if err := p.GetErr(); err != nil {
		return err
	}
	return errors.New("process terminated before becoming ready")
}

// startCommand publishes only a started command. Readers never observe the
// exec.Cmd while Start mutates its Process field, or ProcessState during Wait.
func (p *process) startCommand(cmd *exec.Cmd) error {
	cmd.Stdout = p.logWriter
	cmd.Stderr = p.logWriter
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan struct{})
	p.mu.Lock()
	p.cmd = cmd
	p.done = done
	p.exitErr = nil
	p.startTime = time.Now()
	p.mu.Unlock()

	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		if err != nil {
			p.exitErr = err
		}
		close(done)
		p.mu.Unlock()
		if err != nil {
			// On Windows, killing the process results in "exit status 1" which isn't an error for us
			if runtime.GOOS == "windows" {
				errStr := strings.ToLower(err.Error())
				if strings.Contains(errStr, "exit status 1") {
					// Suppress noisy log on graceful stop
					return
				}
			}
			logger.Error("Failure in running xray-core:", err)
		}
	}()

	return nil
}

// writeConfigAtomic writes beside the destination and renames only after the
// complete file has been flushed. A failed marshal/write/rename leaves the
// previous runtime config intact.
func writeConfigAtomic(configPath string, data []byte) error {
	dir := filepath.Dir(configPath)
	tmp, err := os.CreateTemp(dir, ".xray-config-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		return err
	}
	return nil
}

// Stop terminates the running Xray process.
func (p *process) Stop() error {
	p.lifecycleMu.Lock()
	defer p.lifecycleMu.Unlock()
	p.mu.RLock()
	cmd := p.cmd
	done := p.done
	configPath := p.configPath
	p.mu.RUnlock()
	if !p.IsRunning() || cmd == nil || cmd.Process == nil {
		return errors.New("xray is not running")
	}

	var err error
	if runtime.GOOS == "windows" {
		err = cmd.Process.Kill()
	} else {
		err = cmd.Process.Signal(syscall.SIGTERM)
	}
	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	select {
	case <-done:
		if configPath != "" && configPath != GetConfigPath() {
			_ = os.Remove(configPath)
		}
		return nil
	case <-time.After(5 * time.Second):
		if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return err
		}
	}
	select {
	case <-done:
		if configPath != "" && configPath != GetConfigPath() {
			_ = os.Remove(configPath)
		}
		return nil
	case <-time.After(2 * time.Second):
		return errors.New("xray did not exit after termination")
	}
}

// writeCrashReport writes a crash report to the binary folder with a timestamped filename.
func writeCrashReport(m []byte) error {
	crashReportPath := config.GetBinFolderPath() + "/core_crash_" + time.Now().Format("20060102_150405") + ".log"
	return os.WriteFile(crashReportPath, m, 0o600)
}
