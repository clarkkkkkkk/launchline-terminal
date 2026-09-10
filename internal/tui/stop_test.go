package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/launchline/launchline/internal/app"
)

type stopTestLauncher struct{ noLaunch }

func (stopTestLauncher) LaunchInWorkspace(context.Context, string, app.Application) error { return nil }
func (stopTestLauncher) StopInWorkspace(context.Context, string, app.Application) (int, error) {
	return 1, nil
}

func TestStopPromptAndResponsiveResults(t *testing.T) {
	cfg := app.DefaultConfig()
	cfg.Applications = []app.Application{{ID: "a", Name: "Editor", Path: "editor"}}
	cfg.Workspaces = []app.Workspace{{ID: "w", Name: "My Work", Applications: []string{"a"}}}
	cfg.DefaultWorkspaceID = "w"
	service := app.NewService(&tuiRepo{cfg: cfg})
	model, err := New(service, app.NewLaunchService(service, stopTestLauncher{}))
	if err != nil {
		t.Fatal(err)
	}
	command := submit(model, `/stop "My Work"`)
	if command == nil || model.screen != stoppingScreen {
		t.Fatal("stop did not start asynchronously")
	}
	model.Update(command())
	if !model.stop.done || model.stop.summary.Requested() != 1 || !strings.Contains(model.View(), "close requested") {
		t.Fatalf("results: %s", model.View())
	}
	for _, size := range []tea.WindowSizeMsg{{Width: 28, Height: 14}, {Width: 46, Height: 18}, {Width: 80, Height: 24}, {Width: 120, Height: 32}} {
		model.Update(size)
		view := model.View()
		if len(strings.Split(view, "\n")) > size.Height {
			t.Fatalf("%+v vertical overflow:\n%s", size, view)
		}
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > size.Width {
				t.Fatalf("horizontal overflow: %s", line)
			}
		}
	}
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if model.screen != dashboardScreen {
		t.Fatal("Esc did not return to prompt")
	}
}
