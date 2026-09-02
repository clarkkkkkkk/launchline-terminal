package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/launchline/launchline/internal/app"
)

func TestLoadCreatesAndRoundTripsConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	repo := NewFileRepository(path)
	cfg, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != app.CurrentSchemaVersion {
		t.Fatalf("version = %d", cfg.Version)
	}
	cfg.CompactLogo = true
	cfg.Applications = append(cfg.Applications, app.Application{ID: "app_1", Name: "Editor", Path: "/bin/editor"})
	if err := repo.Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !got.CompactLogo || len(got.Applications) != 1 || got.Applications[0].Name != "Editor" {
		t.Fatalf("round trip mismatch: %#v", got)
	}
}

func TestMalformedConfigIsPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	original := []byte(`{"version":1, broken`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewFileRepository(path).Load()
	var corrupt *CorruptError
	if !errors.As(err, &corrupt) {
		t.Fatalf("expected CorruptError, got %v", err)
	}
	if corrupt.BackupPath == "" || !strings.Contains(corrupt.BackupPath, ".corrupt-") {
		t.Fatalf("missing safety copy: %#v", corrupt)
	}
	backup, readErr := os.ReadFile(corrupt.BackupPath)
	if readErr != nil || string(backup) != string(original) {
		t.Fatalf("safety copy mismatch: %q, %v", backup, readErr)
	}
	stillThere, _ := os.ReadFile(path)
	if string(stillThere) != string(original) {
		t.Fatal("original corrupt file was changed")
	}
}

func TestSaveRejectsInvalidConfigAndLeavesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	repo := NewFileRepository(path)
	if _, err := repo.Load(); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	invalid := app.DefaultConfig()
	invalid.Version = 99
	if err := repo.Save(invalid); err == nil {
		t.Fatal("expected validation error")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed save changed configuration")
	}
}

func TestLoadRecoversInterruptedWindowsStyleReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	previous := path + ".previous"
	data := []byte("{\n  \"version\": 1,\n  \"applications\": [],\n  \"workspaces\": []\n}\n")
	if err := os.WriteFile(previous, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewFileRepository(path).Load()
	if err != nil || cfg.Version != app.CurrentSchemaVersion {
		t.Fatalf("recovery failed: %#v %v", cfg, err)
	}
	if _, err := os.Stat(previous); !os.IsNotExist(err) {
		t.Fatalf("previous file still exists: %v", err)
	}
}
