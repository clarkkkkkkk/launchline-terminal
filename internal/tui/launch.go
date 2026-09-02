package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/launchline/launchline/internal/app"
)

type launchStartedMsg struct {
	workspace app.Workspace
	apps      []app.Application
	stream    <-chan app.LaunchResult
	err       error
}

type launchResultMsg struct {
	result app.LaunchResult
	ok     bool
}

func (m *Model) updateLaunchSelect(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := sortedWorkspaces(m.cfg.Workspaces)
	switch key.String() {
	case "up", "k":
		m.cursor = moveCursor(m.cursor, len(items), -1)
	case "down", "j":
		m.cursor = moveCursor(m.cursor, len(items), 1)
	case "enter":
		if len(items) > 0 {
			return m, m.beginLaunch(items[m.cursor].ID)
		}
	case "esc":
		m.screen, m.cursor = dashboardScreen, 0
	}
	return m, nil
}

func (m *Model) viewLaunchSelect() (string, string, string) {
	items := sortedWorkspaces(m.cfg.Workspaces)
	if len(items) == 0 {
		body := m.description("Select one workspace to launch.") + "\n\n" + m.theme.Title.Render("No workspace configured.") + "\n" + m.description("Create your first workspace and group the applications you normally start together.") + "\n\n" + m.theme.Accent.Render(">") + " Create Workspace"
		return "Workspace Selection — Choose Workspace", body, "Esc Back   ? Help   Q Quit"
	}
	var body strings.Builder
	body.WriteString(m.description("Select one workspace to launch:") + "\n\n")
	start, end := visibleRange(len(items), m.cursor, m.height)
	for i := start; i < end; i++ {
		defaultMark := ""
		if items[i].ID == m.cfg.DefaultWorkspaceID {
			defaultMark = "  " + m.theme.Success.Render("✓ Default")
		}
		line := m.menuItem(i, items[i].Name, true) + "  " + m.theme.Muted.Render(fmt.Sprintf("%d apps", len(items[i].Applications))) + defaultMark
		body.WriteString(line + "\n")
	}
	return "Workspace Selection — Choose Workspace", body.String(), "↑↓ Move   Enter Launch   Esc Back"
}

func (m *Model) beginLaunch(reference string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.screen = launchingScreen
	m.launch = launchState{results: map[string]app.LaunchResult{}}
	return func() tea.Msg {
		workspace, applications, stream, err := m.launcher.Begin(ctx, reference)
		return launchStartedMsg{workspace: workspace, apps: applications, stream: stream, err: err}
	}
}

func waitLaunch(stream <-chan app.LaunchResult) tea.Cmd {
	return func() tea.Msg {
		result, ok := <-stream
		return launchResultMsg{result: result, ok: ok}
	}
}

func (m *Model) updateLaunching(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case launchStartedMsg:
		if msg.err != nil {
			m.errMessage = msg.err.Error()
			m.launch.done = true
			return m, nil
		}
		m.launch.workspace, m.launch.apps, m.launch.stream = msg.workspace, msg.apps, msg.stream
		return m, tea.Batch(m.spinner.Tick, waitLaunch(msg.stream))
	case launchResultMsg:
		if !msg.ok {
			m.launch.done = true
			m.cancel = nil
			return m, nil
		}
		m.launch.results[msg.result.Application.ID] = msg.result
		return m, waitLaunch(m.launch.stream)
	case spinner.TickMsg:
		if !m.launch.done {
			var command tea.Cmd
			m.spinner, command = m.spinner.Update(msg)
			return m, command
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.cancel != nil {
				m.cancel()
			}
			m.cancel = nil
			m.screen, m.cursor = dashboardScreen, 0
		case "enter":
			if m.launch.done {
				m.screen, m.cursor = dashboardScreen, 0
			}
		case "q":
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) viewLaunching() (string, string, string) {
	name := m.launch.workspace.Name
	if name == "" {
		name = "Workspace"
	}
	var body strings.Builder
	body.WriteString(m.description("Starting each configured application independently.") + "\n\n")
	if len(m.launch.apps) == 0 && !m.launch.done {
		body.WriteString(m.spinner.View() + " Resolving applications…\n")
	}
	for _, item := range m.launch.apps {
		result, complete := m.launch.results[item.ID]
		if !complete {
			body.WriteString(m.spinner.View() + " " + item.Name + "\n")
			continue
		}
		if result.Err != nil {
			body.WriteString(m.theme.Error.Render("× "+item.Name) + "\n")
			body.WriteString("  " + m.theme.Muted.Render(truncate(result.Err.Error(), max(15, m.width-10))) + "\n")
		} else {
			body.WriteString(m.theme.Success.Render("✓ "+item.Name) + "\n")
		}
	}
	if m.launch.done {
		succeeded, failed := 0, 0
		for _, result := range m.launch.results {
			if result.Err == nil {
				succeeded++
			} else {
				failed++
			}
		}
		body.WriteString("\n" + m.theme.Title.Render("Workspace launched.") + "\n\n")
		body.WriteString(fmt.Sprintf("%d applications launched successfully.", succeeded))
		if failed > 0 {
			body.WriteString("\n" + m.theme.Error.Render(fmt.Sprintf("%d applications failed.", failed)))
		}
		if len(m.launch.apps) == 0 && m.errMessage == "" {
			body.WriteString("\n" + m.theme.Warning.Render("This workspace has no applications yet."))
		}
		return "Launch Workspace — " + name, body.String(), "Enter/Esc Workspace   Q Quit"
	}
	return "Launch Workspace — " + name, body.String(), "Launching   Esc Cancel   Q Quit"
}
