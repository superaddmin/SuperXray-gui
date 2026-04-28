package global

import (
	"strings"
	"testing"
	"time"
)

func TestHashStorageUsesSHA256LengthHashes(t *testing.T) {
	storage := NewHashStorage(time.Minute)

	hash := storage.SaveHash("query")

	if len(hash) != 64 {
		t.Fatalf("hash length = %d, want 64", len(hash))
	}
	if !storage.IsHash(hash) {
		t.Fatalf("expected generated hash to pass IsHash: %s", hash)
	}
	if storage.IsHash(strings.Repeat("a", 32)) {
		t.Fatal("32-character MD5-style hash should not pass IsHash")
	}
}
