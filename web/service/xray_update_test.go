package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/util/json_util"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

func TestCoreUpdateUsesConfiguredSupplementalAssetDirectory(t *testing.T) {
	t.Setenv("XRAY_LOCATION_ASSET", "process-assets")
	t.Setenv("xray.location.asset", "process-dotted-assets")
	cfg := &xray.Config{Env: json_util.RawMessage(`{"XRAY_LOCATION_ASSET":"config-assets"}`)}
	dir, err := coreAssetDirectory(cfg)
	want, _ := filepath.Abs("process-dotted-assets")
	if err != nil || dir != want {
		t.Fatalf("asset alias precedence = %s %v", dir, err)
	}
	cfg.Env = json_util.RawMessage(`{"xray.location.asset":"custom-assets"}`)
	dir, err = coreAssetDirectory(cfg)
	want, _ = filepath.Abs("custom-assets")
	if err != nil || dir != want {
		t.Fatalf("config asset override = %s %v", dir, err)
	}
}

func TestCoreUpdatePreservesSupplementalGeoForValidation(t *testing.T) {
	source, stage := t.TempDir(), t.TempDir()
	for _, name := range []string{"geoip.dat", "geosite.dat", "geosite_custom.dat", "geoip_IR.dat", "config.json"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte("original-"+name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(stage, "geoip.dat"), []byte("new-official"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := stageSupplementalGeo(source, stage); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"geoip.dat": "new-official", "geosite_custom.dat": "original-geosite_custom.dat", "geoip_IR.dat": "original-geoip_IR.dat"} {
		got, err := os.ReadFile(filepath.Join(stage, name))
		if err != nil || string(got) != want {
			t.Fatalf("staged %s = %q, %v", name, got, err)
		}
	}
	if _, err := os.Stat(filepath.Join(stage, "config.json")); !os.IsNotExist(err) {
		t.Fatal("staging copied unrelated private configuration")
	}
}

func TestCoreVersionSelectionRejectsOldAndMalformedReleases(t *testing.T) {
	for _, version := range []string{"v26.10.10", "v26.11.1", "v27.1.1"} {
		if !supportedCoreVersion(version) {
			t.Fatalf("rejected supported release %s", version)
		}
	}
	for _, version := range []string{"", "v26.3.10", "v26.10.9", "v26.13.1", "v26.10.10-beta", "v26.10.10/evil", "v26.10.10?x"} {
		if supportedCoreVersion(version) {
			t.Fatalf("accepted unsupported release %s", version)
		}
	}
}

func TestCoreArchiveRequiresOfficialSHA256(t *testing.T) {
	payload := []byte("test archive")
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	path := filepath.Join(t.TempDir(), "Xray-test.zip")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	originalClient := serviceHTTPClient
	t.Cleanup(func() { serviceHTTPClient = originalClient })
	for _, test := range []struct {
		name, body string
		status     int
		valid      bool
	}{
		{"official", "MD5= ignored\nSHA2-256= " + digest + "\n", http.StatusOK, true},
		{"wrong", "SHA2-256= " + strings.Repeat("0", 64), http.StatusOK, false},
		{"unlabelled", digest, http.StatusOK, false},
		{"duplicate", "SHA2-256= " + digest + "\nSHA2-256= " + digest, http.StatusOK, false},
		{"missing", "not found", http.StatusNotFound, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			serviceHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Scheme != "https" || req.URL.Host != "github.com" || !strings.HasSuffix(req.URL.Path, "/v26.10.10/Xray-test.zip.dgst") {
					t.Fatalf("unexpected digest request: %s", req.URL)
				}
				return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})}
			err := (&ServerService{}).verifyXrayArchive(path, "v26.10.10")
			if (err == nil) != test.valid {
				t.Fatalf("verification error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func TestCoreUpdateRestoresOriginalFilesOnPartialInstallAndStartFailure(t *testing.T) {
	for _, failure := range []string{"partial-install", "startup", "stopped"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			var files []coreFileReplacement
			for _, name := range []string{"core", "geo", "optional"} {
				item := coreFileReplacement{target: filepath.Join(dir, name), staged: filepath.Join(dir, name+".new"), backup: filepath.Join(dir, name+".backup")}
				if name != "optional" {
					if err := os.WriteFile(item.target, []byte("old-"+name), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(item.staged, []byte("new-"+name), 0o600); err != nil {
					t.Fatal(err)
				}
				files = append(files, item)
			}
			if failure == "partial-install" {
				if err := os.Remove(files[1].staged); err != nil {
					t.Fatal(err)
				}
			}
			running := failure != "stopped"
			starts := 0
			retain, err := installCoreFiles(files, coreUpdateLifecycle{
				running: func() bool { return running }, stop: func() error { running = false; return nil },
				start: func() error {
					starts++
					if failure == "startup" && starts == 1 {
						return errors.New("candidate startup failed")
					}
					running = true
					return nil
				},
			})
			if failure == "stopped" {
				if err != nil || retain || starts != 0 || running {
					t.Fatalf("stopped core state changed: retain=%v starts=%d err=%v", retain, starts, err)
				}
				return
			}
			if err == nil || retain || !running {
				t.Fatalf("failed update did not recover: retain=%v running=%v err=%v", retain, running, err)
			}
			for _, item := range files[:2] {
				data, readErr := os.ReadFile(item.target)
				if readErr != nil || string(data) != "old-"+filepath.Base(item.target) {
					t.Fatal("old installation was not restored")
				}
			}
			if _, err := os.Stat(files[2].target); !os.IsNotExist(err) {
				t.Fatal("rollback left a newly created file")
			}
		})
	}
}

func TestCoreUpdateRetainsBackupWhenRollbackFails(t *testing.T) {
	dir := t.TempDir()
	item := coreFileReplacement{target: filepath.Join(dir, "core"), staged: filepath.Join(dir, "new"), backup: filepath.Join(dir, "backup")}
	if err := os.WriteFile(item.target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(item.staged, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	running := true
	retain, err := installCoreFiles([]coreFileReplacement{item}, coreUpdateLifecycle{
		running: func() bool { return running }, stop: func() error { running = false; return nil },
		start: func() error {
			if err := os.Remove(item.target); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(item.target, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(item.target, "locked"), []byte("busy"), 0o600); err != nil {
				t.Fatal(err)
			}
			return errors.New("startup failed")
		},
	})
	if !retain || err == nil || !strings.Contains(err.Error(), "remove candidate") {
		t.Fatalf("lost recovery failure: retain=%v err=%v", retain, err)
	}
	data, readErr := os.ReadFile(item.backup)
	if readErr != nil || string(data) != "old" {
		t.Fatal("rollback failure destroyed the original backup")
	}
}
