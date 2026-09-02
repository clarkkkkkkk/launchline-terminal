package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	if !strings.Contains(view, "LAUNCHLINE") || strings.Contains(view, "_        _") {
		t.Fatalf("unexpected narrow dashboard:\n%s", view)
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
