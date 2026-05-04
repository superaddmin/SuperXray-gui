package singbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/config"
	panelcore "github.com/superaddmin/SuperXray-gui/v2/core"
)

const (
	defaultConfigName = "sing-box-config.json"
	validateTimeout   = 10 * time.Second
)

var (
	ErrBinaryNotFound = errors.New("sing-box binary not found")
	ErrConfigNotFound = errors.New("sing-box config not found")
)

type Options struct {
	BinaryPath string
	ConfigPath string
	LogDir     string
}

type Adapter struct {
	mu       sync.Mutex
	options  Options
	cmd      *exec.Cmd
	running  bool
	lastErr  string
	lastExit time.Time
}

func DefaultOptions() Options {
	binaryPath := strings.TrimSpace(os.Getenv("SUPERXRAY_SING_BOX_BINARY"))
	if binaryPath == "" {
		binaryPath = filepath.Join(config.GetBinFolderPath(), defaultBinaryName())
	}

	configPath := strings.TrimSpace(os.Getenv("SUPERXRAY_SING_BOX_CONFIG"))
	if configPath == "" {
		configPath = filepath.Join(config.GetBinFolderPath(), defaultConfigName)
	}

	logDir := strings.TrimSpace(os.Getenv("SUPERXRAY_SING_BOX_LOG_FOLDER"))
	if logDir == "" {
		logDir = config.GetLogFolder()
	}

	return Options{
		BinaryPath: binaryPath,
		ConfigPath: configPath,
		LogDir:     logDir,
	}
}

func NewAdapter(options Options) *Adapter {
	if strings.TrimSpace(options.BinaryPath) == "" {
		options.BinaryPath = filepath.Join(config.GetBinFolderPath(), defaultBinaryName())
	}
	if strings.TrimSpace(options.ConfigPath) == "" {
		options.ConfigPath = filepath.Join(config.GetBinFolderPath(), defaultConfigName)
	}
	if strings.TrimSpace(options.LogDir) == "" {
		options.LogDir = config.GetLogFolder()
	}
	return &Adapter{options: options}
}

func (a *Adapter) Instance() panelcore.Instance {
	return panelcore.Instance{
		ID:             "experimental-sing-box",
		Name:           "experimental-sing-box",
		DisplayName:    "Experimental sing-box",
		CoreType:       panelcore.CoreTypeSingBox,
		Mode:           "experimental",
		Source:         "external-binary",
		LifecycleOwner: "core-manager",
		Capabilities: panelcore.Capabilities{
			Read:                    true,
			Write:                   false,
			Validate:                true,
			Start:                   true,
			Stop:                    true,
			Restart:                 true,
			LifecycleViaCoreManager: true,
		},
		WriteSupported:   false,
		ManagerAttached:  true,
		ExperimentalOnly: true,
	}
}

func (a *Adapter) Status(context.Context) (panelcore.Status, error) {
	if err := a.ensureBinary(); err != nil {
		return a.status(panelcore.StateNotInstalled, err.Error(), 0), nil
	}
	if err := a.ensureConfig(); err != nil {
		return a.status(panelcore.StateNotConfigured, err.Error(), 0), nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running && a.cmd != nil && a.cmd.Process != nil {
		return a.statusLocked(panelcore.StateRunning, "", a.cmd.Process.Pid), nil
	}
	return a.statusLocked(panelcore.StateStopped, a.lastErr, 0), nil
}

func (a *Adapter) Validate(ctx context.Context) (panelcore.LifecycleResult, error) {
	if err := a.ensureBinary(); err != nil {
		return result(panelcore.StateNotInstalled, "", err), err
	}
	if err := a.ensureConfig(); err != nil {
		return result(panelcore.StateNotConfigured, "", err), err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, validateTimeout)
	defer cancel()

	// #nosec G204 -- binary path is configured by the administrator and args are passed without a shell.
	cmd := exec.CommandContext(timeoutCtx, a.options.BinaryPath, "check", "-c", a.options.ConfigPath)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("sing-box check timed out after %s", validateTimeout)
		}
		msg := strings.TrimSpace(output.String())
		if msg == "" {
			msg = err.Error()
		}
		return panelcore.LifecycleResult{State: panelcore.StateError, ErrorMsg: msg}, err
	}
	return panelcore.LifecycleResult{State: panelcore.StateStopped, Msg: "sing-box config is valid"}, nil
}

func (a *Adapter) Start(context.Context) (panelcore.LifecycleResult, error) {
	if err := a.ensureBinary(); err != nil {
		return result(panelcore.StateNotInstalled, "", err), err
	}
	if err := a.ensureConfig(); err != nil {
		return result(panelcore.StateNotConfigured, "", err), err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running && a.cmd != nil && a.cmd.Process != nil {
		return panelcore.LifecycleResult{State: panelcore.StateRunning, PID: a.cmd.Process.Pid, Msg: "sing-box is already running"}, nil
	}

	if err := os.MkdirAll(a.options.LogDir, 0o750); err != nil {
		return result(panelcore.StateError, "", err), err
	}
	stdout, err := os.OpenFile(filepath.Join(a.options.LogDir, "sing-box.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return result(panelcore.StateError, "", err), err
	}
	stderr, err := os.OpenFile(filepath.Join(a.options.LogDir, "sing-box-error.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		_ = stdout.Close()
		return result(panelcore.StateError, "", err), err
	}

	// #nosec G204 -- binary path is configured by the administrator and args are passed without a shell.
	cmd := exec.Command(a.options.BinaryPath, "run", "-c", a.options.ConfigPath)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		return result(panelcore.StateError, "", err), err
	}

	a.cmd = cmd
	a.running = true
	a.lastErr = ""
	go a.wait(cmd, stdout, stderr)

	return panelcore.LifecycleResult{State: panelcore.StateRunning, PID: cmd.Process.Pid, Msg: "sing-box started"}, nil
}

func (a *Adapter) Stop(context.Context) (panelcore.LifecycleResult, error) {
	a.mu.Lock()
	cmd := a.cmd
	if !a.running || cmd == nil || cmd.Process == nil {
		a.mu.Unlock()
		return panelcore.LifecycleResult{State: panelcore.StateStopped, Msg: "sing-box is not running"}, nil
	}
	pid := cmd.Process.Pid
	a.mu.Unlock()

	if err := cmd.Process.Kill(); err != nil {
		return result(panelcore.StateError, "", err), err
	}

	a.mu.Lock()
	a.running = false
	a.lastErr = ""
	a.lastExit = time.Now()
	a.mu.Unlock()

	return panelcore.LifecycleResult{State: panelcore.StateStopped, PID: pid, Msg: "sing-box stopped"}, nil
}

func (a *Adapter) Restart(ctx context.Context) (panelcore.LifecycleResult, error) {
	if _, err := a.Stop(ctx); err != nil {
		return result(panelcore.StateError, "", err), err
	}
	return a.Start(ctx)
}

func (a *Adapter) wait(cmd *exec.Cmd, stdout *os.File, stderr *os.File) {
	defer stdout.Close()
	defer stderr.Close()

	err := cmd.Wait()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cmd == cmd {
		a.running = false
		a.cmd = nil
		a.lastExit = time.Now()
		if err != nil {
			a.lastErr = err.Error()
		}
	}
}

func (a *Adapter) ensureBinary() error {
	info, err := os.Stat(a.options.BinaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrBinaryNotFound, a.options.BinaryPath)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%w: %s", ErrBinaryNotFound, a.options.BinaryPath)
	}
	return nil
}

func (a *Adapter) ensureConfig() error {
	info, err := os.Stat(a.options.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrConfigNotFound, a.options.ConfigPath)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%w: %s", ErrConfigNotFound, a.options.ConfigPath)
	}
	return nil
}

func (a *Adapter) status(state panelcore.State, errorMsg string, pid int) panelcore.Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.statusLocked(state, errorMsg, pid)
}

func (a *Adapter) statusLocked(state panelcore.State, errorMsg string, pid int) panelcore.Status {
	return panelcore.Status{
		State:    state,
		ErrorMsg: errorMsg,
		PID:      pid,
		Binary:   a.options.BinaryPath,
		Config:   a.options.ConfigPath,
	}
}

func result(state panelcore.State, msg string, err error) panelcore.LifecycleResult {
	res := panelcore.LifecycleResult{State: state, Msg: msg}
	if err != nil {
		res.ErrorMsg = err.Error()
	}
	return res
}

func defaultBinaryName() string {
	if runtime.GOOS == "windows" {
		return "sing-box.exe"
	}
	return "sing-box"
}
