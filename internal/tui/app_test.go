package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	launchassets "github.com/launchline/launchline/assets"
	"github.com/launchline/launchline/internal/app"
)

type tuiRepo struct{ cfg app.Config }

func (r *tuiRepo) Load() (app.Config, error) { return r.cfg, nil }
func (r *tuiRepo) Save(cfg app.Config) error { r.cfg = cfg; return nil }
func (r *tuiRepo) Path() string              { return "/tmp/launchline/config.json" }

type noLaunch struct{}

func (noLaunch) Launch(context.Context, app.Application) error { return nil }

func newTestModel(t *testing.T) *Model {
	t.Helper()
	service := app.NewService(&tuiRepo{cfg: app.DefaultConfig()})
	model, err := New(service, app.NewLaunchService(service, noLaunch{}))
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func TestDashboardUsesCompactBrandOnNarrowTerminal(t *testing.T) {
	model := newTestModel(t)
	model.Update(tea.WindowSizeMsg{Width: 42, Height: 18})
	view := model.View()
	if !strings.Contains(view, "LAUNCHLINE") || strings.Contains(view, "██╗") {
		t.Fatalf("unexpected narrow dashboard:\n%s", view)
	}
}

func TestDashboardResponsiveLayoutModes(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		fullLogo bool
	}{
		{name: "wide", width: 120, height: 32, fullLogo: true},
		{name: "normal", width: 90, height: 27, fullLogo: true},
		{name: "standard 80x24", width: 80, height: 24, fullLogo: true},
		{name: "narrow", width: 46, height: 18, fullLogo: false},
		{name: "short", width: 100, height: 18, fullLogo: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := newTestModel(t)
			model.Update(tea.WindowSizeMsg{Width: test.width, Height: test.height})
			view := model.View()
			hasFullLogo := strings.Contains(view, strings.Split(launchassets.LaunchlineLogo(), "\n")[0])
			if hasFullLogo != test.fullLogo {
				t.Fatalf("full logo = %v, want %v:\n%s", hasFullLogo, test.fullLogo, view)
			}
			for _, line := range strings.Split(view, "\n") {
				if width := ansi.StringWidth(line); width > test.width {
					t.Fatalf("line width %d exceeds terminal width %d: %q", width, test.width, line)
				}
			}
			if lines := len(strings.Split(view, "\n")); lines > test.height {
				t.Fatalf("view height %d exceeds terminal height %d:\n%s", lines, test.height, view)
			}
			if !strings.Contains(view, "> launchline") || !strings.Contains(view, "Launchline dev") {
				t.Fatalf("missing command/status context:\n%s", view)
			}
		})
	}
}

func TestStandardTerminalUsesExactLogoAsset(t *testing.T) {
	model := newTestModel(t)
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	viewLines := strings.Split(ansi.Strip(model.View()), "\n")
	logoLines := strings.Split(launchassets.LaunchlineLogo(), "\n")
	if len(viewLines) < len(logoLines) {
		t.Fatalf("dashboard does not contain the complete logo:\n%s", model.View())
	}
	for index, expected := range logoLines {
		if actual := strings.TrimRight(viewLines[index], " "); actual != expected {
			t.Fatalf("logo line %d differs from assets/ascii/launchline.txt\ngot:  %q\nwant: %q", index+1, actual, expected)
		}
	}
}

func TestTextInputOwnsQShortcut(t *testing.T) {
	model := newTestModel(t)
	model.openApplicationForm(nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	got := updated.(*Model)
	if got.appForm.fields[0].Value() != "q" || got.screen != applicationFormScreen {
		t.Fatalf("q did not remain in text input: value=%q screen=%d", got.appForm.fields[0].Value(), got.screen)
	}
}

func TestEveryScreenFitsNarrowTerminal(t *testing.T) {
	model := newTestModel(t)
	model.cfg.Applications = []app.Application{
		{ID: "a", Name: "Cursor", Path: "/usr/local/bin/cursor"},
		{ID: "b", Name: "Chrome", Path: "/usr/local/bin/google-chrome"},
		{ID: "c", Name: "Spotify", Path: "/usr/local/bin/spotify"},
	}
	model.cfg.Workspaces = []app.Workspace{{ID: "w", Name: "Development", Applications: []string{"a", "b", "c"}}}
	model.cfg.DefaultWorkspaceID = "w"
	model.Update(tea.WindowSizeMsg{Width: 46, Height: 18})

	model.openApplicationForm(&model.cfg.Applications[0])
	applicationFormView := model.View()
	model.openWorkspaceForm(&model.cfg.Workspaces[0])
	workspaceNameView := model.View()
	model.wsForm.stage = 1
	workspaceSelectionView := model.View()
	model.launch = launchState{
		workspace: model.cfg.Workspaces[0],
		apps:      model.cfg.Applications,
		results: map[string]app.LaunchResult{
			"a": {Application: model.cfg.Applications[0]},
			"b": {Application: model.cfg.Applications[1]},
			"c": {Application: model.cfg.Applications[2], Err: context.DeadlineExceeded},
		},
		done: true,
	}
	model.confirm = confirmState{kind: "application", name: "Cursor"}

	views := map[string]func() string{
		"dashboard":           func() string { model.screen = dashboardScreen; return model.View() },
		"applications":        func() string { model.screen = applicationsScreen; return model.View() },
		"application form":    func() string { return applicationFormView },
		"workspaces":          func() string { model.screen = workspacesScreen; return model.View() },
		"workspace name":      func() string { return workspaceNameView },
		"workspace selection": func() string { return workspaceSelectionView },
		"launch selection":    func() string { model.screen = launchSelectScreen; return model.View() },
		"launch result":       func() string { model.screen = launchingScreen; return model.View() },
		"settings":            func() string { model.screen = settingsScreen; return model.View() },
		"help":                func() string { model.screen = helpScreen; return model.View() },
		"confirmation":        func() string { model.screen = confirmScreen; return model.View() },
	}
	for name, render := range views {
		t.Run(name, func(t *testing.T) {
			view := render()
			for _, line := range strings.Split(view, "\n") {
				if width := ansi.StringWidth(line); width > model.width {
					t.Fatalf("line width %d exceeds terminal width %d: %q", width, model.width, line)
				}
			}
			if lines := len(strings.Split(view, "\n")); lines > model.height {
				t.Fatalf("view height %d exceeds terminal height %d:\n%s", lines, model.height, view)
			}
		})
	}
}

func TestTinyTerminalShowsResizeStateWithoutOverflow(t *testing.T) {
	model := newTestModel(t)
	model.Update(tea.WindowSizeMsg{Width: 24, Height: 10})
	view := model.View()
	if !strings.Contains(view, "Terminal too small") || !strings.Contains(view, "Resize to continue") {
		t.Fatalf("missing resize guidance:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > model.width {
			t.Fatalf("line width %d exceeds terminal width %d: %q", width, model.width, line)
		}
	}
	if lines := len(strings.Split(view, "\n")); lines > model.height {
		t.Fatalf("view height %d exceeds terminal height %d:\n%s", lines, model.height, view)
	}
}
