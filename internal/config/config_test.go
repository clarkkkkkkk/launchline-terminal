package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

func TestVersionOneMigrationPreservesIdentityAndMembership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	legacy := `{
  "version": 1,
  "default_workspace_id": "ws_1",
  "compact_logo": true,
  "applications": [{"id":"app_1","name":"Editor","path":"/editor","arguments":["--new"]}],
  "workspaces": [{"id":"ws_1","name":"Development","applications":["app_1"]}]
}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewFileRepository(path).Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != app.CurrentSchemaVersion || cfg.DefaultWorkspaceID != "ws_1" || cfg.Applications[0].ID != "app_1" || cfg.Workspaces[0].Applications[0] != "app_1" || !cfg.CompactLogo {
		t.Fatalf("migration changed legacy data: %#v", cfg)
	}
}

func TestLaunchSettingsPersistenceAndLegacyDefaults(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			original := fmt.Sprintf(`{"version":%d,"applications":[{"id":"a","name":"Old","path":"old"}],"workspaces":[{"id":"w","name":"Work","applications":["a"]}],"default_workspace_id":"w"}`, version)
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			repo := NewFileRepository(path)
			cfg, err := repo.Load()
			if err != nil {
				t.Fatal(err)
			}
			if len(cfg.Applications[0].Arguments) != 0 || cfg.Applications[0].WorkingDirectory != "" {
				t.Fatalf("unsafe defaults: %#v", cfg)
			}
			cfg.Applications[0].Arguments = []string{".", "--profile", "Work Profile", "", `C:\Work\`}
			cfg.Applications[0].WorkingDirectory = filepath.Join(t.TempDir(), "not-created")
			if err := repo.Save(cfg); err != nil {
				t.Fatal(err)
			}
			first, _ := os.ReadFile(path)
			got, err := NewFileRepository(path).Load()
			if err != nil || !reflect.DeepEqual(got, cfg) {
				t.Fatalf("roundtrip: %#v, %v", got, err)
			}
			if err := repo.Save(got); err != nil {
				t.Fatal(err)
			}
			second, _ := os.ReadFile(path)
			if string(first) != string(second) {
				t.Fatal("serialization changed on second save")
			}
			got.Applications[0].Arguments = nil
			got.Applications[0].WorkingDirectory = ""
			if err := repo.Save(got); err != nil {
				t.Fatal(err)
			}
			cleared, err := repo.Load()
			if err != nil || len(cleared.Applications[0].Arguments) != 0 || cleared.Applications[0].WorkingDirectory != "" || cleared.Workspaces[0].Applications[0] != "a" {
				t.Fatalf("clear: %#v, %v", cleared, err)
			}
		})
	}
}

func TestInvalidLaunchSettingsJSONPreservesConfig(t *testing.T) {
	for _, field := range []string{`"arguments":"--flag"`, `"arguments":[42]`, `"working_directory":[]`} {
		path := filepath.Join(t.TempDir(), "config.json")
		original := `{"version":2,"applications":[{"id":"a","name":"App","path":"app",` + field + `}],"workspaces":[]}`
		if err := os.WriteFile(path, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := NewFileRepository(path).Load()
		var corrupt *CorruptError
		if !errors.As(err, &corrupt) {
			t.Fatalf("expected safe failure for %s: %v", field, err)
		}
		after, _ := os.ReadFile(path)
		if string(after) != original {
			t.Fatal("original changed")
		}
	}
}
