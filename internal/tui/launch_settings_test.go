package tui

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/launchline/launchline/internal/app"
	"github.com/launchline/launchline/internal/config"
)

type settingsLauncher struct{ received chan app.Application }

func (l settingsLauncher) Launch(_ context.Context, a app.Application) error {
	l.received <- a
	return nil
}

func TestApplicationLaunchSettingsPersistentTUIWorkflow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	service := app.NewService(config.NewFileRepository(path))
	recorder := settingsLauncher{received: make(chan app.Application, 4)}
	launch := app.NewLaunchService(service, recorder)
	model, err := New(service, launch)
	if err != nil {
		t.Fatal(err)
	}
	submit(model, "/add")
	values := []string{"Cursor", "/usr/bin/cursor", `. --profile "Work Profile" "C:\\Work\\"`, t.TempDir()}
	for i, value := range values {
		model.appForm.fields[i].SetValue(value)
		model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	if model.screen != applicationsScreen || model.errMessage != "" {
		t.Fatalf("save failed: %s", model.errMessage)
	}
	cfg, _ := service.Load()
	saved := cfg.Applications[0]
	wantArgs := []string{".", "--profile", "Work Profile", `C:\Work\`}
	if !reflect.DeepEqual(saved.Arguments, wantArgs) || saved.WorkingDirectory != values[3] {
		t.Fatalf("saved: %#v", saved)
	}
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_, body, _ := model.viewApplicationDetails()
	if !strings.Contains(body, "Work Profile") || !strings.Contains(body, "Working directory") {
		t.Fatalf("details: %s", body)
	}
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	// Saving an unchanged edit must preserve every argument and workspace ID.
	for range values {
		model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	cfg, _ = service.Load()
	if !reflect.DeepEqual(saved, cfg.Applications[0]) {
		t.Fatalf("edit changed settings: %#v", cfg.Applications[0])
	}
	workspace, err := service.CreateWorkspace(app.Workspace{Name: "Work", Applications: []string{saved.ID}})
	if err != nil {
		t.Fatal(err)
	}
	// Recreate all services to simulate a restart, then launch through the shared service.
	restarted := app.NewService(config.NewFileRepository(path))
	summary, err := app.NewLaunchService(restarted, recorder).Start(context.Background(), workspace.ID)
	if err != nil || summary.Succeeded() != 1 {
		t.Fatalf("launch: %#v, %v", summary, err)
	}
	received := <-recorder.received
	if !reflect.DeepEqual(received.Arguments, wantArgs) || received.WorkingDirectory != values[3] {
		t.Fatalf("forwarded: %#v", received)
	}
	model.openApplicationForm(&saved)
	model.appForm.fields[2].SetValue(`"broken`)
	model.appForm.focus = 3
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if model.screen != applicationFormScreen || !strings.Contains(model.errMessage, "unclosed quote") || model.appForm.fields[2].Value() != `"broken` {
		t.Fatal("invalid input was not retained")
	}
	cfg, _ = service.Load()
	if !reflect.DeepEqual(cfg.Applications[0], saved) {
		t.Fatal("invalid form changed config")
	}
	model.appForm.fields[2].SetValue("")
	model.appForm.fields[3].SetValue("")
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cfg, _ = restarted.Load()
	if cfg.Applications[0].ID != saved.ID || len(cfg.Applications[0].Arguments) != 0 || cfg.Applications[0].WorkingDirectory != "" || cfg.Workspaces[0].Applications[0] != saved.ID {
		t.Fatalf("clear changed identity: %#v", cfg)
	}
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_, body, _ = model.viewApplicationDetails()
	if !strings.Contains(body, "None") || !strings.Contains(body, "Default") {
		t.Fatalf("empty details: %s", body)
	}
	model.openApplicationForm(&cfg.Applications[0])
	model.appForm.fields[2].SetValue("--discarded")
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	after, _ := service.Load()
	if !reflect.DeepEqual(cfg, after) {
		t.Fatal("cancel saved form")
	}
}

func TestLaunchSettingsFormResponsiveFocus(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 28, Height: 14}, {Width: 46, Height: 18}, {Width: 80, Height: 24}, {Width: 100, Height: 28}, {Width: 120, Height: 40}} {
		model := newTestModel(t)
		model.openApplicationForm(nil)
		model.Update(size) // Resize while the form is already open.
		for focus := range model.appForm.fields {
			model.appForm.focus = focus
			model.errMessage = "Arguments contain an unclosed quote"
			view := model.View()
			if len(strings.Split(view, "\n")) > size.Height {
				t.Fatalf("%+v focus %d overflows vertically:\n%s", size, focus, view)
			}
			for _, line := range strings.Split(view, "\n") {
				if ansi.StringWidth(line) > size.Width {
					t.Fatalf("%+v overflows horizontally: %s", size, line)
				}
			}
			labels := []string{"Name", "Executable", "Arguments", "Working directory"}
			if !strings.Contains(ansi.Strip(view), "● "+labels[focus]) {
				t.Fatalf("focused field missing: %s", view)
			}
		}
	}
}

func TestArgumentEditorDoesNotTruncatePersistedValues(t *testing.T) {
	model := newTestModel(t)
	args := []string{strings.Repeat(`\`, 3000)}
	saved, err := model.config.AddApplication(app.Application{Name: "Long arguments", Path: "editor", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	model.openApplicationForm(&saved)
	for range model.appForm.fields {
		model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	cfg, err := model.config.Load()
	if err != nil || !reflect.DeepEqual(cfg.Applications[0].Arguments, args) {
		t.Fatal("editing truncated stored arguments")
	}
}
