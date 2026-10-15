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
	"strings"
	"time"
)

type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	written := len(p)
	if b.limit > 0 && b.Len() < b.limit {
		n := b.limit - b.Len()
		if len(p) > n {
			p = p[:n]
		}
		_, _ = b.Buffer.Write(p)
	}
	return written, nil
}

// ValidateConfig asks the installed Xray binary to parse a configuration
// without starting a listener. It is intentionally performed before replacing
// the active config file so a schema error cannot take down a working process.
func ValidateConfig(config *Config) error {
	return validateConfigWithBinary(config, GetBinaryPath(), false)
}

// ValidateConfigWithBinary validates a configuration against a staged core and
// its assets before the updater replaces the running installation.
func ValidateConfigWithBinary(config *Config, binaryPath string) error {
	return validateConfigWithBinary(config, binaryPath, true)
}

func validateConfigWithBinary(config *Config, binaryPath string, staged bool) error {
	if config == nil {
		return errors.New("xray configuration is nil")
	}
	binaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return fmt.Errorf("resolve Xray binary path: %w", err)
	}
	validationConfig := *config
	if staged {
		// Xray applies config.env during Build, after reading the process
		// environment. Override both accepted asset aliases on the private
		// validation copy so old assets cannot hide an incomplete update.
		var environment map[string]string
		if len(config.Env) > 0 {
			if err := json.Unmarshal(config.Env, &environment); err != nil {
				return errors.New("xray env must contain string values")
			}
		}
		if environment == nil {
			environment = make(map[string]string)
		}
		delete(environment, "xray.location.asset")
		environment["XRAY_LOCATION_ASSET"] = filepath.Dir(binaryPath)
		validationConfig.Env, err = json.Marshal(environment)
		if err != nil {
			return fmt.Errorf("marshal Xray validation environment: %w", err)
		}
	}
	data, err := json.MarshalIndent(&validationConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal Xray configuration: %w", err)
	}
	dir, err := filepath.Abs(filepath.Dir(GetConfigPath()))
	if err != nil {
		return fmt.Errorf("resolve Xray config directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create Xray config directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".xray-validate-*.json")
	if err != nil {
		return fmt.Errorf("create Xray validation file: %w", err)
	}
	path := tmp.Name()
	defer os.Remove(path)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("protect Xray validation file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write Xray validation file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close Xray validation file: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// #nosec G204 -- configured local binary or private staged release; fixed arguments, no shell.
	cmd := exec.CommandContext(ctx, binaryPath, "run", "-test", "-config", path)
	if staged {
		cmd.Env = xrayAssetEnvironment(filepath.Dir(binaryPath))
	}
	return runValidationCommand(ctx, cmd)
}

func xrayAssetEnvironment(assetDir string) []string {
	environment := make([]string, 0, len(os.Environ())+1)
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if strings.EqualFold(key, "XRAY_LOCATION_ASSET") || strings.EqualFold(key, "xray.location.asset") {
			continue
		}
		environment = append(environment, value)
	}
	return append(environment, "XRAY_LOCATION_ASSET="+assetDir)
}

func runValidationCommand(ctx context.Context, cmd *exec.Cmd) error {
	var output cappedBuffer
	output.limit = 4096
	cmd.Stdout = &output
	cmd.Stderr = &output
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("xray configuration validation timed out: %w", ctx.Err())
		}
		// Core diagnostics may echo passwords, private keys, or an entire
		// offending object. Never return the captured output to panel logs/API.
		return fmt.Errorf("xray rejected configuration; check protocol, transport, certificate and geo-file settings: %w", err)
	}
	return nil
}
