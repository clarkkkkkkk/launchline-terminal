package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	launchassets "github.com/launchline/launchline/assets"
	"github.com/launchline/launchline/internal/app"
	"github.com/launchline/launchline/internal/discovery"
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

func submit(model *Model, value string) tea.Cmd {
	model.screen = dashboardScreen
	model.prompt.SetValue(value)
	_, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return command
}

func TestSlashCommandsDriveHybridScreens(t *testing.T) {
	model := newTestModel(t)
	for input, want := range map[string]screen{
		"/help":         helpScreen,
		"?":             helpScreen,
		"/applications": applicationsScreen,
		"/apps":         applicationsScreen,
		"/workspaces":   workspacesScreen,
		"/settings":     settingsScreen,
		"/add":          applicationFormScreen,
	} {
		submit(model, input)
		if model.screen != want {
			t.Fatalf("%s opened screen %d, want %d", input, model.screen, want)
		}
	}
	submit(model, "/version")
	if !strings.Contains(model.notice, "Launchline dev") {
		t.Fatalf("version notice=%q", model.notice)
	}
	submit(model, "/applicatons")
	if !strings.Contains(model.errMessage, "/applications") {
		t.Fatalf("suggestion=%q", model.errMessage)
	}
	submit(model, "/clear")
	if model.errMessage != "" || model.notice != "" {
		t.Fatal("clear did not reset session messages")
	}
	if command := submit(model, "/exit"); command == nil {
		t.Fatal("exit did not return a quit command")
	}
}

func TestPromptCompletionHistoryAndQuotedStart(t *testing.T) {
	model := newTestModel(t)
	model.cfg.Workspaces = []app.Workspace{{ID: "w", Name: "Mobile Development"}}
	model.prompt.SetValue("/app")
	model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if model.prompt.Value() != "/applications" {
		t.Fatalf("completion=%q", model.prompt.Value())
	}
	submit(model, "/help")
	model.screen = dashboardScreen
	model.Update(tea.KeyMsg{Type: tea.KeyUp})
	if model.prompt.Value() != "/help" {
		t.Fatalf("history=%q", model.prompt.Value())
	}
	submit(model, `/workspace "Mobile Development"`)
	if model.screen != workspaceFormScreen || model.wsForm.name.Value() != "Mobile Development" {
		t.Fatalf("workspace command did not focus editor: screen=%d name=%q", model.screen, model.wsForm.name.Value())
	}
	if command := submit(model, `/start "Mobile Development"`); command == nil || model.screen != launchingScreen {
		t.Fatalf("quoted start did not begin: screen=%d", model.screen)
	}
}

func TestPromptWorkspaceCompletionAdvancesOnRepeatedTab(t *testing.T) {
	model := newTestModel(t)
	model.cfg.Workspaces = []app.Workspace{{ID: "w", Name: "Development"}}
	model.prompt.SetValue("/work")
	model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if model.prompt.Value() != "/workspaces" {
		t.Fatalf("first completion=%q", model.prompt.Value())
	}
	model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if model.prompt.Value() != "/workspaces" {
		t.Fatalf("second completion=%q", model.prompt.Value())
	}
	model.prompt.SetValue("/workspace ")
	model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if model.prompt.Value() != "/workspace Development" {
		t.Fatalf("argument completion=%q", model.prompt.Value())
	}
}

func TestExistingWorkspaceOpensOnApplicationSelectionAndTogglesMembership(t *testing.T) {
	cfg := app.DefaultConfig()
	cfg.Applications = []app.Application{
		{ID: "alpha", Name: "Alpha", Path: "/alpha"},
		{ID: "beta", Name: "Beta", Path: "/beta"},
	}
	cfg.Workspaces = []app.Workspace{{ID: "work", Name: "Work", Applications: []string{"alpha"}}}
	cfg.DefaultWorkspaceID = "work"
	repo := &tuiRepo{cfg: cfg}
	service := app.NewService(repo)
	model, err := New(service, app.NewLaunchService(service, noLaunch{}))
	if err != nil {
		t.Fatal(err)
	}
	model.openWorkspaceForm(&model.cfg.Workspaces[0])
	if model.wsForm.stage != 1 {
		t.Fatal("existing workspace did not open on its application checklist")
	}
	model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	if model.wsForm.selected["alpha"] {
		t.Fatal("Space did not unselect the focused application")
	}
	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	if !model.wsForm.selected["beta"] {
		t.Fatal("Space did not select the second application")
	}
	model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := repo.cfg.Workspaces[0].Applications; !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Fatalf("saved membership=%#v", got)
	}
}

func applicationUXModel(t *testing.T) (*Model, *tuiRepo) {
	t.Helper()
	cfg := app.DefaultConfig()
	cfg.Applications = []app.Application{{ID: "manual", Name: "Alpha", Path: "/alpha", Arguments: []string{"--new"}}}
	repo := &tuiRepo{cfg: cfg}
	service := app.NewService(repo)
	model, err := New(service, app.NewLaunchService(service, noLaunch{}))
	if err != nil {
		t.Fatal(err)
	}
	model.catalog.Applications = discovery.Normalize([]discovery.Application{{Name: "Cursor", Target: "/usr/bin/cursor", Kind: discovery.KindExecutable, Source: "xdg-desktop", Platform: "linux"}})
	model.screen = applicationsScreen
	model.search.Blur()
	return model, repo
}

func TestApplicationsKeyboardActionsAreContextual(t *testing.T) {
	model, repo := applicationUXModel(t)
	_, _, footer := model.viewApplications()
	const wantFooter = "↑↓ Navigate   / Search   Enter Details   E Edit   D Delete   A Add   Esc Back"
	if footer != wantFooter {
		t.Fatalf("applications footer=%q", footer)
	}

	model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	if model.screen != applicationsScreen || len(repo.cfg.Applications) != 1 {
		t.Fatal("Space performed an action on the normal Applications screen")
	}

	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if model.screen != applicationDetailsScreen || model.appDetail.name != "Alpha" {
		t.Fatalf("Enter did not open details: screen=%d detail=%q", model.screen, model.appDetail.name)
	}
	_, body, footer := model.viewApplicationDetails()
	if footer != "E Edit   Esc Back" || !strings.Contains(body, "Manual · editable") || !strings.Contains(body, "/alpha") {
		t.Fatalf("manual details missing metadata or controls: footer=%q body=%q", footer, body)
	}
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if model.screen != applicationFormScreen || model.appForm.id != "manual" {
		t.Fatal("E did not open manual application editing")
	}

	model, _ = applicationUXModel(t)
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if model.screen != confirmScreen || model.confirm.id != "manual" {
		t.Fatal("D did not open deletion confirmation")
	}

	model, _ = applicationUXModel(t)
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if model.screen != applicationFormScreen || model.appForm.id != "" {
		t.Fatal("A did not open manual application registration")
	}

	model, _ = applicationUXModel(t)
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if model.screen != dashboardScreen {
		t.Fatal("Esc did not return from Applications")
	}
}

func TestDiscoveredApplicationDetailsAreReadOnly(t *testing.T) {
	model, _ := applicationUXModel(t)
	model.cursor = 1
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if model.screen != applicationDetailsScreen || model.appDetail.name != "Cursor" {
		t.Fatalf("discovered details did not open: screen=%d detail=%q", model.screen, model.appDetail.name)
	}
	_, body, footer := model.viewApplicationDetails()
	if footer != "Esc Back" || !strings.Contains(body, "xdg-desktop") || !strings.Contains(body, "linux") || !strings.Contains(body, "cannot be edited") {
		t.Fatalf("discovered details are unclear: footer=%q body=%q", footer, body)
	}
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if model.screen != applicationDetailsScreen {
		t.Fatal("E unexpectedly edited a discovered application")
	}
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if model.screen != applicationsScreen {
		t.Fatal("Esc did not return from application details")
	}
}

func TestApplicationsSearchHasExplicitFocusAndOwnsText(t *testing.T) {
	model, _ := applicationUXModel(t)
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !model.search.Focused() {
		t.Fatal("/ did not focus application search")
	}
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if model.search.Value() != "a d" {
		t.Fatalf("focused search did not own text input: %q", model.search.Value())
	}
	_, _, footer := model.viewApplications()
	if footer != "Type Search   ↑↓ Results   Enter Details   Esc Done" {
		t.Fatalf("search footer=%q", footer)
	}
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if model.search.Focused() || model.search.Value() != "" || model.screen != applicationsScreen {
		t.Fatal("first Esc did not clear and exit search")
	}
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if model.screen != dashboardScreen {
		t.Fatal("second Esc did not leave Applications")
	}
}

func TestApplicationsSearchResultsCanBeNavigatedAndOpened(t *testing.T) {
	model, _ := applicationUXModel(t)
	model.catalog.Applications = discovery.Normalize(append(model.catalog.Applications,
		discovery.Application{Name: "Chrome", Target: "/usr/bin/chrome", Kind: discovery.KindExecutable, Source: "fixture", Platform: "linux"},
	))

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if model.search.Focused() || model.cursor != 0 || model.search.Value() != "c" {
		t.Fatalf("Down did not move focus to the first filtered result: focused=%v cursor=%d query=%q", model.search.Focused(), model.cursor, model.search.Value())
	}
	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if model.cursor != 1 {
		t.Fatalf("Down did not navigate filtered results: cursor=%d", model.cursor)
	}
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if model.screen != applicationDetailsScreen || model.appDetail.name != "Cursor" {
		t.Fatalf("Enter did not open the highlighted filtered result: screen=%d detail=%q", model.screen, model.appDetail.name)
	}

	model, _ = applicationUXModel(t)
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, character := range "cursor" {
		model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{character}})
	}
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if model.screen != applicationDetailsScreen || model.appDetail.name != "Cursor" {
		t.Fatalf("Enter did not open the exact search result: screen=%d detail=%q", model.screen, model.appDetail.name)
	}
}

func TestWorkspaceSelectionCancelAndContextualHints(t *testing.T) {
	model, repo := applicationUXModel(t)
	repo.cfg.Workspaces = []app.Workspace{{ID: "work", Name: "Work", Applications: []string{"manual"}}}
	model.cfg = repo.cfg
	model.openWorkspaceForm(&model.cfg.Workspaces[0])
	_, _, footer := model.viewWorkspaceForm()
	if footer != "↑↓ Navigate   Space Toggle   Enter Save   Esc Cancel" {
		t.Fatalf("workspace selection footer=%q", footer)
	}
	model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if model.screen != workspacesScreen || !reflect.DeepEqual(repo.cfg.Workspaces[0].Applications, []string{"manual"}) {
		t.Fatal("Esc did not cancel workspace membership changes")
	}

	model.returnTo = applicationsScreen
	if help := model.viewHelp(); !strings.Contains(help, "Enter Details") || !strings.Contains(help, "E Edit") {
		t.Fatalf("applications help is not contextual: %q", help)
	}
	model.returnTo = workspaceFormScreen
	model.wsForm.stage = 1
	if help := model.viewHelp(); !strings.Contains(help, "Space Toggle") || !strings.Contains(help, "Enter Save") {
		t.Fatalf("workspace help is not contextual: %q", help)
	}
	model.confirm = confirmState{kind: "application", name: "Alpha"}
	_, _, confirmFooter := model.viewConfirm()
	if confirmFooter != "Y Confirm   N Cancel" {
		t.Fatalf("confirmation footer=%q", confirmFooter)
	}
}

func TestDiscoveryRefreshAndApplicationFiltering(t *testing.T) {
	repo := &tuiRepo{cfg: app.DefaultConfig()}
	service := app.NewService(repo)
	discoveryService := discovery.NewService(
		discovery.NewFileCatalogRepository(filepath.Join(t.TempDir(), "catalog.json")),
		discovery.DiscovererFunc(func(context.Context) ([]discovery.Application, error) {
			return []discovery.Application{{Name: "Cursor", Target: "/usr/bin/cursor", Kind: discovery.KindExecutable, Platform: "linux", Source: "fixture"}, {Name: "Cura", Target: "/usr/bin/cura", Kind: discovery.KindExecutable, Platform: "linux", Source: "fixture"}}, nil
		}),
	)
	model, err := NewWithDiscovery(service, app.NewLaunchService(service, noLaunch{}), discoveryService, "v0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	command := model.Init()
	if command == nil || !model.refreshing {
		t.Fatal("refresh did not start asynchronously")
	}
	message := command()
	model.Update(message)
	if model.refreshing || len(model.catalog.Applications) != 2 {
		t.Fatalf("refresh state: %#v", model.catalog)
	}
	if command := submit(model, "/refresh"); command == nil || !model.refreshing {
		t.Fatal("slash refresh did not start asynchronously")
	}
	submit(model, "/applications")
	model.search.SetValue("curs")
	choices := model.applicationChoices(model.search.Value())
	if len(choices) != 1 || choices[0].name != "Cursor" {
		t.Fatalf("filter=%#v", choices)
	}
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

func TestPopulatedDashboardAndCatalogStayWithinStandardTerminal(t *testing.T) {
	model := newTestModel(t)
	var applicationIDs []string
	for i := 0; i < 42; i++ {
		item := app.Application{ID: fmt.Sprintf("app_%d", i), Name: fmt.Sprintf("Application %02d", i), Path: fmt.Sprintf("/app/%d", i)}
		model.cfg.Applications = append(model.cfg.Applications, item)
		applicationIDs = append(applicationIDs, item.ID)
	}
	model.cfg.Workspaces = []app.Workspace{{ID: "work", Name: "Development", Applications: applicationIDs}}
	model.cfg.DefaultWorkspaceID = "work"
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	for _, target := range []screen{dashboardScreen, applicationsScreen} {
		model.screen = target
		view := model.View()
		if lines := len(strings.Split(view, "\n")); lines > model.height {
			t.Fatalf("screen %d height %d exceeds %d:\n%s", target, lines, model.height, view)
		}
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

func TestSecondaryScreensUseExactLogoAssetWhenItFits(t *testing.T) {
	model := newTestModel(t)
	model.screen = applicationsScreen
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 32})
	viewLines := strings.Split(ansi.Strip(model.View()), "\n")
	logoLines := strings.Split(launchassets.LaunchlineLogo(), "\n")
	if len(viewLines) < len(logoLines) {
		t.Fatalf("Applications does not contain the complete logo:\n%s", model.View())
	}
	for index, expected := range logoLines {
		if actual := strings.TrimRight(viewLines[index], " "); actual != expected {
			t.Fatalf("secondary logo line %d differs from assets/ascii/launchline.txt\ngot:  %q\nwant: %q", index+1, actual, expected)
		}
	}
}

func TestDashboardPromptHasResponsiveHorizontalRules(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 42, Height: 18}} {
		model := newTestModel(t)
		model.Update(size)
		lines := strings.Split(ansi.Strip(model.View()), "\n")
		rules := 0
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && strings.Trim(trimmed, "─") == "" {
				rules++
				if ansi.StringWidth(line) > size.Width {
					t.Fatalf("rule width exceeds terminal: %q", line)
				}
			}
		}
		if rules != 2 {
			t.Fatalf("expected two prompt rules at %dx%d, got %d:\n%s", size.Width, size.Height, rules, model.View())
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

	model.appDetail = applicationChoice{name: "Cursor", configured: &model.cfg.Applications[0]}
	model.screen = applicationDetailsScreen
	applicationDetailsView := model.View()
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
		"application details": func() string { return applicationDetailsView },
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
			if first := strings.TrimSpace(strings.Split(ansi.Strip(view), "\n")[0]); first != "LAUNCHLINE" {
				t.Fatalf("top brand=%q, want LAUNCHLINE:\n%s", first, view)
			}
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
