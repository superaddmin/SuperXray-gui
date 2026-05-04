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

func TestValidateImportDBFileSizeRejectsTinyUpload(t *testing.T) {
	if err := validateImportDBFileSize(minImportDBFileSize - 1); err == nil {
		t.Fatal("expected tiny upload to be rejected")
	}
}

func TestValidateImportDBUploadMetadataAllowsSQLiteNames(t *testing.T) {
	for _, filename := range []string{"x-ui.db", "backup.sqlite", "backup.sqlite3"} {
		if err := validateImportDBUploadMetadata(filename, minImportDBFileSize); err != nil {
			t.Fatalf("expected %q to be allowed: %v", filename, err)
		}
	}
}

func TestValidateImportDBUploadMetadataRejectsUnsafeNames(t *testing.T) {
	for _, filename := range []string{"../x-ui.db", `nested\x-ui.db`, "x-ui.txt", "x-ui.db.exe", ""} {
		if err := validateImportDBUploadMetadata(filename, minImportDBFileSize); err == nil {
			t.Fatalf("expected %q to be rejected", filename)
		}
	}
}
