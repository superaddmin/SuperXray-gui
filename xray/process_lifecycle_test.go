package xray

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// This helper exercises real process creation, output pipes and Wait on every
// supported OS without requiring an installed Xray binary or network listener.
func TestProcessHelper(t *testing.T) {
	if os.Getenv("SUPERXRAY_PROCESS_HELPER") != "1" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "wait":
		time.Sleep(30 * time.Second)
	case "fail":
		os.Exit(23)
	case "success":
		os.Exit(0)
	case "diagnostic":
		fmt.Fprint(os.Stderr, "private-fixture-credential"+strings.Repeat("x", 8192))
		os.Exit(3)
	case "output":
		fmt.Fprint(os.Stdout, strings.Repeat("x", 8192))
	}
	os.Exit(0)
}

func processHelperCommand(t *testing.T, mode string) *exec.Cmd {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestProcessHelper$", "--", mode)
	cmd.Env = append(os.Environ(), "SUPERXRAY_PROCESS_HELPER=1", "GORACE=atexit_sleep_ms=0")
	return cmd
}

func TestStartWithConfigRestoresPreviousFileOnFailure(t *testing.T) {
	for _, mode := range []string{"missing-executable", "fail", "success"} {
		t.Run(mode, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.json")
			previous := []byte("previous working configuration\n")
			if err := os.WriteFile(configPath, previous, 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := processHelperCommand(t, mode)
			if mode == "missing-executable" {
				cmd = exec.Command(filepath.Join(t.TempDir(), "missing-xray"))
			}
			p := newProcess(&Config{})
			if err := p.startWithConfig(configPath, []byte("candidate"), cmd); err == nil {
				t.Fatal("startup unexpectedly succeeded")
			}
			if p.IsRunning() {
				t.Fatal("failed child still reported running")
			}
			actual, err := os.ReadFile(configPath)
			if err != nil || !bytes.Equal(actual, previous) {
				t.Fatalf("previous file not restored: %q, %v", actual, err)
			}
		})
	}
}

func TestStartWithConfigRemovesFailedFirstConfiguration(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	p := newProcess(&Config{})
	if err := p.startWithConfig(configPath, []byte("candidate"), processHelperCommand(t, "fail")); err == nil {
		t.Fatal("startup unexpectedly succeeded")
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("failed initial startup left a config file: %v", err)
	}
}

func TestProcessLifecycleConcurrentReaders(t *testing.T) {
	p := &Process{newProcess(&Config{})}
	configPath := filepath.Join(t.TempDir(), "config.json")
	done := make(chan struct{})
	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for {
				select {
				case <-done:
					return
				default:
					p.IsRunning()
					p.GetErr()
					p.GetResult()
					p.GetUptime()
					p.GetVersion()
					p.GetAPIPort()
					p.GetConfig()
					p.SetOnlineClients([]string{"fixture"})
					p.GetOnlineClients()
				}
			}
		})
	}
	t.Cleanup(func() {
		close(done)
		readers.Wait()
		if p.IsRunning() {
			_ = p.Stop()
		}
	})
	for range 2 {
		if err := p.startWithConfig(configPath, []byte("candidate"), processHelperCommand(t, "wait")); err != nil {
			t.Fatalf("start: %v", err)
		}
		if !p.IsRunning() {
			t.Fatal("started child not running")
		}
		if err := p.Stop(); err != nil {
			t.Fatalf("stop: %v", err)
		}
		if p.IsRunning() {
			t.Fatal("Stop returned before the child exited")
		}
	}
}

func TestWriteConfigAtomicDoesNotRemoveDestinationOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "occupied")
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(destination, "keep")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeConfigAtomic(destination, []byte("candidate")); err == nil {
		t.Fatal("rename over directory unexpectedly succeeded")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("original destination damaged: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".xray-config-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files leaked: %v, %v", matches, err)
	}
}
