package service

import "testing"

func TestParseCommandValueRejectsMalformedOutput(t *testing.T) {
	if _, err := parseCommandValue([]string{"Private key"}, 0, "private key"); err == nil {
		t.Fatal("parseCommandValue accepted output without ':'")
	}
	if _, err := parseCommandValue([]string{}, 0, "private key"); err == nil {
		t.Fatal("parseCommandValue accepted missing output")
	}
}

func TestValidateECHServerNameRejectsArgumentLikeInput(t *testing.T) {
	if err := validateECHServerName("--help"); err == nil {
		t.Fatal("validateECHServerName accepted argument-like input")
	}
	if err := validateECHServerName("example.com"); err != nil {
		t.Fatalf("validateECHServerName rejected valid input: %v", err)
	}
}
