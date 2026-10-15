package xray

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCappedBufferReportsConsumedBytes(t *testing.T) {
	output := &cappedBuffer{limit: 4}
	for _, input := range []string{"123456789", "another chunk"} {
		n, err := output.Write([]byte(input))
		if err != nil || n != len(input) {
			t.Fatalf("Write = %d, %v; want %d, nil", n, err, len(input))
		}
	}
	if output.String() != "1234" {
		t.Fatalf("output limit not enforced: %q", output.String())
	}
}

func TestValidationDrainsOutputAndRedactsFailures(t *testing.T) {
	if err := runValidationCommand(context.Background(), processHelperCommand(t, "output")); err != nil {
		t.Fatalf("valid process with long output failed: %v", err)
	}
	err := runValidationCommand(context.Background(), processHelperCommand(t, "diagnostic"))
	if err == nil {
		t.Fatal("rejected configuration unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), "private-fixture-credential") || len(err.Error()) > 512 {
		t.Fatal("validation error exposed raw core diagnostics")
	}
}

func TestValidationTimeoutKillsProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	helper := processHelperCommand(t, "wait")
	cmd := exec.CommandContext(ctx, helper.Path, helper.Args[1:]...)
	cmd.Env = helper.Env
	started := time.Now()
	err := runValidationCommand(ctx, cmd)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("validation error = %v, want deadline exceeded", err)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("validation process exceeded its timeout")
	}
	if cmd.ProcessState == nil {
		t.Fatal("timed out validation was not reaped")
	}
}

func TestCandidateEnvironmentOverridesAssetAliases(t *testing.T) {
	t.Setenv("XRAY_LOCATION_ASSET", "old-assets")
	t.Setenv("xray.location.asset", "another-old-assets")
	environment := xrayAssetEnvironment("candidate-assets")
	count := 0
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if strings.EqualFold(key, "XRAY_LOCATION_ASSET") || strings.EqualFold(key, "xray.location.asset") {
			count++
			if entry != "XRAY_LOCATION_ASSET=candidate-assets" {
				t.Fatalf("old assets leaked into candidate environment: %q", entry)
			}
		}
	}
	if count != 1 {
		t.Fatalf("candidate has %d asset overrides, want 1", count)
	}
}
