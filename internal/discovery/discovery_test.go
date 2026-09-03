package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNormalizeStableIDsAndExactTargetDeduplication(t *testing.T) {
	items := Normalize([]Application{
		{Name: " Cursor ", Target: " /usr/bin/cursor ", Kind: KindExecutable, Source: "path", Platform: "linux"},
		{Name: "Cursor", Target: "/usr/bin/cursor", Kind: KindExecutable, Source: "xdg-desktop", Platform: "linux"},
		{Name: "Cursor Remote", Target: "/opt/cursor", Kind: KindExecutable, Source: "xdg-desktop", Platform: "linux"},
	})
	if len(items) != 2 {
		t.Fatalf("got %d applications: %#v", len(items), items)
	}
	if items[0].ID != StableID("linux", KindExecutable, "/usr/bin/cursor", nil) || items[0].Source != "xdg-desktop" {
		t.Fatalf("unstable or lower-priority result: %#v", items[0])
	}
}

func TestDesktopEntryParsingAndExecPlaceholders(t *testing.T) {
	fixture := `[Desktop Entry]
Type=Application
Name=Cursor
Exec="/opt/Cursor Editor/cursor" --new-window %F --url=%u %%done
Hidden=false
NoDisplay=false
`
	entry, err := parseDesktopEntry(strings.NewReader(fixture))
	if err != nil || entry.Name != "Cursor" {
		t.Fatalf("entry: %#v %v", entry, err)
	}
	target, args, err := parseDesktopExec(entry.Exec)
	if err != nil {
		t.Fatal(err)
	}
	if target != "/opt/Cursor Editor/cursor" || !reflect.DeepEqual(args, []string{"--new-window", "--url=", "%done"}) {
		t.Fatalf("target=%q args=%#v", target, args)
	}
	for _, flag := range []string{"Hidden=true", "NoDisplay=true", "Terminal=true", "Type=Link"} {
		text := "[Desktop Entry]\nType=Application\nName=Hidden\nExec=hidden\n" + flag + "\n"
		if _, err := parseDesktopEntry(strings.NewReader(text)); err == nil {
			t.Fatalf("expected %s to be filtered", flag)
		}
	}
}

func TestCatalogRoundTripAndMalformedSafetyCopy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	repository := NewFileCatalogRepository(path)
	want := Catalog{Version: CatalogVersion, RefreshedAt: time.Unix(10, 0).UTC(), Applications: []Application{{Name: "Editor", Target: "/editor", Kind: KindExecutable, Platform: "linux", Source: "fixture"}}}
	if err := repository.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := repository.Load()
	if err != nil || len(got.Applications) != 1 || got.Applications[0].ID == "" {
		t.Fatalf("catalog=%#v err=%v", got, err)
	}
	if err := os.WriteFile(path, []byte(`{"version":1,broken`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = repository.Load()
	var corrupt *CatalogCorruptError
	if !errors.As(err, &corrupt) || corrupt.BackupPath == "" {
		t.Fatalf("expected corrupt safety copy, got %v", err)
	}
}

type memoryCatalog struct{ catalog Catalog }

func (m *memoryCatalog) Load() (Catalog, error) { return m.catalog, nil }
func (m *memoryCatalog) Save(c Catalog) error   { m.catalog = c; return nil }
func (m *memoryCatalog) Path() string           { return "catalog.json" }

func TestRefreshRetainsCacheOnPartialFailureAndRemovesOnCompleteRefresh(t *testing.T) {
	cached := Normalize([]Application{{Name: "Cached", Target: "/cached", Kind: KindExecutable, Platform: "linux"}})
	repository := &memoryCatalog{catalog: Catalog{Version: 1, Applications: cached}}
	service := NewService(repository, DiscovererFunc(func(context.Context) ([]Application, error) {
		return []Application{{Name: "New", Target: "/new", Kind: KindExecutable, Platform: "linux"}}, &PartialError{Errors: []error{errors.New("optional source failed")}}
	}))
	result, err := service.Refresh(context.Background())
	if err == nil || len(result.Catalog.Applications) != 2 {
		t.Fatalf("partial refresh did not retain cache: %#v %v", result, err)
	}
	service.discoverer = DiscovererFunc(func(context.Context) ([]Application, error) {
		return []Application{{Name: "New", Target: "/new", Kind: KindExecutable, Platform: "linux"}}, nil
	})
	result, err = service.Refresh(context.Background())
	if err != nil || len(result.Catalog.Applications) != 1 || result.Removed != 1 {
		t.Fatalf("complete refresh did not remove disappearance: %#v %v", result, err)
	}
}

func TestPlatformMetadataNormalizationFromFixtures(t *testing.T) {
	plist := []byte(`<?xml version="1.0"?><plist><dict><key>CFBundleName</key><string>Fallback</string><key>CFBundleDisplayName</key><string>Cursor</string></dict></plist>`)
	bundle := macBundleApplication("/Applications/Cursor.app", plist)
	if bundle.Name != "Cursor" || bundle.Kind != KindBundle || bundle.Platform != "darwin" {
		t.Fatalf("mac bundle fixture: %#v", bundle)
	}
	shortcut := windowsShortcutApplication(filepath.Join("fixtures", "Cursor.lnk"))
	if shortcut.Name != "Cursor" || shortcut.Kind != KindShortcut || shortcut.Platform != "windows" {
		t.Fatalf("Windows shortcut fixture: %#v", shortcut)
	}
	windows := Normalize([]Application{shortcut, shortcut})
	if len(windows) != 1 || windows[0].ID == "" {
		t.Fatalf("Windows fixture normalization: %#v", windows)
	}
}
