package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRelativeUnderRejectsPathEscapingRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside.txt")

	if _, err := RelativeUnder(root, outside); err == nil {
		t.Fatal("expected path outside root to be rejected")
	}

	if _, err := RelativeUnder(root, filepath.Join(root, "..", "outside.txt")); err == nil {
		t.Fatal("expected traversal outside root to be rejected")
	}
}

func TestOpenFileUnderAllowsNestedFileInsideRoot(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "nested", "config.json")

	f, err := OpenFileUnder(root, target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("OpenFileUnder returned error: %v", err)
	}
	if _, err := f.WriteString("ok"); err != nil {
		t.Fatalf("write returned error: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close returned error: %v", err)
	}

	got, err := ReadFileUnder(root, target)
	if err != nil {
		t.Fatalf("ReadFileUnder returned error: %v", err)
	}
	if string(got) != "ok" {
		t.Fatalf("expected %q, got %q", "ok", string(got))
	}
}
