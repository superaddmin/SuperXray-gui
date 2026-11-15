package main

import (
	"io"
	"os"
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
