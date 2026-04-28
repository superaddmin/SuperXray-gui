package controller

import "testing"

func TestValidateImportDBFileSizeRejectsOversizedUpload(t *testing.T) {
	if err := validateImportDBFileSize(maxImportDBFileSize + 1); err == nil {
		t.Fatal("expected oversized upload to be rejected")
	}
}

func TestValidateImportDBFileSizeAllowsConfiguredLimit(t *testing.T) {
	if err := validateImportDBFileSize(maxImportDBFileSize); err != nil {
		t.Fatalf("expected upload at configured limit to be allowed: %v", err)
	}
}
