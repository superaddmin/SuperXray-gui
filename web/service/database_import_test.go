package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReplacePreparedDatabaseSuccessRemovesRollback(t *testing.T) {
	dir := t.TempDir()
	active := filepath.Join(dir, "active.db")
	candidate := filepath.Join(dir, "candidate.db")
	if err := os.WriteFile(active, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	opened := false
	lifecycle := databaseImportLifecycle{
		checkpoint: func() error { return nil },
		close:      func() error { return nil },
		open:       func(string) error { opened = true; return nil },
	}
	if err := replacePreparedDatabase(candidate, active, lifecycle); err != nil {
		t.Fatalf("replace failed: %v", err)
	}
	got, _ := os.ReadFile(active)
	if string(got) != "new" || !opened {
		t.Fatalf("active database = %q, opened = %v", got, opened)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".superxray-rollback-*.db"))
	if len(matches) != 0 {
		t.Fatalf("rollback files remain: %v", matches)
	}
}

func TestReplacePreparedDatabaseRestoresOriginalWhenImportedOpenFails(t *testing.T) {
	dir := t.TempDir()
	active := filepath.Join(dir, "active.db")
	candidate := filepath.Join(dir, "candidate.db")
	if err := os.WriteFile(active, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	openCalls := 0
	lifecycle := databaseImportLifecycle{
		checkpoint: func() error { return nil },
		close:      func() error { return nil },
		open: func(string) error {
			openCalls++
			if openCalls == 1 {
				return errors.New("candidate migration failed")
			}
			return nil
		},
	}
	if err := replacePreparedDatabase(candidate, active, lifecycle); err == nil {
		t.Fatal("replace unexpectedly succeeded")
	}
	got, _ := os.ReadFile(active)
	if string(got) != "old" {
		t.Fatalf("original database was not restored: %q", got)
	}
}

func TestReplacePreparedDatabaseRestoresOriginalWhenImportedStartFails(t *testing.T) {
	dir := t.TempDir()
	active := filepath.Join(dir, "active.db")
	candidate := filepath.Join(dir, "candidate.db")
	if err := os.WriteFile(active, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	starts := 0
	lifecycle := databaseImportLifecycle{
		running:    true,
		checkpoint: func() error { return nil },
		stop:       func() error { return nil },
		close:      func() error { return nil },
		open:       func(string) error { return nil },
		start: func() error {
			starts++
			if starts == 1 {
				return errors.New("candidate Xray rejected")
			}
			return nil
		},
	}
	if err := replacePreparedDatabase(candidate, active, lifecycle); err == nil {
		t.Fatal("replace unexpectedly succeeded")
	}
	got, _ := os.ReadFile(active)
	if string(got) != "old" {
		t.Fatalf("original database was not restored after start failure: %q", got)
	}
}
