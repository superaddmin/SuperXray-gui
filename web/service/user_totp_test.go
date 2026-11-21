package service

import (
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/crypto"
	"github.com/xlzd/gotp"
)

func TestLoginReadsFormattedAndMalformedStoredTOTP(t *testing.T) {
	users := newFakeUserRepository()
	password, err := crypto.HashPasswordAsBcrypt("fixture")
	if err != nil {
		t.Fatal(err)
	}
	users.users[1] = &model.User{Id: 1, Username: "admin", Password: password}
	settings := NewSettingService(newFakeSettingRepository())
	if err := settings.SetTwoFactorEnable(true); err != nil {
		t.Fatal(err)
	}
	svc := NewUserService(users, *settings)
	for _, tc := range []struct {
		secret string
		valid  bool
	}{
		{"jbsw\u00a0y3dp-ehpk 3pxp=", true}, {"bad!", false}, {"===", false}, {"A", false},
	} {
		if err := settings.SetTwoFactorToken(tc.secret); err != nil {
			t.Fatal(err)
		}
		user, err := svc.CheckUser("admin", "fixture", gotp.NewDefaultTOTP(testOriginalTOTPSecret).Now())
		if tc.valid != (user != nil && err == nil) {
			t.Errorf("unexpected TOTP login result for formatted=%t", tc.valid)
		}
	}
}
