package controller

import (
	"errors"
	"strings"
	"testing"
)

func TestRedactLoginFailurePasswordNeverReturnsRawPassword(t *testing.T) {
	rawPassword := "correct-horse-battery-staple"

	got := redactLoginFailurePassword(rawPassword, errors.New("invalid credentials"), "2FA failed")

	if strings.Contains(got, rawPassword) {
		t.Fatalf("failed login notification must not contain raw password: %q", got)
	}
	if got != "***" {
		t.Fatalf("failed login notification = %q, want fixed mask", got)
	}
}

func TestRedactLoginFailurePasswordKeeps2FAReasonWithoutRawPassword(t *testing.T) {
	rawPassword := "real-admin-password"

	got := redactLoginFailurePassword(rawPassword, errors.New("invalid 2fa code"), "2FA failed")

	if strings.Contains(got, rawPassword) {
		t.Fatalf("2FA failure notification must not contain raw password: %q", got)
	}
	if got != "*** (2FA failed)" {
		t.Fatalf("2FA failure notification = %q, want masked password with reason", got)
	}
}
