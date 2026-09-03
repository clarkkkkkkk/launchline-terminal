package app

import (
	"errors"
	"reflect"
	"testing"
)

type memoryRepo struct{ cfg Config }

func (m *memoryRepo) Load() (Config, error) { return cloneConfig(m.cfg), nil }
func (m *memoryRepo) Save(cfg Config) error { m.cfg = cloneConfig(cfg); return nil }
func (m *memoryRepo) Path() string          { return "memory.json" }

func cloneConfig(cfg Config) Config {
	out := cfg
	out.Applications = append([]Application(nil), cfg.Applications...)
	out.Workspaces = append([]Workspace(nil), cfg.Workspaces...)
	for i := range out.Applications {
		out.Applications[i].Arguments = append([]string(nil), out.Applications[i].Arguments...)
	}
	for i := range out.Workspaces {
		out.Workspaces[i].Applications = append([]string(nil), out.Workspaces[i].Applications...)
	}
	return out
}

func TestApplicationCRUDAndWorkspaceReferenceIntegrity(t *testing.T) {
	repo := &memoryRepo{cfg: DefaultConfig()}
	service := NewService(repo)
	first, err := service.AddApplication(Application{Name: "Editor", Path: "/editor", Arguments: []string{"--new"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddApplication(Application{Name: "editor", Path: "/other"}); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	workspace, err := service.CreateWorkspace(Workspace{Name: "Development", Applications: []string{first.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if repo.cfg.DefaultWorkspaceID != workspace.ID {
		t.Fatal("first workspace was not made default")
	}
	updated, err := service.UpdateApplication(first.ID, Application{Name: "Code", Path: "/code"})
	if err != nil || updated.ID != first.ID {
		t.Fatalf("identity not preserved: %#v %v", updated, err)
	}
	if err := service.DeleteApplication(first.ID); err != nil {
		t.Fatal(err)
	}
	if len(repo.cfg.Applications) != 0 || len(repo.cfg.Workspaces[0].Applications) != 0 {
		t.Fatalf("delete did not cascade references: %#v", repo.cfg)
	}
}

func TestWorkspaceCRUDAndDefaultBehavior(t *testing.T) {
	repo := &memoryRepo{cfg: DefaultConfig()}
	service := NewService(repo)
	a, _ := service.CreateWorkspace(Workspace{Name: "A"})
	b, _ := service.CreateWorkspace(Workspace{Name: "B"})
	if err := service.SetDefaultWorkspace(b.ID); err != nil {
		t.Fatal(err)
	}
	resolved, _, err := service.ResolveWorkspace("")
	if err != nil || resolved.ID != b.ID {
		t.Fatalf("default resolve: %#v %v", resolved, err)
	}
	updated, err := service.UpdateWorkspace(b.ID, Workspace{Name: "Build"})
	if err != nil || updated.ID != b.ID {
		t.Fatalf("workspace update: %#v %v", updated, err)
	}
	if err := service.DeleteWorkspace(b.ID); err != nil {
		t.Fatal(err)
	}
	if repo.cfg.DefaultWorkspaceID != a.ID {
		t.Fatalf("expected remaining workspace as default, got %q", repo.cfg.DefaultWorkspaceID)
	}
	if err := service.DeleteWorkspace(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.ResolveWorkspace(""); !errors.Is(err, ErrNoDefault) {
		t.Fatalf("expected ErrNoDefault, got %v", err)
	}
}

func TestWorkspaceRejectsDuplicateNameAndUnknownReference(t *testing.T) {
	repo := &memoryRepo{cfg: DefaultConfig()}
	service := NewService(repo)
	if _, err := service.CreateWorkspace(Workspace{Name: "Focus"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateWorkspace(Workspace{Name: "focus"}); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("expected duplicate workspace error, got %v", err)
	}
	if _, err := service.CreateWorkspace(Workspace{Name: "Broken", Applications: []string{"missing"}}); err == nil {
		t.Fatal("expected unknown application reference error")
	}
}

func TestArgumentsParsing(t *testing.T) {
	want := []string{"--profile", "Work Profile", "", `a\b`, `C:\Program Files\Editor`}
	got, err := ParseArguments(`--profile "Work Profile" '' a\\b "C:\Program Files\Editor"`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if _, err := ParseArguments(`"broken`); err == nil {
		t.Fatal("expected unclosed quote error")
	}
}

func TestDiscoveredApplicationLinkAndReconcilePreserveManualAndWorkspace(t *testing.T) {
	repo := &memoryRepo{cfg: DefaultConfig()}
	service := NewService(repo)
	manual, err := service.AddApplication(Application{Name: "Custom", Path: "/custom"})
	if err != nil {
		t.Fatal(err)
	}
	discovered, err := service.LinkDiscoveredApplication(Application{Name: "Cursor", Path: "/cursor", DiscoveryID: "discovered_cursor", Source: "fixture", Kind: "executable"})
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := service.CreateWorkspace(Workspace{Name: "Development", Applications: []string{manual.ID, discovered.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ReconcileDiscoveredApplications(map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	cfg, _ := service.Load()
	if cfg.Applications[0].Unavailable || !cfg.Applications[1].Unavailable || !reflect.DeepEqual(cfg.Workspaces[0].Applications, workspace.Applications) {
		t.Fatalf("reconcile damaged config: %#v", cfg)
	}
	if err := service.ReconcileDiscoveredCatalog(map[string]Application{"discovered_cursor": {Name: "Cursor", Path: "/updated/cursor", Arguments: []string{"--new"}, DiscoveryID: "discovered_cursor", Source: "updated", Kind: "executable"}}); err != nil {
		t.Fatal(err)
	}
	cfg, _ = service.Load()
	if cfg.Applications[1].Unavailable || cfg.Applications[1].Path != "/updated/cursor" || !reflect.DeepEqual(cfg.Applications[1].Arguments, []string{"--new"}) {
		t.Fatalf("catalog metadata was not refreshed: %#v", cfg.Applications[1])
	}
	refreshed, err := service.LinkDiscoveredApplication(Application{Name: "Cursor", Path: "/new/cursor", DiscoveryID: "discovered_cursor", Source: "fixture", Kind: "executable"})
	if err != nil || refreshed.ID != discovered.ID || refreshed.Unavailable {
		t.Fatalf("relink did not preserve ID: %#v %v", refreshed, err)
	}
}

func TestManualAndDiscoveredApplicationsWithSameDisplayNameCoexist(t *testing.T) {
	repo := &memoryRepo{cfg: DefaultConfig()}
	service := NewService(repo)
	manual, err := service.AddApplication(Application{Name: "Cursor", Path: "/custom/cursor"})
	if err != nil {
		t.Fatal(err)
	}
	discovered, err := service.LinkDiscoveredApplication(Application{Name: "Cursor", Path: "/usr/bin/cursor", DiscoveryID: "cursor-system"})
	if err != nil {
		t.Fatal(err)
	}
	if manual.ID == discovered.ID || discovered.Name != "Cursor (discovered)" || len(repo.cfg.Applications) != 2 {
		t.Fatalf("manual app was not preserved distinctly: %#v", repo.cfg.Applications)
	}
}
