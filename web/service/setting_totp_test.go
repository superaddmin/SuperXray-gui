package service

import (
	"strings"
	"testing"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/xlzd/gotp"
)

const testOriginalTOTPSecret = "JBSWY3DPEHPK3PXP"
const testReplacementTOTPSecret = "KRUGS4ZANFZSAYJA"

func TestSettingsTOTPChangesRequireProofAndNeverExposeStoredSecret(t *testing.T) {
	setupSettingServiceTestDB(t)
	svc := &SettingService{}
	if err := svc.SetTwoFactorToken(testOriginalTOTPSecret); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetTwoFactorEnable(true); err != nil {
		t.Fatal(err)
	}
	current, err := svc.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	if current.TwoFactorToken != "" {
		t.Fatal("settings response exposed the saved secret")
	}
	// An ordinary save of the redacted response preserves the second factor.
	if err := svc.UpdateAllSetting(current); err != nil {
		t.Fatal(err)
	}
	assertTOTPState(t, svc, true, testOriginalTOTPSecret)
	for _, enabled := range []bool{true, false} {
		for _, code := range []string{"", "invalid", gotp.NewDefaultTOTP(testReplacementTOTPSecret).At(time.Now().Unix() - 300)} {
			changed := *current
			changed.TwoFactorEnable = enabled
			changed.TwoFactorToken = testReplacementTOTPSecret
			changed.WebPort = 9999
			if err := svc.UpdateAllSetting(&changed, code); err == nil {
				t.Fatal("security change accepted without a current code")
			}
			assertTOTPState(t, svc, true, testOriginalTOTPSecret)
			if port, _ := svc.GetPort(); port == 9999 {
				t.Fatal("rejected request partially saved unrelated settings")
			}
		}
	}
	current.TwoFactorToken = testReplacementTOTPSecret
	if err := svc.UpdateAllSetting(current, gotp.NewDefaultTOTP(testOriginalTOTPSecret).Now()); err != nil {
		t.Fatal(err)
	}
	assertTOTPState(t, svc, true, testReplacementTOTPSecret)
	current.TwoFactorEnable = false
	current.TwoFactorToken = ""
	if err := svc.UpdateAllSetting(current, gotp.NewDefaultTOTP(testReplacementTOTPSecret).Now()); err != nil {
		t.Fatal(err)
	}
	assertTOTPState(t, svc, false, "")
}

func TestSettingsTOTPEnrollmentValidatesNewSecretAndCode(t *testing.T) {
	setupSettingServiceTestDB(t)
	svc := &SettingService{}
	settings := validAllSettingForServiceTest()
	settings.TwoFactorEnable = true
	for _, invalidSecret := range []string{"", "invalid!", "AAAA", "AAAAAAAAAAAAAAAA!"} {
		settings.TwoFactorToken = invalidSecret
		if err := svc.UpdateAllSetting(settings, "123456"); err == nil {
			t.Fatal("invalid enrollment accepted")
		}
		assertTOTPState(t, svc, false, "")
	}
	settings.TwoFactorToken = testOriginalTOTPSecret
	if err := svc.UpdateAllSetting(settings); err == nil {
		t.Fatal("enrollment without code accepted")
	}
	settings.TwoFactorToken = strings.ToLower(testOriginalTOTPSecret) + "="
	if err := svc.UpdateAllSetting(settings, gotp.NewDefaultTOTP(testOriginalTOTPSecret).Now()); err != nil {
		t.Fatal(err)
	}
	assertTOTPState(t, svc, true, testOriginalTOTPSecret)
	var count int64
	if err := database.GetDB().Model(&model.Setting{}).Where("key = ?", "twoFactorCode").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("confirmation code was persisted")
	}
}

func TestSettingsTOTPWriteFailureRollsBackEnableFlagAndOtherSettings(t *testing.T) {
	setupSettingServiceTestDB(t)
	svc := &SettingService{}
	if err := database.GetDB().Exec(`CREATE TRIGGER reject_totp_insert BEFORE INSERT ON settings WHEN NEW.key = 'twoFactorToken' BEGIN SELECT RAISE(ABORT, 'injected write failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	settings := validAllSettingForServiceTest()
	settings.WebPort = 9999
	settings.TwoFactorEnable = true
	settings.TwoFactorToken = testOriginalTOTPSecret
	if err := svc.UpdateAllSetting(settings, gotp.NewDefaultTOTP(testOriginalTOTPSecret).Now()); err == nil {
		t.Fatal("expected injected write failure")
	}
	assertTOTPState(t, svc, false, "")
	if port, _ := svc.GetPort(); port != 2053 {
		t.Fatal("failed security save partially persisted other settings")
	}
}

func TestSettingsMalformedStoredTOTPSecretReturnsErrorWithoutPanic(t *testing.T) {
	setupSettingServiceTestDB(t)
	svc := &SettingService{}
	if err := svc.SetTwoFactorToken("invalid!"); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetTwoFactorEnable(true); err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateAllSetting(validAllSettingForServiceTest(), "123456"); err == nil {
		t.Fatal("malformed stored secret accepted")
	}
}

func assertTOTPState(t *testing.T, svc *SettingService, enabled bool, secret string) {
	t.Helper()
	gotEnabled, err := svc.GetTwoFactorEnable()
	if err != nil {
		t.Fatal(err)
	}
	gotSecret, err := svc.GetTwoFactorToken()
	if err != nil {
		t.Fatal(err)
	}
	if gotEnabled != enabled || gotSecret != secret {
		t.Fatal("stored two-factor state differs from expected state")
	}
}
