package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/config"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/crypto"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"
)

func captureBootstrapOutput(t *testing.T, action func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	original := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = original })
	action()
	os.Stdout = original
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	output, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(output)
}

func TestSettingCLIInitializesEmptyPanelWithoutLoggingCredentials(t *testing.T) {
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	t.Cleanup(func() { _ = database.CloseDB() })
	if err := updateSetting(0, "", "", "", "", false); err != nil {
		t.Fatalf("initialize settings: %v", err)
	}
	output := captureBootstrapOutput(t, func() { showSetting(true) })
	if !strings.Contains(output, "hasDefaultCredential: true") {
		t.Fatal("fresh panel must let install.sh detect unconfigured credentials")
	}
	var count int64
	if err := database.GetDB().Model(&model.User{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("settings inspection created a user: count=%d, err=%v", count, err)
	}
	userService := service.UserService{}
	if user, err := userService.CheckUser("admin", "admin", ""); err == nil || user != nil {
		t.Fatal("unconfigured panel accepted fixed default credentials")
	}

	const username = "bootstrap-test-operator"
	const password = "bootstrap-test-passphrase"
	output = captureBootstrapOutput(t, func() {
		if err := updateSetting(0, username, password, "", "", false); err != nil {
			t.Fatalf("configure credentials: %v", err)
		}
		showSetting(true)
	})
	if strings.Contains(output, username) || strings.Contains(output, password) {
		t.Fatal("credential setup printed sensitive credential values")
	}
	if !strings.Contains(output, "hasDefaultCredential: false") {
		t.Fatal("configured panel is still reported as requiring credential setup")
	}
	if err := service.InitDB(config.GetDBPath()); err != nil {
		t.Fatalf("reopen configured database: %v", err)
	}
	if user, err := userService.CheckUser(username, password, ""); err != nil || user == nil {
		t.Fatalf("configured login did not survive restart: %v", err)
	}
	if err := database.GetDB().Model(&model.User{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("configured database must contain one user: count=%d, err=%v", count, err)
	}
}

func TestSettingCLISnapshot(t *testing.T) {
	if os.Getenv("SUPERXRAY_TEST_SETTING_CLI") == "1" {
		os.Args = []string{os.Args[0], "setting", "-show", "-getCert"}
		main()
		os.Exit(0)
	}
	for _, test := range []struct {
		name, key, value, path, initialize string
	}{
		{"absent", "", "", "/super/", "false"},
		{"explicit empty", "webBasePath", "", "/super/", "true"},
		{"explicit root", "webBasePath", "/", "/super/", "true"},
		{"short path", "webBasePath", "/a", "/a/", "false"},
		{"stored default", "webBasePath", "/super/", "/super/", "false"},
		{"invalid port", "webPort", "invalid", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("XUI_DB_FOLDER", t.TempDir())
			if err := database.InitDB(config.GetDBPath()); err != nil {
				t.Fatal(err)
			}
			if test.key != "" {
				if err := database.GetDB().Create(&model.Setting{Key: test.key, Value: test.value}).Error; err != nil {
					t.Fatal(err)
				}
			}
			if err := database.CloseDB(); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestSettingCLISnapshot$")
			cmd.Env = append(os.Environ(), "SUPERXRAY_TEST_SETTING_CLI=1")
			output, err := cmd.CombinedOutput()
			if test.initialize == "" {
				if err == nil || strings.Contains(string(output), "webBasePath:") || !strings.Contains(string(output), "get current port") {
					t.Fatalf("expected read error without partial settings, got %v, %s", err, output)
				}
				return
			}
			if err != nil {
				t.Fatalf("setting -show -getCert: %v, %s", err, output)
			}
			lines := "\n" + strings.ReplaceAll(string(output), "\r\n", "\n")
			for _, line := range []string{"webBasePath: " + test.path, "initializeWebBasePath: " + test.initialize, "cert: "} {
				if !strings.Contains(lines, "\n"+line+"\n") {
					t.Errorf("snapshot missing %q in %s", line, output)
				}
			}
		})
	}
}

func TestSettingCLIWriteFailureRollsBackCredentialsAndPort(t *testing.T) {
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	t.Cleanup(func() { _ = database.CloseDB() })
	if err := updateSetting(54321, "previous-user", "fixture-initial-password", "", "", false); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Exec(`CREATE TRIGGER reject_install_path BEFORE INSERT ON settings
		WHEN NEW.key = 'webBasePath' BEGIN SELECT RAISE(ABORT, 'fixture write failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	output := captureBootstrapOutput(t, func() {
		err := updateSetting(43210, "replacement-user", "fixture-replacement-password", "/new/", "", false)
		if err == nil || !strings.Contains(err.Error(), "fixture write failure") {
			t.Fatalf("expected forced settings write failure, got %v", err)
		}
	})
	if strings.Contains(output, "成功") {
		t.Fatal("reported a successful change before the transaction committed")
	}
	settings := service.SettingService{}
	if port, err := settings.GetPort(); err != nil || port != 54321 {
		t.Fatalf("previous port was not preserved: port=%d err=%v", port, err)
	}
	users := service.UserService{}
	if user, err := users.CheckUser("previous-user", "fixture-initial-password", ""); err != nil || user == nil {
		t.Fatalf("previous credentials stopped working: %v", err)
	}
}

func TestSettingCLIRecognizesExistingCredentialsWithoutChangingThem(t *testing.T) {
	for _, test := range []struct {
		name     string
		username string
		password string
		want     string
	}{
		{"legacy defaults", "admin", "admin", "hasDefaultCredential: true"},
		{"chosen credentials", "existing-operator", "existing-test-password", "hasDefaultCredential: false"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := database.InitDB(filepath.Join(t.TempDir(), "x-ui.db")); err != nil {
				t.Fatalf("InitDB: %v", err)
			}
			t.Cleanup(func() { _ = database.CloseDB() })
			if err := database.GetDB().Where("1 = 1").Delete(&model.User{}).Error; err != nil {
				t.Fatalf("clear fixture users: %v", err)
			}
			hash, err := crypto.HashPasswordAsBcrypt(test.password)
			if err != nil {
				t.Fatalf("hash fixture: %v", err)
			}
			fixture := model.User{Username: test.username, Password: hash}
			if err := database.GetDB().Create(&fixture).Error; err != nil {
				t.Fatalf("create fixture: %v", err)
			}
			output := captureBootstrapOutput(t, func() { showSetting(true) })
			if !strings.Contains(output, test.want) {
				t.Fatal("existing credential classification changed")
			}
			var stored model.User
			if err := database.GetDB().First(&stored, fixture.Id).Error; err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			if stored.Username != fixture.Username || stored.Password != fixture.Password {
				t.Fatal("settings inspection changed existing credentials")
			}
		})
	}
}
